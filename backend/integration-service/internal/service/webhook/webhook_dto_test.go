package webhook

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

func TestCreateWebhookRequestToModel(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("valid request converts to model", func(t *testing.T) {
		t.Parallel()
		req := &CreateWebhookRequest{
			UserID:                  userID,
			Name:                    "My Webhook",
			URL:                     "https://example.com/webhook",
			Method:                  "POST",
			Headers:                 map[string]string{"X-Custom": "header"},
			Priority:                model.WebhookPriorityHigh,
			SeverityFilter:          []string{"critical"},
			MaxPayloadSizeBytes:     512000,
			PayloadHandlingStrategy: model.PayloadHandlingStrategyReject,
		}

		w, err := req.ToModel()
		require.NoError(t, err)
		require.NotNil(t, w)
		assert.Equal(t, userID, w.UserID)
		assert.Equal(t, "My Webhook", w.Name)
		assert.Equal(t, "https://example.com/webhook", w.URL)
		assert.Equal(t, "POST", w.Method)
		assert.Equal(t, map[string]string{"X-Custom": "header"}, w.Headers)
		assert.Equal(t, model.WebhookPriorityHigh, w.Priority)
		assert.Equal(t, []string{"critical"}, w.SeverityFilter)
		assert.Equal(t, 512000, w.MaxPayloadSizeBytes)
		assert.Equal(t, model.PayloadHandlingStrategyReject, w.PayloadHandlingStrategy)
	})

	t.Run("invalid url returns error", func(t *testing.T) {
		t.Parallel()
		req := &CreateWebhookRequest{
			UserID: userID,
			Name:   "Test",
			URL:    "not-a-url",
			Method: "POST",
		}
		_, err := req.ToModel()
		require.Error(t, err)
	})

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		req := &CreateWebhookRequest{
			UserID: userID,
			Name:   "",
			URL:    "https://example.com/webhook",
			Method: "POST",
		}
		_, err := req.ToModel()
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmptyWebhookName)
	})
}

func TestUpdateWebhookRequestApplyToModel(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	webhook := &model.WebhookIntegration{
		ID:                      uuid.New(),
		UserID:                  userID,
		Name:                    "Old Name",
		URL:                     "https://old.example.com",
		Method:                  "POST",
		Enabled:                 true,
		Priority:                model.WebhookPriorityNormal,
		MaxPayloadSizeBytes:     1048576,
		PayloadHandlingStrategy: model.PayloadHandlingStrategyTruncate,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	t.Run("apply all fields", func(t *testing.T) {
		t.Parallel()
		// Clone the webhook for this test
		w := *webhook
		newName := "New Name"
		newURL := "https://new.example.com"
		newMethod := "PUT"
		enabled := false
		priority := model.WebhookPriorityHigh
		size := 512000
		strategy := model.PayloadHandlingStrategyReject

		req := &UpdateWebhookRequest{
			ID:                      w.ID,
			UserID:                  userID,
			Name:                    &newName,
			URL:                     &newURL,
			Method:                  &newMethod,
			Headers:                 map[string]string{"X-New": "header"},
			Enabled:                 &enabled,
			Priority:                &priority,
			SeverityFilter:          []string{"critical"},
			MaxPayloadSizeBytes:     &size,
			PayloadHandlingStrategy: &strategy,
		}

		req.ApplyToModel(&w)

		assert.Equal(t, "New Name", w.Name)
		assert.Equal(t, "https://new.example.com", w.URL)
		assert.Equal(t, "PUT", w.Method)
		assert.Equal(t, map[string]string{"X-New": "header"}, w.Headers)
		assert.False(t, w.Enabled)
		assert.Equal(t, model.WebhookPriorityHigh, w.Priority)
		assert.Equal(t, []string{"critical"}, w.SeverityFilter)
		assert.Equal(t, 512000, w.MaxPayloadSizeBytes)
		assert.Equal(t, model.PayloadHandlingStrategyReject, w.PayloadHandlingStrategy)
	})

	t.Run("apply nil fields leaves model unchanged", func(t *testing.T) {
		t.Parallel()
		w := *webhook
		req := &UpdateWebhookRequest{
			ID:     w.ID,
			UserID: userID,
			// All optional fields nil
		}
		req.ApplyToModel(&w)
		assert.Equal(t, "Old Name", w.Name)
		assert.Equal(t, "https://old.example.com", w.URL)
		assert.Equal(t, "POST", w.Method)
	})
}

func TestWebhookStatsDTOFromModel(t *testing.T) {
	t.Parallel()

	now := time.Now()
	avgTime := 200
	successAt := now.Add(-1 * time.Second)
	failAt := now.Add(-2 * time.Second)

	w := &model.WebhookIntegration{
		ID:                uuid.New(),
		UserID:            uuid.New(),
		TotalSent:         100,
		SuccessfulSent:    80,
		FailedSent:        20,
		FailureCount:      20,
		AvgResponseTimeMs: &avgTime,
		LastSentAt:        &now,
		LastSuccessAt:     &successAt,
		LastFailureAt:     &failAt,
	}

	stats := WebhookStatsFromModel(w)

	require.NotNil(t, stats)
	assert.Equal(t, 100, stats.TotalSent)
	assert.Equal(t, 80, stats.SuccessfulSent)
	assert.Equal(t, 20, stats.FailedSent)
	assert.Equal(t, 20, stats.FailureCount)
	assert.Equal(t, 200, stats.AvgResponseTimeMs)
	assert.Equal(t, 80.0, stats.SuccessRate)
	require.NotNil(t, stats.LastSentAt)
	require.NotNil(t, stats.LastSuccessAt)
	require.NotNil(t, stats.LastFailureAt)
}

func TestWebhookStatsDTOFromModelNilFields(t *testing.T) {
	t.Parallel()

	w := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TotalSent: 0,
	}

	stats := WebhookStatsFromModel(w)

	require.NotNil(t, stats)
	assert.Equal(t, 0.0, stats.SuccessRate)
	assert.Equal(t, 0, stats.AvgResponseTimeMs)
	assert.Nil(t, stats.LastSentAt)
	assert.Nil(t, stats.LastSuccessAt)
	assert.Nil(t, stats.LastFailureAt)
}
