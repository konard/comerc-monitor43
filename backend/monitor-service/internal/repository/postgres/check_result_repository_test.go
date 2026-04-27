// Package postgres предоставляет реализацию репозиториев для работы с PostgreSQL.
// Unit тесты используют sqlmock для мокания SQL-запросов.
package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

// TestCheckResultRepository_Create_Success тестирует успешное создание результата проверки.
func TestCheckResultRepository_Create_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	responseTime := 150
	statusCode := 200

	result := &domain.CheckResult{
		ID:             uuid.New(),
		MonitorID:      monitorID,
		Status:         domain.StatusUp,
		ResponseTimeMs: &responseTime,
		StatusCode:     &statusCode,
		CheckedAt:      time.Now(),
		CreatedAt:      time.Now(),
	}

	// sqlx.NamedExec конвертирует :param в $1, $2, etc., поэтому ожидаем простой INSERT
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO check_results`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), result)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_Create_Error тестирует ошибку при создании результата проверки.
func TestCheckResultRepository_Create_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	result := &domain.CheckResult{
		ID:        uuid.New(),
		MonitorID: monitorID,
		Status:    domain.StatusUp,
		CheckedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO check_results`)).
		WillReturnError(sql.ErrConnDone)

	err = repo.Create(context.Background(), result)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByID_Success тестирует успешное получение результата по ID.
func TestCheckResultRepository_GetByID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	resultID := uuid.New()
	monitorID := uuid.New()
	responseTime := 150
	statusCode := 200
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).AddRow(
		resultID, monitorID, "UP", int64(responseTime), int64(statusCode),
		nil, nil, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE id = $1
	`)).WithArgs(resultID).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), resultID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, resultID, result.ID)
	assert.Equal(t, domain.StatusUp, result.Status)
	assert.Equal(t, responseTime, *result.ResponseTimeMs)
	assert.Equal(t, statusCode, *result.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByID_NotFound тестирует случай, когда результат не найден.
func TestCheckResultRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	resultID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE id = $1
	`)).WithArgs(resultID).WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(context.Background(), resultID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorID_Success тестирует получение результатов по monitor_id.
func TestCheckResultRepository_GetByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	now := time.Now()
	responseTime := 150
	statusCode := 200

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).
		AddRow(
			uuid.New(), monitorID, "UP", int64(responseTime), int64(statusCode),
			nil, nil, now, now,
		).
		AddRow(
			uuid.New(), monitorID, "DOWN", nil, nil,
			"connection timeout", nil, now.Add(-time.Minute), now.Add(-time.Minute),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorID_Empty тестирует получение пустого списка результатов.
func TestCheckResultRepository_GetByMonitorID_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriod_Success тестирует получение результатов за период.
func TestCheckResultRepository_GetByMonitorIDAndPeriod_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	responseTime := 150

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).AddRow(
		uuid.New(), monitorID, "UP", int64(responseTime), nil,
		nil, nil, time.Now(), time.Now(),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
	`)).WithArgs(monitorID, from, to).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriod(context.Background(), monitorID, from, to)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetLatestByMonitorID_Success тестирует получение последних результатов.
func TestCheckResultRepository_GetLatestByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	now := time.Now()
	responseTime := 150

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).
		AddRow(
			uuid.New(), monitorID, "UP", int64(responseTime), nil,
			nil, nil, now, now,
		).
		AddRow(
			uuid.New(), monitorID, "UP", int64(responseTime), nil,
			nil, nil, now.Add(-time.Minute), now.Add(-time.Minute),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`)).WithArgs(monitorID, 5).WillReturnRows(rows)

	results, err := repo.GetLatestByMonitorID(context.Background(), monitorID, 5)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_DeleteOld_Success тестирует удаление старых результатов.
func TestCheckResultRepository_DeleteOld_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM check_results WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(sqlmock.NewResult(0, 150))

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(150), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_DeleteOld_ZeroRows тестирует удаление, когда нет старых записей.
func TestCheckResultRepository_DeleteOld_ZeroRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM check_results WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_CountByMonitorID_Success тестирует подсчёт результатов монитора.
func TestCheckResultRepository_CountByMonitorID_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM check_results WHERE monitor_id = $1`)).
		WithArgs(monitorID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1000))

	count, err := repo.CountByMonitorID(context.Background(), monitorID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_Create_WithErrorMessage тестирует создание результата с ошибкой.
func TestCheckResultRepository_Create_WithErrorMessage(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	errorMsg := "connection timeout"

	result := &domain.CheckResult{
		ID:           uuid.New(),
		MonitorID:    monitorID,
		Status:       domain.StatusDown,
		ErrorMessage: &errorMsg,
		CheckedAt:    time.Now(),
		CreatedAt:    time.Now(),
	}

	// sqlx.NamedExec конвертирует :param в $1, $2, etc., поэтому ожидаем простой INSERT
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO check_results`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), result)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorID_Pagination тестирует пагинацию.
func TestCheckResultRepository_GetByMonitorID_Pagination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
	}{
		{
			name:          "first page",
			limit:         10,
			offset:        0,
			expectedCount: 10,
		},
		{
			name:          "second page",
			limit:         10,
			offset:        10,
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			closeTestDB(t, db)

			repo := NewCheckResultRepository(db)

			monitorID := uuid.New()
			now := time.Now()

			rows := sqlmock.NewRows([]string{
				"id", "monitor_id", "status", "response_time_ms", "status_code",
				"error_message", "error_code", "checked_at", "created_at",
			})

			for i := 0; i < tt.expectedCount; i++ {
				rows.AddRow(
					uuid.New(), monitorID, "UP", nil, nil,
					nil, nil, now, now,
				)
			}

			mock.ExpectQuery(regexp.QuoteMeta(`
				SELECT id, monitor_id, status, response_time_ms, status_code,
					error_message, error_code, checked_at, created_at
				FROM check_results
				WHERE monitor_id = $1
				ORDER BY checked_at DESC
				LIMIT $2 OFFSET $3
			`)).WithArgs(monitorID, tt.limit, tt.offset).WillReturnRows(rows)

			results, err := repo.GetByMonitorID(context.Background(), monitorID, tt.limit, tt.offset)
			assert.NoError(t, err)
			assert.Len(t, results, tt.expectedCount)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestCheckResultRepository_GetByMonitorIDAndPeriod_NoResults тестирует получение результатов за период без данных.
func TestCheckResultRepository_GetByMonitorIDAndPeriod_NoResults(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
	`)).WithArgs(monitorID, from, to).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriod(context.Background(), monitorID, from, to)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByID_InvalidStatus тестирует получение результата с невалидным статусом.
func TestCheckResultRepository_GetByID_InvalidStatus(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	resultID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).AddRow(
		resultID, monitorID, "INVALID_STATUS", nil, nil,
		nil, nil, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE id = $1
	`)).WithArgs(resultID).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), resultID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetLatestByMonitorID_Empty тестирует получение последних результатов для монитора без данных.
func TestCheckResultRepository_GetLatestByMonitorID_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`)).WithArgs(monitorID, 5).WillReturnRows(rows)

	results, err := repo.GetLatestByMonitorID(context.Background(), monitorID, 5)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorID_Error тестирует ошибку при получении результатов.
func TestCheckResultRepository_GetByMonitorID_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2 OFFSET $3
	`)).WithArgs(monitorID, 10, 0).WillReturnError(sql.ErrConnDone)

	results, err := repo.GetByMonitorID(context.Background(), monitorID, 10, 0)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_DeleteOld_Error тестирует ошибку при удалении старых результатов.
func TestCheckResultRepository_DeleteOld_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM check_results WHERE created_at < $1`)).
		WithArgs(olderThan).
		WillReturnError(sql.ErrConnDone)

	deleted, err := repo.DeleteOld(context.Background(), olderThan)
	assert.Error(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_CountByMonitorID_Error тестирует ошибку при подсчёте результатов.
func TestCheckResultRepository_CountByMonitorID_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM check_results WHERE monitor_id = $1`)).
		WithArgs(monitorID).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.CountByMonitorID(context.Background(), monitorID)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriod_Error тестирует ошибку при получении результатов за период.
func TestCheckResultRepository_GetByMonitorIDAndPeriod_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
	`)).WithArgs(monitorID, from, to).WillReturnError(sql.ErrConnDone)

	results, err := repo.GetByMonitorIDAndPeriod(context.Background(), monitorID, from, to)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetLatestByMonitorID_Error тестирует ошибку при получении последних результатов.
func TestCheckResultRepository_GetLatestByMonitorID_Error(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`)).WithArgs(monitorID, 5).WillReturnError(sql.ErrConnDone)

	results, err := repo.GetLatestByMonitorID(context.Background(), monitorID, 5)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_dbListToCheckResults_InvalidStatus тестирует конвертацию списка с невалидным статусом.
func TestCheckResultRepository_dbListToCheckResults_InvalidStatus(t *testing.T) {
	t.Parallel()
	repo := &checkResultRepository{}

	now := time.Now()
	dbResults := []dbCheckResult{
		{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			Status:    string(domain.StatusUp),
			CheckedAt: now,
			CreatedAt: now,
		},
		{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			Status:    "INVALID_STATUS",
			CheckedAt: now,
			CreatedAt: now,
		},
	}

	_, err := repo.dbListToCheckResults(dbResults)
	assert.Error(t, err)
}

// TestCheckResultRepository_GetByID_ScanError тестирует ошибку при сканировании.
func TestCheckResultRepository_GetByID_ScanError(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	resultID := uuid.New()

	// Возвращаем неправильный тип данных для response_time_ms
	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).AddRow(
		resultID, uuid.New(), "UP", "not_a_number", nil,
		nil, nil, time.Now(), time.Now(),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE id = $1
	`)).WithArgs(resultID).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), resultID)
	assert.Error(t, err)
	assert.Nil(t, result)
}
