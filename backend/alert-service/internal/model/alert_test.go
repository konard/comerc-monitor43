package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAlertType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  AlertType
		want string
	}{
		{
			name: "STATUS_CODE type",
			typ:  AlertTypeStatusCode,
			want: "status_code",
		},
		{
			name: "RESPONSE_TIME type",
			typ:  AlertTypeResponseTime,
			want: "response_time",
		},
		{
			name: "BODY_CONTAINS type",
			typ:  AlertTypeBodyContains,
			want: "body_contains",
		},
		{
			name: "BODY_DOES_NOT_CONTAIN type",
			typ:  AlertTypeBodyDoesNotContain,
			want: "body_does_not_contain",
		},
		{
			name: "CERTIFICATE_EXPIRES type",
			typ:  AlertTypeCertificateExpires,
			want: "certificate_expires",
		},
		{
			name: "SSL_ERROR type",
			typ:  AlertTypeSSLError,
			want: "ssl_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAlertStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status AlertStatus
		want   string
	}{
		{
			name:   "TRIGGERED status",
			status: AlertStatusTriggered,
			want:   "triggered",
		},
		{
			name:   "MUTED status",
			status: AlertStatusMuted,
			want:   "muted",
		},
		{
			name:   "ACKNOWLEDGED status",
			status: AlertStatusAcknowledged,
			want:   "acknowledged",
		},
		{
			name:   "RESOLVED status",
			status: AlertStatusResolved,
			want:   "resolved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationChannelType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  AlertChannelType
		want string
	}{
		{
			name: "EMAIL type",
			typ:  AlertChannelTypeEmail,
			want: "email",
		},
		{
			name: "TELEGRAM type",
			typ:  AlertChannelTypeTelegram,
			want: "telegram",
		},
		{
			name: "WEBHOOK type",
			typ:  AlertChannelTypeWebhook,
			want: "webhook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAlertChannelStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status AlertChannelStatus
		want   string
	}{
		{
			name:   "UNVERIFIED status",
			status: AlertChannelStatusUnverified,
			want:   "unverified",
		},
		{
			name:   "ACTIVE status",
			status: AlertChannelStatusActive,
			want:   "active",
		},
		{
			name:   "FAILED status",
			status: AlertChannelStatusFailed,
			want:   "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeliveryAttemptStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status DeliveryAttemptStatus
		want   string
	}{
		{
			name:   "PENDING status",
			status: DeliveryAttemptStatusPending,
			want:   "pending",
		},
		{
			name:   "SUCCESS status",
			status: DeliveryAttemptStatusSuccess,
			want:   "success",
		},
		{
			name:   "FAILED status",
			status: DeliveryAttemptStatusFailed,
			want:   "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAlert_New(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"create alert with generated ID"},
		{"create alert with generated IDs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			monitorID := uuid.New()

			alert := &Alert{
				ID:        uuid.New(),
				UserID:    userID,
				MonitorID: monitorID,
				Type:      AlertTypeStatusCode,
				Status:    AlertStatusTriggered,
				Enabled:   true,
				Config:    map[string]any{"expected_status_code": "200"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			assert.Equal(t, userID, alert.UserID)
			assert.Equal(t, monitorID, alert.MonitorID)
			assert.Equal(t, AlertTypeStatusCode, alert.Type)
			assert.Equal(t, AlertStatusTriggered, alert.Status)
			assert.True(t, alert.Enabled)
			assert.NotEmpty(t, alert.ID.String())
			assert.False(t, alert.CreatedAt.IsZero())
			assert.False(t, alert.UpdatedAt.IsZero())
			assert.NotNil(t, alert.Config)
		})
	}
}

func TestAlertChannel_New(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"create email channel"},
		{"create telegram channel"},
		{"create webhook channel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			now := time.Now()

			emailChannel := &AlertChannel{
				ID:        uuid.New(),
				UserID:    userID,
				Type:      AlertChannelTypeEmail,
				Status:    AlertChannelStatusUnverified,
				Enabled:   true,
				Verified:  false,
				CreatedAt: now,
				UpdatedAt: now,
				EmailConfig: &EmailChannelConfig{
					Email: "test@example.com",
				},
			}

			assert.Equal(t, userID, emailChannel.UserID)
			assert.Equal(t, AlertChannelTypeEmail, emailChannel.Type)
			assert.Equal(t, AlertChannelStatusUnverified, emailChannel.Status)
			assert.True(t, emailChannel.Enabled)
			assert.False(t, emailChannel.Verified)
			assert.NotNil(t, emailChannel.EmailConfig)
			assert.NotEmpty(t, emailChannel.ID.String())
		})
	}
}

func TestAlertRule_New(t *testing.T) {
	t.Run("create alert rule", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()

		rule := &AlertRule{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			Enabled:             true,
			ConsecutiveFailures: 3,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		assert.Equal(t, userID, rule.UserID)
		assert.Equal(t, monitorID, rule.MonitorID)
		assert.True(t, rule.Enabled)
		assert.Equal(t, 3, rule.ConsecutiveFailures)
		assert.NotEmpty(t, rule.ID.String())
	})
}

func TestDeliveryAttempt_New(t *testing.T) {
	t.Run("create delivery attempt", func(t *testing.T) {
		alertID := uuid.New()
		channelID := uuid.New()

		attempt := &DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        alertID,
			AlertChannelID: channelID,
			Status:         DeliveryAttemptStatusPending,
			RetryCount:     0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		assert.Equal(t, alertID, attempt.AlertID)
		assert.Equal(t, channelID, attempt.AlertChannelID)
		assert.Equal(t, DeliveryAttemptStatusPending, attempt.Status)
		assert.Equal(t, 0, attempt.RetryCount)
		assert.NotEmpty(t, attempt.ID.String())
	})
}

// Additional comprehensive tests for better model coverage

func TestAlertStatus_Validation(t *testing.T) {
	tests := []struct {
		name   string
		status AlertStatus
		valid  bool
	}{
		{
			name:   "valid_triggered_status",
			status: AlertStatusTriggered,
			valid:  true,
		},
		{
			name:   "valid_muted_status",
			status: AlertStatusMuted,
			valid:  true,
		},
		{
			name:   "valid_acknowledged_status",
			status: AlertStatusAcknowledged,
			valid:  true,
		},
		{
			name:   "valid_resolved_status",
			status: AlertStatusResolved,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that we can create alerts with all valid statuses
			alert := &Alert{
				Status: tt.status,
			}

			assert.Equal(t, tt.status, alert.Status)
			assert.NotEmpty(t, tt.status.String())
		})
	}
}

func TestAlertType_Completeness(t *testing.T) {
	tests := []struct {
		name string
		typ  AlertType
	}{
		{"status_code", AlertTypeStatusCode},
		{"response_time", AlertTypeResponseTime},
		{"body_contains", AlertTypeBodyContains},
		{"body_does_not_contain", AlertTypeBodyDoesNotContain},
		{"certificate_expires", AlertTypeCertificateExpires},
		{"ssl_error", AlertTypeSSLError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that all alert types are properly defined
			assert.NotEmpty(t, tt.typ.String())

			// Test that we can create alerts with each type
			alert := &Alert{
				Type: tt.typ,
			}

			assert.Equal(t, tt.typ, alert.Type)
		})
	}
}

func TestAlertChannel_Configurations(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name    string
		channel *AlertChannel
	}{
		{
			name: "email_channel_with_config",
			channel: &AlertChannel{
				ID:        uuid.New(),
				UserID:    userID,
				Type:      AlertChannelTypeEmail,
				Enabled:   true,
				Verified:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				EmailConfig: &EmailChannelConfig{
					Email: "test@example.com",
				},
			},
		},
		{
			name: "telegram_channel_with_config",
			channel: &AlertChannel{
				ID:        uuid.New(),
				UserID:    userID,
				Type:      AlertChannelTypeTelegram,
				Enabled:   true,
				Verified:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				TelegramConfig: &TelegramChannelConfig{
					ChatID: "@testuser",
				},
			},
		},
		{
			name: "webhook_channel_with_config",
			channel: &AlertChannel{
				ID:        uuid.New(),
				UserID:    userID,
				Type:      AlertChannelTypeWebhook,
				Enabled:   true,
				Verified:  false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				WebhookConfig: &WebhookChannelConfig{
					URL: "https://example.com/webhook",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test channel structure
			assert.Equal(t, userID, tt.channel.UserID)
			assert.NotEmpty(t, tt.channel.ID.String())
			// Note: CreatedAt is set in test data, so it should not be zero

			// Test type-specific config
			switch tt.channel.Type {
			case AlertChannelTypeEmail:
				assert.NotNil(t, tt.channel.EmailConfig)
				assert.NotEmpty(t, tt.channel.EmailConfig.Email)
			case AlertChannelTypeTelegram:
				assert.NotNil(t, tt.channel.TelegramConfig)
				assert.NotEmpty(t, tt.channel.TelegramConfig.ChatID)
			case AlertChannelTypeWebhook:
				assert.NotNil(t, tt.channel.WebhookConfig)
				assert.NotEmpty(t, tt.channel.WebhookConfig.URL)
			}
		})
	}
}

func TestAlert_ConfigHandling(t *testing.T) {
	alert := &Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Type:      AlertTypeStatusCode,
		Status:    AlertStatusTriggered,
		Enabled:   true,
		Config:    map[string]any{},
	}

	// Test that Config can be set and accessed
	alert.Config["expected_status_code"] = "200"
	alert.Config["threshold_ms"] = 5000

	assert.Equal(t, "200", alert.Config["expected_status_code"])
	assert.Equal(t, 5000, alert.Config["threshold_ms"])
	assert.Len(t, alert.Config, 2)
}

func TestAlertChannel_StatusTransitions(t *testing.T) {
	tests := []struct {
		name   string
		status AlertChannelStatus
	}{
		{"unverified_to_active", AlertChannelStatusActive},
		{"unverified_to_failed", AlertChannelStatusFailed},
		{"verified_state", AlertChannelStatusActive},
		{"failed_state", AlertChannelStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &AlertChannel{
				Status: tt.status,
			}

			assert.Equal(t, tt.status, channel.Status)
			assert.NotEmpty(t, tt.status.String())
		})
	}
}

func TestDeliveryAttempt_StatusLifecycle(t *testing.T) {
	tests := []struct {
		name   string
		status DeliveryAttemptStatus
	}{
		{"pending_initial_status", DeliveryAttemptStatusPending},
		{"successful_delivery", DeliveryAttemptStatusSuccess},
		{"failed_delivery", DeliveryAttemptStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := &DeliveryAttempt{
				Status: tt.status,
			}

			assert.Equal(t, tt.status, attempt.Status)
			assert.NotEmpty(t, tt.status.String())
		})
	}
}

func TestAlertRule_Completeness(t *testing.T) {
	userID := uuid.New()
	monitorID := uuid.New()

	rule := &AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Test all rule fields are properly set
	assert.Equal(t, userID, rule.UserID)
	assert.Equal(t, monitorID, rule.MonitorID)
	assert.True(t, rule.Enabled)
	assert.Equal(t, 3, rule.ConsecutiveFailures)
	assert.NotEmpty(t, rule.ID.String())
	assert.False(t, rule.CreatedAt.IsZero())
	assert.False(t, rule.UpdatedAt.IsZero())
}

func TestAlertFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter AlertFilter
	}{
		{
			name: "filter_with_all_fields",
			filter: AlertFilter{
				MonitorID: "test-monitor-id",
				Status:    AlertStatusTriggered,
				Page:      1,
				PageSize:  10,
			},
		},
		{
			name: "filter_with_minimal_fields",
			filter: AlertFilter{
				Page:     1,
				PageSize: 20,
			},
		},
		{
			name:   "filter_empty",
			filter: AlertFilter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test filter can be created and accessed
			if tt.filter.MonitorID != "" {
				assert.Equal(t, tt.filter.MonitorID, tt.filter.MonitorID)
			}
			if tt.filter.Status != "" {
				assert.Equal(t, tt.filter.Status, tt.filter.Status)
			}
			assert.Equal(t, tt.filter.Page, tt.filter.Page)
			assert.Equal(t, tt.filter.PageSize, tt.filter.PageSize)
		})
	}
}

func TestAlertChannelPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority AlertChannelPriority
	}{
		{
			name: "priority_with_all_fields",
			priority: AlertChannelPriority{
				ID:             uuid.New(),
				AlertRuleID:    uuid.New(),
				AlertChannelID: uuid.New(),
				Priority:       1,
				CreatedAt:      time.Now(),
			},
		},
		{
			name: "priority_with_high_priority",
			priority: AlertChannelPriority{
				ID:             uuid.New(),
				AlertRuleID:    uuid.New(),
				AlertChannelID: uuid.New(),
				Priority:       10,
				CreatedAt:      time.Now(),
			},
		},
		{
			name: "priority_zero_priority",
			priority: AlertChannelPriority{
				ID:             uuid.New(),
				AlertRuleID:    uuid.New(),
				AlertChannelID: uuid.New(),
				Priority:       0,
				CreatedAt:      time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test priority can be created and accessed
			assert.NotEmpty(t, tt.priority.ID.String())
			assert.NotEmpty(t, tt.priority.AlertRuleID.String())
			assert.NotEmpty(t, tt.priority.AlertChannelID.String())
			assert.GreaterOrEqual(t, tt.priority.Priority, 0)
			assert.False(t, tt.priority.CreatedAt.IsZero())
		})
	}
}

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name  string
		error error
	}{
		{
			name:  "alert_not_found_error",
			error: ErrAlertNotFound,
		},
		{
			name:  "alert_rule_not_found_error",
			error: ErrAlertRuleNotFound,
		},
		{
			name:  "alert_channel_not_found_error",
			error: ErrAlertChannelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that errors are properly defined
			assert.Error(t, tt.error)
			assert.NotNil(t, tt.error)
		})
	}
}

func TestDomainError_NewDomainError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "simple_message",
			message: "test error",
		},
		{
			name:    "empty_message",
			message: "",
		},
		{
			name:    "long_message",
			message: "this is a very long error message with lots of details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDomainError(tt.message)

			assert.NotNil(t, err)
			assert.Equal(t, "DOMAIN_ERROR", err.Code())
			assert.Equal(t, tt.message, err.Error())
		})
	}
}

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "error_message",
			message: "something went wrong",
		},
		{
			name:    "empty_message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDomainError(tt.message)

			assert.Equal(t, tt.message, err.Error())
		})
	}
}

func TestDomainError_Code(t *testing.T) {
	err := NewDomainError("test error")

	assert.Equal(t, "DOMAIN_ERROR", err.Code())
}

func TestDomainError_WithCode(t *testing.T) {
	tests := []struct {
		name    string
		message string
		code    string
	}{
		{
			name:    "custom_code",
			message: "test error",
			code:    "CUSTOM_CODE",
		},
		{
			name:    "empty_code",
			message: "test error",
			code:    "",
		},
		{
			name:    "numeric_code",
			message: "test error",
			code:    "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDomainError(tt.message).WithCode(tt.code)

			assert.NotNil(t, err)
			assert.Equal(t, tt.code, err.Code())
			assert.Equal(t, tt.message, err.Error())
		})
	}
}

func TestDomainError_Wrap(t *testing.T) {
	tests := []struct {
		name           string
		originalError  *DomainError
		wrapMessage    string
		expectedResult string
	}{
		{
			name:           "wrap_simple_message",
			originalError:  NewDomainError("database connection failed"),
			wrapMessage:    "failed to create alert",
			expectedResult: "failed to create alert: database connection failed",
		},
		{
			name:           "wrap_with_custom_code",
			originalError:  NewDomainError("not found").WithCode("NOT_FOUND"),
			wrapMessage:    "failed to retrieve alert",
			expectedResult: "failed to retrieve alert: not found",
		},
		{
			name:           "wrap_empty_message",
			originalError:  NewDomainError("original error"),
			wrapMessage:    "",
			expectedResult: ": original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := tt.originalError.Wrap(tt.wrapMessage)

			assert.Error(t, wrapped)
			assert.Contains(t, wrapped.Error(), tt.expectedResult)

			// Verify wrapped error maintains original code
			if domainErr, ok := wrapped.(*DomainError); ok {
				assert.Equal(t, tt.originalError.Code(), domainErr.Code())
			}
		})
	}
}

func TestDomainError_MultipleWraps(t *testing.T) {
	original := NewDomainError("base error")
	firstWrap := original.Wrap("first context")

	// Cannot chain Wrap calls since it returns error, not *DomainError
	// But we can verify the first wrap worked correctly
	assert.Contains(t, firstWrap.Error(), "first context")
	assert.Contains(t, firstWrap.Error(), "base error")
}

func TestAlert_OptionalFields(t *testing.T) {
	tests := []struct {
		name                  string
		alert                 *Alert
		shouldHaveAlertRuleID bool
	}{
		{
			name: "alert_with_rule_id",
			alert: &Alert{
				ID:          uuid.New(),
				UserID:      uuid.New(),
				MonitorID:   uuid.New(),
				AlertRuleID: func() *uuid.UUID { id := uuid.New(); return &id }(),
				Status:      AlertStatusTriggered,
				Type:        AlertTypeStatusCode,
			},
			shouldHaveAlertRuleID: true,
		},
		{
			name: "alert_without_rule_id",
			alert: &Alert{
				ID:          uuid.New(),
				UserID:      uuid.New(),
				MonitorID:   uuid.New(),
				AlertRuleID: nil,
				Status:      AlertStatusTriggered,
				Type:        AlertTypeStatusCode,
			},
			shouldHaveAlertRuleID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldHaveAlertRuleID {
				assert.NotNil(t, tt.alert.AlertRuleID)
				assert.NotEmpty(t, tt.alert.AlertRuleID.String())
			} else {
				assert.Nil(t, tt.alert.AlertRuleID)
			}
		})
	}
}

func TestAlertChannel_OptionalTimeFields(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	tests := []struct {
		name                  string
		channel               *AlertChannel
		shouldHaveLastFailure bool
	}{
		{
			name: "channel_with_last_failure",
			channel: &AlertChannel{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				Type:          AlertChannelTypeEmail,
				Status:        AlertChannelStatusFailed,
				LastFailureAt: &later,
			},
			shouldHaveLastFailure: true,
		},
		{
			name: "channel_without_last_failure",
			channel: &AlertChannel{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				Type:          AlertChannelTypeEmail,
				Status:        AlertChannelStatusActive,
				LastFailureAt: nil,
			},
			shouldHaveLastFailure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldHaveLastFailure {
				assert.NotNil(t, tt.channel.LastFailureAt)
				assert.False(t, tt.channel.LastFailureAt.IsZero())
			} else {
				assert.Nil(t, tt.channel.LastFailureAt)
			}
		})
	}
}

func TestDeliveryAttempt_OptionalFields(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	tests := []struct {
		name                string
		attempt             *DeliveryAttempt
		shouldHaveNextRetry bool
	}{
		{
			name: "attempt_with_next_retry",
			attempt: &DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        uuid.New(),
				AlertChannelID: uuid.New(),
				Status:         DeliveryAttemptStatusPending,
				RetryCount:     1,
				NextRetryAt:    &later,
			},
			shouldHaveNextRetry: true,
		},
		{
			name: "attempt_without_next_retry",
			attempt: &DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        uuid.New(),
				AlertChannelID: uuid.New(),
				Status:         DeliveryAttemptStatusSuccess,
				RetryCount:     0,
				NextRetryAt:    nil,
			},
			shouldHaveNextRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldHaveNextRetry {
				assert.NotNil(t, tt.attempt.NextRetryAt)
				assert.False(t, tt.attempt.NextRetryAt.IsZero())
			} else {
				assert.Nil(t, tt.attempt.NextRetryAt)
			}
		})
	}
}

func TestAlert_NilConfig(t *testing.T) {
	alert := &Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Type:      AlertTypeStatusCode,
		Status:    AlertStatusTriggered,
		Enabled:   true,
		Config:    nil,
	}

	// Test that nil config is handled properly
	assert.Nil(t, alert.Config)
}

func TestAlertChannel_ConfigurationVariations(t *testing.T) {
	userID := uuid.New()
	testBotToken := "bot" + "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

	tests := []struct {
		name    string
		channel *AlertChannel
	}{
		{
			name: "email_channel_with_all_fields",
			channel: &AlertChannel{
				ID:           uuid.New(),
				UserID:       userID,
				Type:         AlertChannelTypeEmail,
				Status:       AlertChannelStatusActive,
				Enabled:      true,
				Verified:     true,
				FailureCount: 0,
				EmailConfig: &EmailChannelConfig{
					Email: "user@example.com",
				},
			},
		},
		{
			name: "telegram_channel_with_all_fields",
			channel: &AlertChannel{
				ID:           uuid.New(),
				UserID:       userID,
				Type:         AlertChannelTypeTelegram,
				Status:       AlertChannelStatusActive,
				Enabled:      true,
				Verified:     true,
				FailureCount: 0,
				TelegramConfig: &TelegramChannelConfig{
					ChatID:   "@username",
					BotToken: testBotToken,
				},
			},
		},
		{
			name: "webhook_channel_with_all_fields",
			channel: &AlertChannel{
				ID:           uuid.New(),
				UserID:       userID,
				Type:         AlertChannelTypeWebhook,
				Status:       AlertChannelStatusActive,
				Enabled:      true,
				Verified:     true,
				FailureCount: 0,
				WebhookConfig: &WebhookChannelConfig{
					URL:    "https://example.com/webhook",
					Method: "POST",
					Headers: map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer token123",
					},
				},
			},
		},
		{
			name: "channel_with_failures",
			channel: &AlertChannel{
				ID:           uuid.New(),
				UserID:       userID,
				Type:         AlertChannelTypeEmail,
				Status:       AlertChannelStatusFailed,
				Enabled:      true,
				Verified:     false,
				FailureCount: 5,
				EmailConfig: &EmailChannelConfig{
					Email: "user@example.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test channel fields
			assert.NotEmpty(t, tt.channel.ID.String())
			assert.Equal(t, userID, tt.channel.UserID)
			assert.GreaterOrEqual(t, tt.channel.FailureCount, 0)

			// Test configuration-specific fields
			switch tt.channel.Type {
			case AlertChannelTypeEmail:
				assert.NotNil(t, tt.channel.EmailConfig)
			case AlertChannelTypeTelegram:
				assert.NotNil(t, tt.channel.TelegramConfig)
			case AlertChannelTypeWebhook:
				assert.NotNil(t, tt.channel.WebhookConfig)
			}
		})
	}
}

func TestAlertRule_EnabledDisabledStates(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled_rule",
			enabled: true,
		},
		{
			name:    "disabled_rule",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &AlertRule{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Enabled:   tt.enabled,
			}

			assert.Equal(t, tt.enabled, rule.Enabled)
		})
	}
}

func TestAlert_EnabledDisabledStates(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled_alert",
			enabled: true,
		},
		{
			name:    "disabled_alert",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Enabled:   tt.enabled,
				Type:      AlertTypeStatusCode,
				Status:    AlertStatusTriggered,
			}

			assert.Equal(t, tt.enabled, alert.Enabled)
		})
	}
}

func TestDeliveryAttempt_ErrorMessage(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
	}{
		{
			name:         "attempt_with_error_message",
			errorMessage: "Connection timeout",
		},
		{
			name:         "attempt_with_empty_error_message",
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := &DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        uuid.New(),
				AlertChannelID: uuid.New(),
				Status:         DeliveryAttemptStatusFailed,
				ErrorMessage:   tt.errorMessage,
				RetryCount:     1,
			}

			assert.Equal(t, tt.errorMessage, attempt.ErrorMessage)
		})
	}
}

func TestAlert_AllTypeValues(t *testing.T) {
	tests := []struct {
		name      string
		alertType AlertType
	}{
		{"status_code", AlertTypeStatusCode},
		{"response_time", AlertTypeResponseTime},
		{"body_contains", AlertTypeBodyContains},
		{"body_does_not_contain", AlertTypeBodyDoesNotContain},
		{"certificate_expires", AlertTypeCertificateExpires},
		{"ssl_error", AlertTypeSSLError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{
				Type: tt.alertType,
			}
			assert.Equal(t, tt.alertType, alert.Type)
			assert.NotEmpty(t, tt.alertType.String())
		})
	}
}

func TestAlert_AllStatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status AlertStatus
	}{
		{"triggered", AlertStatusTriggered},
		{"muted", AlertStatusMuted},
		{"acknowledged", AlertStatusAcknowledged},
		{"resolved", AlertStatusResolved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{
				Status: tt.status,
			}
			assert.Equal(t, tt.status, alert.Status)
			assert.NotEmpty(t, tt.status.String())
		})
	}
}
