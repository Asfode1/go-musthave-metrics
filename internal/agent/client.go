package agent

import (
	"fmt"
	"io"
	"net/http"
	"time"
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
	var valueStr string
	switch v := metric.Value.(type) {
	case int64:
		valueStr = fmt.Sprintf("%d", v)
	case float64:
		valueStr = fmt.Sprintf("%g", v)
	default:
		return fmt.Errorf("unsupported metric value type: %T", v)
	}

	url := fmt.Sprintf("%s/update/%s/%s/%s", c.baseURL, metric.Type, metric.Name, valueStr)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain") // Чтобы сервер корректно обработал запрос

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Чтобы прочитать тело ответа и избежать EOF ошибок
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// SendMetrics отправляет все метрики на сервер
func (c *Client) SendMetrics(metrics []MetricValue) error {
	for _, metric := range metrics {
		if err := c.SendMetric(metric); err != nil {
			return fmt.Errorf("failed to send metric %s: %w", metric.Name, err)
		}
	}
	return nil
}
