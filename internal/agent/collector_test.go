package agent

import (
	"testing"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
)

func TestCollector_Collect(t *testing.T) {
	collector := NewCollector()
	metrics := collector.Collect()

	if len(metrics) == 0 {
		t.Fatal("Expected at least one metric")
	}

	// Проверяем наличие обязательных метрик
	hasPollCount := false
	hasRandomValue := false
	hasAlloc := false

	for _, metric := range metrics {
		if metric.Name == "PollCount" && metric.Type == model.Counter {
			hasPollCount = true
			if val, ok := metric.Value.(int64); !ok || val != 1 {
				t.Errorf("Expected PollCount to be 1, got %v", metric.Value)
			}
		}
		if metric.Name == "RandomValue" && metric.Type == model.Gauge {
			hasRandomValue = true
		}
		if metric.Name == "Alloc" && metric.Type == model.Gauge {
			hasAlloc = true
		}
	}

	if !hasPollCount {
		t.Error("PollCount metric not found")
	}
	if !hasRandomValue {
		t.Error("RandomValue metric not found")
	}
	if !hasAlloc {
		t.Error("Alloc metric not found")
	}
}

func TestCollector_GetPollCount(t *testing.T) {
	collector := NewCollector()

	// Первый вызов Collect должен увеличить счетчик до 1
	collector.Collect()
	if collector.GetPollCount() != 1 {
		t.Errorf("Expected pollCount to be 1, got %d", collector.GetPollCount())
	}

	// Второй вызов должен увеличить до 2
	collector.Collect()
	if collector.GetPollCount() != 2 {
		t.Errorf("Expected pollCount to be 2, got %d", collector.GetPollCount())
	}
}

func TestCollector_Collect_AllMetrics(t *testing.T) {
	collector := NewCollector()
	metrics := collector.Collect()

	expectedMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "PollCount", "RandomValue",
	}

	metricMap := make(map[string]bool)
	for _, metric := range metrics {
		metricMap[metric.Name] = true
	}

	for _, expected := range expectedMetrics {
		if !metricMap[expected] {
			t.Errorf("Expected metric %s not found", expected)
		}
	}
}
