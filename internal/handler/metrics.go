package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
	"github.com/Asfode1/go-musthave-metrics/internal/storage"
)

// MetricsHandler обработчик для работы с метриками
type MetricsHandler struct {
	storage *storage.MemStorage
}

// NewMetricsHandler создает новый обработчик метрик
func NewMetricsHandler(storage *storage.MemStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

// Update обрабатывает POST /update/<ТИП>/<ИМЯ>/<ЗНАЧЕНИЕ>
func (h *MetricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Парсим путь: /update/<ТИП>/<ИМЯ>/<ЗНАЧЕНИЕ>
	path := strings.TrimPrefix(r.URL.Path, "/update/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")

	// Проверяем количество частей пути
	if len(parts) != 3 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	metricType := parts[0]
	metricName := parts[1]
	metricValue := parts[2]

	// Проверяем наличие имени метрики (должно быть до парсинга значения)
	if metricName == "" {
		http.Error(w, "Metric name is required", http.StatusNotFound)
		return
	}

	// Проверяем наличие значения
	if metricValue == "" {
		http.Error(w, "Metric value is required", http.StatusBadRequest)
		return
	}

	// Обрабатываем в зависимости от типа метрики
	switch metricType {
	case model.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value", http.StatusBadRequest)
			return
		}
		// PollCount представляет абсолютное значение количества сборов метрик,
		// поэтому используем SetCounter вместо UpdateCounter для корректной работы при перезапуске агента
		if metricName == "PollCount" {
			h.storage.SetCounter(metricName, value)
		} else {
			h.storage.UpdateCounter(metricName, value)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}
}
