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

func TestAlertRuleRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertRuleRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			Enabled:             true,
			ConsecutiveFailures: 5,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, rule)
		require.NoError(t, err)
		assert.NotEmpty(t, rule.ID)
	})

	t.Run("GetByID", func(t *testing.T) {
		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			Enabled:             true,
			ConsecutiveFailures: 3,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, rule)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, rule.ID.String())
		require.NoError(t, err)
		assert.Equal(t, rule.ID, retrieved.ID)
		assert.Equal(t, rule.UserID, retrieved.UserID)
		assert.Equal(t, rule.MonitorID, retrieved.MonitorID)
		assert.Equal(t, rule.Enabled, retrieved.Enabled)
		assert.Equal(t, rule.ConsecutiveFailures, retrieved.ConsecutiveFailures)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New().String())
		assert.Error(t, err)
	})

	t.Run("Update", func(t *testing.T) {
		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			Enabled:             true,
			ConsecutiveFailures: 1,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, rule)
		require.NoError(t, err)

		rule.Enabled = false
		rule.ConsecutiveFailures = 10
		rule.UpdatedAt = time.Now()

		err = repo.Update(ctx, rule)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, rule.ID.String())
		require.NoError(t, err)
		assert.False(t, retrieved.Enabled)
		assert.Equal(t, 10, retrieved.ConsecutiveFailures)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		rule := &model.AlertRule{
			ID:        uuid.New(),
			Enabled:   false,
			UpdatedAt: time.Now(),
		}

		err := repo.Update(ctx, rule)
		assert.Error(t, err)
		assert.Equal(t, model.ErrAlertRuleNotFound, err)
	})

	t.Run("Delete", func(t *testing.T) {
		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			Enabled:             true,
			ConsecutiveFailures: 2,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, rule)
		require.NoError(t, err)

		err = repo.Delete(ctx, rule.ID.String())
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, rule.ID.String())
		assert.Error(t, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New().String())
		assert.Error(t, err)
		assert.Equal(t, model.ErrAlertRuleNotFound, err)
	})

	t.Run("List", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()

		// Create multiple rules
		for i := 0; i < 3; i++ {
			rule := &model.AlertRule{
				ID:                  uuid.New(),
				UserID:              userID,
				MonitorID:           monitorID,
				Enabled:             true,
				ConsecutiveFailures: i + 1,
				CreatedAt:           time.Now().Add(-time.Duration(i) * time.Minute),
				UpdatedAt:           time.Now(),
			}
			err := repo.Create(ctx, rule)
			require.NoError(t, err)
		}

		// List all rules
		rules, err := repo.List(ctx, userID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(rules), 3)
	})

	t.Run("GetByUserIDAndMonitorID", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()

		// Create rule for specific user and monitor
		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			Enabled:             true,
			ConsecutiveFailures: 5,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := repo.Create(ctx, rule)
		require.NoError(t, err)

		// Get rule by user ID and monitor ID
		retrievedRule, err := repo.GetByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)
		assert.Equal(t, rule.ID, retrievedRule.ID)
		assert.Equal(t, userID, retrievedRule.UserID)
		assert.Equal(t, monitorID, retrievedRule.MonitorID)
		assert.Equal(t, true, retrievedRule.Enabled)
		assert.Equal(t, 5, retrievedRule.ConsecutiveFailures)
	})
}
