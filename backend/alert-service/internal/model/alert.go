package model

import (
	"time"

	"github.com/google/uuid"
)

// Ошибки доменной модели
var (
	ErrAlertNotFound        = NewDomainError("alert not found").WithCode("ERR_ALERT_NOT_FOUND")
	ErrAlertRuleNotFound    = NewDomainError("alert rule not found").WithCode("ERR_ALERT_RULE_NOT_FOUND")
	ErrAlertChannelNotFound = NewDomainError("alert channel not found").WithCode("ERR_CHANNEL_NOT_FOUND")
)

// AlertStatus представляет статус алерта
type AlertStatus string

const (
	AlertStatusTriggered    AlertStatus = "triggered"
	AlertStatusMuted        AlertStatus = "muted"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// String возвращает строковое представление AlertStatus
func (s AlertStatus) String() string {
	return string(s)
}

// AlertType представляет тип алерта
type AlertType string

const (
	AlertTypeStatusCode         AlertType = "status_code"
	AlertTypeResponseTime       AlertType = "response_time"
	AlertTypeBodyContains       AlertType = "body_contains"
	AlertTypeBodyDoesNotContain AlertType = "body_does_not_contain"
	AlertTypeCertificateExpires AlertType = "certificate_expires"
	AlertTypeSSLError           AlertType = "ssl_error"
	AlertTypeFlapping           AlertType = "flapping"
)

// String возвращает строковое представление AlertType
func (t AlertType) String() string {
	return string(t)
}

// Alert представляет алерт в доменной модели
type Alert struct {
	ID                  uuid.UUID   `db:"id"`
	UserID              uuid.UUID   `db:"user_id"`
	MonitorID           uuid.UUID   `db:"monitor_id"`
	AlertRuleID         *uuid.UUID  `db:"alert_rule_id"`
	Status              AlertStatus `db:"status"`
	Type                AlertType   `db:"type"`
	Enabled             bool        `db:"enabled"`
	ConsecutiveFailures int         `db:"consecutive_failures"`
	ThresholdMs         int         `db:"threshold_ms"`
	CreatedAt           time.Time   `db:"created_at"`
	UpdatedAt           time.Time   `db:"updated_at"`

	// Поля подтверждения алерта
	AcknowledgedBy *uuid.UUID `db:"acknowledged_by"`
	AcknowledgedAt *time.Time `db:"acknowledged_at"`

	// Type-specific configuration (stored as TEXT/JSON in DB per DOD 4.1)
	Config map[string]any `db:"config"`
}

// AlertRule представляет правило алерта
type AlertRule struct {
	ID                  uuid.UUID `db:"id"`
	UserID              uuid.UUID `db:"user_id"`
	MonitorID           uuid.UUID `db:"monitor_id"`
	Enabled             bool      `db:"enabled"`
	ConsecutiveFailures int       `db:"consecutive_failures"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// AlertFilter представляет фильтр для поиска алертов
type AlertFilter struct {
	MonitorID string
	Status    AlertStatus
	Page      int
	PageSize  int
}

// AlertChannelType представляет тип канала уведомлений
type AlertChannelType string

const (
	AlertChannelTypeEmail    AlertChannelType = "email"
	AlertChannelTypeTelegram AlertChannelType = "telegram"
	AlertChannelTypeWebhook  AlertChannelType = "webhook"
)

// String возвращает строковое представление AlertChannelType
func (t AlertChannelType) String() string {
	return string(t)
}

// AlertChannelStatus представляет статус канала уведомлений
type AlertChannelStatus string

const (
	AlertChannelStatusUnverified AlertChannelStatus = "unverified"
	AlertChannelStatusActive     AlertChannelStatus = "active"
	AlertChannelStatusFailed     AlertChannelStatus = "failed"
)

// String возвращает строковое представление AlertChannelStatus
func (s AlertChannelStatus) String() string {
	return string(s)
}

// AlertChannel представляет канал уведомлений
type AlertChannel struct {
	ID            uuid.UUID          `db:"id"`
	UserID        uuid.UUID          `db:"user_id"`
	Name          string             `db:"name"`
	Type          AlertChannelType   `db:"type"`
	Status        AlertChannelStatus `db:"status"`
	Enabled       bool               `db:"enabled"`
	Verified      bool               `db:"verified"`
	FailureCount  int                `db:"failure_count"`
	LastFailureAt *time.Time         `db:"last_failure_at"`
	CreatedAt     time.Time          `db:"created_at"`
	UpdatedAt     time.Time          `db:"updated_at"`

	// Type-specific configurations
	TelegramConfig *TelegramChannelConfig `db:"telegram_config"`
	EmailConfig    *EmailChannelConfig    `db:"email_config"`
	WebhookConfig  *WebhookChannelConfig  `db:"webhook_config"`
}

// TelegramChannelConfig представляет конфигурацию для Telegram
type TelegramChannelConfig struct {
	ChatID   string `json:"chat_id"`
	BotToken string `json:"bot_token"`
}

// EmailChannelConfig представляет конфигурацию для Email
type EmailChannelConfig struct {
	Email string `json:"email"`
}

// WebhookChannelConfig представляет конфигурацию для Webhook
type WebhookChannelConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
}

// DeliveryAttemptStatus представляет статус попытки доставки
type DeliveryAttemptStatus string

const (
	DeliveryAttemptStatusPending    DeliveryAttemptStatus = "pending"
	DeliveryAttemptStatusSuccess    DeliveryAttemptStatus = "success"
	DeliveryAttemptStatusFailed     DeliveryAttemptStatus = "failed"
	DeliveryAttemptStatusSuppressed DeliveryAttemptStatus = "suppressed"
)

// String возвращает строковое представление DeliveryAttemptStatus
func (s DeliveryAttemptStatus) String() string {
	return string(s)
}

// DeliveryAttempt представляет попытку доставки уведомления
type DeliveryAttempt struct {
	ID             uuid.UUID             `db:"id"`
	AlertID        uuid.UUID             `db:"alert_id"`
	AlertChannelID uuid.UUID             `db:"alert_channel_id"`
	Status         DeliveryAttemptStatus `db:"status"`
	ErrorMessage   string                `db:"error_message"`
	RetryCount     int                   `db:"retry_count"`
	NextRetryAt    *time.Time            `db:"next_retry_at"`
	CreatedAt      time.Time             `db:"created_at"`
	UpdatedAt      time.Time             `db:"updated_at"`
}

// AlertChannelPriority представляет приоритет канала для правила
type AlertChannelPriority struct {
	ID             uuid.UUID `db:"id"`
	AlertRuleID    uuid.UUID `db:"alert_rule_id"`
	AlertChannelID uuid.UUID `db:"alert_channel_id"`
	Priority       int       `db:"priority"`
	CreatedAt      time.Time `db:"created_at"`
}
