package httpclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// Client HTTP клиент для доставки webhook.
type Client struct {
	httpClient *http.Client
	log        *slog.Logger
}

// NewClient создаёт новый HTTP client для webhook доставки.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: slog.Default(),
	}
}

// SendWebhook отправляет webhook на указанный URL.
func (c *Client) SendWebhook(ctx context.Context, req *WebhookRequest) (*WebhookResponse, error) {
	// 1. Валидируем URL
	if err := validateURL(req.URL); err != nil {
		return nil, errors.Wrap(err, "invalid webhook URL")
	}

	// 2. Подготавливаем payload
	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal webhook payload")
	}

	// 3. Проверяем размер payload
	if req.MaxPayloadSizeBytes > 0 && len(payload) > req.MaxPayloadSizeBytes {
		if !req.TruncateOnOverflow {
			return nil, model.ErrWebhookPayloadTooLarge
		}
		payload = truncatePayload(payload, req.MaxPayloadSizeBytes)
	}

	// 4. Создаём HTTP запрос
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	// 5. Устанавливаем headers
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 6. Добавляем signature если указан
	if req.SecretKey != nil && *req.SecretKey != "" {
		timestamp := time.Now().Unix()
		signature := generateHMACSignature(payload, *req.SecretKey, timestamp)

		httpReq.Header.Set("X-Webhook-Signature", signature)
		httpReq.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", timestamp))
	}

	// 7. Отправляем запрос
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, c.handleRequestError(err)
	}

	responseTimeMs := int(time.Since(startTime).Milliseconds())

	// 8. Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		c.log.WarnContext(ctx, "failed to close webhook response body", "error", closeErr)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to read webhook response body")
	}

	// 9. Проверяем статус код
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.handleHTTPError(resp.StatusCode, string(body))
	}

	c.log.InfoContext(ctx, "webhook sent successfully",
		"url", req.URL,
		"status_code", resp.StatusCode,
		"response_time_ms", responseTimeMs,
	)

	return &WebhookResponse{
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: responseTimeMs,
		Body:           string(body),
	}, nil
}

// WebhookRequest запрос для отправки webhook.
type WebhookRequest struct {
	URL                 string
	Method              string
	Headers             map[string]string
	Payload             any
	SecretKey           *string
	MaxPayloadSizeBytes int
	TruncateOnOverflow  bool
}

// WebhookResponse ответ от webhook endpoint.
type WebhookResponse struct {
	StatusCode     int
	ResponseTimeMs int
	Body           string
}

// validateURL валидирует webhook URL.
func validateURL(webhookURL string) error {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return model.ErrWebhookInvalidURL
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return model.ErrWebhookInvalidURL
	}

	// Проверка на инъекции
	if containsInjection(webhookURL) {
		return model.ErrWebhookInjection
	}

	return nil
}

// containsInjection проверяет URL на наличие инъекций.
func containsInjection(urlStr string) bool {
	suspiciousPatterns := []string{
		"<script", "</script>", "javascript:", "onerror=", "onload=",
		"../", "..\\", "file://", "data:",
	}

	urlLower := strings.ToLower(urlStr)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}

	return false
}

// truncatePayload обрезает payload до указанного размера.
func truncatePayload(payload []byte, maxSize int) []byte {
	if len(payload) <= maxSize {
		return payload
	}

	return payload[:maxSize]
}

// generateHMACSignature генерирует HMAC-SHA256 подпись.
func generateHMACSignature(payload []byte, secret string, timestamp int64) string {
	sigPayload := fmt.Sprintf("%d.%s", timestamp, payload)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(sigPayload))

	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// handleRequestError обрабатывает ошибку HTTP запроса.
func (c *Client) handleRequestError(err error) error {
	if err.Error() == "context deadline exceeded" {
		return model.ErrWebhookTimeout
	}

	if urlErr, ok := err.(*url.Error); ok {
		if urlErr.Timeout() {
			return model.ErrWebhookTimeout
		}
	}

	return errors.Wrap(err, "webhook request failed")
}

// handleHTTPError обрабатывает HTTP ошибку.
func (c *Client) handleHTTPError(statusCode int, body string) error {
	// 4xx ошибки - permanent, no retry
	if statusCode >= 400 && statusCode < 500 {
		switch statusCode {
		case http.StatusUnauthorized:
			return model.ErrWebhookAuthFailed
		case http.StatusNotFound:
			return model.ErrWebhookInvalidURL
		case http.StatusRequestTimeout:
			return model.ErrWebhookTimeout
		default:
			return model.ErrDeliveryPermanentError
		}
	}

	// 5xx ошибки - transient, retry
	if statusCode >= 500 {
		return model.ErrWebhook5xxError
	}

	return fmt.Errorf("webhook returned status %d: %s", statusCode, body)
}

// TestWebhook тестирует webhook endpoint.
func (c *Client) TestWebhook(ctx context.Context, webhookURL string, timeoutSeconds int) error {
	if err := validateURL(webhookURL); err != nil {
		return err
	}

	// Создаём тестовый запрос с таймаутом
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	testPayload := map[string]any{
		"test":      true,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	payload, err := json.Marshal(testPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Отправляем
	resp, err := client.Do(req)
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			return model.ErrWebhookTimeout
		}
		return c.handleRequestError(err)
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		c.log.WarnContext(ctx, "failed to close test webhook response body", "error", closeErr)
	}

	// Проверяем статус
	if resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusGatewayTimeout {
		return model.ErrWebhookTimeout
	}

	if resp.StatusCode >= 400 {
		return c.handleHTTPError(resp.StatusCode, "")
	}

	return nil
}
