// Тесты покрытия ветвей tracer+metrics для postgres-репозиториев.
// Покрывает ветви `if r.db.tracer != nil` и `if r.db.metrics != nil` во всех репозиториях.
package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// newMockDBWithTracerAndMetrics создаёт *DB с noop-трейсером и метриками для покрытия обеих ветвей.
func newMockDBWithTracerAndMetrics(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("mock db close: %v", err)
		}
	})
	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &DB{DB: sqlxDB}
	dbWrapper.tracer = noop.NewTracerProvider().Tracer("test")
	metrics := apptelemetry.NewMetrics("test")
	dbWrapper.metrics = metrics
	return dbWrapper, mock
}

// ─── AlertMuteRepository — tracer+metrics paths ───────────────────────────────

func TestAlertMuteRepository_Create_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_mutes").WillReturnResult(sqlmock.NewResult(1, 1))

	mute := &model.AlertMute{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Scope:     model.MuteScopeUser,
		CreatedAt: time.Now(),
		CreatedBy: uuid.New(),
	}
	err := repo.Create(context.Background(), mute)
	assert.NoError(t, err)
}

func TestAlertMuteRepository_Delete_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertMuteRepository_Delete_TracerMetrics_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertMuteRepository_GetActiveByMonitorID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	monitorID := uuid.New()
	userID := uuid.New()
	rows := sqlmock.NewRows([]string{"id", "user_id", "monitor_id", "scope", "muted_until", "created_at", "created_by"}).
		AddRow(uuid.New(), userID, monitorID, model.MuteScopeUser, nil, now, userID)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetActiveByMonitorID(context.Background(), monitorID.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAlertMuteRepository_GetActiveByUserID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "user_id", "monitor_id", "scope", "muted_until", "created_at", "created_by"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetActiveByUserID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAlertMuteRepository_DeleteByUserIDAndMonitorID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(2, 2))

	err := repo.DeleteByUserIDAndMonitorID(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertMuteRepository_DeleteExpired_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(3, 3))

	n, err := repo.DeleteExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestAlertMuteRepository_IsMuted_TracerMetrics_True(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	muted, err := repo.IsMuted(context.Background(), uuid.New().String(), uuid.New().String())
	require.NoError(t, err)
	assert.True(t, muted)
}

func TestAlertMuteRepository_IsMuted_TracerMetrics_False(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertMuteRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	muted, err := repo.IsMuted(context.Background(), uuid.New().String(), uuid.New().String())
	require.NoError(t, err)
	assert.False(t, muted)
}

// ─── AlertRuleRepository — tracer+metrics paths ───────────────────────────────

func TestAlertRuleRepository_Create_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_rules").WillReturnResult(sqlmock.NewResult(1, 1))

	rule := &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		MonitorID:           uuid.New(),
		Enabled:             true,
		ConsecutiveFailures: 3,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	err := repo.Create(context.Background(), rule)
	assert.NoError(t, err)
}

func TestAlertRuleRepository_GetByID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	ruleID := uuid.New()
	rows := sqlmock.NewRows([]string{"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at"}).
		AddRow(ruleID, uuid.New(), uuid.New(), true, 3, now, now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), ruleID.String())
	require.NoError(t, err)
	assert.Equal(t, ruleID, result.ID)
}

func TestAlertRuleRepository_GetByUserIDAndMonitorID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at"}).
		AddRow(uuid.New(), uuid.New(), uuid.New(), true, 2, now, now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetByUserIDAndMonitorID(context.Background(), uuid.New().String(), uuid.New().String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAlertRuleRepository_List_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.List(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAlertRuleRepository_Update_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("UPDATE alert_rules").WillReturnResult(sqlmock.NewResult(1, 1))

	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Enabled:   false,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), rule)
	assert.NoError(t, err)
}

func TestAlertRuleRepository_Update_TracerMetrics_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("UPDATE alert_rules").WillReturnResult(sqlmock.NewResult(0, 0))

	rule := &model.AlertRule{ID: uuid.New(), UserID: uuid.New(), UpdatedAt: time.Now()}
	err := repo.Update(context.Background(), rule)
	assert.ErrorIs(t, err, model.ErrAlertRuleNotFound)
}

func TestAlertRuleRepository_Delete_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_rules").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRuleRepository_Delete_TracerMetrics_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_rules").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertRuleNotFound)
}

// ─── AuditLogRepository — success paths ──────────────────────────────────────

func TestAuditLogRepository_Create_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAuditLogRepository(mockDB)

	mock.ExpectExec("INSERT INTO audit_logs").WillReturnResult(sqlmock.NewResult(1, 1))

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
	err := repo.Create(context.Background(), entry)
	assert.NoError(t, err)
}

func TestAuditLogRepository_List_TracerMetrics_Success_Empty(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAuditLogRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "user_id", "action", "resource_type", "resource_id", "fields", "ip_address", "created_at"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.List(context.Background(), "alert", uuid.New().String(), 10)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAuditLogRepository_List_TracerMetrics_WithRows(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAuditLogRepository(mockDB)

	logID := uuid.New().String()
	userID := uuid.New().String()
	resourceID := uuid.New().String()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{"id", "user_id", "action", "resource_type", "resource_id", "fields", "ip_address", "created_at"}).
		AddRow(logID, &userID, model.AuditActionAlertTriggered, "alert", &resourceID, []byte(`{"key":"val"}`), nil, now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.List(context.Background(), "alert", resourceID, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, model.AuditActionAlertTriggered, result[0].Action)
}

// ─── AlertEscalationRepository — tracer+metrics paths ────────────────────────

func TestAlertEscalationRepository_Create_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertEscalationRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_escalations").WillReturnResult(sqlmock.NewResult(1, 1))

	escalation := &model.AlertEscalation{
		ID:             uuid.New(),
		AlertID:        uuid.New(),
		Level:          1,
		Reason:         "timeout",
		TimeoutMinutes: 30,
		CreatedAt:      time.Now(),
	}
	err := repo.Create(context.Background(), escalation)
	assert.NoError(t, err)
}

func TestAlertEscalationRepository_GetLatestByAlertID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewAlertEscalationRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"id", "alert_id", "level", "escalated_to_channel_id", "reason", "timeout_minutes", "created_at"}).
		AddRow(uuid.New(), uuid.New(), 1, nil, "timeout_no_ack", 30, now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetLatestByAlertID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Level)
}

// ─── MonitorStatusChangeRepository — tracer+metrics paths ────────────────────

func TestMonitorStatusChangeRepository_Create_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	mock.ExpectExec("INSERT INTO monitor_status_changes").WillReturnResult(sqlmock.NewResult(1, 1))

	change := &model.MonitorStatusChange{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		OldStatus: "UP",
		NewStatus: "DOWN",
		CreatedAt: time.Now(),
	}
	err := repo.Create(context.Background(), change)
	assert.NoError(t, err)
}

func TestMonitorStatusChangeRepository_CountInWindow_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountInWindow(context.Background(), uuid.New().String(), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestMonitorStatusChangeRepository_GetLatestByMonitorID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"id", "monitor_id", "old_status", "new_status", "created_at"}).
		AddRow(uuid.New(), uuid.New(), "UP", "DOWN", now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetLatestByMonitorID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Equal(t, "DOWN", result.NewStatus)
}

// ─── DeliveryAttemptRepository — tracer+metrics paths ────────────────────────

func TestDeliveryAttemptRepository_Create_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("INSERT INTO delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))

	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        uuid.New(),
		AlertChannelID: uuid.New(),
		Status:         model.DeliveryAttemptStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := repo.Create(context.Background(), attempt)
	assert.NoError(t, err)
}

func TestDeliveryAttemptRepository_GetByID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()
	rows := sqlmock.NewRows([]string{"id", "alert_id", "alert_channel_id", "status", "error_message", "retry_count", "next_retry_at", "created_at", "updated_at"}).
		AddRow(id, uuid.New(), uuid.New(), model.DeliveryAttemptStatusPending, "", 0, nil, now, now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), id.String())
	require.NoError(t, err)
	assert.Equal(t, id, result.ID)
}

func TestDeliveryAttemptRepository_List_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "alert_id", "alert_channel_id", "status", "error_message", "retry_count", "next_retry_at", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.List(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeliveryAttemptRepository_ListPending_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "alert_id", "alert_channel_id", "status", "error_message", "retry_count", "next_retry_at", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.ListPending(context.Background(), 100)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeliveryAttemptRepository_Update_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("UPDATE delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))

	attempt := &model.DeliveryAttempt{
		ID:        uuid.New(),
		Status:    model.DeliveryAttemptStatusSuccess,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), attempt)
	assert.NoError(t, err)
}

func TestDeliveryAttemptRepository_Update_TracerMetrics_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("UPDATE delivery_attempts").WillReturnResult(sqlmock.NewResult(0, 0))

	attempt := &model.DeliveryAttempt{ID: uuid.New(), Status: model.DeliveryAttemptStatusFailed, UpdatedAt: time.Now()}
	err := repo.Update(context.Background(), attempt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeliveryAttemptRepository_Delete_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestDeliveryAttemptRepository_DeleteOldAttempts_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnResult(sqlmock.NewResult(5, 5))

	cutoff := time.Now().Add(-30 * 24 * time.Hour).Unix()
	err := repo.DeleteOldAttempts(context.Background(), cutoff)
	assert.NoError(t, err)
}

// ─── MaintenanceWindowRepository — List and CheckOverlapping success paths ───

func TestMaintenanceWindowRepository_List_TracerMetrics_Empty(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	windows, total, err := repo.List(context.Background(), uuid.New().String(), model.MaintenanceWindowFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, windows)
}

func TestMaintenanceWindowRepository_List_TracerMetrics_WithRows(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	}).AddRow(
		uuid.New(), uuid.New(), nil, "Test Window",
		string(model.MaintenanceStatusScheduled), string(model.RecurrenceTypeOnce),
		false, false, true, false,
		pq.StringArray{"monitor-1", "monitor-2"},
		now, now.Add(time.Hour), nil, nil,
		1, nil, now, now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	windows, total, err := repo.List(context.Background(), uuid.New().String(), model.MaintenanceWindowFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, windows, 1)
	assert.Equal(t, "Test Window", windows[0].Name)
}

func TestMaintenanceWindowRepository_List_WithFilter_Status(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	filter := model.MaintenanceWindowFilter{Status: model.MaintenanceStatusScheduled}
	windows, total, err := repo.List(context.Background(), uuid.New().String(), filter)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, windows)
}

func TestMaintenanceWindowRepository_List_WithFilter_Dates(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	now := time.Now()

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	filter := model.MaintenanceWindowFilter{
		StartDate: &now,
		EndDate:   &now,
		Page:      2,
		PageSize:  5,
	}
	windows, _, err := repo.List(context.Background(), uuid.New().String(), filter)
	require.NoError(t, err)
	assert.Empty(t, windows)
}

func TestMaintenanceWindowRepository_List_QueryError(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.MaintenanceWindowFilter{})
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_GetByID_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()

	row := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	}).AddRow(
		id, uuid.New(), nil, "Window",
		string(model.MaintenanceStatusActive), string(model.RecurrenceTypeOnce),
		false, true, true, false,
		pq.StringArray{},
		now, now.Add(time.Hour), &now, nil,
		1, nil, now, now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(row)

	result, err := repo.GetByID(context.Background(), id.String())
	require.NoError(t, err)
	assert.Equal(t, id, result.ID)
}

func TestMaintenanceWindowRepository_GetByID_TracerMetrics_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{}))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotFound)
}

func TestMaintenanceWindowRepository_GetActiveByMonitorID_TracerMetrics_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{}))

	result, err := repo.GetActiveByMonitorID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMaintenanceWindowRepository_GetActiveByMonitorID_TracerMetrics_Found(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()

	row := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	}).AddRow(
		id, uuid.New(), nil, "Active Window",
		string(model.MaintenanceStatusActive), string(model.RecurrenceTypeOnce),
		false, false, true, false,
		pq.StringArray{"monitor-id"},
		now, now.Add(time.Hour), &now, nil,
		1, nil, now, now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(row)

	result, err := repo.GetActiveByMonitorID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}

func TestMaintenanceWindowRepository_CheckOverlapping_TracerMetrics_Found(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()

	row := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	}).AddRow(
		id, uuid.New(), nil, "Existing Window",
		string(model.MaintenanceStatusActive), string(model.RecurrenceTypeOnce),
		false, false, false, false,
		pq.StringArray{"m1"},
		now, now.Add(2*time.Hour), nil, nil,
		1, nil, now, now,
	)
	mock.ExpectQuery("SELECT").WillReturnRows(row)

	overlaps, mw, err := repo.CheckOverlapping(context.Background(), []string{"m1"}, now, now.Add(time.Hour), "")
	require.NoError(t, err)
	assert.True(t, overlaps)
	require.NotNil(t, mw)
	assert.Equal(t, id, mw.ID)
}

func TestMaintenanceWindowRepository_Update_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("UPDATE maintenance_windows").WillReturnResult(sqlmock.NewResult(1, 1))

	now := time.Now()
	mw := &model.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Updated Window",
		Status:    model.MaintenanceStatusCompleted,
		StartsAt:  now,
		EndsAt:    now.Add(time.Hour),
		Version:   2,
		UpdatedAt: now,
	}
	err := repo.Update(context.Background(), mw)
	assert.NoError(t, err)
}

func TestMaintenanceWindowRepository_Delete_TracerMetrics_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracerAndMetrics(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("DELETE FROM maintenance_windows").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}
