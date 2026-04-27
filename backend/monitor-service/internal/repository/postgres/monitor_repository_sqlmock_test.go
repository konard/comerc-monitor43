package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

// monitorColumns возвращает список колонок таблицы monitors.
func monitorColumns() []string {
	return []string{
		"id", "user_id", "name", "url", "check_type", "interval_seconds", "timeout_seconds",
		"status", "working_hours_start", "working_hours_end", "working_days",
		"degraded_response_time_threshold", "degraded_failure_rate_threshold",
		"last_check_at", "created_at", "updated_at",
	}
}

// addMonitorRow добавляет строку монитора в mock rows.
func addMonitorRow(rows *sqlmock.Rows, id, userID uuid.UUID, name, url, status string) *sqlmock.Rows {
	now := time.Now()
	return rows.AddRow(
		id, userID, name, url, "HTTP", 60, 30,
		status, nil, nil, pq.StringArray{},
		nil, nil,
		nil, now, now,
	)
}

// TestMonitorRepository_Create_Success тестирует успешное создание монитора.
func TestMonitorRepository_Create_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitor := &domain.Monitor{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		Name:            "Test Monitor",
		URL:             "https://example.com",
		CheckType:       "HTTP",
		IntervalSeconds: 60,
		TimeoutSeconds:  30,
		Status:          domain.StatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO monitors`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), monitor)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Create_Error тестирует ошибку при создании монитора.
func TestMonitorRepository_Create_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitor := &domain.Monitor{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Test Monitor",
		URL:       "https://example.com",
		CheckType: "HTTP",
		Status:    domain.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO monitors`)).
		WillReturnError(sql.ErrConnDone)

	err = repo.Create(context.Background(), monitor)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_GetByID_Success тестирует успешное получение монитора по ID.
func TestMonitorRepository_GetByID_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows(monitorColumns())
	addMonitorRow(rows, monitorID, userID, "Test Monitor", "https://example.com", "UP")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE id = $1
	`)).WithArgs(monitorID).WillReturnRows(rows)

	monitor, err := repo.GetByID(context.Background(), monitorID)

	assert.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, monitorID, monitor.ID)
	assert.Equal(t, domain.StatusUp, monitor.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_GetByID_NotFound тестирует случай, когда монитор не найден.
func TestMonitorRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE id = $1
	`)).WithArgs(monitorID).WillReturnError(sql.ErrNoRows)

	monitor, err := repo.GetByID(context.Background(), monitorID)

	assert.Error(t, err)
	assert.Nil(t, monitor)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_GetByID_DBError тестирует ошибку БД при получении монитора.
func TestMonitorRepository_GetByID_DBError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE id = $1
	`)).WithArgs(monitorID).WillReturnError(sql.ErrConnDone)

	monitor, err := repo.GetByID(context.Background(), monitorID)

	assert.Error(t, err)
	assert.Nil(t, monitor)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_GetByUserIDAndName_Success тестирует получение монитора по userID и имени.
func TestMonitorRepository_GetByUserIDAndName_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows(monitorColumns())
	addMonitorRow(rows, monitorID, userID, "My Monitor", "https://example.com", "UP")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1 AND name = $2
	`)).WithArgs(userID, "My Monitor").WillReturnRows(rows)

	monitor, err := repo.GetByUserIDAndName(context.Background(), userID, "My Monitor")

	assert.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, monitorID, monitor.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_GetByUserIDAndName_NotFound тестирует случай, когда монитор не найден.
func TestMonitorRepository_GetByUserIDAndName_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1 AND name = $2
	`)).WithArgs(userID, "Unknown Monitor").WillReturnError(sql.ErrNoRows)

	monitor, err := repo.GetByUserIDAndName(context.Background(), userID, "Unknown Monitor")

	assert.Error(t, err)
	assert.Nil(t, monitor)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListByUserID_Success тестирует успешный листинг мониторов.
func TestMonitorRepository_ListByUserID_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	rows := sqlmock.NewRows(monitorColumns())
	addMonitorRow(rows, uuid.New(), userID, "Monitor 1", "https://example1.com", "UP")
	addMonitorRow(rows, uuid.New(), userID, "Monitor 2", "https://example2.com", "DOWN")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(userID, 10, 0).WillReturnRows(rows)

	monitors, err := repo.ListByUserID(context.Background(), userID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, monitors, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListByUserID_Error тестирует ошибку при листинге.
func TestMonitorRepository_ListByUserID_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(userID, 10, 0).WillReturnError(sql.ErrConnDone)

	monitors, err := repo.ListByUserID(context.Background(), userID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, monitors)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListByUserIDAndStatus_Success тестирует листинг по статусу.
func TestMonitorRepository_ListByUserIDAndStatus_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	rows := sqlmock.NewRows(monitorColumns())
	addMonitorRow(rows, uuid.New(), userID, "Active Monitor", "https://example.com", "UP")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`)).WithArgs(userID, domain.StatusUp).WillReturnRows(rows)

	monitors, err := repo.ListByUserIDAndStatus(context.Background(), userID, domain.StatusUp)

	assert.NoError(t, err)
	assert.Len(t, monitors, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListActive_Success тестирует листинг активных мониторов.
func TestMonitorRepository_ListActive_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	rows := sqlmock.NewRows(monitorColumns())
	addMonitorRow(rows, uuid.New(), uuid.New(), "Monitor 1", "https://example1.com", "UP")
	addMonitorRow(rows, uuid.New(), uuid.New(), "Monitor 2", "https://example2.com", "DOWN")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
		ORDER BY created_at DESC
	`)).WillReturnRows(rows)

	monitors, err := repo.ListActive(context.Background())

	assert.NoError(t, err)
	assert.Len(t, monitors, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListActive_Error тестирует ошибку при листинге активных мониторов.
func TestMonitorRepository_ListActive_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
		ORDER BY created_at DESC
	`)).WillReturnError(sql.ErrConnDone)

	monitors, err := repo.ListActive(context.Background())

	assert.Error(t, err)
	assert.Nil(t, monitors)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListDueForCheck_Success тестирует листинг мониторов для проверки.
func TestMonitorRepository_ListDueForCheck_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	rows := sqlmock.NewRows(monitorColumns())
	addMonitorRow(rows, uuid.New(), uuid.New(), "Due Monitor", "https://example.com", "UP")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
			AND (last_check_at IS NULL OR last_check_at + (interval_seconds || ' seconds')::interval < NOW())
		ORDER BY last_check_at ASC NULLS FIRST
		LIMIT $1
	`)).WithArgs(10).WillReturnRows(rows)

	monitors, err := repo.ListDueForCheck(context.Background(), 10)

	assert.NoError(t, err)
	assert.Len(t, monitors, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListDueForCheck_Error тестирует ошибку при запросе.
func TestMonitorRepository_ListDueForCheck_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
			AND (last_check_at IS NULL OR last_check_at + (interval_seconds || ' seconds')::interval < NOW())
		ORDER BY last_check_at ASC NULLS FIRST
		LIMIT $1
	`)).WithArgs(10).WillReturnError(sql.ErrConnDone)

	monitors, err := repo.ListDueForCheck(context.Background(), 10)

	assert.Error(t, err)
	assert.Nil(t, monitors)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Update_Success тестирует успешное обновление монитора.
func TestMonitorRepository_Update_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitor := &domain.Monitor{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		Name:            "Updated Monitor",
		URL:             "https://updated.com",
		CheckType:       "HTTP",
		IntervalSeconds: 120,
		TimeoutSeconds:  60,
		Status:          domain.StatusUp,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE monitors SET`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Update(context.Background(), monitor)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Update_NotFound тестирует случай, когда монитор не найден при обновлении.
func TestMonitorRepository_Update_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitor := &domain.Monitor{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		Name:            "Updated Monitor",
		URL:             "https://updated.com",
		CheckType:       "HTTP",
		IntervalSeconds: 120,
		TimeoutSeconds:  60,
		Status:          domain.StatusUp,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE monitors SET`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Update(context.Background(), monitor)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Update_Error тестирует ошибку при обновлении.
func TestMonitorRepository_Update_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitor := &domain.Monitor{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Monitor",
		URL:       "https://example.com",
		CheckType: "HTTP",
		Status:    domain.StatusUp,
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE monitors SET`)).
		WillReturnError(sql.ErrConnDone)

	err = repo.Update(context.Background(), monitor)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_UpdateStatus_Success тестирует успешное обновление статуса.
func TestMonitorRepository_UpdateStatus_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE monitors SET
			status = $1,
			updated_at = $2
		WHERE id = $3
	`)).WithArgs(domain.StatusDown, sqlmock.AnyArg(), monitorID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateStatus(context.Background(), monitorID, domain.StatusDown)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_UpdateStatus_NotFound тестирует случай, когда монитор не найден.
func TestMonitorRepository_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE monitors SET
			status = $1,
			updated_at = $2
		WHERE id = $3
	`)).WithArgs(domain.StatusDown, sqlmock.AnyArg(), monitorID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateStatus(context.Background(), monitorID, domain.StatusDown)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_UpdateStatus_Error тестирует ошибку при обновлении статуса.
func TestMonitorRepository_UpdateStatus_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE monitors SET
			status = $1,
			updated_at = $2
		WHERE id = $3
	`)).WithArgs(domain.StatusDown, sqlmock.AnyArg(), monitorID).
		WillReturnError(sql.ErrConnDone)

	err = repo.UpdateStatus(context.Background(), monitorID, domain.StatusDown)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_UpdateLastCheck_Success тестирует успешное обновление last_check_at.
func TestMonitorRepository_UpdateLastCheck_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()
	lastCheckAt := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE monitors SET
			last_check_at = $1,
			updated_at = $2
		WHERE id = $3
	`)).WithArgs(lastCheckAt, sqlmock.AnyArg(), monitorID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateLastCheck(context.Background(), monitorID, lastCheckAt)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_UpdateLastCheck_NotFound тестирует случай, когда монитор не найден.
func TestMonitorRepository_UpdateLastCheck_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()
	lastCheckAt := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE monitors SET
			last_check_at = $1,
			updated_at = $2
		WHERE id = $3
	`)).WithArgs(lastCheckAt, sqlmock.AnyArg(), monitorID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateLastCheck(context.Background(), monitorID, lastCheckAt)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Delete_Success тестирует успешное удаление монитора.
func TestMonitorRepository_Delete_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM monitors WHERE id = $1`)).
		WithArgs(monitorID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(context.Background(), monitorID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Delete_NotFound тестирует случай, когда монитор не найден при удалении.
func TestMonitorRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM monitors WHERE id = $1`)).
		WithArgs(monitorID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(context.Background(), monitorID)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_Delete_Error тестирует ошибку при удалении монитора.
func TestMonitorRepository_Delete_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM monitors WHERE id = $1`)).
		WithArgs(monitorID).
		WillReturnError(sql.ErrConnDone)

	err = repo.Delete(context.Background(), monitorID)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_CountByUserID_Success тестирует успешный подсчёт мониторов.
func TestMonitorRepository_CountByUserID_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM monitors WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.CountByUserID(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_CountByUserID_Error тестирует ошибку при подсчёте.
func TestMonitorRepository_CountByUserID_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM monitors WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.CountByUserID(context.Background(), userID)

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ExistsByName_True тестирует проверку существования монитора (найден).
func TestMonitorRepository_ExistsByName_True(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM monitors WHERE user_id = $1 AND name = $2)`)).
		WithArgs(userID, "Existing Monitor").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.ExistsByName(context.Background(), userID, "Existing Monitor")

	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ExistsByName_False тестирует проверку существования монитора (не найден).
func TestMonitorRepository_ExistsByName_False(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM monitors WHERE user_id = $1 AND name = $2)`)).
		WithArgs(userID, "Non-existent Monitor").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.ExistsByName(context.Background(), userID, "Non-existent Monitor")

	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ExistsByName_Error тестирует ошибку при проверке существования.
func TestMonitorRepository_ExistsByName_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM monitors WHERE user_id = $1 AND name = $2)`)).
		WithArgs(userID, "Some Monitor").
		WillReturnError(sql.ErrConnDone)

	exists, err := repo.ExistsByName(context.Background(), userID, "Some Monitor")

	assert.Error(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_GetByID_InvalidStatus тестирует ошибку при невалидном статусе.
func TestMonitorRepository_GetByID_InvalidStatus(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	monitorID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(monitorColumns()).
		AddRow(
			monitorID, userID, "Test Monitor", "https://example.com", "HTTP", 60, 30,
			"INVALID_STATUS", nil, nil, pq.StringArray{},
			nil, nil,
			nil, now, now,
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE id = $1
	`)).WithArgs(monitorID).WillReturnRows(rows)

	monitor, err := repo.GetByID(context.Background(), monitorID)

	assert.Error(t, err)
	assert.Nil(t, monitor)
	assert.Contains(t, err.Error(), "INVALID_STATUS")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_ListByUserIDAndStatus_Error тестирует ошибку при листинге по статусу.
func TestMonitorRepository_ListByUserIDAndStatus_Error(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	userID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, name, url, check_type, interval_seconds, timeout_seconds,
			status, working_hours_start, working_hours_end, working_days,
			degraded_response_time_threshold, degraded_failure_rate_threshold,
			last_check_at, created_at, updated_at
		FROM monitors
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`)).WithArgs(userID, domain.StatusUp).WillReturnError(sql.ErrConnDone)

	monitors, err := repo.ListByUserIDAndStatus(context.Background(), userID, domain.StatusUp)

	assert.Error(t, err)
	assert.Nil(t, monitors)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMonitorRepository_NewMonitorRepository тестирует создание репозитория.
func TestMonitorRepository_NewMonitorRepository(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewMonitorRepository(db)

	assert.NotNil(t, repo)
	assert.Implements(t, (*interfaces.MonitorRepository)(nil), repo)
}
