package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

func TestAlertChannelRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertChannelRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		channel := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Type:         model.AlertChannelTypeEmail,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			EmailConfig:  &model.EmailChannelConfig{Email: "test@example.com"},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := repo.Create(ctx, channel)
		require.NoError(t, err)
		assert.NotEmpty(t, channel.ID)
	})

	t.Run("GetByID", func(t *testing.T) {
		channel := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Type:         model.AlertChannelTypeEmail,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			EmailConfig:  &model.EmailChannelConfig{Email: "test@example.com"},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, channel.ID.String())
		require.NoError(t, err)
		assert.Equal(t, channel.ID, retrieved.ID)
		assert.Equal(t, channel.UserID, retrieved.UserID)
		assert.Equal(t, channel.Type, retrieved.Type)
		assert.Equal(t, channel.Status, retrieved.Status)
		assert.Equal(t, channel.Enabled, retrieved.Enabled)
		assert.Equal(t, channel.Verified, retrieved.Verified)
	})

	t.Run("Update", func(t *testing.T) {
		channel := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Type:         model.AlertChannelTypeTelegram,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			TelegramConfig: &model.TelegramChannelConfig{
				ChatID:   "123456789",
				BotToken: "test-token",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		channel.Status = model.AlertChannelStatusFailed
		channel.FailureCount = 5
		channel.UpdatedAt = time.Now()

		err = repo.Update(ctx, channel)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, channel.ID.String())
		require.NoError(t, err)
		assert.Equal(t, model.AlertChannelStatusFailed, retrieved.Status)
		assert.Equal(t, 5, retrieved.FailureCount)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		channel := &model.AlertChannel{
			ID:        uuid.New(),
			Status:    model.AlertChannelStatusActive,
			UpdatedAt: time.Now(),
		}

		err := repo.Update(ctx, channel)
		assert.Error(t, err)
		assert.Equal(t, model.ErrAlertChannelNotFound, err)
	})

	t.Run("Delete", func(t *testing.T) {
		channel := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Type:         model.AlertChannelTypeWebhook,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			WebhookConfig: &model.WebhookChannelConfig{
				URL:    "https://example.com/webhook",
				Method: "POST",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		err = repo.Delete(ctx, channel.ID.String())
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, channel.ID.String())
		assert.Error(t, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New().String())
		assert.Error(t, err)
		assert.Equal(t, model.ErrAlertChannelNotFound, err)
	})

	t.Run("ListByUserID", func(t *testing.T) {
		userID := uuid.New()

		// Create multiple channels
		for i := 0; i < 3; i++ {
			channel := &model.AlertChannel{
				ID:           uuid.New(),
				UserID:       userID,
				Type:         model.AlertChannelTypeEmail,
				Status:       model.AlertChannelStatusActive,
				Enabled:      true,
				Verified:     true,
				FailureCount: i,
				EmailConfig:  &model.EmailChannelConfig{Email: "test@example.com"},
				CreatedAt:    time.Now().Add(-time.Duration(i) * time.Minute),
				UpdatedAt:    time.Now(),
			}
			err := repo.Create(ctx, channel)
			require.NoError(t, err)
		}

		// List channels by user ID
		channels, err := repo.ListByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(channels), 3)
	})

	t.Run("GetByUserIDAndType", func(t *testing.T) {
		userID := uuid.New()

		// Create email channel
		emailChannel := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       userID,
			Type:         model.AlertChannelTypeEmail,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			EmailConfig:  &model.EmailChannelConfig{Email: "test@example.com"},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := repo.Create(ctx, emailChannel)
		require.NoError(t, err)

		// Get email channel by user ID
		retrievedEmailChannel, err := repo.GetByUserIDAndType(ctx, userID.String(), model.AlertChannelTypeEmail)
		require.NoError(t, err)
		assert.Equal(t, emailChannel.ID, retrievedEmailChannel.ID)
		assert.Equal(t, model.AlertChannelTypeEmail, retrievedEmailChannel.Type)
		assert.Equal(t, userID, retrievedEmailChannel.UserID)

		// Create telegram channel for a different user
		otherUserID := uuid.New()
		telegramChannel := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       otherUserID,
			Type:         model.AlertChannelTypeTelegram,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			TelegramConfig: &model.TelegramChannelConfig{
				ChatID:   "123456789",
				BotToken: "test-token",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = repo.Create(ctx, telegramChannel)
		require.NoError(t, err)

		// Get telegram channel by other user ID
		retrievedTelegramChannel, err := repo.GetByUserIDAndType(ctx, otherUserID.String(), model.AlertChannelTypeTelegram)
		require.NoError(t, err)
		assert.Equal(t, telegramChannel.ID, retrievedTelegramChannel.ID)
		assert.Equal(t, model.AlertChannelTypeTelegram, retrievedTelegramChannel.Type)
		assert.Equal(t, otherUserID, retrievedTelegramChannel.UserID)
	})

	t.Run("ExistsDuplicate", func(t *testing.T) {
		userID := uuid.New()
		email := "duplicate@example.com"

		// Create first channel
		channel1 := &model.AlertChannel{
			ID:           uuid.New(),
			UserID:       userID,
			Type:         model.AlertChannelTypeEmail,
			Status:       model.AlertChannelStatusActive,
			Enabled:      true,
			Verified:     true,
			FailureCount: 0,
			EmailConfig:  &model.EmailChannelConfig{Email: email},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := repo.Create(ctx, channel1)
		require.NoError(t, err)

		// Check if duplicate exists
		exists, err := repo.ExistsDuplicate(ctx, userID.String(), model.AlertChannelTypeEmail, email)
		require.NoError(t, err)
		assert.True(t, exists, "Duplicate should be detected")

		// Check non-duplicate email
		exists, err = repo.ExistsDuplicate(ctx, userID.String(), model.AlertChannelTypeEmail, "different@example.com")
		require.NoError(t, err)
		assert.False(t, exists, "Different email should not be detected as duplicate")

		// Check non-duplicate user
		otherUserID := uuid.New()
		exists, err = repo.ExistsDuplicate(ctx, otherUserID.String(), model.AlertChannelTypeEmail, email)
		require.NoError(t, err)
		assert.False(t, exists, "Same email for different user should not be duplicate")
	})
}
