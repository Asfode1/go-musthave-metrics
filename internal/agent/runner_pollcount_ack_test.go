package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
)

func TestRunner_reportOnce_AcksPollCountEvenIfOtherMetricsFailed(t *testing.T) {
	var (
		mu    sync.Mutex
		seen  = map[string]int{}
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b, err := io.ReadAll(gr)
			_ = gr.Close()
			_ = r.Body.Close()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			body = b
		} else {
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			body = b
		}

		var m model.Metrics
		if err := json.Unmarshal(body, &m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		seen[m.ID]++
		mu.Unlock()

		if m.ID == "badMetric" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := &Runner{
		collector: NewCollector(),
		client:    NewClient(srv.URL),
		config:    &RunnerConfig{},
	}

	// Делаем несколько Collect, чтобы накопить PollCount.
	for i := 0; i < 5; i++ {
		_ = r.collector.Collect()
	}
	if got := r.collector.PollCountSnapshot(); got != 5 {
		t.Fatalf("expected pollCount=5 before report, got %d", got)
	}

	metrics := []MetricValue{
		{Type: model.Gauge, Name: "okMetric", Value: float64(1.23)},
		{Type: model.Gauge, Name: "badMetric", Value: float64(9.99)},
	}

	err := r.reportOnce(metrics)
	if err == nil {
		t.Fatalf("expected combined error, got nil")
	}

	// Важно: PollCount должен быть ack-нут даже при частичной ошибке других метрик,
	// если сам PollCount успешно доставлен.
	if got := r.collector.PollCountSnapshot(); got != 0 {
		t.Fatalf("expected pollCount=0 after report (acked), got %d", got)
	}

	mu.Lock()
	defer mu.Unlock()
	if seen["PollCount"] != 1 {
		t.Fatalf("expected PollCount to be sent once, got %d", seen["PollCount"])
	}
	if seen["badMetric"] != 1 {
		t.Fatalf("expected badMetric to be sent once, got %d", seen["badMetric"])
	}
	if seen["okMetric"] != 1 {
		t.Fatalf("expected okMetric to be sent once, got %d", seen["okMetric"])
	}
}

func TestRunner_reportOnce_DoesNotAckPollCountIfPollCountSendFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b, err := io.ReadAll(gr)
			_ = gr.Close()
			_ = r.Body.Close()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			body = b
		} else {
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			body = b
		}

		var m model.Metrics
		if err := json.Unmarshal(body, &m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if m.ID == "PollCount" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := &Runner{
		collector: NewCollector(),
		client:    NewClient(srv.URL),
		config:    &RunnerConfig{},
	}

	for i := 0; i < 3; i++ {
		_ = r.collector.Collect()
	}
	if got := r.collector.PollCountSnapshot(); got != 3 {
		t.Fatalf("expected pollCount=3 before report, got %d", got)
	}

	err := r.reportOnce([]MetricValue{
		{Type: model.Gauge, Name: "okMetric", Value: float64(1.23)},
	})
	if err == nil {
		t.Fatalf("expected error (PollCount failed), got nil")
	}

	// PollCount не должен ack-аться, если его отправка не удалась.
	if got := r.collector.PollCountSnapshot(); got != 3 {
		t.Fatalf("expected pollCount to remain 3 after failed PollCount send, got %d", got)
	}
}

