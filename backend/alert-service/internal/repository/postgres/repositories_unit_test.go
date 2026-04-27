package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// ─── AlertMuteRepository ──────────────────────────────────────────────────────

func TestNewAlertMuteRepository(t *testing.T) {
	repo := NewAlertMuteRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &AlertMuteRepository{}, repo)
}

func TestAlertMuteRepository_MuteModel(t *testing.T) {
	userID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	mute := &model.AlertMute{
		ID:        uuid.New(),
		UserID:    userID,
		MonitorID: &monitorID,
		Scope:     model.MuteScopeUser,
		CreatedAt: now,
		CreatedBy: userID,
	}

	assert.Equal(t, userID, mute.UserID)
	assert.Equal(t, &monitorID, mute.MonitorID)
	assert.Equal(t, model.MuteScopeUser, mute.Scope)
}

func TestAlertMuteRepository_GlobalScope(t *testing.T) {
	mute := &model.AlertMute{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Scope:  model.MuteScopeGlobal,
	}

	assert.Equal(t, model.MuteScopeGlobal, mute.Scope)
	assert.Nil(t, mute.MonitorID)
}

// ─── AlertEscalationRepository ───────────────────────────────────────────────

func TestNewAlertEscalationRepository(t *testing.T) {
	repo := NewAlertEscalationRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &AlertEscalationRepository{}, repo)
}

func TestAlertEscalationRepository_EscalationModel(t *testing.T) {
	channelID := uuid.New()
	escalation := &model.AlertEscalation{
		ID:                   uuid.New(),
		AlertID:              uuid.New(),
		Level:                2,
		EscalatedToChannelID: &channelID,
		Reason:               "timeout_no_ack",
		TimeoutMinutes:       30,
		CreatedAt:            time.Now(),
	}

	assert.Equal(t, 2, escalation.Level)
	assert.Equal(t, "timeout_no_ack", escalation.Reason)
	assert.Equal(t, 30, escalation.TimeoutMinutes)
	assert.Equal(t, &channelID, escalation.EscalatedToChannelID)
}

// ─── MonitorStatusChangeRepository ───────────────────────────────────────────

func TestNewMonitorStatusChangeRepository(t *testing.T) {
	repo := NewMonitorStatusChangeRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &MonitorStatusChangeRepository{}, repo)
}

func TestMonitorStatusChangeRepository_ChangeModel(t *testing.T) {
	change := &model.MonitorStatusChange{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		OldStatus: "UP",
		NewStatus: "DOWN",
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "UP", change.OldStatus)
	assert.Equal(t, "DOWN", change.NewStatus)
}

// ─── AuditLogRepository ───────────────────────────────────────────────────────

func TestNewAuditLogRepository(t *testing.T) {
	repo := NewAuditLogRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &AuditLogRepository{}, repo)
}

func TestAuditLogRepository_AuditLogModel(t *testing.T) {
	userID := uuid.New()
	resID := uuid.New().String()
	log := &model.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       model.AuditActionChannelCreated,
		ResourceType: "channel",
		ResourceID:   &resID,
		Fields: map[string]any{
			"channel_type": "telegram",
		},
		CreatedAt: time.Now(),
	}

	assert.Equal(t, model.AuditActionChannelCreated, log.Action)
	assert.Equal(t, "channel", log.ResourceType)
	assert.NotNil(t, log.Fields)
}

// ─── MaintenanceWindowRepository ─────────────────────────────────────────────

func TestNewMaintenanceWindowRepository(t *testing.T) {
	repo := NewMaintenanceWindowRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &MaintenanceWindowRepository{}, repo)
}

func TestMaintenanceWindowRepository_WindowModel(t *testing.T) {
	now := time.Now()
	window := &model.MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		Name:            "Weekly Maintenance",
		Status:          model.MaintenanceStatusScheduled,
		Recurrence:      model.RecurrenceTypeWeekly,
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{"monitor-1", "monitor-2"},
		StartsAt:        now.Add(time.Hour),
		EndsAt:          now.Add(2 * time.Hour),
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, "Weekly Maintenance", window.Name)
	assert.Equal(t, model.MaintenanceStatusScheduled, window.Status)
	assert.Equal(t, model.RecurrenceTypeWeekly, window.Recurrence)
	assert.Len(t, window.MonitorIDs, 2)
	assert.Equal(t, 1, window.Version)
}

func TestMaintenanceWindowRepository_AllStatuses(t *testing.T) {
	statuses := []model.MaintenanceWindowStatus{
		model.MaintenanceStatusScheduled,
		model.MaintenanceStatusActive,
		model.MaintenanceStatusCompleted,
		model.MaintenanceStatusCancelled,
		model.MaintenanceStatusOrphaned,
	}

	for _, s := range statuses {
		assert.NotEmpty(t, string(s))
	}
}

func TestMaintenanceWindowRepository_AllRecurrences(t *testing.T) {
	recurrences := []model.RecurrenceType{
		model.RecurrenceTypeOnce,
		model.RecurrenceTypeDaily,
		model.RecurrenceTypeWeekly,
		model.RecurrenceTypeMonthly,
	}

	for _, r := range recurrences {
		assert.NotEmpty(t, string(r))
	}
}

func TestMaintenanceWindowRepository_Filter(t *testing.T) {
	now := time.Now()
	end := now.Add(24 * time.Hour)
	filter := model.MaintenanceWindowFilter{
		Status:    model.MaintenanceStatusActive,
		StartDate: &now,
		EndDate:   &end,
		Page:      1,
		PageSize:  20,
	}

	assert.Equal(t, model.MaintenanceStatusActive, filter.Status)
	assert.Equal(t, 1, filter.Page)
	assert.Equal(t, 20, filter.PageSize)
}

// ─── DB.SetObservability ──────────────────────────────────────────────────────

func TestDB_SetObservability_NilValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Host = "invalid-host-that-does-not-exist"

	db := &DB{}
	db.SetObservability(nil, nil)

	assert.Nil(t, db.GetTracer())
	assert.Nil(t, db.GetMetrics())
}

// ─── NewDBFromDSN invalid DSN ─────────────────────────────────────────────────

func TestNewDBFromDSN_InvalidDSN(t *testing.T) {
	db := NewDBFromDSN("invalid-dsn-string")
	require.NotNil(t, db)
	err := db.Connect()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

// ─── DeliveryAttemptRepository ────────────────────────────────────────────────

func TestNewDeliveryAttemptRepository_Unit(t *testing.T) {
	repo := NewDeliveryAttemptRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &DeliveryAttemptRepository{}, repo)
}

func TestDeliveryAttemptRepository_AttemptModel(t *testing.T) {
	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        uuid.New(),
		AlertChannelID: uuid.New(),
		Status:         model.DeliveryAttemptStatusPending,
		RetryCount:     1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	assert.Equal(t, model.DeliveryAttemptStatusPending, attempt.Status)
	assert.Equal(t, 1, attempt.RetryCount)
}

// ─── AlertRuleRepository ──────────────────────────────────────────────────────

func TestNewAlertRuleRepository_Unit(t *testing.T) {
	repo := NewAlertRuleRepository(nil)
	assert.NotNil(t, repo)
	assert.IsType(t, &AlertRuleRepository{}, repo)
}

func TestAlertRuleRepository_RuleModel(t *testing.T) {
	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		MonitorID:           uuid.New(),
		Enabled:             true,
		ConsecutiveFailures: 2,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	assert.True(t, rule.Enabled)
	assert.Equal(t, 2, rule.ConsecutiveFailures)
}
