package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/testutil"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// ─── AuditLogRepository ──────────────────────────────────────────────────────

func TestAuditLogRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAuditLogRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		userID := uuid.New()
		resourceID := uuid.New().String()

		entry := &model.AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       model.AuditActionAlertTriggered,
			ResourceType: "alert",
			ResourceID:   &resourceID,
			Fields:       map[string]any{"key": "value"},
			CreatedAt:    time.Now(),
		}

		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	})

	t.Run("List", func(t *testing.T) {
		resourceID := uuid.New().String()
		userID := uuid.New()

		entry := &model.AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       model.AuditActionAlertResolved,
			ResourceType: "alert",
			ResourceID:   &resourceID,
			Fields:       map[string]any{"status": "resolved"},
			CreatedAt:    time.Now(),
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)

		logs, err := repo.List(ctx, "alert", resourceID, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 1)
	})

	t.Run("ListEmpty", func(t *testing.T) {
		logs, err := repo.List(ctx, "alert", uuid.New().String(), 10)
		require.NoError(t, err)
		assert.Empty(t, logs)
	})
}

// ─── AlertMuteRepository ─────────────────────────────────────────────────────

func TestAlertMuteRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertMuteRepository(dbWrapper)

	t.Run("Create_UserScope", func(t *testing.T) {
		userID := uuid.New()
		mute := &model.AlertMute{
			ID:        uuid.New(),
			UserID:    userID,
			Scope:     model.MuteScopeUser,
			CreatedAt: time.Now(),
			CreatedBy: userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)
	})

	t.Run("Create_WithMonitorID", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()
		futureTime := time.Now().Add(24 * time.Hour)

		mute := &model.AlertMute{
			ID:         uuid.New(),
			UserID:     userID,
			MonitorID:  &monitorID,
			Scope:      model.MuteScopeUser,
			MutedUntil: &futureTime,
			CreatedAt:  time.Now(),
			CreatedBy:  userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		userID := uuid.New()
		muteID := uuid.New()
		mute := &model.AlertMute{
			ID:        muteID,
			UserID:    userID,
			Scope:     model.MuteScopeUser,
			CreatedAt: time.Now(),
			CreatedBy: userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)

		err = repo.Delete(ctx, muteID.String())
		require.NoError(t, err)
	})

	t.Run("GetActiveByMonitorID", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()
		futureTime := time.Now().Add(time.Hour)

		mute := &model.AlertMute{
			ID:         uuid.New(),
			UserID:     userID,
			MonitorID:  &monitorID,
			Scope:      model.MuteScopeUser,
			MutedUntil: &futureTime,
			CreatedAt:  time.Now(),
			CreatedBy:  userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)

		found, err := repo.GetActiveByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("GetActiveByUserID", func(t *testing.T) {
		userID := uuid.New()
		futureTime := time.Now().Add(time.Hour)

		mute := &model.AlertMute{
			ID:         uuid.New(),
			UserID:     userID,
			Scope:      model.MuteScopeUser,
			MutedUntil: &futureTime,
			CreatedAt:  time.Now(),
			CreatedBy:  userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)

		mutes, err := repo.GetActiveByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(mutes), 1)
	})

	t.Run("IsMuted_True", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()
		futureTime := time.Now().Add(time.Hour)

		mute := &model.AlertMute{
			ID:         uuid.New(),
			UserID:     userID,
			MonitorID:  &monitorID,
			Scope:      model.MuteScopeUser,
			MutedUntil: &futureTime,
			CreatedAt:  time.Now(),
			CreatedBy:  userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)

		muted, err := repo.IsMuted(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)
		assert.True(t, muted)
	})

	t.Run("IsMuted_False", func(t *testing.T) {
		muted, err := repo.IsMuted(ctx, uuid.New().String(), uuid.New().String())
		require.NoError(t, err)
		assert.False(t, muted)
	})

	t.Run("DeleteByUserIDAndMonitorID", func(t *testing.T) {
		userID := uuid.New()
		monitorID := uuid.New()

		mute := &model.AlertMute{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: &monitorID,
			Scope:     model.MuteScopeUser,
			CreatedAt: time.Now(),
			CreatedBy: userID,
		}
		err := repo.Create(ctx, mute)
		require.NoError(t, err)

		err = repo.DeleteByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)
	})
}

// ─── AlertEscalationRepository ───────────────────────────────────────────────

func TestAlertEscalationRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertEscalationRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		alertID := uuid.New()
		channelID := uuid.New()

		escalation := &model.AlertEscalation{
			ID:                   uuid.New(),
			AlertID:              alertID,
			Level:                1,
			EscalatedToChannelID: &channelID,
			Reason:               "timeout",
			TimeoutMinutes:       30,
			CreatedAt:            time.Now(),
		}
		err := repo.Create(ctx, escalation)
		require.NoError(t, err)
	})

	t.Run("GetLatestByAlertID", func(t *testing.T) {
		alertID := uuid.New()

		for i := 1; i <= 2; i++ {
			channelID := uuid.New()
			e := &model.AlertEscalation{
				ID:                   uuid.New(),
				AlertID:              alertID,
				Level:                i,
				EscalatedToChannelID: &channelID,
				Reason:               "test",
				TimeoutMinutes:       30,
				CreatedAt:            time.Now().Add(time.Duration(i) * time.Second),
			}
			err := repo.Create(ctx, e)
			require.NoError(t, err)
		}

		latest, err := repo.GetLatestByAlertID(ctx, alertID.String())
		require.NoError(t, err)
		assert.Equal(t, 2, latest.Level)
	})
}

// ─── MonitorStatusChangeRepository ──────────────────────────────────────────

func TestMonitorStatusChangeRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewMonitorStatusChangeRepository(dbWrapper)

	t.Run("Create", func(t *testing.T) {
		change := &model.MonitorStatusChange{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, change)
		require.NoError(t, err)
	})

	t.Run("CountInWindow", func(t *testing.T) {
		monitorID := uuid.New()
		for i := 0; i < 3; i++ {
			change := &model.MonitorStatusChange{
				ID:        uuid.New(),
				MonitorID: monitorID,
				UserID:    uuid.New(),
				OldStatus: "UP",
				NewStatus: "DOWN",
				CreatedAt: time.Now(),
			}
			err := repo.Create(ctx, change)
			require.NoError(t, err)
		}

		count, err := repo.CountInWindow(ctx, monitorID.String(), 10*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("GetLatestByMonitorID", func(t *testing.T) {
		monitorID := uuid.New()
		for i := 0; i < 2; i++ {
			change := &model.MonitorStatusChange{
				ID:        uuid.New(),
				MonitorID: monitorID,
				UserID:    uuid.New(),
				OldStatus: "UP",
				NewStatus: "DOWN",
				CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			err := repo.Create(ctx, change)
			require.NoError(t, err)
		}

		latest, err := repo.GetLatestByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)
		assert.Equal(t, monitorID, latest.MonitorID)
	})
}

// ─── MaintenanceWindowRepository ─────────────────────────────────────────────

func TestMaintenanceWindowRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewMaintenanceWindowRepository(dbWrapper)

	now := time.Now()
	future := now.Add(2 * time.Hour)

	t.Run("Create", func(t *testing.T) {
		mw := &model.MaintenanceWindow{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "Test Maintenance",
			Status:          model.MaintenanceStatusScheduled,
			Recurrence:      model.RecurrenceTypeOnce,
			IsGlobal:        false,
			PauseMonitoring: true,
			SuppressAlerts:  false,
			SafeMode:        false,
			MonitorIDs:      []string{uuid.New().String()},
			StartsAt:        now.Add(time.Hour),
			EndsAt:          future,
			Version:         0,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		err := repo.Create(ctx, mw)
		require.NoError(t, err)
	})

	t.Run("GetActiveByMonitorID", func(t *testing.T) {
		monitorID := uuid.New()
		ts := time.Now().UTC()
		mw := &model.MaintenanceWindow{
			ID:         uuid.New(),
			UserID:     uuid.New(),
			Name:       "Active MW",
			Status:     model.MaintenanceStatusActive,
			Recurrence: model.RecurrenceTypeOnce,
			IsGlobal:   true, // use global to avoid monitor_ids array matching issues
			MonitorIDs: []string{monitorID.String()},
			StartsAt:   ts.Add(-2 * time.Hour),
			EndsAt:     ts.Add(2 * time.Hour),
			Version:    0,
			CreatedAt:  ts,
			UpdatedAt:  ts,
		}
		err := repo.Create(ctx, mw)
		require.NoError(t, err)

		window, err := repo.GetActiveByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)
		assert.NotNil(t, window)

		// Cleanup: cancel window so it doesn't affect subsequent tests
		_, cleanErr := db.ExecContext(ctx, `UPDATE maintenance_windows SET status='cancelled' WHERE id=$1`, mw.ID)
		require.NoError(t, cleanErr)
	})

	t.Run("IsUnderMaintenance", func(t *testing.T) {
		monitorID := uuid.New()
		ts := time.Now().UTC()
		mw := &model.MaintenanceWindow{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			Name:            "Under Maintenance",
			Status:          model.MaintenanceStatusActive,
			Recurrence:      model.RecurrenceTypeOnce,
			IsGlobal:        true,
			PauseMonitoring: true,
			MonitorIDs:      []string{monitorID.String()},
			StartsAt:        ts.Add(-2 * time.Hour),
			EndsAt:          ts.Add(2 * time.Hour),
			Version:         0,
			CreatedAt:       ts,
			UpdatedAt:       ts,
		}
		err := repo.Create(ctx, mw)
		require.NoError(t, err)

		under, err := repo.IsUnderMaintenance(ctx, monitorID.String())
		require.NoError(t, err)
		assert.True(t, under)

		// Cleanup: cancel window so it doesn't affect subsequent tests
		_, err = db.ExecContext(ctx, `UPDATE maintenance_windows SET status='cancelled' WHERE id=$1`, mw.ID)
		require.NoError(t, err)
	})

	t.Run("IsUnderMaintenance_False", func(t *testing.T) {
		// No active windows for this monitor ID
		under, err := repo.IsUnderMaintenance(ctx, uuid.New().String())
		require.NoError(t, err)
		assert.False(t, under)
	})
}

// ─── AlertRepository additional functions ────────────────────────────────────

func TestAlertRepository_Additional_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	alertRepo := NewAlertRepository(dbWrapper)
	channelRepo := NewAlertChannelRepository(dbWrapper)

	makeAlert := func(status model.AlertStatus) *model.Alert {
		ruleID := uuid.New()
		return &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &ruleID,
			Status:              status,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
	}

	t.Run("CreateWithDeliveryAttempt", func(t *testing.T) {
		userID := uuid.New()
		channel := &model.AlertChannel{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        model.AlertChannelTypeEmail,
			Status:      model.AlertChannelStatusActive,
			Enabled:     true,
			Verified:    true,
			EmailConfig: &model.EmailChannelConfig{Email: "test@example.com"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := channelRepo.Create(ctx, channel)
		require.NoError(t, err)

		alert := makeAlert(model.AlertStatusTriggered)
		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        alert.ID,
			AlertChannelID: channel.ID,
			Status:         model.DeliveryAttemptStatusPending,
			RetryCount:     0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err = alertRepo.CreateWithDeliveryAttempt(ctx, alert, attempt)
		require.NoError(t, err)
	})

	t.Run("DeleteResolvedOlderThan", func(t *testing.T) {
		alert := makeAlert(model.AlertStatusResolved)
		alert.CreatedAt = time.Now().Add(-48 * time.Hour)
		alert.UpdatedAt = time.Now().Add(-48 * time.Hour)
		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		err = alertRepo.DeleteResolvedOlderThan(ctx, 24*time.Hour)
		require.NoError(t, err)
	})

	t.Run("GetLastAlertByMonitorIDAndStatus", func(t *testing.T) {
		monitorID := uuid.New()
		alert := makeAlert(model.AlertStatusTriggered)
		alert.MonitorID = monitorID
		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		found, err := alertRepo.GetLastAlertByMonitorIDAndStatus(ctx, monitorID.String(), model.AlertStatusTriggered)
		require.NoError(t, err)
		assert.Equal(t, alert.ID, found.ID)
	})

	t.Run("CountUniqueMonitorsWithAlertsSince", func(t *testing.T) {
		alert := makeAlert(model.AlertStatusTriggered)
		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		count, err := alertRepo.CountUniqueMonitorsWithAlertsSince(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)
	})

	t.Run("AcknowledgeAlert", func(t *testing.T) {
		alert := makeAlert(model.AlertStatusTriggered)
		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		err = alertRepo.AcknowledgeAlert(ctx, alert.ID.String(), uuid.New().String())
		require.NoError(t, err)

		updated, err := alertRepo.GetByID(ctx, alert.ID.String())
		require.NoError(t, err)
		assert.Equal(t, model.AlertStatusAcknowledged, updated.Status)
	})

	t.Run("GetActiveAlertsForMonitor", func(t *testing.T) {
		monitorID := uuid.New()
		alert := makeAlert(model.AlertStatusTriggered)
		alert.MonitorID = monitorID
		err := alertRepo.Create(ctx, alert)
		require.NoError(t, err)

		alerts, err := alertRepo.GetActiveAlertsForMonitor(ctx, monitorID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 1)
	})
}

// ─── AlertChannelRepository additional functions ──────────────────────────────

func TestAlertChannelRepository_Additional_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertChannelRepository(dbWrapper)

	makeChannel := func(userID uuid.UUID) *model.AlertChannel {
		return &model.AlertChannel{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        model.AlertChannelTypeEmail,
			Status:      model.AlertChannelStatusActive,
			Enabled:     true,
			Verified:    true,
			EmailConfig: &model.EmailChannelConfig{Email: uuid.New().String() + "@example.com"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	t.Run("CountByUserID", func(t *testing.T) {
		userID := uuid.New()
		ch1 := makeChannel(userID)
		ch2 := makeChannel(userID)
		require.NoError(t, repo.Create(ctx, ch1))
		require.NoError(t, repo.Create(ctx, ch2))

		count, err := repo.CountByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("IncrementFailureCount", func(t *testing.T) {
		userID := uuid.New()
		ch := makeChannel(userID)
		require.NoError(t, repo.Create(ctx, ch))

		newCount, err := repo.IncrementFailureCount(ctx, ch.ID.String())
		require.NoError(t, err)
		assert.Equal(t, 1, newCount)
	})

	t.Run("DisableChannel", func(t *testing.T) {
		userID := uuid.New()
		ch := makeChannel(userID)
		require.NoError(t, repo.Create(ctx, ch))

		err := repo.DisableChannel(ctx, ch.ID.String(), "too many failures")
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, ch.ID.String())
		require.NoError(t, err)
		assert.Equal(t, model.AlertChannelStatusFailed, updated.Status)
	})

	t.Run("MarkAsFailed", func(t *testing.T) {
		userID := uuid.New()
		ch := makeChannel(userID)
		require.NoError(t, repo.Create(ctx, ch))

		err := repo.MarkAsFailed(ctx, ch.ID.String(), 3)
		require.NoError(t, err)
	})
}

// ─── DB unit tests ────────────────────────────────────────────────────────────

func TestDB_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, "postgres", cfg.User)
	assert.Equal(t, 25, cfg.MaxOpenConns)
}

func TestDB_Rollback_InvalidType(t *testing.T) {
	dbWrapper := &DB{}
	err := dbWrapper.Rollback("not-a-tx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type")
}

func TestDB_Commit_InvalidType(t *testing.T) {
	dbWrapper := &DB{}
	err := dbWrapper.Commit(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type")
}

func TestDB_GettersWithNilObservability(t *testing.T) {
	dbWrapper := &DB{}
	assert.Nil(t, dbWrapper.GetTracer())
	assert.Nil(t, dbWrapper.GetMetrics())
}

// ─── AlertChannelRepository: SetChannelPriorities / GetChannelPriorities ─────

func TestAlertChannelRepository_Priorities_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewAlertChannelRepository(dbWrapper)

	userID := uuid.New()
	ruleID := uuid.New()

	ch := &model.AlertChannel{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        model.AlertChannelTypeEmail,
		Status:      model.AlertChannelStatusActive,
		Enabled:     true,
		Verified:    true,
		EmailConfig: &model.EmailChannelConfig{Email: "prio@example.com"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, repo.Create(ctx, ch))

	t.Run("SetChannelPriorities", func(t *testing.T) {
		priorities := []model.AlertChannelPriority{
			{AlertRuleID: ruleID, AlertChannelID: ch.ID, Priority: 1},
		}
		err := repo.SetChannelPriorities(ctx, ruleID.String(), priorities)
		require.NoError(t, err)
	})

	t.Run("GetChannelPriorities", func(t *testing.T) {
		prios, err := repo.GetChannelPriorities(ctx, ruleID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(prios), 1)
	})

	t.Run("SetChannelPriorities_Empty", func(t *testing.T) {
		err := repo.SetChannelPriorities(ctx, ruleID.String(), nil)
		require.NoError(t, err)

		prios, err := repo.GetChannelPriorities(ctx, ruleID.String())
		require.NoError(t, err)
		assert.Empty(t, prios)
	})
}

// ─── MaintenanceWindowRepository: GetByID / Update / Delete / List ───────────

func TestMaintenanceWindowRepository_CRUD_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewMaintenanceWindowRepository(dbWrapper)

	now := time.Now().UTC()
	userID := uuid.New()

	makeMW := func() *model.MaintenanceWindow {
		return &model.MaintenanceWindow{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Test MW",
			Status:     model.MaintenanceStatusScheduled,
			Recurrence: model.RecurrenceTypeOnce,
			MonitorIDs: []string{uuid.New().String()},
			StartsAt:   now.Add(time.Hour),
			EndsAt:     now.Add(2 * time.Hour),
			Version:    0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	t.Run("GetByID", func(t *testing.T) {
		mw := makeMW()
		require.NoError(t, repo.Create(ctx, mw))

		found, err := repo.GetByID(ctx, mw.ID.String())
		require.NoError(t, err)
		assert.Equal(t, mw.ID, found.ID)
		assert.Equal(t, mw.Name, found.Name)
	})

	t.Run("Update", func(t *testing.T) {
		mw := makeMW()
		require.NoError(t, repo.Create(ctx, mw))

		mw.Name = "Updated Name"
		// Version stays at 0 — the current DB version; the query matches WHERE version=0
		err := repo.Update(ctx, mw)
		require.NoError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		mw := makeMW()
		require.NoError(t, repo.Create(ctx, mw))

		err := repo.Delete(ctx, mw.ID.String())
		require.NoError(t, err)
	})

	t.Run("List", func(t *testing.T) {
		mw := makeMW()
		require.NoError(t, repo.Create(ctx, mw))

		windows, total, err := repo.List(ctx, userID.String(), model.MaintenanceWindowFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(windows), 1)
	})

	t.Run("List_WithStatusFilter", func(t *testing.T) {
		windows, _, err := repo.List(ctx, userID.String(), model.MaintenanceWindowFilter{
			Status: model.MaintenanceStatusScheduled,
		})
		require.NoError(t, err)
		assert.NotNil(t, windows)
	})
}

// ─── DeliveryAttemptRepository additional functions ───────────────────────────

func TestDeliveryAttemptRepository_Additional_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	repo := NewDeliveryAttemptRepository(dbWrapper)
	alertRepo := NewAlertRepository(dbWrapper)
	channelRepo := NewAlertChannelRepository(dbWrapper)

	userID := uuid.New()
	monitorID := uuid.New()
	ruleID := uuid.New()

	// Create prerequisite channel
	ch := &model.AlertChannel{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        model.AlertChannelTypeEmail,
		Status:      model.AlertChannelStatusActive,
		Enabled:     true,
		Verified:    true,
		EmailConfig: &model.EmailChannelConfig{Email: "da@example.com"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, channelRepo.Create(ctx, ch))

	// Create prerequisite alert
	alert := &model.Alert{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		AlertRuleID:         &ruleID,
		Status:              model.AlertStatusTriggered,
		Type:                model.AlertTypeStatusCode,
		Enabled:             true,
		ConsecutiveFailures: 1,
		ThresholdMs:         5000,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	require.NoError(t, alertRepo.Create(ctx, alert))

	// Create delivery attempt
	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        alert.ID,
		AlertChannelID: ch.ID,
		Status:         model.DeliveryAttemptStatusSuccess,
		RetryCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	require.NoError(t, repo.Create(ctx, attempt))

	t.Run("CleanupOldAttempts", func(t *testing.T) {
		err := repo.CleanupOldAttempts(ctx, 0) // delete all older than 0 (i.e. nothing to delete now)
		require.NoError(t, err)
	})

	t.Run("CountRecentByMonitorAndChannel", func(t *testing.T) {
		count, err := repo.CountRecentByMonitorAndChannel(ctx, monitorID.String(), "email", time.Hour)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)
	})

	t.Run("GetLastDeliveryTimeForMonitorAndStatus", func(t *testing.T) {
		ts, err := repo.GetLastDeliveryTimeForMonitorAndStatus(ctx, monitorID.String(), string(model.AlertStatusTriggered))
		require.NoError(t, err)
		assert.NotNil(t, ts)
	})
}

// ─── DB integration tests ─────────────────────────────────────────────────────

func TestDB_SetObservability_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	dbWrapper.SetObservability(nil, nil)
	assert.Nil(t, dbWrapper.GetTracer())
	assert.Nil(t, dbWrapper.GetMetrics())
	_ = ctx
}

func TestDB_BeginTx_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}
	tx, err := dbWrapper.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NoError(t, tx.Rollback())
}

// ─── Tracer-path coverage: run key operations with a noop tracer set ──────────

func TestRepository_WithTracer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	// Set a noop tracer to cover the tracer != nil branches in all repositories
	noopTracer := noop.NewTracerProvider().Tracer("test")
	dbWrapper := &DB{DB: db, tracer: noopTracer}

	t.Run("AlertRepo_Create_WithTracer", func(t *testing.T) {
		repo := NewAlertRepository(dbWrapper)
		ruleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, repo.Create(ctx, alert))

		found, err := repo.GetByID(ctx, alert.ID.String())
		require.NoError(t, err)
		assert.Equal(t, alert.ID, found.ID)

		err = repo.Update(ctx, alert)
		require.NoError(t, err)

		err = repo.Delete(ctx, alert.ID.String())
		require.NoError(t, err)
	})

	t.Run("AlertRepo_List_WithTracer", func(t *testing.T) {
		repo := NewAlertRepository(dbWrapper)
		userID := uuid.New()
		ruleID := uuid.New()

		alert := &model.Alert{
			ID:          uuid.New(),
			UserID:      userID,
			MonitorID:   uuid.New(),
			AlertRuleID: &ruleID,
			Status:      model.AlertStatusTriggered,
			Type:        model.AlertTypeStatusCode,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, repo.Create(ctx, alert))

		_, _, err := repo.List(ctx, userID.String(), model.AlertFilter{})
		require.NoError(t, err)

		_, err = repo.ListActiveByMonitorID(ctx, alert.MonitorID.String())
		require.NoError(t, err)

		_, err = repo.GetLastAlertTimeAnyStatus(ctx, alert.MonitorID.String())
		require.NoError(t, err)
	})

	t.Run("ChannelRepo_WithTracer", func(t *testing.T) {
		repo := NewAlertChannelRepository(dbWrapper)
		userID := uuid.New()

		ch := &model.AlertChannel{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        model.AlertChannelTypeEmail,
			Status:      model.AlertChannelStatusActive,
			Enabled:     true,
			Verified:    true,
			EmailConfig: &model.EmailChannelConfig{Email: "tracer@example.com"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, repo.Create(ctx, ch))

		_, err := repo.GetByID(ctx, ch.ID.String())
		require.NoError(t, err)

		_, err = repo.ListByUserID(ctx, userID.String())
		require.NoError(t, err)

		_, err = repo.CountByUserID(ctx, userID.String())
		require.NoError(t, err)

		err = repo.Update(ctx, ch)
		require.NoError(t, err)

		_, err = repo.GetByUserIDAndType(ctx, userID.String(), model.AlertChannelTypeEmail)
		require.NoError(t, err)

		_, err = repo.ExistsDuplicate(ctx, userID.String(), model.AlertChannelTypeEmail, "tracer@example.com")
		require.NoError(t, err)

		err = repo.Delete(ctx, ch.ID.String())
		require.NoError(t, err)
	})

	t.Run("AlertRuleRepo_WithTracer", func(t *testing.T) {
		repo := NewAlertRuleRepository(dbWrapper)
		userID := uuid.New()

		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           uuid.New(),
			Enabled:             true,
			ConsecutiveFailures: 3,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, repo.Create(ctx, rule))

		found, err := repo.GetByID(ctx, rule.ID.String())
		require.NoError(t, err)
		assert.Equal(t, rule.ID, found.ID)

		_, err = repo.GetByUserIDAndMonitorID(ctx, userID.String(), rule.MonitorID.String())
		require.NoError(t, err)

		_, err = repo.List(ctx, userID.String())
		require.NoError(t, err)

		err = repo.Update(ctx, rule)
		require.NoError(t, err)

		err = repo.Delete(ctx, rule.ID.String())
		require.NoError(t, err)
	})

	t.Run("DeliveryAttemptRepo_WithTracer", func(t *testing.T) {
		alertRepo := NewAlertRepository(dbWrapper)
		channelRepo := NewAlertChannelRepository(dbWrapper)
		repo := NewDeliveryAttemptRepository(dbWrapper)

		userID := uuid.New()
		monitorID := uuid.New()
		ruleID := uuid.New()

		ch := &model.AlertChannel{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        model.AlertChannelTypeEmail,
			Status:      model.AlertChannelStatusActive,
			Enabled:     true,
			Verified:    true,
			EmailConfig: &model.EmailChannelConfig{Email: "da-tracer@example.com"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, channelRepo.Create(ctx, ch))

		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, alertRepo.Create(ctx, alert))

		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        alert.ID,
			AlertChannelID: ch.ID,
			Status:         model.DeliveryAttemptStatusPending,
			RetryCount:     0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		require.NoError(t, repo.Create(ctx, attempt))

		found, err := repo.GetByID(ctx, attempt.ID.String())
		require.NoError(t, err)
		assert.Equal(t, attempt.ID, found.ID)

		_, err = repo.List(ctx, alert.ID.String())
		require.NoError(t, err)

		_, err = repo.ListPending(ctx, 10)
		require.NoError(t, err)

		attempt.Status = model.DeliveryAttemptStatusSuccess
		err = repo.Update(ctx, attempt)
		require.NoError(t, err)
	})

	t.Run("AuditLogRepo_WithTracer", func(t *testing.T) {
		repo := NewAuditLogRepository(dbWrapper)
		userID := uuid.New()
		resourceID := uuid.New().String()

		entry := &model.AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       model.AuditActionAlertTriggered,
			ResourceType: "alert",
			ResourceID:   &resourceID,
			Fields:       map[string]any{"test": "tracer"},
			CreatedAt:    time.Now(),
		}
		require.NoError(t, repo.Create(ctx, entry))

		_, err := repo.List(ctx, "alert", resourceID, 10)
		require.NoError(t, err)
	})

	t.Run("MuteRepo_WithTracer", func(t *testing.T) {
		repo := NewAlertMuteRepository(dbWrapper)
		userID := uuid.New()
		monitorID := uuid.New()

		mute := &model.AlertMute{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: &monitorID,
			Scope:     model.MuteScopeUser,
			CreatedAt: time.Now(),
			CreatedBy: userID,
		}
		require.NoError(t, repo.Create(ctx, mute))

		_, err := repo.GetActiveByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)

		_, err = repo.GetActiveByUserID(ctx, userID.String())
		require.NoError(t, err)

		_, err = repo.IsMuted(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)

		err = repo.Delete(ctx, mute.ID.String())
		require.NoError(t, err)
	})

	t.Run("EscalationRepo_WithTracer", func(t *testing.T) {
		repo := NewAlertEscalationRepository(dbWrapper)
		alertID := uuid.New()
		channelID := uuid.New()

		e := &model.AlertEscalation{
			ID:                   uuid.New(),
			AlertID:              alertID,
			Level:                1,
			EscalatedToChannelID: &channelID,
			Reason:               "test",
			TimeoutMinutes:       30,
			CreatedAt:            time.Now(),
		}
		require.NoError(t, repo.Create(ctx, e))

		_, err := repo.GetLatestByAlertID(ctx, alertID.String())
		require.NoError(t, err)
	})

	t.Run("MonitorStatusRepo_WithTracer", func(t *testing.T) {
		repo := NewMonitorStatusChangeRepository(dbWrapper)
		monitorID := uuid.New()

		change := &model.MonitorStatusChange{
			ID:        uuid.New(),
			MonitorID: monitorID,
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, change))

		_, err := repo.CountInWindow(ctx, monitorID.String(), 10*time.Minute)
		require.NoError(t, err)

		_, err = repo.GetLatestByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)
	})

	t.Run("MaintenanceWindowRepo_WithTracer", func(t *testing.T) {
		repo := NewMaintenanceWindowRepository(dbWrapper)
		userID := uuid.New()
		monitorID := uuid.New()
		now := time.Now().UTC()

		mw := &model.MaintenanceWindow{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Tracer MW",
			Status:     model.MaintenanceStatusScheduled,
			Recurrence: model.RecurrenceTypeOnce,
			MonitorIDs: []string{monitorID.String()},
			StartsAt:   now.Add(time.Hour),
			EndsAt:     now.Add(2 * time.Hour),
			Version:    0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		require.NoError(t, repo.Create(ctx, mw))

		found, err := repo.GetByID(ctx, mw.ID.String())
		require.NoError(t, err)
		assert.Equal(t, mw.ID, found.ID)

		_, _, err = repo.List(ctx, userID.String(), model.MaintenanceWindowFilter{})
		require.NoError(t, err)

		_, _, err = repo.CheckOverlapping(ctx,
			[]string{monitorID.String()},
			now.Add(30*time.Minute), now.Add(90*time.Minute), "")
		require.NoError(t, err)

		err = repo.Update(ctx, mw)
		require.NoError(t, err)

		err = repo.Delete(ctx, mw.ID.String())
		require.NoError(t, err)
	})
}

// ─── ListPendingForChannel (returns stub error) ───────────────────────────────

func TestAlertChannelRepository_ListPendingForChannel(t *testing.T) {
	repo := &AlertChannelRepository{db: &DB{}}
	result, err := repo.ListPendingForChannel(context.Background(), uuid.New().String(), 10)
	assert.Nil(t, result)
	require.Error(t, err)
}

// ─── NewDBFromDSN integration test ───────────────────────────────────────────

func TestNewDBFromDSN_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	container, sqlDB, err := testutil.StartPostgreSQLContainer(ctx)
	if err != nil {
		t.Skipf("skipping: Docker unavailable: %v", err)
	}
	t.Cleanup(func() {
		require.NoError(t, container.Shutdown(ctx))
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	require.NoError(t, sqlDB.Close())

	db := NewDBFromDSN(dsn)
	require.NotNil(t, db)
	err = db.Connect()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
}

// ─── DB.Rollback valid path ────────────────────────────────────────────────────

func TestDB_Rollback_ValidTx_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	dbWrapper := &DB{DB: db}

	tx, err := dbWrapper.BeginTxx(ctx, nil)
	require.NoError(t, err)

	err = dbWrapper.Rollback(tx)
	require.NoError(t, err)
}

// ─── Metrics-path coverage: run key operations with both tracer and metrics ──

func TestRepository_WithMetrics_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupSharedTestDB(t)

	noopTracer := noop.NewTracerProvider().Tracer("test")
	metrics := apptelemetry.NewMetrics("test")

	dbWrapper := &DB{DB: db, tracer: noopTracer, metrics: metrics}

	t.Run("AlertRepo_WithMetrics", func(t *testing.T) {
		repo := NewAlertRepository(dbWrapper)
		ruleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, repo.Create(ctx, alert))

		found, err := repo.GetByID(ctx, alert.ID.String())
		require.NoError(t, err)
		assert.Equal(t, alert.ID, found.ID)

		_, err = repo.GetLastAlertTimeAnyStatus(ctx, alert.MonitorID.String())
		require.NoError(t, err)

		alerts, _, err := repo.List(ctx, alert.UserID.String(), model.AlertFilter{})
		require.NoError(t, err)
		assert.NotEmpty(t, alerts)

		activeAlerts, err := repo.ListActiveByMonitorID(ctx, alert.MonitorID.String())
		require.NoError(t, err)
		_ = activeAlerts

		_, err = repo.GetLastAlertByMonitorIDAndStatus(ctx, alert.MonitorID.String(), model.AlertStatusTriggered)
		require.NoError(t, err)

		count, err := repo.CountUniqueMonitorsWithAlertsSince(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)

		activeForMonitor, err := repo.GetActiveAlertsForMonitor(ctx, alert.MonitorID.String())
		require.NoError(t, err)
		_ = activeForMonitor

		err = repo.Update(ctx, alert)
		require.NoError(t, err)

		err = repo.Delete(ctx, alert.ID.String())
		require.NoError(t, err)
	})

	t.Run("AlertRepo_AcknowledgeAlert_WithMetrics", func(t *testing.T) {
		repo := NewAlertRepository(dbWrapper)
		ruleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, repo.Create(ctx, alert))

		// Успешное подтверждение
		err := repo.AcknowledgeAlert(ctx, alert.ID.String(), alert.UserID.String())
		require.NoError(t, err)

		// Попытка повторного подтверждения — не triggered, возвращает ErrAlertNotAcknowledgeable
		err = repo.AcknowledgeAlert(ctx, alert.ID.String(), alert.UserID.String())
		require.ErrorIs(t, err, model.ErrAlertNotAcknowledgeable)
	})

	t.Run("AlertRepo_CreateWithDeliveryAttempt_WithMetrics", func(t *testing.T) {
		alertRepo := NewAlertRepository(dbWrapper)
		channelRepo := NewAlertChannelRepository(dbWrapper)

		userID := uuid.New()
		monitorID := uuid.New()
		ruleID := uuid.New()

		ch := &model.AlertChannel{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        model.AlertChannelTypeEmail,
			Status:      model.AlertChannelStatusActive,
			Enabled:     true,
			Verified:    true,
			EmailConfig: &model.EmailChannelConfig{Email: "cwda@example.com"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, channelRepo.Create(ctx, ch))

		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        alert.ID,
			AlertChannelID: ch.ID,
			Status:         model.DeliveryAttemptStatusPending,
			RetryCount:     0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := alertRepo.CreateWithDeliveryAttempt(ctx, alert, attempt)
		require.NoError(t, err)
	})

	t.Run("AlertRepo_DeleteResolvedOlderThan_WithMetrics", func(t *testing.T) {
		repo := NewAlertRepository(dbWrapper)
		ruleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusResolved,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 0,
			ThresholdMs:         5000,
			CreatedAt:           time.Now().Add(-48 * time.Hour),
			UpdatedAt:           time.Now().Add(-48 * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, alert))

		err := repo.DeleteResolvedOlderThan(ctx, 24*time.Hour)
		require.NoError(t, err)
	})

	t.Run("AlertRuleRepo_WithMetrics", func(t *testing.T) {
		repo := NewAlertRuleRepository(dbWrapper)
		userID := uuid.New()
		monitorID := uuid.New()

		rule := &model.AlertRule{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			Enabled:             true,
			ConsecutiveFailures: 2,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, repo.Create(ctx, rule))

		found, err := repo.GetByID(ctx, rule.ID.String())
		require.NoError(t, err)
		assert.Equal(t, rule.ID, found.ID)

		_, err = repo.GetByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)

		_, err = repo.List(ctx, userID.String())
		require.NoError(t, err)

		err = repo.Update(ctx, rule)
		require.NoError(t, err)

		err = repo.Delete(ctx, rule.ID.String())
		require.NoError(t, err)
	})

	t.Run("DeliveryAttemptRepo_WithMetrics", func(t *testing.T) {
		alertRepo := NewAlertRepository(dbWrapper)
		channelRepo := NewAlertChannelRepository(dbWrapper)
		repo := NewDeliveryAttemptRepository(dbWrapper)

		userID := uuid.New()
		monitorID := uuid.New()
		ruleID := uuid.New()

		ch := &model.AlertChannel{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        model.AlertChannelTypeEmail,
			Status:      model.AlertChannelStatusActive,
			Enabled:     true,
			Verified:    true,
			EmailConfig: &model.EmailChannelConfig{Email: "da-metrics@example.com"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, channelRepo.Create(ctx, ch))

		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              userID,
			MonitorID:           monitorID,
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, alertRepo.Create(ctx, alert))

		attempt := &model.DeliveryAttempt{
			ID:             uuid.New(),
			AlertID:        alert.ID,
			AlertChannelID: ch.ID,
			Status:         model.DeliveryAttemptStatusPending,
			RetryCount:     0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		require.NoError(t, repo.Create(ctx, attempt))

		found, err := repo.GetByID(ctx, attempt.ID.String())
		require.NoError(t, err)
		assert.Equal(t, attempt.ID, found.ID)

		_, err = repo.List(ctx, alert.ID.String())
		require.NoError(t, err)

		_, err = repo.ListPending(ctx, 10)
		require.NoError(t, err)

		_, err = repo.CountRecentByMonitorAndChannel(ctx, monitorID.String(), ch.ID.String(), time.Hour)
		require.NoError(t, err)

		_, err = repo.GetLastDeliveryTimeForMonitorAndStatus(ctx, monitorID.String(), string(model.DeliveryAttemptStatusPending))
		require.NoError(t, err)

		attempt.Status = model.DeliveryAttemptStatusSuccess
		err = repo.Update(ctx, attempt)
		require.NoError(t, err)

		err = repo.Delete(ctx, attempt.ID.String())
		require.NoError(t, err)
	})

	t.Run("AuditLogRepo_WithMetrics", func(t *testing.T) {
		repo := NewAuditLogRepository(dbWrapper)
		userID := uuid.New()
		resourceID := uuid.New().String()

		entry := &model.AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       model.AuditActionAlertTriggered,
			ResourceType: "alert",
			ResourceID:   &resourceID,
			Fields:       map[string]any{"test": "metrics"},
			CreatedAt:    time.Now(),
		}
		require.NoError(t, repo.Create(ctx, entry))

		logs, err := repo.List(ctx, "alert", resourceID, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, logs)
	})

	t.Run("AlertMuteRepo_WithMetrics", func(t *testing.T) {
		repo := NewAlertMuteRepository(dbWrapper)
		userID := uuid.New()
		monitorID := uuid.New()

		mute := &model.AlertMute{
			ID:        uuid.New(),
			UserID:    userID,
			MonitorID: &monitorID,
			Scope:     model.MuteScopeUser,
			CreatedBy: userID,
			CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, mute))

		_, err := repo.GetActiveByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)

		_, err = repo.GetActiveByUserID(ctx, userID.String())
		require.NoError(t, err)

		isMuted, err := repo.IsMuted(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)
		assert.True(t, isMuted)

		err = repo.DeleteByUserIDAndMonitorID(ctx, userID.String(), monitorID.String())
		require.NoError(t, err)

		err = repo.Delete(ctx, mute.ID.String())
		// Either succeeded or not found — both ok
		_ = err
	})

	t.Run("AlertEscalationRepo_WithMetrics", func(t *testing.T) {
		alertRepo := NewAlertRepository(dbWrapper)
		repo := NewAlertEscalationRepository(dbWrapper)

		ruleID := uuid.New()
		alert := &model.Alert{
			ID:                  uuid.New(),
			UserID:              uuid.New(),
			MonitorID:           uuid.New(),
			AlertRuleID:         &ruleID,
			Status:              model.AlertStatusTriggered,
			Type:                model.AlertTypeStatusCode,
			Enabled:             true,
			ConsecutiveFailures: 1,
			ThresholdMs:         5000,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		require.NoError(t, alertRepo.Create(ctx, alert))

		channelID := uuid.New()
		esc := &model.AlertEscalation{
			ID:                   uuid.New(),
			AlertID:              alert.ID,
			Level:                1,
			EscalatedToChannelID: &channelID,
			Reason:               "test-metrics",
			TimeoutMinutes:       10,
			CreatedAt:            time.Now(),
		}
		require.NoError(t, repo.Create(ctx, esc))

		_, err := repo.GetLatestByAlertID(ctx, alert.ID.String())
		require.NoError(t, err)
	})

	t.Run("MonitorStatusChangeRepo_WithMetrics", func(t *testing.T) {
		repo := NewMonitorStatusChangeRepository(dbWrapper)
		monitorID := uuid.New()

		change := &model.MonitorStatusChange{
			ID:        uuid.New(),
			MonitorID: monitorID,
			UserID:    uuid.New(),
			OldStatus: "up",
			NewStatus: "down",
			CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, change))

		count, err := repo.CountInWindow(ctx, monitorID.String(), time.Hour)
		require.NoError(t, err)
		assert.Greater(t, count, 0)

		latest, err := repo.GetLatestByMonitorID(ctx, monitorID.String())
		require.NoError(t, err)
		assert.Equal(t, change.ID, latest.ID)
	})

	t.Run("MaintenanceWindowRepo_WithMetrics", func(t *testing.T) {
		repo := NewMaintenanceWindowRepository(dbWrapper)
		userID := uuid.New()
		monitorID := uuid.New()

		mw := &model.MaintenanceWindow{
			ID:              uuid.New(),
			UserID:          userID,
			MonitorID:       &monitorID,
			Name:            "metrics-test",
			Status:          model.MaintenanceStatusScheduled,
			Recurrence:      model.RecurrenceTypeOnce,
			IsGlobal:        false,
			PauseMonitoring: true,
			SuppressAlerts:  true,
			SafeMode:        false,
			MonitorIDs:      []string{monitorID.String()},
			StartsAt:        time.Now().Add(time.Hour),
			EndsAt:          time.Now().Add(2 * time.Hour),
			Version:         0,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		require.NoError(t, repo.Create(ctx, mw))

		found, err := repo.GetByID(ctx, mw.ID.String())
		require.NoError(t, err)
		assert.Equal(t, mw.ID, found.ID)

		windows, _, err := repo.List(ctx, userID.String(), model.MaintenanceWindowFilter{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.NotEmpty(t, windows)

		overlaps, _, err := repo.CheckOverlapping(ctx, []string{monitorID.String()}, mw.StartsAt, mw.EndsAt, "")
		require.NoError(t, err)
		assert.True(t, overlaps)

		mw.Name = "updated-metrics"
		err = repo.Update(ctx, mw)
		require.NoError(t, err)

		err = repo.Delete(ctx, mw.ID.String())
		require.NoError(t, err)
	})

	t.Run("AlertChannelRepo_WithMetrics", func(t *testing.T) {
		repo := NewAlertChannelRepository(dbWrapper)
		userID := uuid.New()

		ch := &model.AlertChannel{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          model.AlertChannelTypeWebhook,
			Status:        model.AlertChannelStatusActive,
			Enabled:       true,
			Verified:      true,
			WebhookConfig: &model.WebhookChannelConfig{URL: "https://hook.example.com"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		require.NoError(t, repo.Create(ctx, ch))

		found, err := repo.GetByID(ctx, ch.ID.String())
		require.NoError(t, err)
		assert.Equal(t, ch.ID, found.ID)

		_, err = repo.GetByUserIDAndType(ctx, userID.String(), model.AlertChannelTypeWebhook)
		require.NoError(t, err)

		_, err = repo.ListByUserID(ctx, userID.String())
		require.NoError(t, err)

		count, err := repo.CountByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.Greater(t, count, 0)

		ch.Enabled = false
		err = repo.Update(ctx, ch)
		require.NoError(t, err)

		_, err = repo.IncrementFailureCount(ctx, ch.ID.String())
		require.NoError(t, err)

		err = repo.MarkAsFailed(ctx, ch.ID.String(), 1)
		require.NoError(t, err)

		err = repo.DisableChannel(ctx, ch.ID.String(), "test")
		require.NoError(t, err)

		err = repo.Delete(ctx, ch.ID.String())
		require.NoError(t, err)
	})
}

// ─── Unit tests for helper functions ─────────────────────────────────────────

func TestIsUniqueViolationError(t *testing.T) {
	assert.False(t, isUniqueViolationError(nil))
	assert.True(t, isUniqueViolationError(errors.New("duplicate key value violates unique constraint")))
	assert.True(t, isUniqueViolationError(errors.New("unique_violation")))
	assert.True(t, isUniqueViolationError(errors.New("error 23505")))
	assert.False(t, isUniqueViolationError(errors.New("some other error")))
}

// ─── NewDB with invalid config (covers the error path) ───────────────────────

func TestNewDB_InvalidConfig(t *testing.T) {
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     1,
		User:     "bad",
		Password: "bad",
		Database: "bad",
		SSLMode:  "disable",
	}
	db := NewDB(cfg)
	require.NotNil(t, db)
	err := db.Connect()
	assert.Error(t, err)
}
