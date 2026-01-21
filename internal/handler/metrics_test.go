package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	// PollCount должен использовать SetCounter (замену), а не UpdateCounter (суммирование)
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

	// Второе обновление PollCount должно заменить значение, а не добавить
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	w2 := httptest.NewRecorder()
	handler.Update(w2, req2)

	value, ok = memStorage.GetCounter("PollCount")
	if !ok {
		t.Fatal("PollCount should exist")
	}
	if value != 5 {
		t.Errorf("Expected 5 (replaced), got %d", value)
	}
}
