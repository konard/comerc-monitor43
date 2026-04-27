package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// ─── AlertMuteRepository ──────────────────────────────────────────────────────

func TestAlertMuteRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_mutes").WillReturnError(errors.New("db error"))

	mute := &model.AlertMute{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Scope:     model.MuteScopeUser,
		CreatedAt: time.Now(),
		CreatedBy: uuid.New(),
	}

	err := repo.Create(context.Background(), mute)
	assert.Error(t, err)
}

func TestAlertMuteRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
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

func TestAlertMuteRepository_Delete_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertMuteRepository_Delete_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertMuteRepository_GetActiveByMonitorID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetActiveByMonitorID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertMuteRepository_GetActiveByMonitorID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "scope", "muted_until", "created_at", "created_by",
	}).AddRow(id, userID, nil, "user", nil, now, userID)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetActiveByMonitorID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAlertMuteRepository_GetActiveByUserID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetActiveByUserID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertMuteRepository_GetActiveByUserID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "scope", "muted_until", "created_at", "created_by",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetActiveByUserID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestAlertMuteRepository_DeleteByUserIDAndMonitorID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnError(errors.New("db error"))

	err := repo.DeleteByUserIDAndMonitorID(context.Background(), uuid.New().String(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertMuteRepository_DeleteByUserIDAndMonitorID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteByUserIDAndMonitorID(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertMuteRepository_DeleteExpired_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnError(errors.New("db error"))

	_, err := repo.DeleteExpired(context.Background())
	assert.Error(t, err)
}

func TestAlertMuteRepository_DeleteExpired_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_mutes").WillReturnResult(sqlmock.NewResult(3, 3))

	n, err := repo.DeleteExpired(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestAlertMuteRepository_IsMuted_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.IsMuted(context.Background(), uuid.New().String(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertMuteRepository_IsMuted_Mock_True(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	muted, err := repo.IsMuted(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
	assert.True(t, muted)
}

func TestAlertMuteRepository_IsMuted_Mock_False(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertMuteRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	muted, err := repo.IsMuted(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
	assert.False(t, muted)
}

// ─── AlertEscalationRepository ───────────────────────────────────────────────

func TestAlertEscalationRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertEscalationRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_escalations").WillReturnError(errors.New("db error"))

	escalation := &model.AlertEscalation{
		ID:             uuid.New(),
		AlertID:        uuid.New(),
		Level:          1,
		Reason:         "timeout",
		TimeoutMinutes: 15,
		CreatedAt:      time.Now(),
	}

	err := repo.Create(context.Background(), escalation)
	assert.Error(t, err)
}

func TestAlertEscalationRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertEscalationRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_escalations").WillReturnResult(sqlmock.NewResult(1, 1))

	escalation := &model.AlertEscalation{
		ID:             uuid.New(),
		AlertID:        uuid.New(),
		Level:          1,
		Reason:         "timeout",
		TimeoutMinutes: 15,
		CreatedAt:      time.Now(),
	}

	err := repo.Create(context.Background(), escalation)
	assert.NoError(t, err)
}

func TestAlertEscalationRepository_GetLatestByAlertID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertEscalationRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetLatestByAlertID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertEscalationRepository_GetLatestByAlertID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertEscalationRepository(mockDB)

	id := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "alert_id", "level", "escalated_to_channel_id", "reason", "timeout_minutes", "created_at",
	}).AddRow(id, uuid.New(), 2, nil, "timeout_no_ack", 30, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetLatestByAlertID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

// ─── MonitorStatusChangeRepository ───────────────────────────────────────────

func TestMonitorStatusChangeRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	mock.ExpectExec("INSERT INTO monitor_status_changes").WillReturnError(errors.New("db error"))

	change := &model.MonitorStatusChange{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		OldStatus: "UP",
		NewStatus: "DOWN",
		CreatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), change)
	assert.Error(t, err)
}

func TestMonitorStatusChangeRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	mock.ExpectExec("INSERT INTO monitor_status_changes").WillReturnResult(sqlmock.NewResult(1, 1))

	change := &model.MonitorStatusChange{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		OldStatus: "UP",
		NewStatus: "DOWN",
		CreatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), change)
	assert.NoError(t, err)
}

func TestMonitorStatusChangeRepository_CountInWindow_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountInWindow(context.Background(), uuid.New().String(), time.Hour)
	assert.Error(t, err)
}

func TestMonitorStatusChangeRepository_CountInWindow_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountInWindow(context.Background(), uuid.New().String(), time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestMonitorStatusChangeRepository_GetLatestByMonitorID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetLatestByMonitorID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestMonitorStatusChangeRepository_GetLatestByMonitorID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMonitorStatusChangeRepository(mockDB)

	id := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "user_id", "old_status", "new_status", "created_at",
	}).AddRow(id, uuid.New(), uuid.New(), "UP", "DOWN", now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetLatestByMonitorID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

// ─── MaintenanceWindowRepository ─────────────────────────────────────────────

func TestMaintenanceWindowRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("INSERT INTO maintenance_windows").WillReturnError(errors.New("db error"))

	now := time.Now()
	mw := &model.MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		Name:            "Test Window",
		Status:          model.MaintenanceStatusScheduled,
		Recurrence:      model.RecurrenceTypeOnce,
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{uuid.New().String()},
		StartsAt:        now.Add(time.Hour),
		EndsAt:          now.Add(2 * time.Hour),
		Version:         0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := repo.Create(context.Background(), mw)
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("INSERT INTO maintenance_windows").WillReturnResult(sqlmock.NewResult(1, 1))

	now := time.Now()
	mw := &model.MaintenanceWindow{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "Test Window",
		Status:     model.MaintenanceStatusScheduled,
		Recurrence: model.RecurrenceTypeOnce,
		MonitorIDs: []string{uuid.New().String()},
		StartsAt:   now.Add(time.Hour),
		EndsAt:     now.Add(2 * time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := repo.Create(context.Background(), mw)
	assert.NoError(t, err)
}

func TestMaintenanceWindowRepository_GetByID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_GetActiveByMonitorID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetActiveByMonitorID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_IsUnderMaintenance_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.IsUnderMaintenance(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_IsUnderMaintenance_Mock_True(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	active, err := repo.IsUnderMaintenance(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.True(t, active)
}

func TestMaintenanceWindowRepository_Update_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("UPDATE maintenance_windows").WillReturnError(errors.New("db error"))

	now := time.Now()
	mw := &model.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Updated",
		Status:    model.MaintenanceStatusScheduled,
		UpdatedAt: now,
		Version:   1,
	}

	err := repo.Update(context.Background(), mw)
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_Update_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("UPDATE maintenance_windows").WillReturnResult(sqlmock.NewResult(0, 0))

	now := time.Now()
	mw := &model.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Updated",
		Status:    model.MaintenanceStatusScheduled,
		UpdatedAt: now,
		Version:   1,
	}

	err := repo.Update(context.Background(), mw)
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_Update_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("UPDATE maintenance_windows").WillReturnResult(sqlmock.NewResult(1, 1))

	now := time.Now()
	mw := &model.MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Updated",
		Status:    model.MaintenanceStatusActive,
		UpdatedAt: now,
		Version:   1,
	}

	err := repo.Update(context.Background(), mw)
	assert.NoError(t, err)
}

func TestMaintenanceWindowRepository_Delete_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("DELETE FROM maintenance_windows").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_Delete_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("DELETE FROM maintenance_windows").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrMaintenanceWindowNotFound)
}

func TestMaintenanceWindowRepository_Delete_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectExec("DELETE FROM maintenance_windows").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestMaintenanceWindowRepository_List_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.MaintenanceWindowFilter{})
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_CheckOverlapping_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	now := time.Now()
	_, _, err := repo.CheckOverlapping(context.Background(), []string{"monitor-1"}, now, now.Add(time.Hour), "")
	assert.Error(t, err)
}

func TestMaintenanceWindowRepository_CheckOverlapping_Mock_NoOverlap(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewMaintenanceWindowRepository(mockDB)

	// Empty rows causes sql.ErrNoRows which signals no overlap
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "name", "status", "recurrence",
		"is_global", "pause_monitoring", "suppress_alerts", "safe_mode",
		"monitor_ids", "starts_at", "ends_at", "activated_at", "completed_at",
		"version", "reason", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	now := time.Now()
	overlaps, mw, err := repo.CheckOverlapping(context.Background(), []string{"monitor-1"}, now, now.Add(time.Hour), "")
	assert.NoError(t, err)
	assert.False(t, overlaps)
	assert.Nil(t, mw)
}

// ─── AlertChannelRepository success paths for low-coverage methods ────────────

func TestAlertChannelRepository_ListByUserID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config",
		"webhook_config", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.ListByUserID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestAlertChannelRepository_GetByUserIDAndType_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("sql: no rows"))

	_, err := repo.GetByUserIDAndType(context.Background(), uuid.New().String(), model.AlertChannelTypeEmail)
	assert.Error(t, err)
}

// ─── DB.BeginTx ──────────────────────────────────────────────────────────────

func TestDB_BeginTx_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))

	_, err := mockDB.BeginTx(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestDB_BeginTx_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := mockDB.BeginTx(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.NoError(t, tx.Commit())
}
