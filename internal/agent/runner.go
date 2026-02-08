package agent

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/Asfode1/go-musthave-metrics/internal/config"
	"github.com/Asfode1/go-musthave-metrics/internal/model"
)

// Runner управляет сбором и отправкой метрик
type Runner struct {
	collector *Collector
	client    *Client
	config    *RunnerConfig
}

// RunnerConfig конфигурация для Runner
type RunnerConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerURL      string
}

// NewRunner создает новый Runner
func NewRunner(config *RunnerConfig) *Runner {
	return &Runner{
		collector: NewCollector(),
		client:    NewClient(config.ServerURL),
		config:    config,
	}
}

// NewRunnerFromAgentConfig создает Runner из конфигурации агента
func NewRunnerFromAgentConfig(cfg *config.AgentConfig) *Runner {
	return NewRunner(&RunnerConfig{
		ServerURL:      cfg.ServerURL,
		PollInterval:   cfg.PollInterval,
		ReportInterval: cfg.ReportInterval,
	})
}

// Start запускает сбор и отправку метрик
func (r *Runner) Start() {
	var (
		metricsMu sync.RWMutex
		metrics   []MetricValue
	)

	// Чтобы собирать метрики с заданной периодичностью независимо от отправки
	go func() {
		ticker := time.NewTicker(r.config.PollInterval)
		defer ticker.Stop()

		for range ticker.C {
			newMetrics := r.collector.Collect()
			// PollCount будет рассчитываться и подтверждаться (ack) в репортере,
			// чтобы избежать потерь инкрементов при отправке.
			newMetrics = withoutMetricName(newMetrics, "PollCount")
			metricsMu.Lock()
			metrics = newMetrics
			metricsMu.Unlock()
		}
	}()

	// Чтобы отправлять метрики на сервер с заданной периодичностью независимо от сбора
	go func() {
		ticker := time.NewTicker(r.config.ReportInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Чтобы использовать актуальные метрики для отправки
			metricsMu.RLock()
			currentMetrics := make([]MetricValue, len(metrics))
			copy(currentMetrics, metrics)
			metricsMu.RUnlock()

			if err := r.reportOnce(currentMetrics); err != nil {
				log.Printf("Failed to send metrics: %v", err)
			} else {
				log.Printf("Successfully sent metrics")
			}
		}
	}()

	log.Println("Agent started")
	log.Printf("Poll interval: %v", r.config.PollInterval)
	log.Printf("Report interval: %v", r.config.ReportInterval)
	log.Printf("Server URL: %s", r.config.ServerURL)
}

// reportOnce отправляет метрики и отдельно подтверждает PollCount.
//
// Критично: PollCount подтверждается (ack) по факту успешной отправки самого PollCount,
// а не только при полном успехе отправки всех метрик. Иначе при частичном успехе
// PollCount может быть принят сервером, но не будет ack на стороне агента, что приведет
// к повторной отправке того же delta и двойному учету на сервере (counter += delta).
func (r *Runner) reportOnce(metrics []MetricValue) error {
	// На всякий случай гарантируем, что PollCount не уйдет в общем батче,
	// иначе он может отправиться дважды (в SendMetrics и отдельным запросом ниже).
	metrics = withoutMetricName(metrics, "PollCount")

	// Сначала отправляем все метрики кроме PollCount. Ошибки отдельных метрик не мешают
	// попытаться доставить остальные (SendMetrics продолжает отправку).
	errOther := r.client.SendMetrics(metrics)

	// Затем отправляем PollCount отдельно и делаем ack только если он действительно доставлен.
	pollCount := r.collector.PollCountSnapshot()
	var errPoll error
	if pollCount > 0 {
		errPoll = r.client.SendMetric(MetricValue{
			Type:  model.Counter,
			Name:  "PollCount",
			Value: pollCount,
		})
		if errPoll == nil {
			r.collector.AckPollCount(pollCount)
		}
	}

	if errOther == nil && errPoll == nil {
		return nil
	}
	return errors.Join(errOther, errPoll)
}

func withoutMetricName(in []MetricValue, name string) []MetricValue {
	if len(in) == 0 {
		return in
	}
	out := make([]MetricValue, 0, len(in))
	for _, m := range in {
		if m.Name == name {
			continue
		}
		out = append(out, m)
	}
	return out
}
