package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
	alertproto "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

func mustParseTestUUID(value string) uuid.UUID {
	return uuid.MustParse(value)
}

// TestAlertToProto_ValidInput проверяет корректную конвертацию domain.Alert в proto.Alert
func TestAlertToProto_ValidInput(t *testing.T) {
	alertID := mustParseTestUUID("550e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")
	monitorID := mustParseTestUUID("770e8400-e29b-41d4-a716-446655440000")
	now := time.Now()

	alert := &model.Alert{
		ID:                  alertID,
		UserID:              userID,
		MonitorID:           monitorID,
		Type:                model.AlertTypeStatusCode,
		Status:              model.AlertStatusTriggered,
		Enabled:             true,
		ConsecutiveFailures: 3,
		ThresholdMs:         5000,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	protoAlert, err := AlertToProto(alert)

	require.NoError(t, err)
	require.NotNil(t, protoAlert)
	assert.Equal(t, alertID.String(), protoAlert.Id)
	assert.Equal(t, userID.String(), protoAlert.UserId)
	assert.Equal(t, monitorID.String(), protoAlert.MonitorId)
	assert.Equal(t, alertproto.AlertType_ALERT_TYPE_STATUS_CODE, protoAlert.Type)
	assert.Equal(t, true, protoAlert.Enabled)
	assert.Equal(t, int32(3), protoAlert.ConsecutiveFailures)
	assert.Equal(t, int32(5000), protoAlert.ThresholdMs)
	assert.Equal(t, now.Unix(), protoAlert.CreatedAt.Seconds)
	assert.Equal(t, now.Unix(), protoAlert.UpdatedAt.Seconds)
}

// TestAlertToProto_AcknowledgedStatus проверяет конвертацию со статусом ACKNOWLEDGED
func TestAlertToProto_AcknowledgedStatus(t *testing.T) {
	alertID := mustParseTestUUID("550e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")
	monitorID := mustParseTestUUID("770e8400-e29b-41d4-a716-446655440000")

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusAcknowledged,
	}

	protoAlert, err := AlertToProto(alert)

	require.NoError(t, err)
	require.NotNil(t, protoAlert)
	assert.Equal(t, true, protoAlert.Enabled) // Acknowledged means enabled in proto
}

// TestAlertToProto_MutedStatus проверяет конвертацию со статусом MUTED
func TestAlertToProto_MutedStatus(t *testing.T) {
	alertID := mustParseTestUUID("550e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")
	monitorID := mustParseTestUUID("770e8400-e29b-41d4-a716-446655440000")

	alert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: monitorID,
		Type:      model.AlertTypeStatusCode,
		Status:    model.AlertStatusMuted,
	}

	protoAlert, err := AlertToProto(alert)

	require.NoError(t, err)
	require.NotNil(t, protoAlert)
	assert.Equal(t, false, protoAlert.Enabled) // Muted means disabled in proto
}

// TestAlertToProto_NilInput проверяет обработку nil
func TestAlertToProto_NilInput(t *testing.T) {
	protoAlert, err := AlertToProto(nil)

	assert.NoError(t, err)
	assert.Nil(t, protoAlert)
}

// TestAlertToProto_AllAlertTypes проверяет конвертацию всех типов алертов в proto
func TestAlertToProto_AllAlertTypes(t *testing.T) {
	alertID := mustParseTestUUID("550e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")
	monitorID := mustParseTestUUID("770e8400-e29b-41d4-a716-446655440000")

	baseAlert := &model.Alert{
		ID:        alertID,
		UserID:    userID,
		MonitorID: monitorID,
		Status:    model.AlertStatusTriggered,
		Enabled:   true,
	}

	testCases := []struct {
		domainType   model.AlertType
		expectedType alertproto.AlertType
	}{
		{model.AlertTypeStatusCode, alertproto.AlertType_ALERT_TYPE_STATUS_CODE},
		{model.AlertTypeResponseTime, alertproto.AlertType_ALERT_TYPE_RESPONSE_TIME},
		{model.AlertTypeBodyContains, alertproto.AlertType_ALERT_TYPE_BODY_CONTAINS},
		{model.AlertTypeBodyDoesNotContain, alertproto.AlertType_ALERT_TYPE_BODY_DOES_NOT_CONTAIN},
		{model.AlertTypeCertificateExpires, alertproto.AlertType_ALERT_TYPE_CERTIFICATE_EXPIRES},
		{model.AlertTypeSSLError, alertproto.AlertType_ALERT_TYPE_SSL_ERROR},
	}

	for _, tc := range testCases {
		t.Run(tc.domainType.String(), func(t *testing.T) {
			baseAlert.Type = tc.domainType
			protoAlert, err := AlertToProto(baseAlert)

			require.NoError(t, err)
			require.NotNil(t, protoAlert)
			assert.Equal(t, tc.expectedType, protoAlert.Type)
		})
	}
}

// TestProtoToAlertChannel_Email проверяет конвертацию Email канала
func TestProtoToAlertChannel_Email(t *testing.T) {
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	protoChannel := &alertproto.NotificationChannel{
		Type:     alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL,
		Address:  "user@example.com",
		Verified: true,
	}

	channel, err := ProtoToAlertChannel(protoChannel, userID)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, model.AlertChannelTypeEmail, channel.Type)
	assert.Equal(t, "user@example.com", channel.EmailConfig.Email)
	assert.Equal(t, model.AlertChannelStatusUnverified, channel.Status)
	assert.Equal(t, true, channel.Enabled)
	assert.Equal(t, true, channel.Verified)
}

// TestProtoToAlertChannel_Telegram проверяет конвертацию Telegram канала
func TestProtoToAlertChannel_Telegram(t *testing.T) {
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	protoChannel := &alertproto.NotificationChannel{
		Type:     alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM,
		Address:  "123456789",
		Verified: true,
	}

	channel, err := ProtoToAlertChannel(protoChannel, userID)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, model.AlertChannelTypeTelegram, channel.Type)
	assert.Equal(t, "123456789", channel.TelegramConfig.ChatID)
	assert.Equal(t, true, channel.Verified)
}

// TestProtoToAlertChannel_Webhook проверяет конвертацию Webhook канала
func TestProtoToAlertChannel_Webhook(t *testing.T) {
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	protoChannel := &alertproto.NotificationChannel{
		Type:     alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK,
		Address:  "https://example.com/webhook",
		Verified: true,
	}

	channel, err := ProtoToAlertChannel(protoChannel, userID)

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, model.AlertChannelTypeWebhook, channel.Type)
	assert.Equal(t, "https://example.com/webhook", channel.WebhookConfig.URL)
	assert.Equal(t, "POST", channel.WebhookConfig.Method)
	assert.Equal(t, true, channel.Verified)
}

// TestProtoToAlertChannel_AllChannelTypes проверяет конвертацию всех типов каналов
func TestProtoToAlertChannel_AllChannelTypes(t *testing.T) {
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	testCases := []struct {
		protoType      alertproto.NotificationChannelType
		expectedType   model.AlertChannelType
		expectedConfig any
	}{
		{
			alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL,
			model.AlertChannelTypeEmail,
			&model.EmailChannelConfig{Email: "user@example.com"},
		},
		{
			alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM,
			model.AlertChannelTypeTelegram,
			&model.TelegramChannelConfig{ChatID: "123456789"},
		},
		{
			alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK,
			model.AlertChannelTypeWebhook,
			&model.WebhookChannelConfig{URL: "https://example.com/webhook", Method: "POST"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedType.String(), func(t *testing.T) {
			protoChannel := &alertproto.NotificationChannel{
				Type:     tc.protoType,
				Address:  "test@example.com",
				Verified: true,
			}

			channel, err := ProtoToAlertChannel(protoChannel, userID)

			require.NoError(t, err)
			require.NotNil(t, channel)
			assert.Equal(t, tc.expectedType, channel.Type)
		})
	}
}

// TestProtoToAlertChannel_NilInput проверяет обработку nil
func TestProtoToAlertChannel_NilInput(t *testing.T) {
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	channel, err := ProtoToAlertChannel(nil, userID)

	assert.NoError(t, err)
	assert.Nil(t, channel)
}

// TestAlertChannelToProto_Email проверяет конвертацию Email канала в proto
func TestAlertChannelToProto_Email(t *testing.T) {
	channelID := mustParseTestUUID("880e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	channel := &model.AlertChannel{
		ID:     channelID,
		UserID: userID,
		Type:   model.AlertChannelTypeEmail,
		EmailConfig: &model.EmailChannelConfig{
			Email: "user@example.com",
		},
		Verified: true,
	}

	protoChannel, err := AlertChannelToProto(channel)

	require.NoError(t, err)
	require.NotNil(t, protoChannel)
	assert.Equal(t, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL, protoChannel.Type)
	assert.Equal(t, "user@example.com", protoChannel.Address)
	assert.Equal(t, true, protoChannel.Verified)
}

// TestAlertChannelToProto_Telegram проверяет конвертацию Telegram канала в proto
func TestAlertChannelToProto_Telegram(t *testing.T) {
	channelID := mustParseTestUUID("880e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	channel := &model.AlertChannel{
		ID:     channelID,
		UserID: userID,
		Type:   model.AlertChannelTypeTelegram,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "123456789",
		},
		Verified: true,
	}

	protoChannel, err := AlertChannelToProto(channel)

	require.NoError(t, err)
	require.NotNil(t, protoChannel)
	assert.Equal(t, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM, protoChannel.Type)
	assert.Equal(t, "123456789", protoChannel.Address)
	assert.Equal(t, true, protoChannel.Verified)
}

// TestAlertChannelToProto_Webhook проверяет конвертацию Webhook канала в proto
func TestAlertChannelToProto_Webhook(t *testing.T) {
	channelID := mustParseTestUUID("880e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	channel := &model.AlertChannel{
		ID:     channelID,
		UserID: userID,
		Type:   model.AlertChannelTypeWebhook,
		WebhookConfig: &model.WebhookChannelConfig{
			URL:    "https://example.com/webhook",
			Method: "POST",
		},
		Verified: true,
	}

	protoChannel, err := AlertChannelToProto(channel)

	require.NoError(t, err)
	require.NotNil(t, protoChannel)
	assert.Equal(t, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK, protoChannel.Type)
	assert.Equal(t, "https://example.com/webhook", protoChannel.Address)
	assert.Equal(t, true, protoChannel.Verified)
}

// TestAlertChannelToProto_NilConfig проверяет конвертацию с nil конфигом
func TestAlertChannelToProto_NilConfig(t *testing.T) {
	channelID := mustParseTestUUID("880e8400-e29b-41d4-a716-446655440000")
	userID := mustParseTestUUID("660e8400-e29b-41d4-a716-446655440000")

	channel := &model.AlertChannel{
		ID:     channelID,
		UserID: userID,
		Type:   model.AlertChannelTypeEmail,
		// EmailConfig is nil
		Verified: true,
	}

	protoChannel, err := AlertChannelToProto(channel)

	require.NoError(t, err)
	require.NotNil(t, protoChannel)
	assert.Equal(t, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL, protoChannel.Type)
	assert.Equal(t, "", protoChannel.Address) // Empty address when config is nil
	assert.Equal(t, true, protoChannel.Verified)
}

// TestAlertChannelToProto_NilInput проверяет обработку nil
func TestAlertChannelToProto_NilInput(t *testing.T) {
	protoChannel, err := AlertChannelToProto(nil)

	assert.NoError(t, err)
	assert.Nil(t, protoChannel)
}

// TestProtoToAlertType_AllTypes проверяет конвертацию всех типов алертов из proto
func TestProtoToAlertType_AllTypes(t *testing.T) {
	testCases := []struct {
		protoType    alertproto.AlertType
		expectedType model.AlertType
	}{
		{alertproto.AlertType_ALERT_TYPE_STATUS_CODE, model.AlertTypeStatusCode},
		{alertproto.AlertType_ALERT_TYPE_RESPONSE_TIME, model.AlertTypeResponseTime},
		{alertproto.AlertType_ALERT_TYPE_BODY_CONTAINS, model.AlertTypeBodyContains},
		{alertproto.AlertType_ALERT_TYPE_BODY_DOES_NOT_CONTAIN, model.AlertTypeBodyDoesNotContain},
		{alertproto.AlertType_ALERT_TYPE_CERTIFICATE_EXPIRES, model.AlertTypeCertificateExpires},
		{alertproto.AlertType_ALERT_TYPE_SSL_ERROR, model.AlertTypeSSLError},
		{alertproto.AlertType_ALERT_TYPE_UNSPECIFIED, model.AlertTypeStatusCode}, // Default
	}

	for _, tc := range testCases {
		t.Run(tc.expectedType.String(), func(t *testing.T) {
			result := protoToAlertType(tc.protoType)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

// TestAlertTypeToProto_AllTypes проверяет конвертацию всех типов алертов в proto
func TestAlertTypeToProto_AllTypes(t *testing.T) {
	testCases := []struct {
		domainType   model.AlertType
		expectedType alertproto.AlertType
	}{
		{model.AlertTypeStatusCode, alertproto.AlertType_ALERT_TYPE_STATUS_CODE},
		{model.AlertTypeResponseTime, alertproto.AlertType_ALERT_TYPE_RESPONSE_TIME},
		{model.AlertTypeBodyContains, alertproto.AlertType_ALERT_TYPE_BODY_CONTAINS},
		{model.AlertTypeBodyDoesNotContain, alertproto.AlertType_ALERT_TYPE_BODY_DOES_NOT_CONTAIN},
		{model.AlertTypeCertificateExpires, alertproto.AlertType_ALERT_TYPE_CERTIFICATE_EXPIRES},
		{model.AlertTypeSSLError, alertproto.AlertType_ALERT_TYPE_SSL_ERROR},
		{model.AlertType("unknown"), alertproto.AlertType_ALERT_TYPE_UNSPECIFIED}, // Default to UNSPECIFIED
	}

	for _, tc := range testCases {
		t.Run(tc.domainType.String(), func(t *testing.T) {
			result := alertTypeToProto(tc.domainType)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

// TestProtoToAlertStatus_AllStatuses проверяет конвертацию всех статусов из proto
func TestProtoToAlertStatus_AllStatuses(t *testing.T) {
	testCases := []struct {
		protoStatus    alertproto.AlertStatus
		expectedStatus model.AlertStatus
	}{
		{alertproto.AlertStatus_ALERT_STATUS_ACTIVE, model.AlertStatusTriggered},
		{alertproto.AlertStatus_ALERT_STATUS_DISABLED, model.AlertStatusMuted},
		{alertproto.AlertStatus_ALERT_STATUS_PAUSED, model.AlertStatusAcknowledged},
		{alertproto.AlertStatus_ALERT_STATUS_UNSPECIFIED, model.AlertStatusTriggered}, // Default
	}

	for _, tc := range testCases {
		t.Run(tc.expectedStatus.String(), func(t *testing.T) {
			result := protoToAlertStatus(tc.protoStatus)
			assert.Equal(t, tc.expectedStatus, result)
		})
	}
}

// TestAlertStatusToProto_AllStatuses проверяет конвертацию всех статусов в proto
func TestAlertStatusToProto_AllStatuses(t *testing.T) {
	testCases := []struct {
		domainStatus   model.AlertStatus
		expectedStatus alertproto.AlertStatus
	}{
		{model.AlertStatusTriggered, alertproto.AlertStatus_ALERT_STATUS_ACTIVE},
		{model.AlertStatusMuted, alertproto.AlertStatus_ALERT_STATUS_DISABLED},
		{model.AlertStatusAcknowledged, alertproto.AlertStatus_ALERT_STATUS_PAUSED},
		{model.AlertStatusResolved, alertproto.AlertStatus_ALERT_STATUS_UNSPECIFIED},
		{model.AlertStatus("unknown"), alertproto.AlertStatus_ALERT_STATUS_UNSPECIFIED}, // Default to UNSPECIFIED
	}

	for _, tc := range testCases {
		t.Run(tc.domainStatus.String(), func(t *testing.T) {
			result := alertStatusToProto(tc.domainStatus)
			assert.Equal(t, tc.expectedStatus, result)
		})
	}
}

// TestProtoToChannelType_AllTypes проверяет конвертацию всех типов каналов из proto
func TestProtoToChannelType_AllTypes(t *testing.T) {
	testCases := []struct {
		protoType    alertproto.NotificationChannelType
		expectedType model.AlertChannelType
	}{
		{alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL, model.AlertChannelTypeEmail},
		{alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM, model.AlertChannelTypeTelegram},
		{alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK, model.AlertChannelTypeWebhook},
		{alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_SMS, model.AlertChannelTypeWebhook},       // Fallback
		{alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED, model.AlertChannelTypeEmail}, // Default
	}

	for _, tc := range testCases {
		t.Run(tc.expectedType.String(), func(t *testing.T) {
			result := protoToChannelType(tc.protoType)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

// TestChannelTypeToProto_AllTypes проверяет конвертацию всех типов каналов в proto
func TestChannelTypeToProto_AllTypes(t *testing.T) {
	testCases := []struct {
		domainType   model.AlertChannelType
		expectedType alertproto.NotificationChannelType
	}{
		{model.AlertChannelTypeEmail, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL},
		{model.AlertChannelTypeTelegram, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM},
		{model.AlertChannelTypeWebhook, alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK},
		{model.AlertChannelType("unknown"), alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED}, // Default to UNSPECIFIED
	}

	for _, tc := range testCases {
		t.Run(tc.domainType.String(), func(t *testing.T) {
			result := channelTypeToProto(tc.domainType)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}
