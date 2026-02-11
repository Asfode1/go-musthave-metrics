package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
	"github.com/Asfode1/go-musthave-metrics/internal/storage"
)

// MetricsHandler обработчик для работы с метриками
type MetricsHandler struct {
	storage *storage.MemStorage
	saveFn  func() error
}

type httpError struct {
	status int
	msg    string
}

func (e httpError) Error() string {
	return e.msg
}

// NewMetricsHandler создает новый обработчик метрик
func NewMetricsHandler(storage *storage.MemStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

// NewMetricsHandlerWithSaver создает новый обработчик и настраивает синхронное сохранение.
// saveFn вызывается после успешного обновления метрики (если не nil).
func NewMetricsHandlerWithSaver(storage *storage.MemStorage, saveFn func() error) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
		saveFn:  saveFn,
	}
}

// UpdateJSON обрабатывает POST /update, принимает метрику в JSON и сохраняет её.
func (h *MetricsHandler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r.Header.Get("Content-Type")) && r.Header.Get("Content-Type") != "" {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var m model.Metrics
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := validateMetric(m); err != nil {
		http.Error(w, err.msg, err.status)
		return
	}
	m = h.applyMetric(m)

	if h.saveFn != nil {
		if err := h.saveFn(); err != nil {
			http.Error(w, "Failed to persist metrics", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(m)
}

// UpdateBatchJSON обрабатывает POST /updates, принимает массив метрик в JSON и сохраняет их.
func (h *MetricsHandler) UpdateBatchJSON(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r.Header.Get("Content-Type")) && r.Header.Get("Content-Type") != "" {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var metrics []model.Metrics
	if err := json.Unmarshal(body, &metrics); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	for _, m := range metrics {
		if err := validateMetric(m); err != nil {
			http.Error(w, err.msg, err.status)
			return
		}
	}

	for i, m := range metrics {
		metrics[i] = h.applyMetric(m)
	}

	if h.saveFn != nil {
		if err := h.saveFn(); err != nil {
			http.Error(w, "Failed to persist metrics", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(metrics)
}

// Value обрабатывает GET /value/<ТИП>/<ИМЯ>
func (h *MetricsHandler) Value(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/value/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	metricType := strings.TrimSpace(parts[0])
	metricName := strings.TrimSpace(parts[1])
	if metricName == "" {
		http.Error(w, "Metric name is required", http.StatusNotFound)
		return
	}

	var out string
	switch metricType {
	case model.Counter:
		value, ok := h.storage.GetCounter(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		out = strconv.FormatInt(value, 10)
	case model.Gauge:
		value, ok := h.storage.GetGauge(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		out = strconv.FormatFloat(value, 'f', -1, 64)
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}

// ValueJSON обрабатывает POST /value, возвращает значение метрики по JSON-запросу.
func (h *MetricsHandler) ValueJSON(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r.Header.Get("Content-Type")) && r.Header.Get("Content-Type") != "" {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var req model.Metrics
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "Metric id is required", http.StatusNotFound)
		return
	}

	resp := model.Metrics{ID: req.ID, MType: req.MType}
	switch req.MType {
	case model.Counter:
		v, ok := h.storage.GetCounter(req.ID)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		resp.Delta = &v

	case model.Gauge:
		v, ok := h.storage.GetGauge(req.ID)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		resp.Value = &v

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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

	metricType := strings.TrimSpace(parts[0])
	metricName := strings.TrimSpace(parts[1])
	metricValue := strings.TrimSpace(parts[2])

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
		h.storage.UpdateCounter(metricName, value)

	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metricName, value)

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if h.saveFn != nil {
		if err := h.saveFn(); err != nil {
			http.Error(w, "Failed to persist metrics", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func isJSONContentType(v string) bool {
	// допускаем параметры типа charset
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(v)), "application/json")
}

func validateMetric(m model.Metrics) *httpError {
	if strings.TrimSpace(m.ID) == "" {
		return &httpError{status: http.StatusNotFound, msg: "Metric id is required"}
	}
	switch m.MType {
	case model.Counter:
		if m.Delta == nil {
			return &httpError{status: http.StatusBadRequest, msg: "Delta is required for counter"}
		}
	case model.Gauge:
		if m.Value == nil {
			return &httpError{status: http.StatusBadRequest, msg: "Value is required for gauge"}
		}
	default:
		return &httpError{status: http.StatusBadRequest, msg: "Invalid metric type"}
	}
	return nil
}

func (h *MetricsHandler) applyMetric(m model.Metrics) model.Metrics {
	switch m.MType {
	case model.Counter:
		h.storage.UpdateCounter(m.ID, *m.Delta)
		if v, ok := h.storage.GetCounter(m.ID); ok {
			m.Delta = &v
		}
	case model.Gauge:
		h.storage.UpdateGauge(m.ID, *m.Value)
		if v, ok := h.storage.GetGauge(m.ID); ok {
			m.Value = &v
		}
	}
	return m
}
