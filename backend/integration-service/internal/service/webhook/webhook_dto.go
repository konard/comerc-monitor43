package webhook

import (
	"github.com/google/uuid"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// CreateWebhookRequest DTO для создания webhook.
type CreateWebhookRequest struct {
	UserID                  uuid.UUID
	Name                    string
	URL                     string
	Method                  string
	Headers                 map[string]string
	SecretKey               *string // Не зашифрован, от клиента
	Priority                model.WebhookPriority
	SeverityFilter          []string
	MaxPayloadSizeBytes     int
	PayloadHandlingStrategy model.PayloadHandlingStrategy
}

// ToModel конвертирует DTO в domain model.
func (r *CreateWebhookRequest) ToModel() (*model.WebhookIntegration, error) {
	webhook, err := model.NewWebhookIntegration(r.UserID, r.Name, r.URL, r.Method)
	if err != nil {
		return nil, err
	}

	webhook.Headers = r.Headers
	if r.Priority != "" {
		webhook.Priority = r.Priority
	}

	if len(r.SeverityFilter) > 0 {
		webhook.SeverityFilter = r.SeverityFilter
	}

	if r.MaxPayloadSizeBytes > 0 {
		webhook.MaxPayloadSizeBytes = r.MaxPayloadSizeBytes
	}

	if r.PayloadHandlingStrategy != "" {
		webhook.PayloadHandlingStrategy = r.PayloadHandlingStrategy
	}

	return webhook, nil
}

// UpdateWebhookRequest DTO для обновления webhook.
type UpdateWebhookRequest struct {
	ID                      uuid.UUID
	UserID                  uuid.UUID
	Name                    *string
	URL                     *string
	Method                  *string
	Headers                 map[string]string
	Enabled                 *bool
	Priority                *model.WebhookPriority
	SeverityFilter          []string
	MaxPayloadSizeBytes     *int
	PayloadHandlingStrategy *model.PayloadHandlingStrategy
}

// ApplyToModel применяет изменения к модели.
func (r *UpdateWebhookRequest) ApplyToModel(webhook *model.WebhookIntegration) {
	if r.Name != nil {
		webhook.Name = *r.Name
	}

	if r.URL != nil {
		webhook.URL = *r.URL
	}

	if r.Method != nil {
		webhook.Method = *r.Method
	}

	if r.Headers != nil {
		webhook.Headers = r.Headers
	}

	if r.Enabled != nil {
		webhook.Enabled = *r.Enabled
	}

	if r.Priority != nil {
		webhook.Priority = *r.Priority
	}

	if r.SeverityFilter != nil {
		webhook.SeverityFilter = r.SeverityFilter
	}

	if r.MaxPayloadSizeBytes != nil {
		webhook.MaxPayloadSizeBytes = *r.MaxPayloadSizeBytes
	}

	if r.PayloadHandlingStrategy != nil {
		webhook.PayloadHandlingStrategy = *r.PayloadHandlingStrategy
	}
}

// ListWebhooksRequest DTO для списка webhooks.
type ListWebhooksRequest struct {
	UserID    uuid.UUID
	PageSize  int
	PageToken string
}

// WebhookStatsDTO DTO для статистики webhook.
type WebhookStatsDTO struct {
	TotalSent         int
	SuccessfulSent    int
	FailedSent        int
	AvgResponseTimeMs int
	LastSentAt        *int64
	LastSuccessAt     *int64
	LastFailureAt     *int64
	FailureCount      int
	SuccessRate       float64
}

// FromModel создаёт DTO из domain model.
func WebhookStatsFromModel(webhook *model.WebhookIntegration) *WebhookStatsDTO {
	stats := &WebhookStatsDTO{
		TotalSent:      webhook.TotalSent,
		SuccessfulSent: webhook.SuccessfulSent,
		FailedSent:     webhook.FailedSent,
		FailureCount:   webhook.FailureCount,
	}

	if webhook.AvgResponseTimeMs != nil {
		stats.AvgResponseTimeMs = *webhook.AvgResponseTimeMs
	}

	if webhook.LastSentAt != nil {
		ts := webhook.LastSentAt.Unix()
		stats.LastSentAt = &ts
	}

	if webhook.LastSuccessAt != nil {
		ts := webhook.LastSuccessAt.Unix()
		stats.LastSuccessAt = &ts
	}

	if webhook.LastFailureAt != nil {
		ts := webhook.LastFailureAt.Unix()
		stats.LastFailureAt = &ts
	}

	// Вычисляем success rate
	if webhook.TotalSent > 0 {
		stats.SuccessRate = (float64(webhook.SuccessfulSent) / float64(webhook.TotalSent)) * 100
	}

	return stats
}

// TestWebhookRequest DTO для тестирования webhook.
type TestWebhookRequest struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

// TestWebhookResponse DTO ответа на тестирование webhook.
type TestWebhookResponse struct {
	Success        bool
	StatusCode     int
	ResponseTimeMs int
	ErrorMessage   string
}

// CloneWebhookRequest DTO для клонирования webhook.
type CloneWebhookRequest struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	NewName string
	NewURL  string
}
