package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// WebhookSender интерфейс для отправки webhook уведомлений (используется для тестирования)
type WebhookSender interface {
	Send(ctx context.Context, url string, payload any) error
}

// WebhookSenderWithHeaders интерфейс для отправки webhook с кастомными заголовками
type WebhookSenderWithHeaders interface {
	WebhookSender
	SendWithHeaders(ctx context.Context, url string, payload any, headers map[string]string) error
}

// WebhookClient отправляет webhook уведомления
type WebhookClient struct {
	client  *http.Client
	timeout time.Duration
}

// NewWebhookClient создаёт новый Webhook клиент
func NewWebhookClient(timeout time.Duration) *WebhookClient {
	return &WebhookClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Send отправляет webhook
func (c *WebhookClient) Send(ctx context.Context, url string, payload any) (err error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Monitor-Alert-Service/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp, &err, "failed to close webhook response body")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("webhook returned status %d and response body read failed: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendWithHeaders отправляет webhook с кастомными заголовками
func (c *WebhookClient) SendWithHeaders(
	ctx context.Context,
	url string,
	payload any,
	headers map[string]string,
) (err error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Monitor-Alert-Service/1.0")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp, &err, "failed to close webhook response body")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("webhook returned status %d and response body read failed: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *WebhookClient) SendWithConfig(ctx context.Context, url string, payload any, config *model.WebhookChannelConfig) (err error) {
	method := config.Method
	if method == "" {
		method = "POST"
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Monitor-Alert-Service/1.0")

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp, &err, "failed to close webhook response body")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("webhook returned status %d and response body read failed: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
