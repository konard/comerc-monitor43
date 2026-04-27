package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WebhookStatus статус webhook интеграции.
type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusInactive WebhookStatus = "inactive"
	WebhookStatusFailed   WebhookStatus = "failed"
	WebhookStatusDisabled WebhookStatus = "disabled"
)

// WebhookPriority приоритет webhook для доставки.
type WebhookPriority string

const (
	WebhookPriorityHigh   WebhookPriority = "high"
	WebhookPriorityNormal WebhookPriority = "normal"
	WebhookPriorityLow    WebhookPriority = "low"
)

// PayloadHandlingStrategy стратегия обработки большого payload.
type PayloadHandlingStrategy string

const (
	PayloadHandlingStrategyTruncate PayloadHandlingStrategy = "truncate"
	PayloadHandlingStrategyReject   PayloadHandlingStrategy = "reject"
)

// WebhookIntegration представляет webhook интеграцию.
type WebhookIntegration struct {
	// ID уникальный идентификатор
	ID uuid.UUID
	// UserID идентификатор владельца
	UserID uuid.UUID
	// имя интеграции
	Name string
	// URL webhook endpoint
	URL string
	// Method HTTP метод
	Method string
	// Headers custom HTTP headers
	Headers map[string]string
	// SecretKey секретный ключ для HMAC (зашифрован)
	SecretKey *string
	// Enabled включена ли интеграция
	Enabled bool
	// Status статус интеграции
	Status WebhookStatus
	// Priority приоритет доставки
	Priority WebhookPriority
	// SeverityFilter фильтр по severity для отправки
	SeverityFilter []string
	// MaxPayloadSizeBytes максимальный размер payload
	MaxPayloadSizeBytes int
	// PayloadHandlingStrategy стратегия обработки большого payload
	PayloadHandlingStrategy PayloadHandlingStrategy

	// Statistics
	TotalSent         int
	SuccessfulSent    int
	FailedSent        int
	AvgResponseTimeMs *int
	LastSentAt        *time.Time
	LastSuccessAt     *time.Time
	LastFailureAt     *time.Time

	// Failure tracking
	FailureCount        int
	ConsecutiveFailures int

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewWebhookIntegration создаёт новую webhook интеграцию.
func NewWebhookIntegration(userID uuid.UUID, name, webhookURL string, method string) (*WebhookIntegration, error) {
	if err := validateWebhookName(name); err != nil {
		return nil, err
	}

	if err := validateWebhookURL(webhookURL); err != nil {
		return nil, err
	}

	if err := validateHTTPMethod(method); err != nil {
		return nil, err
	}

	now := time.Now()

	return &WebhookIntegration{
		ID:                      uuid.New(),
		UserID:                  userID,
		Name:                    name,
		URL:                     webhookURL,
		Method:                  method,
		Headers:                 make(map[string]string),
		Enabled:                 true,
		Status:                  WebhookStatusActive,
		Priority:                WebhookPriorityNormal,
		SeverityFilter:          []string{"critical", "warning", "degraded"},
		MaxPayloadSizeBytes:     1048576, // 1MB
		PayloadHandlingStrategy: PayloadHandlingStrategyTruncate,
		TotalSent:               0,
		SuccessfulSent:          0,
		FailedSent:              0,
		FailureCount:            0,
		ConsecutiveFailures:     0,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}

// validateWebhookName проверяет валидность имени webhook.
func validateWebhookName(name string) error {
	if name == "" {
		return ErrEmptyWebhookName
	}
	if len(name) > 255 {
		return ErrWebhookNameTooLong
	}
	return nil
}

// validateWebhookURL проверяет валидность webhook URL.
func validateWebhookURL(webhookURL string) error {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return ErrWebhookInvalidURL
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrWebhookInvalidURL
	}

	// Проверка на инъекции
	if containsInjection(webhookURL) {
		return ErrWebhookInjection
	}

	return nil
}

// validateHTTPMethod проверяет валидность HTTP метода.
func validateHTTPMethod(method string) error {
	method = strings.ToUpper(method)
	validMethods := map[string]bool{
		http.MethodGet:   true,
		http.MethodPost:  true,
		http.MethodPut:   true,
		http.MethodPatch: true,
	}

	if !validMethods[method] {
		return ErrWebhookInvalidMethod
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

// CanDeliver возвращает true, если webhook может доставлять сообщения.
func (w *WebhookIntegration) CanDeliver() bool {
	return w.Enabled && w.Status == WebhookStatusActive
}

// ShouldDeliverForSeverity проверяет, следует ли доставлять для указанного severity.
func (w *WebhookIntegration) ShouldDeliverForSeverity(severity string) bool {
	if len(w.SeverityFilter) == 0 {
		return true
	}

	for _, allowed := range w.SeverityFilter {
		if strings.EqualFold(allowed, severity) {
			return true
		}
	}

	return false
}

// MarkAsFailed помечает webhook как неудачный.
func (w *WebhookIntegration) MarkAsFailed() {
	now := time.Now()
	w.FailureCount++
	w.ConsecutiveFailures++
	w.LastFailureAt = &now
	w.UpdatedAt = now

	// Автоматически отключаем после 5 неудачных попыток
	if w.ConsecutiveFailures >= 5 {
		w.Status = WebhookStatusFailed
		w.Enabled = false
	}
}

// MarkAsSuccessful помечает webhook как успешно доставленный.
func (w *WebhookIntegration) MarkAsSuccessful(responseTimeMs int) {
	now := time.Now()
	w.TotalSent++
	w.SuccessfulSent++
	w.LastSentAt = &now
	w.LastSuccessAt = &now
	w.ConsecutiveFailures = 0
	w.UpdatedAt = now

	// Обновляем среднее время ответа
	if w.AvgResponseTimeMs == nil {
		w.AvgResponseTimeMs = &responseTimeMs
	} else {
		// Скользящее среднее: new_avg = 0.9 * old_avg + 0.1 * new_value
		newAvg := float64(*w.AvgResponseTimeMs)*0.9 + float64(responseTimeMs)*0.1
		rounded := int(newAvg + 0.5)
		w.AvgResponseTimeMs = &rounded
	}
}

// MarkAsDisabled помечает webhook как отключённый.
func (w *WebhookIntegration) MarkAsDisabled() {
	w.Status = WebhookStatusDisabled
	w.Enabled = false
	w.UpdatedAt = time.Now()
}

// MarkAsActive помечает webhook как активный.
func (w *WebhookIntegration) MarkAsActive() {
	w.Status = WebhookStatusActive
	w.Enabled = true
	w.ConsecutiveFailures = 0
	w.UpdatedAt = time.Now()
}

// UpdateStats обновляет статистику из delivery attempts.
func (w *WebhookIntegration) UpdateStats(totalSent, successfulSent, failedSent int, avgResponseTimeMs int) {
	w.TotalSent = totalSent
	w.SuccessfulSent = successfulSent
	w.FailedSent = failedSent
	w.AvgResponseTimeMs = &avgResponseTimeMs
	w.UpdatedAt = time.Now()
}

// GenerateSignature генерирует HMAC-SHA256 подпись для payload.
func (w *WebhookIntegration) GenerateSignature(payload []byte, timestamp int64) (string, error) {
	if w.SecretKey == nil || *w.SecretKey == "" {
		return "", nil // Без секретного ключа подпись не генерируем
	}

	// Создаём payload для подписи: timestamp + payload
	sigPayload := fmt.Sprintf("%d.%s", timestamp, payload)

	h := hmac.New(sha256.New, []byte(*w.SecretKey))
	h.Write([]byte(sigPayload))

	return "sha256=" + hex.EncodeToString(h.Sum(nil)), nil
}

// ValidatePayloadSize проверяет размер payload.
func (w *WebhookIntegration) ValidatePayloadSize(size int) error {
	if size > w.MaxPayloadSizeBytes {
		return ErrWebhookPayloadTooLarge
	}
	return nil
}

// ShouldTruncatePayload возвращает true, если нужно обрезать payload.
func (w *WebhookIntegration) ShouldTruncatePayload() bool {
	return w.PayloadHandlingStrategy == PayloadHandlingStrategyTruncate
}

// TruncatePayload обрезает payload до максимального размера.
func (w *WebhookIntegration) TruncatePayload(payload []byte) []byte {
	if len(payload) <= w.MaxPayloadSizeBytes {
		return payload
	}

	return payload[:w.MaxPayloadSizeBytes]
}

// MarshalJSON для сериализации SeverityFilter как JSON массива.
func (w *WebhookIntegration) MarshalJSON() ([]byte, error) {
	type Alias WebhookIntegration
	return json.Marshal(&struct {
		SeverityFilter []string `json:"severity_filter"`
		*Alias
	}{
		SeverityFilter: w.SeverityFilter,
		Alias:          (*Alias)(w),
	})
}

// WebhookDeliveryAttempt представляет попытку доставки webhook.
type WebhookDeliveryAttempt struct {
	ID                 uuid.UUID
	WebhookID          uuid.UUID
	AlertID            uuid.UUID
	Status             string
	RetryCount         int
	NextRetryAt        *time.Time
	HTTPStatusCode     *int
	ResponseTimeMs     *int
	ErrorMessage       *string
	ErrorCode          *string
	PayloadSizeBytes   int
	PayloadTruncated   bool
	SignatureAlgorithm *string
	CreatedAt          time.Time
	SentAt             *time.Time
}

// NewWebhookDeliveryAttempt создаёт новую попытку доставки.
func NewWebhookDeliveryAttempt(webhookID, alertID uuid.UUID) *WebhookDeliveryAttempt {
	return &WebhookDeliveryAttempt{
		ID:         uuid.New(),
		WebhookID:  webhookID,
		AlertID:    alertID,
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}
}

// MarkAsSent помечает попытку как отправленную.
func (d *WebhookDeliveryAttempt) MarkAsSent(statusCode int, responseTimeMs int) {
	d.Status = "sent"
	d.HTTPStatusCode = &statusCode
	d.ResponseTimeMs = &responseTimeMs
	now := time.Now()
	d.SentAt = &now
}

// MarkAsFailed помечает попытку как неудачную.
func (d *WebhookDeliveryAttempt) MarkAsFailed(errorCode string, errorMessage string) {
	d.Status = "failed"
	d.ErrorCode = &errorCode
	d.ErrorMessage = &errorMessage
}

// ScheduleRetry планирует повторную попытку.
func (d *WebhookDeliveryAttempt) ScheduleRetry(nextRetryAt time.Time, retryCount int) {
	d.Status = "retry_scheduled"
	d.NextRetryAt = &nextRetryAt
	d.RetryCount = retryCount
}
