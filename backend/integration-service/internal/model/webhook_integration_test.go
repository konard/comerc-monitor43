package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookIntegration(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name        string
		webhookName string
		webhookURL  string
		method      string
		wantErr     error
	}{
		{
			name:        "valid webhook",
			webhookName: "Test Webhook",
			webhookURL:  "https://example.com/webhook",
			method:      "POST",
			wantErr:     nil,
		},
		{
			name:        "empty name",
			webhookName: "",
			webhookURL:  "https://example.com/webhook",
			method:      "POST",
			wantErr:     ErrEmptyWebhookName,
		},
		{
			name:        "name too long",
			webhookName: string(make([]byte, 256)),
			webhookURL:  "https://example.com/webhook",
			method:      "POST",
			wantErr:     ErrWebhookNameTooLong,
		},
		{
			name:        "invalid URL",
			webhookName: "Test",
			webhookURL:  "invalid-url",
			method:      "POST",
			wantErr:     ErrWebhookInvalidURL,
		},
		{
			name:        "injection in URL",
			webhookName: "Test",
			webhookURL:  "https://example.com/<script>alert('xss')</script>",
			method:      "POST",
			wantErr:     ErrWebhookInjection,
		},
		{
			name:        "invalid method",
			webhookName: "Test",
			webhookURL:  "https://example.com/webhook",
			method:      "INVALID",
			wantErr:     ErrWebhookInvalidMethod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook, err := NewWebhookIntegration(userID, tt.webhookName, tt.webhookURL, tt.method)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, webhook)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, webhook)
				assert.NotEqual(t, uuid.Nil, webhook.ID)
				assert.Equal(t, userID, webhook.UserID)
				assert.Equal(t, tt.webhookName, webhook.Name)
				assert.Equal(t, tt.webhookURL, webhook.URL)
				assert.Equal(t, tt.method, webhook.Method)
				assert.True(t, webhook.Enabled)
				assert.Equal(t, WebhookStatusActive, webhook.Status)
				assert.Equal(t, WebhookPriorityNormal, webhook.Priority)
				assert.Equal(t, 1048576, webhook.MaxPayloadSizeBytes) // 1MB default
			}
		})
	}
}

func TestWebhookIntegration_CanDeliver(t *testing.T) {
	tests := []struct {
		name     string
		webhook  *WebhookIntegration
		expected bool
	}{
		{
			name:     "active and enabled",
			webhook:  &WebhookIntegration{Enabled: true, Status: WebhookStatusActive},
			expected: true,
		},
		{
			name:     "disabled",
			webhook:  &WebhookIntegration{Enabled: false, Status: WebhookStatusActive},
			expected: false,
		},
		{
			name:     "failed status",
			webhook:  &WebhookIntegration{Enabled: true, Status: WebhookStatusFailed},
			expected: false,
		},
		{
			name:     "inactive status",
			webhook:  &WebhookIntegration{Enabled: true, Status: WebhookStatusInactive},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.webhook.CanDeliver()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebhookIntegration_ShouldDeliverForSeverity(t *testing.T) {
	userID := uuid.New()

	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	// Default severity filter includes critical, warning, degraded
	assert.True(t, webhook.ShouldDeliverForSeverity("critical"))
	assert.True(t, webhook.ShouldDeliverForSeverity("warning"))
	assert.True(t, webhook.ShouldDeliverForSeverity("degraded"))
	assert.False(t, webhook.ShouldDeliverForSeverity("info"))

	// Custom severity filter
	webhook.SeverityFilter = []string{"critical"}

	assert.True(t, webhook.ShouldDeliverForSeverity("critical"))
	assert.False(t, webhook.ShouldDeliverForSeverity("warning"))

	// Empty filter = allow all
	webhook.SeverityFilter = []string{}

	assert.True(t, webhook.ShouldDeliverForSeverity("critical"))
	assert.True(t, webhook.ShouldDeliverForSeverity("info"))
}

func TestWebhookIntegration_MarkAsFailed(t *testing.T) {
	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	initialFailureCount := webhook.FailureCount

	// Mark as failed 4 times - should stay active
	for i := 0; i < 4; i++ {
		webhook.MarkAsFailed()
		assert.Equal(t, WebhookStatusActive, webhook.Status)
		assert.True(t, webhook.Enabled)
	}

	assert.Equal(t, initialFailureCount+4, webhook.FailureCount)
	assert.Equal(t, 4, webhook.ConsecutiveFailures)

	// 5th failure - should be disabled
	webhook.MarkAsFailed()

	assert.Equal(t, WebhookStatusFailed, webhook.Status)
	assert.False(t, webhook.Enabled)
	assert.Equal(t, 5, webhook.ConsecutiveFailures)
}

func TestWebhookIntegration_MarkAsSuccessful(t *testing.T) {
	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	// Simulate some failures
	webhook.MarkAsFailed()
	webhook.MarkAsFailed()
	webhook.ConsecutiveFailures = 2

	// Mark as successful
	responseTimeMs := 150
	webhook.MarkAsSuccessful(responseTimeMs)

	assert.Equal(t, 1, webhook.TotalSent)
	assert.Equal(t, 1, webhook.SuccessfulSent)
	assert.Equal(t, 0, webhook.ConsecutiveFailures)
	assert.NotNil(t, webhook.LastSentAt)
	assert.NotNil(t, webhook.LastSuccessAt)
	assert.NotNil(t, webhook.AvgResponseTimeMs)
	assert.Equal(t, responseTimeMs, *webhook.AvgResponseTimeMs)
}

func TestWebhookIntegration_UpdateStats(t *testing.T) {
	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	webhook.UpdateStats(100, 95, 5, 120)

	assert.Equal(t, 100, webhook.TotalSent)
	assert.Equal(t, 95, webhook.SuccessfulSent)
	assert.Equal(t, 5, webhook.FailedSent)
	assert.Equal(t, 120, *webhook.AvgResponseTimeMs)
}

func TestWebhookIntegration_GenerateSignature(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()

	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)
	webhook.SecretKey = &secret

	payload := []byte(`{"test": "data"}`)
	timestamp := int64(1234567890)

	signature, err := webhook.GenerateSignature(payload, timestamp)

	require.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Contains(t, signature, "sha256=")
}

func TestWebhookIntegration_ValidatePayloadSize(t *testing.T) {
	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	// Default max size is 1MB
	validPayload := make([]byte, 1048576)   // 1MB
	invalidPayload := make([]byte, 1048577) // 1MB + 1 byte

	err = webhook.ValidatePayloadSize(len(validPayload))
	assert.NoError(t, err)

	err = webhook.ValidatePayloadSize(len(invalidPayload))
	assert.ErrorIs(t, err, ErrWebhookPayloadTooLarge)
}

func TestWebhookIntegration_TruncatePayload(t *testing.T) {
	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	webhook.PayloadHandlingStrategy = PayloadHandlingStrategyTruncate

	// Payload larger than max size
	largePayload := make([]byte, 2*1048576) // 2MB

	truncated := webhook.TruncatePayload(largePayload)

	assert.Equal(t, webhook.MaxPayloadSizeBytes, len(truncated))
	assert.Equal(t, largePayload[:webhook.MaxPayloadSizeBytes], truncated)
}

func TestNewWebhookDeliveryAttempt(t *testing.T) {
	webhookID := uuid.New()
	alertID := uuid.New()

	attempt := NewWebhookDeliveryAttempt(webhookID, alertID)

	assert.NotEqual(t, uuid.Nil, attempt.ID)
	assert.Equal(t, webhookID, attempt.WebhookID)
	assert.Equal(t, alertID, attempt.AlertID)
	assert.Equal(t, "pending", attempt.Status)
	assert.Equal(t, 0, attempt.RetryCount)
	assert.NotNil(t, attempt.CreatedAt)
}

func TestWebhookDeliveryAttempt_MarkAsSent(t *testing.T) {
	webhookID := uuid.New()
	alertID := uuid.New()

	attempt := NewWebhookDeliveryAttempt(webhookID, alertID)

	statusCode := 200
	responseTimeMs := 150

	attempt.MarkAsSent(statusCode, responseTimeMs)

	assert.Equal(t, "sent", attempt.Status)
	assert.Equal(t, &statusCode, attempt.HTTPStatusCode)
	assert.Equal(t, &responseTimeMs, attempt.ResponseTimeMs)
	assert.NotNil(t, attempt.SentAt)
}

func TestWebhookDeliveryAttempt_MarkAsFailed(t *testing.T) {
	webhookID := uuid.New()
	alertID := uuid.New()

	attempt := NewWebhookDeliveryAttempt(webhookID, alertID)

	errorCode := "TIMEOUT"
	errorMessage := "Connection timeout"

	attempt.MarkAsFailed(errorCode, errorMessage)

	assert.Equal(t, "failed", attempt.Status)
	assert.Equal(t, &errorCode, attempt.ErrorCode)
	assert.Equal(t, &errorMessage, attempt.ErrorMessage)
}

func TestWebhookDeliveryAttempt_ScheduleRetry(t *testing.T) {
	webhookID := uuid.New()
	alertID := uuid.New()

	attempt := NewWebhookDeliveryAttempt(webhookID, alertID)

	nextRetryAt := time.Now().Add(30 * time.Second)
	retryCount := 1

	attempt.ScheduleRetry(nextRetryAt, retryCount)

	assert.Equal(t, "retry_scheduled", attempt.Status)
	assert.Equal(t, &nextRetryAt, attempt.NextRetryAt)
	assert.Equal(t, retryCount, attempt.RetryCount)
}

func TestWebhookIntegrationMarkAsDisabled(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	webhook.MarkAsDisabled()

	assert.Equal(t, WebhookStatusDisabled, webhook.Status)
	assert.False(t, webhook.Enabled)
}

func TestWebhookIntegrationMarkAsActive(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	// Disable first
	webhook.MarkAsDisabled()
	assert.Equal(t, WebhookStatusDisabled, webhook.Status)

	// Then re-activate
	webhook.ConsecutiveFailures = 3
	webhook.MarkAsActive()

	assert.Equal(t, WebhookStatusActive, webhook.Status)
	assert.True(t, webhook.Enabled)
	assert.Equal(t, 0, webhook.ConsecutiveFailures)
}

func TestWebhookIntegrationShouldTruncatePayload(t *testing.T) {
	t.Parallel()

	t.Run("truncate strategy returns true", func(t *testing.T) {
		t.Parallel()
		w := &WebhookIntegration{PayloadHandlingStrategy: PayloadHandlingStrategyTruncate}
		assert.True(t, w.ShouldTruncatePayload())
	})

	t.Run("reject strategy returns false", func(t *testing.T) {
		t.Parallel()
		w := &WebhookIntegration{PayloadHandlingStrategy: PayloadHandlingStrategyReject}
		assert.False(t, w.ShouldTruncatePayload())
	})
}

func TestWebhookIntegrationTruncatePayloadNoTruncation(t *testing.T) {
	t.Parallel()

	w := &WebhookIntegration{MaxPayloadSizeBytes: 1000}
	payload := []byte("small payload")
	result := w.TruncatePayload(payload)
	assert.Equal(t, payload, result)
}

func TestWebhookIntegrationMarshalJSON(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)
	webhook.SeverityFilter = []string{"critical", "warning"}

	data, err := webhook.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), "severity_filter")
	assert.Contains(t, string(data), "critical")
}

func TestWebhookIntegrationMarkAsSuccessfulUpdatesAvg(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	webhook, err := NewWebhookIntegration(userID, "Test", "https://example.com/webhook", "POST")
	require.NoError(t, err)

	// First successful delivery sets initial avg
	webhook.MarkAsSuccessful(100)
	require.NotNil(t, webhook.AvgResponseTimeMs)
	assert.Equal(t, 100, *webhook.AvgResponseTimeMs)

	// Second successful delivery updates sliding avg
	webhook.MarkAsSuccessful(200)
	// new_avg = 0.9 * 100 + 0.1 * 200 = 90 + 20 = 110
	assert.Equal(t, 110, *webhook.AvgResponseTimeMs)
}

func TestWebhookIntegrationGenerateSignatureNoSecret(t *testing.T) {
	t.Parallel()

	w := &WebhookIntegration{SecretKey: nil}
	sig, err := w.GenerateSignature([]byte("payload"), 12345)
	require.NoError(t, err)
	assert.Empty(t, sig)
}
