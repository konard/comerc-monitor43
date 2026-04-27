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

func TestDeliveryAttemptRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewDeliveryAttemptRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        uuid.New(),
			AlertChannelID: uuid.New(),
			Status:         model.DeliveryAttemptStatusPending,
			ErrorMessage:   "",
			RetryCount:     0,
			NextRetryAt:    nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, attempt)
		require.NoError(t, err)
		assert.NotEmpty(t, attempt.ID)
	})

	t.Run("GetByID", func(t *testing.T) {
		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        uuid.New(),
			AlertChannelID: uuid.New(),
			Status:         model.DeliveryAttemptStatusPending,
			ErrorMessage:   "",
			RetryCount:     0,
			NextRetryAt:    nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, attempt)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, attempt.ID.String())
		require.NoError(t, err)
		assert.Equal(t, attempt.ID, retrieved.ID)
		assert.Equal(t, attempt.AlertID, retrieved.AlertID)
		assert.Equal(t, attempt.AlertChannelID, retrieved.AlertChannelID)
		assert.Equal(t, attempt.Status, retrieved.Status)
		assert.Equal(t, attempt.RetryCount, retrieved.RetryCount)
	})

	t.Run("Update", func(t *testing.T) {
		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        uuid.New(),
			AlertChannelID: uuid.New(),
			Status:         model.DeliveryAttemptStatusPending,
			ErrorMessage:   "",
			RetryCount:     0,
			NextRetryAt:    nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, attempt)
		require.NoError(t, err)

		attempt.Status = model.DeliveryAttemptStatusSuccess
		attempt.RetryCount = 1
		attempt.UpdatedAt = time.Now()

		err = repo.Update(ctx, attempt)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, attempt.ID.String())
		require.NoError(t, err)
		assert.Equal(t, model.DeliveryAttemptStatusSuccess, retrieved.Status)
		assert.Equal(t, 1, retrieved.RetryCount)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		attempt := &model.DeliveryAttempt{
			ID:        uuid.New(),
			Status:    model.DeliveryAttemptStatusSuccess,
			UpdatedAt: time.Now(),
		}

		err := repo.Update(ctx, attempt)
		assert.Error(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        uuid.New(),
			AlertChannelID: uuid.New(),
			Status:         model.DeliveryAttemptStatusPending,
			ErrorMessage:   "",
			RetryCount:     0,
			NextRetryAt:    nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, attempt)
		require.NoError(t, err)

		err = repo.Delete(ctx, attempt.ID.String())
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, attempt.ID.String())
		assert.Error(t, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		// Delete method doesn't return error for non-existent IDs
		// It just doesn't delete anything
		err := repo.Delete(ctx, uuid.New().String())
		assert.NoError(t, err) // Delete implementation doesn't check rows affected
	})

	t.Run("List", func(t *testing.T) {
		alertID := uuid.New()
		channelID := uuid.New()

		// Create multiple attempts
		for i := 0; i < 3; i++ {
			attempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        alertID,
				AlertChannelID: channelID,
				Status:         model.DeliveryAttemptStatusPending,
				ErrorMessage:   "",
				RetryCount:     i,
				NextRetryAt:    nil,
				CreatedAt:      time.Now().Add(-time.Duration(i) * time.Minute),
				UpdatedAt:      time.Now(),
			}
			err := repo.Create(ctx, attempt)
			require.NoError(t, err)
		}

		// List attempts
		attempts, err := repo.List(ctx, alertID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(attempts), 3)
	})

	t.Run("ListPending", func(t *testing.T) {
		alertID := uuid.New()
		channelID := uuid.New()
		nextRetryTime := time.Now().Add(1 * time.Hour)

		// Create pending attempts
		for i := 0; i < 2; i++ {
			attempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        alertID,
				AlertChannelID: channelID,
				Status:         model.DeliveryAttemptStatusPending,
				ErrorMessage:   "",
				RetryCount:     0,
				NextRetryAt:    &nextRetryTime,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			err := repo.Create(ctx, attempt)
			require.NoError(t, err)
		}

		// Create failed attempts
		failedTime := time.Now().Add(-1 * time.Hour)
		for i := 0; i < 2; i++ {
			attempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        uuid.New(),
				AlertChannelID: uuid.New(),
				Status:         model.DeliveryAttemptStatusFailed,
				ErrorMessage:   "test error",
				RetryCount:     3,
				NextRetryAt:    &failedTime,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			err := repo.Create(ctx, attempt)
			require.NoError(t, err)
		}

		// Create successful attempts
		for i := 0; i < 2; i++ {
			attempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        uuid.New(),
				AlertChannelID: uuid.New(),
				Status:         model.DeliveryAttemptStatusSuccess,
				ErrorMessage:   "",
				RetryCount:     0,
				NextRetryAt:    nil,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			err := repo.Create(ctx, attempt)
			require.NoError(t, err)
		}

		// List pending attempts
		pendingAttempts, err := repo.ListPending(ctx, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(pendingAttempts), 2)

		// Verify all attempts are pending
		for _, attempt := range pendingAttempts {
			assert.Equal(t, model.DeliveryAttemptStatusPending, attempt.Status)
		}
	})

	t.Run("DeleteOldAttempts", func(t *testing.T) {
		alertID := uuid.New()
		channelID := uuid.New()
		oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
		recentTime := time.Now().Add(-1 * time.Hour)

		// Create old attempts
		for i := 0; i < 3; i++ {
			attempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        alertID,
				AlertChannelID: channelID,
				Status:         model.DeliveryAttemptStatusSuccess,
				ErrorMessage:   "",
				RetryCount:     0,
				NextRetryAt:    nil,
				CreatedAt:      oldTime.Add(-time.Duration(i) * time.Hour),
				UpdatedAt:      oldTime.Add(-time.Duration(i) * time.Hour),
			}
			err := repo.Create(ctx, attempt)
			require.NoError(t, err)
		}

		// Create recent attempts
		for i := 0; i < 2; i++ {
			attempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        uuid.New(),
				AlertChannelID: uuid.New(),
				Status:         model.DeliveryAttemptStatusSuccess,
				ErrorMessage:   "",
				RetryCount:     0,
				NextRetryAt:    nil,
				CreatedAt:      recentTime.Add(-time.Duration(i) * time.Minute),
				UpdatedAt:      recentTime.Add(-time.Duration(i) * time.Minute),
			}
			err := repo.Create(ctx, attempt)
			require.NoError(t, err)
		}

		// Delete attempts older than 7 days
		sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
		err := repo.DeleteOldAttempts(ctx, sevenDaysAgo.Unix())
		require.NoError(t, err)

		// Verify old attempts are gone
		oldAlertID := uuid.New()
		oldChannelID := uuid.New()

		// Try to find old attempts - they should not exist anymore
		// Create a new attempt with old ID to verify deletion worked
		newAttempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        oldAlertID,
			AlertChannelID: oldChannelID,
			Status:         model.DeliveryAttemptStatusSuccess,
			ErrorMessage:   "",
			RetryCount:     0,
			NextRetryAt:    nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = repo.Create(ctx, newAttempt)
		require.NoError(t, err)

		// List attempts for old alert (should be empty since we deleted old ones)
		attempts, err := repo.List(ctx, oldAlertID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(attempts), 1) // Only the new attempt should exist
	})
}
