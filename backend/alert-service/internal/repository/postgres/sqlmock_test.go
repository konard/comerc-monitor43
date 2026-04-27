package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// newMockDB создаёт *DB на основе sqlmock для юнит-тестов без реальной БД
func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("mock db close: %v", err)
		}
	})
	sqlxDB := sqlx.NewDb(db, "postgres")
	return &DB{DB: sqlxDB}, mock
}

// ─── AlertRepository ──────────────────────────────────────────────────────────

func TestAlertRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("INSERT INTO alerts").WillReturnError(errors.New("db error"))

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeStatusCode,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), alert)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create alert")
}

func TestAlertRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("INSERT INTO alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeStatusCode,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), alert)
	assert.NoError(t, err)
}

func TestAlertRepository_GetByID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert by id")
}

func TestAlertRepository_GetByID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type",
		"enabled", "consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(id, userID, monitorID, nil, "triggered", "status_code", true, 0, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), id.String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAlertRepository_GetLastAlertTimeAnyStatus_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetLastAlertTimeAnyStatus(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get last alert")
}

func TestAlertRepository_GetLastAlertTimeAnyStatus_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type",
		"enabled", "consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(id, userID, monitorID, nil, "triggered", "status_code", true, 0, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetLastAlertTimeAnyStatus(context.Background(), monitorID.String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAlertRepository_ListActiveByMonitorID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ListActiveByMonitorID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list active alerts")
}

func TestAlertRepository_ListActiveByMonitorID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type",
		"enabled", "consecutive_failures", "threshold_ms", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.ListActiveByMonitorID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestAlertRepository_List_Mock_CountError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("count error"))

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.AlertFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count alerts")
}

func TestAlertRepository_List_Mock_ListError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("list error"))

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.AlertFilter{Page: 1, PageSize: 10})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list alerts")
}

func TestAlertRepository_List_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	id := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()
	dataRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type",
		"enabled", "consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(id, userID, monitorID, nil, "triggered", "status_code", true, 0, 0, now, now)
	mock.ExpectQuery("SELECT").WillReturnRows(dataRows)

	alerts, total, err := repo.List(context.Background(), userID.String(), model.AlertFilter{
		MonitorID: monitorID.String(),
		Status:    model.AlertStatusTriggered,
		Page:      1,
		PageSize:  10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, alerts, 1)
}

func TestAlertRepository_Update_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnError(errors.New("db error"))

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update alert")
}

func TestAlertRepository_Update_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	alert := &model.Alert{
		ID:        uuid.New(),
		Status:    model.AlertStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertRepository_Update_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	alert := &model.Alert{
		ID:        uuid.New(),
		Status:    model.AlertStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.NoError(t, err)
}

func TestAlertRepository_Delete_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete alert")
}

func TestAlertRepository_Delete_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertRepository_Delete_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_DeleteResolvedOlderThan_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnError(errors.New("db error"))

	err := repo.DeleteResolvedOlderThan(context.Background(), 24*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete resolved alerts")
}

func TestAlertRepository_DeleteResolvedOlderThan_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(5, 5))

	err := repo.DeleteResolvedOlderThan(context.Background(), 24*time.Hour)
	assert.NoError(t, err)
}

func TestAlertRepository_GetLastAlertByMonitorIDAndStatus_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetLastAlertByMonitorIDAndStatus(context.Background(), uuid.New().String(), model.AlertStatusTriggered)
	assert.Error(t, err)
}

func TestAlertRepository_GetLastAlertByMonitorIDAndStatus_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type",
		"enabled", "consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(id, userID, monitorID, nil, "triggered", "status_code", true, 0, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetLastAlertByMonitorIDAndStatus(context.Background(), monitorID.String(), model.AlertStatusTriggered)
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAlertRepository_CountUniqueMonitorsWithAlertsSince_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountUniqueMonitorsWithAlertsSince(context.Background(), time.Now())
	assert.Error(t, err)
}

func TestAlertRepository_CountUniqueMonitorsWithAlertsSince_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(3)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountUniqueMonitorsWithAlertsSince(context.Background(), time.Now().Add(-time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestAlertRepository_AcknowledgeAlert_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnError(errors.New("db error"))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_AcknowledgeAlert_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotAcknowledgeable)
}

func TestAlertRepository_AcknowledgeAlert_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_GetActiveAlertsForMonitor_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetActiveAlertsForMonitor(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_GetActiveAlertsForMonitor_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type",
		"enabled", "consecutive_failures", "threshold_ms", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetActiveAlertsForMonitor(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestAlertRepository_CreateWithDeliveryAttempt_Mock_BeginError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	alert := &model.Alert{ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	attempt := &model.DeliveryAttempt{ID: uuid.New(), AlertID: alert.ID, AlertChannelID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestAlertRepository_CreateWithDeliveryAttempt_Mock_AlertError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO alerts").WillReturnError(errors.New("insert error"))
	mock.ExpectRollback()

	alert := &model.Alert{ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	attempt := &model.DeliveryAttempt{ID: uuid.New(), AlertID: alert.ID, AlertChannelID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.Error(t, err)
}

func TestAlertRepository_CreateWithDeliveryAttempt_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO alerts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	alert := &model.Alert{ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	attempt := &model.DeliveryAttempt{ID: uuid.New(), AlertID: alert.ID, AlertChannelID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.NoError(t, err)
}

// ─── AlertRuleRepository ──────────────────────────────────────────────────────

func TestAlertRuleRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_rules").WillReturnError(errors.New("db error"))

	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create alert rule")
}

func TestAlertRuleRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_rules").WillReturnResult(sqlmock.NewResult(1, 1))

	rule := &model.AlertRule{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), rule)
	assert.NoError(t, err)
}

func TestAlertRuleRepository_GetByID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRuleRepository_GetByID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	id := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at",
	}).AddRow(id, uuid.New(), uuid.New(), true, 2, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), id.String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAlertRuleRepository_GetByUserIDAndMonitorID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByUserIDAndMonitorID(context.Background(), uuid.New().String(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRuleRepository_GetByUserIDAndMonitorID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	id := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at",
	}).AddRow(id, uuid.New(), uuid.New(), true, 1, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetByUserIDAndMonitorID(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestAlertRuleRepository_List_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.List(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRuleRepository_List_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.List(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestAlertRuleRepository_Update_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("UPDATE alert_rules").WillReturnError(errors.New("db error"))

	rule := &model.AlertRule{ID: uuid.New(), Enabled: false, UpdatedAt: time.Now()}
	err := repo.Update(context.Background(), rule)
	assert.Error(t, err)
}

func TestAlertRuleRepository_Update_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("UPDATE alert_rules").WillReturnResult(sqlmock.NewResult(0, 0))

	rule := &model.AlertRule{ID: uuid.New(), Enabled: false, UpdatedAt: time.Now()}
	err := repo.Update(context.Background(), rule)
	assert.ErrorIs(t, err, model.ErrAlertRuleNotFound)
}

func TestAlertRuleRepository_Update_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("UPDATE alert_rules").WillReturnResult(sqlmock.NewResult(1, 1))

	rule := &model.AlertRule{ID: uuid.New(), Enabled: true, UpdatedAt: time.Now()}
	err := repo.Update(context.Background(), rule)
	assert.NoError(t, err)
}

func TestAlertRuleRepository_Delete_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_rules").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRuleRepository_Delete_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_rules").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertRuleNotFound)
}

func TestAlertRuleRepository_Delete_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRuleRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_rules").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

// ─── DeliveryAttemptRepository ────────────────────────────────────────────────

func TestDeliveryAttemptRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("INSERT INTO delivery_attempts").WillReturnError(errors.New("db error"))

	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        uuid.New(),
		AlertChannelID: uuid.New(),
		Status:         model.DeliveryAttemptStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := repo.Create(context.Background(), attempt)
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
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

func TestDeliveryAttemptRepository_GetByID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_GetByID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	id := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "alert_id", "alert_channel_id", "status", "error_message",
		"retry_count", "next_retry_at", "created_at", "updated_at",
	}).AddRow(id, uuid.New(), uuid.New(), "pending", "", 0, nil, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), id.String())
	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestDeliveryAttemptRepository_List_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.List(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_List_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "alert_id", "alert_channel_id", "status", "error_message",
		"retry_count", "next_retry_at", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.List(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestDeliveryAttemptRepository_ListPending_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ListPending(context.Background(), 10)
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_ListPending_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "alert_id", "alert_channel_id", "status", "error_message",
		"retry_count", "next_retry_at", "created_at", "updated_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.ListPending(context.Background(), 10)
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestDeliveryAttemptRepository_Update_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("UPDATE delivery_attempts").WillReturnError(errors.New("db error"))

	attempt := &model.DeliveryAttempt{
		ID:        uuid.New(),
		Status:    model.DeliveryAttemptStatusSuccess,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), attempt)
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_Update_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("UPDATE delivery_attempts").WillReturnResult(sqlmock.NewResult(0, 0))

	attempt := &model.DeliveryAttempt{
		ID:        uuid.New(),
		Status:    model.DeliveryAttemptStatusSuccess,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), attempt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delivery attempt not found")
}

func TestDeliveryAttemptRepository_Update_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
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

func TestDeliveryAttemptRepository_Delete_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_Delete_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestDeliveryAttemptRepository_DeleteOldAttempts_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnError(errors.New("db error"))

	err := repo.DeleteOldAttempts(context.Background(), 90)
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_DeleteOldAttempts_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnResult(sqlmock.NewResult(3, 3))

	err := repo.DeleteOldAttempts(context.Background(), 90)
	assert.NoError(t, err)
}

func TestDeliveryAttemptRepository_CleanupOldAttempts_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectExec("DELETE FROM delivery_attempts").WillReturnResult(sqlmock.NewResult(2, 2))

	err := repo.CleanupOldAttempts(context.Background(), 30*24*time.Hour)
	assert.NoError(t, err)
}

func TestDeliveryAttemptRepository_CountRecentByMonitorAndChannel_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountRecentByMonitorAndChannel(context.Background(), uuid.New().String(), uuid.New().String(), time.Hour)
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_CountRecentByMonitorAndChannel_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountRecentByMonitorAndChannel(context.Background(), uuid.New().String(), uuid.New().String(), time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestDeliveryAttemptRepository_GetLastDeliveryTimeForMonitorAndStatus_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetLastDeliveryTimeForMonitorAndStatus(context.Background(), uuid.New().String(), "delivered")
	assert.Error(t, err)
}

func TestDeliveryAttemptRepository_GetLastDeliveryTimeForMonitorAndStatus_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewDeliveryAttemptRepository(mockDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"created_at"}).AddRow(now)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetLastDeliveryTimeForMonitorAndStatus(context.Background(), uuid.New().String(), "delivered")
	assert.NoError(t, err)
	assert.WithinDuration(t, now, *got, time.Second)
}

// ─── AuditLogRepository ───────────────────────────────────────────────────────

func TestAuditLogRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAuditLogRepository(mockDB)

	mock.ExpectExec("INSERT INTO audit_logs").WillReturnError(errors.New("db error"))

	userID := uuid.New()
	resID := uuid.New().String()
	log := &model.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       model.AuditActionChannelCreated,
		ResourceType: "channel",
		ResourceID:   &resID,
		Fields:       map[string]any{"key": "value"},
		CreatedAt:    time.Now(),
	}

	err := repo.Create(context.Background(), log)
	assert.Error(t, err)
}

func TestAuditLogRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAuditLogRepository(mockDB)

	mock.ExpectExec("INSERT INTO audit_logs").WillReturnResult(sqlmock.NewResult(1, 1))

	userID := uuid.New()
	resID := uuid.New().String()
	log := &model.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       model.AuditActionChannelCreated,
		ResourceType: "channel",
		ResourceID:   &resID,
		Fields:       map[string]any{"key": "value"},
		CreatedAt:    time.Now(),
	}

	err := repo.Create(context.Background(), log)
	assert.NoError(t, err)
}

func TestAuditLogRepository_List_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAuditLogRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.List(context.Background(), "channel", uuid.New().String(), 10)
	assert.Error(t, err)
}

func TestAuditLogRepository_List_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAuditLogRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "action", "resource_type", "resource_id",
		"fields", "ip_address", "created_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.List(context.Background(), "channel", uuid.New().String(), 10)
	assert.NoError(t, err)
	assert.Empty(t, got)
}

// ─── AlertChannelRepository (error paths) ────────────────────────────────────

func TestAlertChannelRepository_Create_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnError(errors.New("db error"))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeTelegram,
		Status:    model.AlertChannelStatusActive,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.Error(t, err)
}

func TestAlertChannelRepository_Create_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeTelegram,
		Status:    model.AlertChannelStatusActive,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_GetByID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_GetByUserIDAndType_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByUserIDAndType(context.Background(), uuid.New().String(), model.AlertChannelTypeTelegram)
	assert.Error(t, err)
}

func TestAlertChannelRepository_ListByUserID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ListByUserID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_Update_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeTelegram,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), channel)
	assert.Error(t, err)
}

func TestAlertChannelRepository_Update_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(0, 0))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeTelegram,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), channel)
	assert.ErrorIs(t, err, model.ErrAlertChannelNotFound)
}

func TestAlertChannelRepository_Update_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeEmail,
		UpdatedAt: time.Now(),
	}
	err := repo.Update(context.Background(), channel)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Delete_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_Delete_Mock_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertChannelNotFound)
}

func TestAlertChannelRepository_Delete_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertChannelRepository_ExistsDuplicate_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ExistsDuplicate(context.Background(), uuid.New().String(), model.AlertChannelTypeTelegram, "-1001234")
	assert.Error(t, err)
}

func TestAlertChannelRepository_ExistsDuplicate_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	// ExistsDuplicate returns telegram_config/email_config/webhook_config columns
	telegramJSON := []byte(`{"chat_id":"-1001234"}`)
	rows := sqlmock.NewRows([]string{"telegram_config", "email_config", "webhook_config"}).
		AddRow(telegramJSON, nil, nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	exists, err := repo.ExistsDuplicate(context.Background(), uuid.New().String(), model.AlertChannelTypeTelegram, "-1001234")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestAlertChannelRepository_MarkAsFailed_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	err := repo.MarkAsFailed(context.Background(), uuid.New().String(), 3)
	assert.Error(t, err)
}

func TestAlertChannelRepository_MarkAsFailed_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.MarkAsFailed(context.Background(), uuid.New().String(), 3)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_CountByUserID_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountByUserID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_CountByUserID_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountByUserID(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestAlertChannelRepository_IncrementFailureCount_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	_, err := repo.IncrementFailureCount(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_IncrementFailureCount_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{"failure_count"}).AddRow(3)
	mock.ExpectQuery("UPDATE alert_channels").WillReturnRows(rows)

	count, err := repo.IncrementFailureCount(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestAlertChannelRepository_DisableChannel_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	err := repo.DisableChannel(context.Background(), uuid.New().String(), "bounce")
	assert.Error(t, err)
}

func TestAlertChannelRepository_DisableChannel_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DisableChannel(context.Background(), uuid.New().String(), "bounce")
	assert.NoError(t, err)
}

func TestAlertChannelRepository_GetChannelPriorities_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetChannelPriorities(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_GetChannelPriorities_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "alert_rule_id", "alert_channel_id", "priority", "created_at",
	})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	got, err := repo.GetChannelPriorities(context.Background(), uuid.New().String())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestAlertChannelRepository_SetChannelPriorities_Mock_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	priorities := []model.AlertChannelPriority{
		{
			ID:             uuid.New(),
			AlertRuleID:    uuid.New(),
			AlertChannelID: uuid.New(),
			Priority:       1,
			CreatedAt:      time.Now(),
		},
	}

	err := repo.SetChannelPriorities(context.Background(), uuid.New().String(), priorities)
	assert.Error(t, err)
}

func TestAlertChannelRepository_SetChannelPriorities_Mock_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO alert_channel_priorities").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	priorities := []model.AlertChannelPriority{
		{
			ID:             uuid.New(),
			AlertRuleID:    uuid.New(),
			AlertChannelID: uuid.New(),
			Priority:       1,
			CreatedAt:      time.Now(),
		},
	}

	err := repo.SetChannelPriorities(context.Background(), uuid.New().String(), priorities)
	assert.NoError(t, err)
}

// ─── DB transaction methods ───────────────────────────────────────────────────

func TestDB_BeginTxx_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection error"))

	_, err := mockDB.BeginTxx(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestDB_BeginTxx_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := mockDB.BeginTxx(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	assert.NoError(t, mockDB.Rollback(tx))
}

func TestDB_Commit_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := mockDB.BeginTxx(context.Background(), nil)
	require.NoError(t, err)

	err = mockDB.Commit(tx)
	assert.NoError(t, err)
}

func TestDB_Rollback_InvalidType_Mock(t *testing.T) {
	mockDB, _ := newMockDB(t)

	err := mockDB.Rollback("not a transaction")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type")
}

func TestDB_Commit_InvalidType_Mock(t *testing.T) {
	mockDB, _ := newMockDB(t)

	err := mockDB.Commit("not a transaction")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type")
}
