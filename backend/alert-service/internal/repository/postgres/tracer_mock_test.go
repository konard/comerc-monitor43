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
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// newMockDBWithTracer создаёт *DB c noop-трейсером для покрытия ветвей span
func newMockDBWithTracer(t *testing.T) (*DB, sqlmock.Sqlmock) {
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
	return dbWrapper, mock
}

// ─── AlertRepository — tracing paths ──────────────────────────────────────────

func TestAlertRepository_Create_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
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

func TestAlertRepository_Create_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
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
}

func TestAlertRepository_GetByID_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	alertID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(alertID, uuid.New(), uuid.New(), nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), alertID.String())
	require.NoError(t, err)
	assert.Equal(t, alertID, result.ID)
}

func TestAlertRepository_GetByID_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_Update_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	alert := &model.Alert{
		ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(),
		Status: model.AlertStatusResolved, UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.NoError(t, err)
}

func TestAlertRepository_Update_Tracer_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	alert := &model.Alert{
		ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(),
		Status: model.AlertStatusResolved, UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertRepository_Update_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnError(errors.New("db error"))

	alert := &model.Alert{
		ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(),
		Status: model.AlertStatusTriggered, UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.Error(t, err)
}

func TestAlertRepository_Delete_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_Delete_Tracer_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertRepository_Delete_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_AcknowledgeAlert_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_AcknowledgeAlert_Tracer_NotAcknowledgeable(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotAcknowledgeable)
}

func TestAlertRepository_AcknowledgeAlert_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnError(errors.New("db error"))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_ListActiveByMonitorID_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), uuid.New(), nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	alerts, err := repo.ListActiveByMonitorID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
}

func TestAlertRepository_ListActiveByMonitorID_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ListActiveByMonitorID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_GetLastAlertTimeAnyStatus_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), uuid.New(), nil, model.AlertStatusResolved, model.AlertTypeStatusCode, true, 0, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, err := repo.GetLastAlertTimeAnyStatus(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_GetLastAlertTimeAnyStatus_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("no rows"))

	_, err := repo.GetLastAlertTimeAnyStatus(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_DeleteResolvedOlderThan_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(3, 3))

	err := repo.DeleteResolvedOlderThan(context.Background(), 24*time.Hour)
	assert.NoError(t, err)
}

func TestAlertRepository_DeleteResolvedOlderThan_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnError(errors.New("db error"))

	err := repo.DeleteResolvedOlderThan(context.Background(), 24*time.Hour)
	assert.Error(t, err)
}

func TestAlertRepository_GetLastAlertByMonitorIDAndStatus_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), uuid.New(), nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, err := repo.GetLastAlertByMonitorIDAndStatus(context.Background(), uuid.New().String(), model.AlertStatusTriggered)
	assert.NoError(t, err)
}

func TestAlertRepository_GetLastAlertByMonitorIDAndStatus_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("no rows"))

	_, err := repo.GetLastAlertByMonitorIDAndStatus(context.Background(), uuid.New().String(), model.AlertStatusTriggered)
	assert.Error(t, err)
}

func TestAlertRepository_CountUniqueMonitorsWithAlertsSince_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountUniqueMonitorsWithAlertsSince(context.Background(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestAlertRepository_CountUniqueMonitorsWithAlertsSince_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountUniqueMonitorsWithAlertsSince(context.Background(), time.Now())
	assert.Error(t, err)
}

func TestAlertRepository_GetActiveAlertsForMonitor_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), uuid.New(), nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, err := repo.GetActiveAlertsForMonitor(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_GetActiveAlertsForMonitor_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetActiveAlertsForMonitor(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertRepository_List_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now().UTC().Truncate(time.Second)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), uuid.New(), nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.AlertFilter{})
	assert.NoError(t, err)
}

func TestAlertRepository_CreateWithDeliveryAttempt_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO alerts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	alert := &model.Alert{
		ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(),
		Status: model.AlertStatusTriggered, Type: model.AlertTypeStatusCode,
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	attempt := &model.DeliveryAttempt{
		ID: uuid.New(), AlertID: alert.ID, AlertChannelID: uuid.New(),
		Status: model.DeliveryAttemptStatusPending, CreatedAt: now, UpdatedAt: now,
	}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.NoError(t, err)
}

func TestAlertRepository_CreateWithDeliveryAttempt_Tracer_DeliveryInsertError(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO alerts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO delivery_attempts").WillReturnError(errors.New("insert error"))
	mock.ExpectRollback()

	alert := &model.Alert{
		ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(),
		Status: model.AlertStatusTriggered, Type: model.AlertTypeStatusCode,
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	attempt := &model.DeliveryAttempt{
		ID: uuid.New(), AlertID: alert.ID, AlertChannelID: uuid.New(),
		Status: model.DeliveryAttemptStatusPending, CreatedAt: now, UpdatedAt: now,
	}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create delivery attempt in transaction")
}

// ─── AlertChannelRepository — tracing paths ───────────────────────────────────

func TestAlertChannelRepository_Create_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	channel := &model.AlertChannel{
		ID: uuid.New(), UserID: uuid.New(), Type: model.AlertChannelTypeTelegram,
		Status: model.AlertChannelStatusUnverified, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Create_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnError(errors.New("db error"))

	channel := &model.AlertChannel{
		ID: uuid.New(), UserID: uuid.New(), Type: model.AlertChannelTypeEmail,
		Status: model.AlertChannelStatusUnverified, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.Error(t, err)
}

func TestAlertChannelRepository_GetByID_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).AddRow(id, uuid.New(), model.AlertChannelTypeTelegram, model.AlertChannelStatusActive,
		true, true, 0, nil, nil, nil, nil, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, err := repo.GetByID(context.Background(), id.String())
	assert.NoError(t, err)
}

func TestAlertChannelRepository_GetByID_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_Update_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	channel := &model.AlertChannel{
		ID: uuid.New(), Status: model.AlertChannelStatusActive, UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), channel)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Update_Tracer_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(0, 0))

	channel := &model.AlertChannel{
		ID: uuid.New(), Status: model.AlertChannelStatusActive, UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), channel)
	assert.ErrorIs(t, err, model.ErrAlertChannelNotFound)
}

func TestAlertChannelRepository_Update_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	channel := &model.AlertChannel{
		ID: uuid.New(), Status: model.AlertChannelStatusActive, UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), channel)
	assert.Error(t, err)
}

func TestAlertChannelRepository_Delete_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Delete_Tracer_NotFound(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertChannelNotFound)
}

func TestAlertChannelRepository_Delete_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_CountByUserID_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountByUserID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestAlertChannelRepository_CountByUserID_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountByUserID(context.Background(), uuid.New().String())
	assert.Error(t, err)
}

func TestAlertChannelRepository_SetChannelPriorities_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	ruleID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").
		WithArgs(ruleID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.SetChannelPriorities(context.Background(), ruleID.String(), nil)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_SetChannelPriorities_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").
		WillReturnError(errors.New("delete error"))
	mock.ExpectRollback()

	err := repo.SetChannelPriorities(context.Background(), uuid.New().String(), nil)
	assert.Error(t, err)
}

func TestAlertChannelRepository_GetChannelPriorities_Tracer_Success(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "alert_rule_id", "alert_channel_id", "priority", "created_at"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetChannelPriorities(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAlertChannelRepository_GetChannelPriorities_Tracer_Error(t *testing.T) {
	mockDB, mock := newMockDBWithTracer(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetChannelPriorities(context.Background(), uuid.New().String())
	assert.Error(t, err)
}
