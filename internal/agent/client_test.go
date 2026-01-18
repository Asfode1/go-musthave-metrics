package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
)

func TestClient_SendMetric(t *testing.T) {
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("Expected Content-Type to be text/plain, got %s", r.Header.Get("Content-Type"))
		}

		expectedPath := "/update/counter/testMetric/42"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	metric := MetricValue{
		Type:  model.Counter,
		Name:  "testMetric",
		Value: int64(42),
	}

	err := client.SendMetric(metric)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_SendMetric_Gauge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/update/gauge/testGauge/3.14"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	metric := MetricValue{
		Type:  model.Gauge,
		Name:  "testGauge",
		Value: float64(3.14),
	}

	err := client.SendMetric(metric)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_SendMetric_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	metric := MetricValue{
		Type:  model.Counter,
		Name:  "testMetric",
		Value: int64(42),
	}

	err := client.SendMetric(metric)
	if err == nil {
		t.Fatal("Expected error for bad status code, got nil")
	}
}

func TestClient_SendMetrics(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	metrics := []MetricValue{
		{Type: model.Counter, Name: "metric1", Value: int64(1)},
		{Type: model.Gauge, Name: "metric2", Value: float64(2.5)},
		{Type: model.Counter, Name: "metric3", Value: int64(3)},
	}

	err := client.SendMetrics(metrics)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if requestCount != len(metrics) {
		t.Errorf("Expected %d requests, got %d", len(metrics), requestCount)
	}
}

func TestClient_SendMetrics_PartialError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Возвращаем ошибку на втором запросе
		if requestCount == 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	metrics := []MetricValue{
		{Type: model.Counter, Name: "metric1", Value: int64(1)},
		{Type: model.Gauge, Name: "metric2", Value: float64(2.5)},
		{Type: model.Counter, Name: "metric3", Value: int64(3)},
	}

	err := client.SendMetrics(metrics)
	if err == nil {
		t.Fatal("Expected error for failed metric, got nil")
	}
}
