package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// ─── AlertRepository.GetByID ──────────────────────────────────────────────────

func TestAlertRepository_GetByID_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	alertID := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	ruleID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(
		alertID, userID, monitorID, ruleID,
		model.AlertStatusTriggered, model.AlertTypeStatusCode, true,
		3, 500, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(alertID.String()).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), alertID.String())
	require.NoError(t, err)
	assert.Equal(t, alertID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, model.AlertStatusTriggered, result.Status)
	assert.Equal(t, 3, result.ConsecutiveFailures)
}

func TestAlertRepository_GetByID_NotFound_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("sql: no rows in result set"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert by id")
}

// ─── AlertRepository.Update ───────────────────────────────────────────────────

func TestAlertRepository_Update_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusResolved,
		Type:      model.AlertTypeStatusCode,
		Enabled:   true,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.NoError(t, err)
}

func TestAlertRepository_Update_NotFound_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertRepository_Update_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnError(errors.New("connection lost"))

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), alert)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update alert")
}

// ─── AlertRepository.Delete ───────────────────────────────────────────────────

func TestAlertRepository_Delete_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_Delete_NotFound_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotFound)
}

func TestAlertRepository_Delete_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete alert")
}

// ─── AlertRepository.AcknowledgeAlert ────────────────────────────────────────

func TestAlertRepository_AcknowledgeAlert_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertRepository_AcknowledgeAlert_NotAcknowledgeable_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertNotAcknowledgeable)
}

func TestAlertRepository_AcknowledgeAlert_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("UPDATE alerts").WillReturnError(errors.New("db error"))

	err := repo.AcknowledgeAlert(context.Background(), uuid.New().String(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acknowledge alert")
}

// ─── AlertRepository.List ─────────────────────────────────────────────────────

func TestAlertRepository_List_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	// ExpectQuery for count
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	// ExpectQuery for list
	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), userID, uuid.New(), nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now).
		AddRow(uuid.New(), userID, uuid.New(), nil, model.AlertStatusResolved, model.AlertTypeResponseTime, true, 0, 500, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	alerts, total, err := repo.List(context.Background(), userID.String(), model.AlertFilter{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, alerts, 2)
}

func TestAlertRepository_List_WithFilters_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	userID := uuid.New()
	monitorID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	listRows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), userID, monitorID, nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(listRows)

	filter := model.AlertFilter{
		MonitorID: monitorID.String(),
		Status:    model.AlertStatusTriggered,
		Page:      1,
		PageSize:  10,
	}

	alerts, total, err := repo.List(context.Background(), userID.String(), filter)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, alerts, 1)
}

func TestAlertRepository_List_CountError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.AlertFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count alerts")
}

func TestAlertRepository_List_SelectError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, _, err := repo.List(context.Background(), uuid.New().String(), model.AlertFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list alerts")
}

// ─── AlertRepository.ListActiveByMonitorID ───────────────────────────────────

func TestAlertRepository_ListActiveByMonitorID_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	monitorID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(uuid.New(), userID, monitorID, nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").WithArgs(monitorID.String()).WillReturnRows(rows)

	alerts, err := repo.ListActiveByMonitorID(context.Background(), monitorID.String())
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
}

func TestAlertRepository_ListActiveByMonitorID_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ListActiveByMonitorID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list active alerts")
}

// ─── AlertRepository.GetLastAlertTimeAnyStatus ───────────────────────────────

func TestAlertRepository_GetLastAlertTimeAnyStatus_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	monitorID := uuid.New()
	alertID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(alertID, uuid.New(), monitorID, nil, model.AlertStatusResolved, model.AlertTypeStatusCode, true, 0, 0, now, now)

	mock.ExpectQuery("SELECT").WithArgs(monitorID.String()).WillReturnRows(rows)

	result, err := repo.GetLastAlertTimeAnyStatus(context.Background(), monitorID.String())
	require.NoError(t, err)
	assert.Equal(t, alertID, result.ID)
}

func TestAlertRepository_GetLastAlertTimeAnyStatus_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("no rows"))

	_, err := repo.GetLastAlertTimeAnyStatus(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get last alert")
}

// ─── AlertRepository.DeleteResolvedOlderThan ─────────────────────────────────

func TestAlertRepository_DeleteResolvedOlderThan_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnResult(sqlmock.NewResult(5, 5))

	err := repo.DeleteResolvedOlderThan(context.Background(), 24*time.Hour)
	assert.NoError(t, err)
}

func TestAlertRepository_DeleteResolvedOlderThan_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectExec("DELETE FROM alerts").WillReturnError(errors.New("db error"))

	err := repo.DeleteResolvedOlderThan(context.Background(), 24*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete resolved alerts")
}

// ─── AlertRepository.GetLastAlertByMonitorIDAndStatus ────────────────────────

func TestAlertRepository_GetLastAlertByMonitorIDAndStatus_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	monitorID := uuid.New()
	alertID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).AddRow(alertID, uuid.New(), monitorID, nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now)

	mock.ExpectQuery("SELECT").
		WithArgs(monitorID.String(), model.AlertStatusTriggered).
		WillReturnRows(rows)

	result, err := repo.GetLastAlertByMonitorIDAndStatus(context.Background(), monitorID.String(), model.AlertStatusTriggered)
	require.NoError(t, err)
	assert.Equal(t, alertID, result.ID)
}

func TestAlertRepository_GetLastAlertByMonitorIDAndStatus_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("no rows"))

	_, err := repo.GetLastAlertByMonitorIDAndStatus(context.Background(), uuid.New().String(), model.AlertStatusResolved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get last alert by monitor and status")
}

// ─── AlertRepository.CountUniqueMonitorsWithAlertsSince ──────────────────────

func TestAlertRepository_CountUniqueMonitorsWithAlertsSince_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(7)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)

	count, err := repo.CountUniqueMonitorsWithAlertsSince(context.Background(), time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 7, count)
}

func TestAlertRepository_CountUniqueMonitorsWithAlertsSince_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountUniqueMonitorsWithAlertsSince(context.Background(), time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count unique monitors with alerts since")
}

// ─── AlertRepository.GetActiveAlertsForMonitor ───────────────────────────────

func TestAlertRepository_GetActiveAlertsForMonitor_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	monitorID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled",
		"consecutive_failures", "threshold_ms", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), userID, monitorID, nil, model.AlertStatusTriggered, model.AlertTypeStatusCode, true, 1, 0, now, now).
		AddRow(uuid.New(), userID, monitorID, nil, model.AlertStatusAcknowledged, model.AlertTypeResponseTime, true, 0, 500, now, now)

	mock.ExpectQuery("SELECT").WithArgs(monitorID.String()).WillReturnRows(rows)

	alerts, err := repo.GetActiveAlertsForMonitor(context.Background(), monitorID.String())
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
}

func TestAlertRepository_GetActiveAlertsForMonitor_DBError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetActiveAlertsForMonitor(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get active alerts for monitor")
}

// ─── AlertRepository.CreateWithDeliveryAttempt ───────────────────────────────

func TestAlertRepository_CreateWithDeliveryAttempt_Success_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO alerts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO delivery_attempts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeStatusCode,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        alert.ID,
		AlertChannelID: uuid.New(),
		Status:         model.DeliveryAttemptStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.NoError(t, err)
}

func TestAlertRepository_CreateWithDeliveryAttempt_BeginError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := repo.CreateWithDeliveryAttempt(context.Background(), &model.Alert{}, &model.DeliveryAttempt{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestAlertRepository_CreateWithDeliveryAttempt_AlertInsertError_Mock(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO alerts").WillReturnError(errors.New("insert error"))
	mock.ExpectRollback()

	alert := &model.Alert{ID: uuid.New(), UserID: uuid.New(), MonitorID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	attempt := &model.DeliveryAttempt{ID: uuid.New(), AlertID: alert.ID, AlertChannelID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}

	err := repo.CreateWithDeliveryAttempt(context.Background(), alert, attempt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create alert in transaction")
}
