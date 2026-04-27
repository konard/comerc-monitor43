package handler

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	alertproto "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertToProto конвертирует domain.Alert в proto.Alert.
func AlertToProto(alert *domain.Alert) (*alertproto.Alert, error) {
	if alert == nil {
		return nil, nil
	}

	// Note: proto.Alert represents an alert rule configuration, not an alert instance
	// We convert domain status to enabled field - triggered/active means enabled, muted means disabled
	enabled := alert.Status == domain.AlertStatusTriggered || alert.Status == domain.AlertStatusAcknowledged

	protoAlert := &alertproto.Alert{
		Id:                  alert.ID.String(),
		UserId:              alert.UserID.String(),
		MonitorId:           alert.MonitorID.String(),
		Type:                alertTypeToProto(alert.Type),
		Enabled:             enabled,
		ThresholdMs:         int32(alert.ThresholdMs),         // #nosec G115 -- bounded by DB constraints (≤2^31)
		ConsecutiveFailures: int32(alert.ConsecutiveFailures), // #nosec G115 -- bounded by DB constraints (1-5)
		CreatedAt:           timestamppb.New(alert.CreatedAt),
		UpdatedAt:           timestamppb.New(alert.UpdatedAt),
	}

	return protoAlert, nil
}

// ProtoToAlertChannel конвертирует proto.NotificationChannel в domain.AlertChannel.
func ProtoToAlertChannel(protoChannel *alertproto.NotificationChannel, userID uuid.UUID) (*domain.AlertChannel, error) {
	if protoChannel == nil {
		return nil, nil
	}

	channel := &domain.AlertChannel{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      protoToChannelType(protoChannel.Type),
		Status:    domain.AlertChannelStatusUnverified,
		Enabled:   true,
		Verified:  protoChannel.Verified,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Устанавливаем конфигурацию в зависимости от типа
	switch channel.Type {
	case domain.AlertChannelTypeTelegram:
		// Для Telegram адрес используется как chat_id
		channel.TelegramConfig = &domain.TelegramChannelConfig{
			ChatID:   protoChannel.Address,
			BotToken: "", // Будет заполнен при создании
		}
	case domain.AlertChannelTypeEmail:
		channel.EmailConfig = &domain.EmailChannelConfig{
			Email: protoChannel.Address,
		}
	case domain.AlertChannelTypeWebhook:
		channel.WebhookConfig = &domain.WebhookChannelConfig{
			URL:    protoChannel.Address,
			Method: "POST",
		}
	}

	return channel, nil
}

// AlertChannelToProto конвертирует domain.AlertChannel в proto.NotificationChannel.
func AlertChannelToProto(channel *domain.AlertChannel) (*alertproto.NotificationChannel, error) {
	if channel == nil {
		return nil, nil
	}

	protoChannel := &alertproto.NotificationChannel{
		Type:     channelTypeToProto(channel.Type),
		Verified: channel.Verified,
	}

	// Определяем адрес в зависимости от типа канала
	switch channel.Type {
	case domain.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			protoChannel.Address = channel.TelegramConfig.ChatID
		}
	case domain.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			protoChannel.Address = channel.EmailConfig.Email
		}
	case domain.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			protoChannel.Address = channel.WebhookConfig.URL
		}
	}

	return protoChannel, nil
}

// DomainAlertRuleToProto конвертирует domain.AlertRule в proto.AlertRule.
func DomainAlertRuleToProto(rule *domain.AlertRule) (*alertproto.AlertRule, error) {
	if rule == nil {
		return nil, nil
	}

	return &alertproto.AlertRule{
		Id:                  rule.ID.String(),
		UserId:              rule.UserID.String(),
		MonitorId:           rule.MonitorID.String(),
		Enabled:             rule.Enabled,
		ConsecutiveFailures: int32(rule.ConsecutiveFailures), // #nosec G115 -- bounded by DB constraints (1-5)
		CreatedAt:           timestamppb.New(rule.CreatedAt),
		UpdatedAt:           timestamppb.New(rule.UpdatedAt),
	}, nil
}

// Helper functions for status conversion

func protoToAlertType(alertType alertproto.AlertType) domain.AlertType {
	switch alertType {
	case alertproto.AlertType_ALERT_TYPE_STATUS_CODE:
		return domain.AlertTypeStatusCode
	case alertproto.AlertType_ALERT_TYPE_RESPONSE_TIME:
		return domain.AlertTypeResponseTime
	case alertproto.AlertType_ALERT_TYPE_BODY_CONTAINS:
		return domain.AlertTypeBodyContains
	case alertproto.AlertType_ALERT_TYPE_BODY_DOES_NOT_CONTAIN:
		return domain.AlertTypeBodyDoesNotContain
	case alertproto.AlertType_ALERT_TYPE_CERTIFICATE_EXPIRES:
		return domain.AlertTypeCertificateExpires
	case alertproto.AlertType_ALERT_TYPE_SSL_ERROR:
		return domain.AlertTypeSSLError
	default:
		return domain.AlertTypeStatusCode
	}
}

func alertTypeToProto(alertType domain.AlertType) alertproto.AlertType {
	switch alertType {
	case domain.AlertTypeStatusCode:
		return alertproto.AlertType_ALERT_TYPE_STATUS_CODE
	case domain.AlertTypeResponseTime:
		return alertproto.AlertType_ALERT_TYPE_RESPONSE_TIME
	case domain.AlertTypeBodyContains:
		return alertproto.AlertType_ALERT_TYPE_BODY_CONTAINS
	case domain.AlertTypeBodyDoesNotContain:
		return alertproto.AlertType_ALERT_TYPE_BODY_DOES_NOT_CONTAIN
	case domain.AlertTypeCertificateExpires:
		return alertproto.AlertType_ALERT_TYPE_CERTIFICATE_EXPIRES
	case domain.AlertTypeSSLError:
		return alertproto.AlertType_ALERT_TYPE_SSL_ERROR
	default:
		return alertproto.AlertType_ALERT_TYPE_UNSPECIFIED
	}
}

func protoToAlertStatus(status alertproto.AlertStatus) domain.AlertStatus {
	switch status {
	case alertproto.AlertStatus_ALERT_STATUS_ACTIVE:
		return domain.AlertStatusTriggered
	case alertproto.AlertStatus_ALERT_STATUS_DISABLED:
		return domain.AlertStatusMuted
	case alertproto.AlertStatus_ALERT_STATUS_PAUSED:
		return domain.AlertStatusAcknowledged
	default:
		return domain.AlertStatusTriggered
	}
}

func alertStatusToProto(status domain.AlertStatus) alertproto.AlertStatus {
	switch status {
	case domain.AlertStatusTriggered:
		return alertproto.AlertStatus_ALERT_STATUS_ACTIVE
	case domain.AlertStatusMuted:
		return alertproto.AlertStatus_ALERT_STATUS_DISABLED
	case domain.AlertStatusAcknowledged:
		return alertproto.AlertStatus_ALERT_STATUS_PAUSED
	case domain.AlertStatusResolved:
		// Resolved не имеет прямого аналога в proto; UNSPECIFIED отличает от DISABLED (muted)
		return alertproto.AlertStatus_ALERT_STATUS_UNSPECIFIED
	default:
		return alertproto.AlertStatus_ALERT_STATUS_UNSPECIFIED
	}
}

func protoToChannelType(channelType alertproto.NotificationChannelType) domain.AlertChannelType {
	switch channelType {
	case alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL:
		return domain.AlertChannelTypeEmail
	case alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM:
		return domain.AlertChannelTypeTelegram
	case alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK:
		return domain.AlertChannelTypeWebhook
	case alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_SMS:
		return domain.AlertChannelTypeWebhook // Fallback to webhook
	default:
		return domain.AlertChannelTypeEmail // Default
	}
}

// channelToProto конвертирует domain.AlertChannel в proto.AlertChannel (single-channel API).
func channelToProto(channel *domain.AlertChannel) (*alertproto.AlertChannel, error) {
	if channel == nil {
		return nil, nil
	}

	cfgBytes, err := marshalChannelConfig(channel)
	if err != nil {
		return nil, err
	}

	out := &alertproto.AlertChannel{
		Id:           channel.ID.String(),
		UserId:       channel.UserID.String(),
		Name:         channel.Name,
		Type:         channelTypeToString(channel.Type),
		Config:       string(cfgBytes),
		Enabled:      channel.Enabled,
		Verified:     channel.Verified,
		Status:       channelStatusToString(channel.Status),
		FailureCount: int32(channel.FailureCount), // #nosec G115 -- bounded
		CreatedAt:    timestamppb.New(channel.CreatedAt),
		UpdatedAt:    timestamppb.New(channel.UpdatedAt),
	}
	return out, nil
}

// channelTypeToString возвращает upper-case строку типа канала для proto.
func channelTypeToString(t domain.AlertChannelType) string {
	switch t {
	case domain.AlertChannelTypeTelegram:
		return "TELEGRAM"
	case domain.AlertChannelTypeEmail:
		return "EMAIL"
	case domain.AlertChannelTypeWebhook:
		return "WEBHOOK"
	default:
		return ""
	}
}

// stringToChannelType разбирает строковое имя типа канала.
func stringToChannelType(s string) (domain.AlertChannelType, bool) {
	switch s {
	case "TELEGRAM", "telegram":
		return domain.AlertChannelTypeTelegram, true
	case "EMAIL", "email":
		return domain.AlertChannelTypeEmail, true
	case "WEBHOOK", "webhook":
		return domain.AlertChannelTypeWebhook, true
	default:
		return "", false
	}
}

// channelStatusToString возвращает строковое представление статуса канала.
func channelStatusToString(s domain.AlertChannelStatus) string {
	switch s {
	case domain.AlertChannelStatusActive:
		return "ACTIVE"
	case domain.AlertChannelStatusFailed:
		return "FAILED"
	case domain.AlertChannelStatusUnverified:
		return "UNVERIFIED"
	default:
		return ""
	}
}

// marshalChannelConfig сериализует type-specific config канала в JSON.
func marshalChannelConfig(channel *domain.AlertChannel) ([]byte, error) {
	switch channel.Type {
	case domain.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			return json.Marshal(channel.TelegramConfig)
		}
	case domain.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			return json.Marshal(channel.EmailConfig)
		}
	case domain.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			return json.Marshal(channel.WebhookConfig)
		}
	}
	return []byte("{}"), nil
}

// applyChannelConfigJSON десериализует JSON-конфигурацию в нужный type-specific блок.
func applyChannelConfigJSON(channel *domain.AlertChannel, cfg string) error {
	if cfg == "" {
		return nil
	}
	switch channel.Type {
	case domain.AlertChannelTypeTelegram:
		var c domain.TelegramChannelConfig
		if err := json.Unmarshal([]byte(cfg), &c); err != nil {
			return err
		}
		channel.TelegramConfig = &c
	case domain.AlertChannelTypeEmail:
		var c domain.EmailChannelConfig
		if err := json.Unmarshal([]byte(cfg), &c); err != nil {
			return err
		}
		channel.EmailConfig = &c
	case domain.AlertChannelTypeWebhook:
		var c domain.WebhookChannelConfig
		if err := json.Unmarshal([]byte(cfg), &c); err != nil {
			return err
		}
		if c.Method == "" {
			c.Method = "POST"
		}
		channel.WebhookConfig = &c
	}
	return nil
}

func channelTypeToProto(channelType domain.AlertChannelType) alertproto.NotificationChannelType {
	switch channelType {
	case domain.AlertChannelTypeEmail:
		return alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL
	case domain.AlertChannelTypeTelegram:
		return alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM
	case domain.AlertChannelTypeWebhook:
		return alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK
	default:
		return alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED
	}
}
