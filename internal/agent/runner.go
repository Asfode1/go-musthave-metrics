package agent

import (
	"log"
	"sync"
	"time"

	"github.com/Asfode1/go-musthave-metrics/internal/config"
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

			// Чтобы сервер получил актуальные данные о состоянии системы
			if err := r.client.SendMetrics(currentMetrics); err != nil {
				log.Printf("Failed to send metrics: %v", err)
			} else {
				log.Printf("Successfully sent %d metrics", len(currentMetrics))
			}
		}
	}()

	log.Println("Agent started")
	log.Printf("Poll interval: %v", r.config.PollInterval)
	log.Printf("Report interval: %v", r.config.ReportInterval)
	log.Printf("Server URL: %s", r.config.ServerURL)
}
