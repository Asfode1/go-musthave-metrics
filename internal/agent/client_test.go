package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
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

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type to be application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding to be gzip, got %s", r.Header.Get("Content-Encoding"))
		}
		if r.Header.Get("Accept-Encoding") != "gzip" {
			t.Errorf("Expected Accept-Encoding to be gzip, got %s", r.Header.Get("Accept-Encoding"))
		}

		if r.URL.Path != "/update" {
			t.Errorf("Expected path /update, got %s", r.URL.Path)
		}

		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		b, err := io.ReadAll(gr)
		_ = gr.Close()
		_ = r.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read gzipped body: %v", err)
		}

		var m model.Metrics
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		if m.ID != "testMetric" || m.MType != model.Counter {
			t.Fatalf("Unexpected metric: %+v", m)
		}
		if m.Delta == nil || *m.Delta != 42 {
			t.Fatalf("Expected delta=42, got %+v", m.Delta)
		}
		if m.Value != nil {
			t.Fatalf("Expected value=nil for counter, got %+v", m.Value)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		zw := gzip.NewWriter(w)
		_ = json.NewEncoder(zw).Encode(m)
		_ = zw.Close()
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
		if r.URL.Path != "/update" {
			t.Errorf("Expected path /update, got %s", r.URL.Path)
		}

		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		b, err := io.ReadAll(gr)
		_ = gr.Close()
		_ = r.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read gzipped body: %v", err)
		}

		var m model.Metrics
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		if m.ID != "testGauge" || m.MType != model.Gauge {
			t.Fatalf("Unexpected metric: %+v", m)
		}
		if m.Value == nil || *m.Value != 3.14 {
			t.Fatalf("Expected value=3.14, got %+v", m.Value)
		}
		if m.Delta != nil {
			t.Fatalf("Expected delta=nil for gauge, got %+v", m.Delta)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		zw := gzip.NewWriter(w)
		_ = json.NewEncoder(zw).Encode(m)
		_ = zw.Close()
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

func TestClient_SendMetricsBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/updates" {
			t.Errorf("Expected path /updates, got %s", r.URL.Path)
		}

		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		b, err := io.ReadAll(gr)
		_ = gr.Close()
		_ = r.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read gzipped body: %v", err)
		}

		var ms []model.Metrics
		if err := json.Unmarshal(b, &ms); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		if len(ms) != 2 {
			t.Fatalf("Expected 2 metrics, got %d", len(ms))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	metrics := []MetricValue{
		{Type: model.Counter, Name: "metric1", Value: int64(1)},
		{Type: model.Gauge, Name: "metric2", Value: float64(2.5)},
	}

	err := client.SendMetricsBatch(metrics)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
