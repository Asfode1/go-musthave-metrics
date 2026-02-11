package agent

import (
	"errors"
	"log"
	"sync/atomic"
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
		metricsStore atomic.Value // []MetricValue
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
			metricsStore.Store(newMetrics)
		}
	}()

	// Чтобы отправлять метрики на сервер с заданной периодичностью независимо от сбора
	go func() {
		ticker := time.NewTicker(r.config.ReportInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Чтобы использовать актуальные метрики для отправки
			currentMetrics := loadMetricsSnapshot(&metricsStore)

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

	// Сначала отправляем все метрики кроме PollCount.
	// Пытаемся батч-отправку, а при ошибке — fallback на поштучную отправку.
	errOther := r.client.SendMetricsBatch(metrics)
	if errOther != nil {
		if errFallback := r.client.SendMetrics(metrics); errFallback == nil {
			errOther = nil
		} else {
			errOther = errors.Join(errOther, errFallback)
		}
	}

	// Затем отправляем PollCount отдельно и делаем ack только если он действительно доставлен.
	pollCount := r.collector.PollCountSnapshot()
	var errPoll error
	if pollCount > 0 {
		errPoll = sendMetricWithRetry(r.client, MetricValue{
			Type:  model.Counter,
			Name:  "PollCount",
			Value: pollCount,
		}, 3, 50*time.Millisecond)
		if errPoll == nil {
			r.collector.AckPollCount(pollCount)
		}
	}

	if errOther == nil && errPoll == nil {
		return nil
	}
	return errors.Join(errOther, errPoll)
}

func loadMetricsSnapshot(store *atomic.Value) []MetricValue {
	if store == nil {
		return nil
	}
	if v := store.Load(); v != nil {
		if metrics, ok := v.([]MetricValue); ok {
			out := make([]MetricValue, len(metrics))
			copy(out, metrics)
			return out
		}
	}
	return nil
}

func sendMetricWithRetry(client *Client, metric MetricValue, attempts int, baseDelay time.Duration) error {
	if client == nil {
		return errors.New("client is nil")
	}
	if attempts <= 0 {
		attempts = 1
	}
	if baseDelay <= 0 {
		baseDelay = 10 * time.Millisecond
	}

	var err error
	delay := baseDelay
	for i := 0; i < attempts; i++ {
		err = client.SendMetric(metric)
		if err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		time.Sleep(delay)
		delay *= 2
	}
	return err
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
