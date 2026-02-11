package agent

import (
	"math/rand"
	"runtime"
	"sync"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
)

// MetricValue представляет значение метрики
type MetricValue struct {
	Type  string
	Name  string
	Value interface{}
}

// Collector собирает метрики из runtime
type Collector struct {
	mu         sync.Mutex
	pollCount  int64
	randomSeed *rand.Rand
}

// NewCollector создает новый сборщик метрик
func NewCollector() *Collector {
	return &Collector{
		randomSeed: rand.New(rand.NewSource(rand.Int63())),
	}
}

// Collect собирает все метрики из runtime и дополнительные метрики
func (c *Collector) Collect() []MetricValue {
	c.mu.Lock()
	c.pollCount++
	pollCount := c.pollCount
	randomValue := c.randomSeed.Float64()
	c.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := make([]MetricValue, 0, 30)

	// Чтобы получить актуальную информацию о состоянии памяти и производительности
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "Alloc", Value: float64(m.Alloc)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "BuckHashSys", Value: float64(m.BuckHashSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "Frees", Value: float64(m.Frees)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "GCCPUFraction", Value: m.GCCPUFraction})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "GCSys", Value: float64(m.GCSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "HeapAlloc", Value: float64(m.HeapAlloc)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "HeapIdle", Value: float64(m.HeapIdle)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "HeapInuse", Value: float64(m.HeapInuse)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "HeapObjects", Value: float64(m.HeapObjects)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "HeapReleased", Value: float64(m.HeapReleased)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "HeapSys", Value: float64(m.HeapSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "LastGC", Value: float64(m.LastGC)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "Lookups", Value: float64(m.Lookups)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "MCacheInuse", Value: float64(m.MCacheInuse)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "MCacheSys", Value: float64(m.MCacheSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "MSpanInuse", Value: float64(m.MSpanInuse)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "MSpanSys", Value: float64(m.MSpanSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "Mallocs", Value: float64(m.Mallocs)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "NextGC", Value: float64(m.NextGC)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "NumForcedGC", Value: float64(m.NumForcedGC)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "NumGC", Value: float64(m.NumGC)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "OtherSys", Value: float64(m.OtherSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "PauseTotalNs", Value: float64(m.PauseTotalNs)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "StackInuse", Value: float64(m.StackInuse)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "StackSys", Value: float64(m.StackSys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "Sys", Value: float64(m.Sys)})
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "TotalAlloc", Value: float64(m.TotalAlloc)})

	// Чтобы отслеживать количество опросов и иметь тестовую метрику
	metrics = append(metrics, MetricValue{Type: model.Counter, Name: "PollCount", Value: pollCount})

	// Чтобы иметь тестовую метрику для проверки работы системы
	metrics = append(metrics, MetricValue{Type: model.Gauge, Name: "RandomValue", Value: randomValue})

	return metrics
}

// PollCountSnapshot возвращает текущее значение pollCount.
// Используется репортером для формирования payload (до отправки).
func (c *Collector) PollCountSnapshot() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pollCount
}

// AckPollCount "подтверждает" отправленное значение PollCount:
// вычитает sent из pollCount, не затрагивая новые инкременты, которые могли
// произойти параллельно с отправкой.
func (c *Collector) AckPollCount(sent int64) {
	if sent <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if sent >= c.pollCount {
		c.pollCount = 0
		return
	}
	c.pollCount -= sent
}

