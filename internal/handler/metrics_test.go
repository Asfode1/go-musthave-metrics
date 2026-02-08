package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
	"github.com/Asfode1/go-musthave-metrics/internal/mw"
	"github.com/Asfode1/go-musthave-metrics/internal/storage"
)

func TestMetricsHandler_Update_Counter(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/testCounter/42", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	value, ok := memStorage.GetCounter("testCounter")
	if !ok {
		t.Fatal("Counter should exist")
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
}

func TestMetricsHandler_Update_Gauge(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/testGauge/3.14", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	value, ok := memStorage.GetGauge("testGauge")
	if !ok {
		t.Fatal("Gauge should exist")
	}
	if value != 3.14 {
		t.Errorf("Expected 3.14, got %f", value)
	}
}

func TestMetricsHandler_Update_InvalidPath(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/test", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMetricsHandler_Update_EmptyMetricName(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/counter//42", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestMetricsHandler_Update_InvalidCounterValue(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/test/invalid", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMetricsHandler_Update_InvalidGaugeValue(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/invalid", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMetricsHandler_Update_InvalidMetricType(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	req := httptest.NewRequest(http.MethodPost, "/update/invalid/test/42", nil)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMetricsHandler_Update_CounterIncrement(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	// Первое обновление
	req1 := httptest.NewRequest(http.MethodPost, "/update/counter/testCounter/10", nil)
	w1 := httptest.NewRecorder()
	handler.Update(w1, req1)

	// Второе обновление (должно добавиться)
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/testCounter/5", nil)
	w2 := httptest.NewRecorder()
	handler.Update(w2, req2)

	value, ok := memStorage.GetCounter("testCounter")
	if !ok {
		t.Fatal("Counter should exist")
	}
	if value != 15 {
		t.Errorf("Expected 15 (10+5), got %d", value)
	}
}

func TestMetricsHandler_Update_GaugeReplace(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	// Первое обновление
	req1 := httptest.NewRequest(http.MethodPost, "/update/gauge/testGauge/3.14", nil)
	w1 := httptest.NewRecorder()
	handler.Update(w1, req1)

	// Второе обновление (должно замениться)
	req2 := httptest.NewRequest(http.MethodPost, "/update/gauge/testGauge/2.71", nil)
	w2 := httptest.NewRecorder()
	handler.Update(w2, req2)

	value, ok := memStorage.GetGauge("testGauge")
	if !ok {
		t.Fatal("Gauge should exist")
	}
	if value != 2.71 {
		t.Errorf("Expected 2.71 (replaced), got %f", value)
	}
}

func TestMetricsHandler_Update_PollCount(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := NewMetricsHandler(memStorage)

	// PollCount должен обрабатываться как обычный counter (приращение)
	req1 := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/3", nil)
	w1 := httptest.NewRecorder()
	handler.Update(w1, req1)

	value, ok := memStorage.GetCounter("PollCount")
	if !ok {
		t.Fatal("PollCount should exist")
	}
	if value != 3 {
		t.Errorf("Expected 3, got %d", value)
	}

	// Второе обновление PollCount должно добавить значение к существующему
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	w2 := httptest.NewRecorder()
	handler.Update(w2, req2)

	value, ok = memStorage.GetCounter("PollCount")
	if !ok {
		t.Fatal("PollCount should exist")
	}
	if value != 8 {
		t.Errorf("Expected 8 (3+5), got %d", value)
	}
}

func TestMetricsHandler_UpdateJSON_Counter(t *testing.T) {
	memStorage := storage.NewMemStorage()
	h := NewMetricsHandler(memStorage)

	delta := int64(42)
	reqBody, _ := json.Marshal(model.Metrics{ID: "jsonCounter", MType: model.Counter, Delta: &delta})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateJSON(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Expected application/json, got %s", ct)
	}

	var resp model.Metrics
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}
	if resp.ID != "jsonCounter" || resp.MType != model.Counter || resp.Delta == nil || *resp.Delta != 42 {
		t.Fatalf("Unexpected response: %+v", resp)
	}
}

func TestMetricsHandler_ValueJSON_Gauge(t *testing.T) {
	memStorage := storage.NewMemStorage()
	h := NewMetricsHandler(memStorage)

	// предварительно сохраним метрику
	memStorage.UpdateGauge("jsonGauge", 3.14)

	reqBody, _ := json.Marshal(model.Metrics{ID: "jsonGauge", MType: model.Gauge})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ValueJSON(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Expected application/json, got %s", ct)
	}

	var resp model.Metrics
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}
	if resp.ID != "jsonGauge" || resp.MType != model.Gauge || resp.Value == nil || *resp.Value != 3.14 {
		t.Fatalf("Unexpected response: %+v", resp)
	}
}

func TestGzipMiddleware_RequestAndResponseJSON(t *testing.T) {
	memStorage := storage.NewMemStorage()
	h := NewMetricsHandler(memStorage)

	// Собираем сервер: gzip middleware + /update JSON
	handler := mw.Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.UpdateJSON(w, r)
	}))

	delta := int64(7)
	rawBody, _ := json.Marshal(model.Metrics{ID: "gzCounter", MType: model.Counter, Delta: &delta})
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(rawBody)
	_ = zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Expected gzipped response")
	}

	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()
	var resp model.Metrics
	if err := json.NewDecoder(gr).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode gzipped JSON: %v", err)
	}
	if resp.ID != "gzCounter" || resp.MType != model.Counter || resp.Delta == nil || *resp.Delta != 7 {
		t.Fatalf("Unexpected response: %+v", resp)
	}
}
