package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
)

// Client HTTP-клиент для отправки метрик на сервер, чтобы обеспечить надежную доставку данных
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient создает новый HTTP-клиент
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Чтобы предотвратить зависание при проблемах с сетью
		},
	}
}

// SendMetric отправляет одну метрику на сервер
func (c *Client) SendMetric(metric MetricValue) error {
	url := fmt.Sprintf("%s/update", c.baseURL)

	payload, err := metricValueToJSON(metric)
	if err != nil {
		return err
	}

	gzPayload, err := gzipBytes(payload)
	if err != nil {
		return fmt.Errorf("failed to gzip payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(gzPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json") // формат JSON
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Чтобы прочитать тело ответа и избежать EOF ошибок
	bodyReader := resp.Body
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip") {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gr.Close()
		bodyReader = gr
	}

	_, err = io.ReadAll(bodyReader)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func metricValueToJSON(metric MetricValue) ([]byte, error) {
	m := model.Metrics{
		ID:    metric.Name,
		MType: metric.Type,
	}

	switch v := metric.Value.(type) {
	case int64:
		// counter
		m.Delta = &v
	case float64:
		// gauge
		m.Value = &v
	default:
		return nil, fmt.Errorf("unsupported metric value type: %T", v)
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metric: %w", err)
	}
	return b, nil
}

// SendMetrics отправляет все метрики на сервер
// Продолжает отправку даже при ошибках отдельных метрик, чтобы максимально доставить данные
func (c *Client) SendMetrics(metrics []MetricValue) error {
	var errs []error
	for _, metric := range metrics {
		if err := c.SendMetric(metric); err != nil {
			errs = append(errs, fmt.Errorf("failed to send metric %s: %w", metric.Name, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
