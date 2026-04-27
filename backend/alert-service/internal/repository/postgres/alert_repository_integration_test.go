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

func TestAlertRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		alertRuleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &alertRuleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 5,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, alert)
		require.NoError(t, err)
		assert.NotEmpty(t, alert.ID)
	})

	t.Run("GetByID", func(t *testing.T) {
		alertRuleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &alertRuleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 3,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, alert)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, alert.ID.String())
		require.NoError(t, err)
		assert.Equal(t, alert.ID, retrieved.ID)
		assert.Equal(t, alert.UserID, retrieved.UserID)
		assert.Equal(t, alert.MonitorID, retrieved.MonitorID)
		assert.Equal(t, alert.Status, retrieved.Status)
		assert.Equal(t, alert.Type, retrieved.Type)
		assert.Equal(t, alert.Enabled, retrieved.Enabled)
		assert.Equal(t, alert.ConsecutiveFailures, retrieved.ConsecutiveFailures)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New().String())
		assert.Error(t, err)
	})

	t.Run("Update", func(t *testing.T) {
		alertRuleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &alertRuleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, alert)
		require.NoError(t, err)

		alert.Status = model.AlertStatusResolved
		alert.Enabled = false
		alert.ConsecutiveFailures = 10
		alert.UpdatedAt = time.Now()

		err = repo.Update(ctx, alert)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, alert.ID.String())
		require.NoError(t, err)
		assert.Equal(t, model.AlertStatusResolved, retrieved.Status)
		assert.False(t, retrieved.Enabled)
		assert.Equal(t, 10, retrieved.ConsecutiveFailures)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		alert := &model.Alert{
			ID:        uuid.New(),
			Status:    model.AlertStatusResolved,
			Type:      model.AlertTypeResponseTime,
			Enabled:   false,
			UpdatedAt: time.Now(),
		}

		err := repo.Update(ctx, alert)
		assert.Error(t, err)
		assert.Equal(t, model.ErrAlertNotFound, err)
	})

	t.Run("Delete", func(t *testing.T) {
		alertRuleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &alertRuleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 2,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		err := repo.Create(ctx, alert)
		require.NoError(t, err)

		err = repo.Delete(ctx, alert.ID.String())
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, alert.ID.String())
		assert.Error(t, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New().String())
		assert.Error(t, err)
		assert.Equal(t, model.ErrAlertNotFound, err)
	})

	t.Run("List", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()

		// Create multiple alerts
		for i := 0; i < 3; i++ {
			alertRuleID := uuid.New()
			alert := &model.Alert{
				ID:                  uuid.New(),
				UserID:              userID,
				MonitorID:           monitorID,
				AlertRuleID:         &alertRuleID,
				Status:              model.AlertStatusTriggered,
				Type:                model.AlertTypeResponseTime,
				Enabled:             true,
				ConsecutiveFailures: i + 1,
				ThresholdMs:         30000,
				CreatedAt:           time.Now().Add(-time.Duration(i) * time.Minute),
				UpdatedAt:           time.Now(),
			}
			err := repo.Create(ctx, alert)
			require.NoError(t, err)
		}

		// List all alerts
		alerts, total, err := repo.List(ctx, userID.String(), model.AlertFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 3)
		assert.GreaterOrEqual(t, total, 3)

		// List with monitor filter
		alerts, total, err = repo.List(ctx, userID.String(), model.AlertFilter{
			MonitorID: monitorID.String(),
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 3)
		assert.GreaterOrEqual(t, total, 3)

		// List with status filter
		alerts, total, err = repo.List(ctx, userID.String(), model.AlertFilter{
			Status: model.AlertStatusTriggered,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 3)
		assert.GreaterOrEqual(t, total, 3)

		// List with pagination
		alerts, total, err = repo.List(ctx, userID.String(), model.AlertFilter{
			PageSize: 2,
			Page:     1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(alerts), 2)
		assert.GreaterOrEqual(t, total, 3)
	})

	t.Run("ListActiveByMonitorID", func(t *testing.T) {
		monitorID := uuid.New()

		// Create active alert
		alertRuleID := uuid.New()
		activeAlert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           monitorID,
			AlertRuleID:         &alertRuleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 5,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := repo.Create(ctx, activeAlert)
		require.NoError(t, err)

		// Create resolved alert
		resolvedAlertRuleID := uuid.New()
		resolvedAlert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           monitorID,
			AlertRuleID:         &resolvedAlertRuleID,
			Status:              model.AlertStatusResolved,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err = repo.Create(ctx, resolvedAlert)
		require.NoError(t, err)

		// List active alerts
		activeAlerts, err := repo.ListActiveByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(activeAlerts), 1)

		// Verify status
		for _, alert := range activeAlerts {
			assert.Equal(t, model.AlertStatusTriggered, alert.Status)
		}
	})

	t.Run("GetLastAlertTimeAnyStatus", func(t *testing.T) {
		monitorID := uuid.New()

		// Create first alert
		firstAlertRuleID := uuid.New()
		firstAlert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           monitorID,
			AlertRuleID:         &firstAlertRuleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 3,
			ThresholdMs:         30000,
			CreatedAt:           time.Now().Add(-1 * time.Hour),
			UpdatedAt:           time.Now(),
		}
		err := repo.Create(ctx, firstAlert)
		require.NoError(t, err)

		// Wait a bit to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		// Create second alert
		secondAlertRuleID := uuid.New()
		secondAlert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           monitorID,
			AlertRuleID:         &secondAlertRuleID,
			Status:              model.AlertStatusResolved,
			Type:                model.AlertTypeResponseTime,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         30000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err = repo.Create(ctx, secondAlert)
		require.NoError(t, err)

		// Get last alert
		lastAlert, err := repo.GetLastAlertTimeAnyStatus(ctx, monitorID.String())
		require.NoError(t, err)
		assert.Equal(t, secondAlert.ID, lastAlert.ID)
	})
}
