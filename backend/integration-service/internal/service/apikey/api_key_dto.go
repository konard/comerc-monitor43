package apikey

import (
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// CreateAPIKeyRequest DTO для создания API ключа.
type CreateAPIKeyRequest struct {
	UserID             uuid.UUID
	Name               string
	Description        *string
	KeyType            model.APIKeyType
	Scopes             []string
	ExpiresAt          *time.Time
	IPWhitelist        []string
	RateLimitPerMinute int
}

// Validate валидирует запрос.
func (r *CreateAPIKeyRequest) Validate() error {
	if r.Name == "" {
		return model.ErrEmptyAPIKeyName
	}

	if len(r.Name) > 255 {
		return model.ErrAPIKeyNameTooLong
	}

	if len(r.Scopes) == 0 {
		return model.ErrInvalidScope
	}

	for _, scope := range r.Scopes {
		if !model.ValidScopes[scope] {
			return model.ErrInvalidScope
		}
	}

	return nil
}

// UpdateAPIKeyRequest DTO для обновления API ключа.
type UpdateAPIKeyRequest struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Name               *string
	Description        *string
	Enabled            *bool
	ExpiresAt          *time.Time
	IPWhitelist        []string
	RateLimitPerMinute *int
}

// ApplyToModel применяет изменения к модели.
func (r *UpdateAPIKeyRequest) ApplyToModel(key *model.APIKey) {
	if r.Name != nil {
		key.Name = *r.Name
	}

	if r.Description != nil {
		key.Description = r.Description
	}

	if r.Enabled != nil {
		if *r.Enabled {
			key.Enable()
		} else {
			key.Disable()
		}
	}

	if r.ExpiresAt != nil {
		key.ExpiresAt = r.ExpiresAt
	}

	if r.IPWhitelist != nil {
		key.IPWhitelist = r.IPWhitelist
	}

	if r.RateLimitPerMinute != nil {
		key.RateLimitPerMinute = *r.RateLimitPerMinute
	}
}

// ListAPIKeysRequest DTO для списка API ключей.
type ListAPIKeysRequest struct {
	UserID    uuid.UUID
	PageSize  int
	PageToken string
}

// APIKeyStatsDTO DTO для статистики API ключа.
type APIKeyStatsDTO struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	SuccessRate        float64
	AvgLatencyMs       int32
	LastUsedAt         *int64
	LastUsedIP         string
	MostUsedEndpoint   string
}

// FromModel создаёт DTO из domain model.
func APIKeyStatsFromModel(key *model.APIKey, avgLatencyMs int32) *APIKeyStatsDTO {
	stats := &APIKeyStatsDTO{
		TotalRequests:      int64(key.TotalRequests),
		SuccessfulRequests: int64(key.SuccessfulRequests),
		FailedRequests:     int64(key.FailedRequests),
		SuccessRate:        key.GetSuccessRate(),
		AvgLatencyMs:       avgLatencyMs,
	}

	if key.LastUsedAt != nil {
		ts := key.LastUsedAt.Unix()
		stats.LastUsedAt = &ts
	}

	if key.LastUsedIP != nil {
		stats.LastUsedIP = *key.LastUsedIP
	}

	if key.MostUsedEndpoint != nil {
		stats.MostUsedEndpoint = *key.MostUsedEndpoint
	}

	return stats
}

// ValidateAPIKeyRequest DTO для валидации API ключа (используется Gateway).
type ValidateAPIKeyRequest struct {
	APIKey    string
	Endpoint  string
	Method    string
	IPAddress string
}

// ValidateAPIKeyResponse DTO ответа на валидацию API ключа.
type ValidateAPIKeyResponse struct {
	Valid     bool
	UserID    uuid.UUID
	Scopes    []string
	Status    model.APIKeyStatus
	ErrorCode string
	ErrorMsg  string
}

// RotateAPIKeySecretRequest DTO для ротации секретного ключа.
type RotateAPIKeySecretRequest struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

// RotateAPIKeySecretResponse DTO ответа на ротацию ключа.
type RotateAPIKeySecretResponse struct {
	APIKey     *model.APIKey
	NewFullKey string // Новый полный ключ - возвращается только один раз!
}

// GetAPIKeyStatsRequest DTO для получения статистики.
type GetAPIKeyStatsRequest struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

// ExportAPIKeyUsageRequest DTO для экспорта истории использования.
type ExportAPIKeyUsageRequest struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	PeriodDays int // За сколько дней экспортировать
}

// APIKeyUsageExportDTO DTO для экспортированных данных.
type APIKeyUsageExportDTO struct {
	APIKeyID    uuid.UUID
	PeriodStart time.Time
	PeriodEnd   time.Time
	Records     []APIKeyUsageRecordDTO
}

// APIKeyUsageRecordDTO DTO записи использования.
type APIKeyUsageRecordDTO struct {
	Timestamp      time.Time
	Endpoint       string
	Method         string
	HTTPStatusCode int
	ResponseTimeMs int
	IPAddress      string
	UserAgent      string
	RequestID      uuid.UUID
	RateLimited    bool
}
