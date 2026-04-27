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

// TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_Success тестирует успешное получение результатов с пагинацией.
func TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	responseTime := 150
	statusCode := 200
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).
		AddRow(uuid.New(), monitorID, "UP", int64(responseTime), int64(statusCode), nil, nil, now, now).
		AddRow(uuid.New(), monitorID, "DOWN", nil, nil, "timeout", nil, now.Add(-time.Minute), now.Add(-time.Minute))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
		LIMIT $4 OFFSET $5
	`)).WithArgs(monitorID, from, to, 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriodPaginated(context.Background(), monitorID, from, to, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_Empty тестирует получение пустого списка.
func TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_Empty(t *testing.T) {
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
		LIMIT $4 OFFSET $5
	`)).WithArgs(monitorID, from, to, 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriodPaginated(context.Background(), monitorID, from, to, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_Error тестирует ошибку при запросе.
func TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_Error(t *testing.T) {
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
		LIMIT $4 OFFSET $5
	`)).WithArgs(monitorID, from, to, 10, 0).WillReturnError(sql.ErrConnDone)

	results, err := repo.GetByMonitorIDAndPeriodPaginated(context.Background(), monitorID, from, to, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_SecondPage тестирует вторую страницу.
func TestCheckResultRepository_GetByMonitorIDAndPeriodPaginated_SecondPage(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-48 * time.Hour)
	to := time.Now()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).
		AddRow(uuid.New(), monitorID, "UP", nil, nil, nil, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		ORDER BY checked_at DESC
		LIMIT $4 OFFSET $5
	`)).WithArgs(monitorID, from, to, 5, 5).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriodPaginated(context.Background(), monitorID, from, to, 5, 5)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_Success тестирует успешную фильтрацию по статусу.
func TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	now := time.Now()
	responseTime := 200
	statusCode := 200

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).
		AddRow(uuid.New(), monitorID, "UP", int64(responseTime), int64(statusCode), nil, nil, now, now).
		AddRow(uuid.New(), monitorID, "UP", int64(responseTime), int64(statusCode), nil, nil, now.Add(-time.Minute), now.Add(-time.Minute))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3 AND status = $4
		ORDER BY checked_at DESC
		LIMIT $5 OFFSET $6
	`)).WithArgs(monitorID, from, to, "UP", 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriodAndStatus(context.Background(), monitorID, from, to, domain.StatusUp, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, domain.StatusUp, r.Status)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_StatusDown тестирует фильтрацию по статусу DOWN.
func TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_StatusDown(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	closeTestDB(t, db)

	repo := NewCheckResultRepository(db)

	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	now := time.Now()
	errMsg := "connection timeout"

	rows := sqlmock.NewRows([]string{
		"id", "monitor_id", "status", "response_time_ms", "status_code",
		"error_message", "error_code", "checked_at", "created_at",
	}).
		AddRow(uuid.New(), monitorID, "DOWN", nil, nil, errMsg, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, monitor_id, status, response_time_ms, status_code,
			error_message, error_code, checked_at, created_at
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3 AND status = $4
		ORDER BY checked_at DESC
		LIMIT $5 OFFSET $6
	`)).WithArgs(monitorID, from, to, "DOWN", 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriodAndStatus(context.Background(), monitorID, from, to, domain.StatusDown, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, domain.StatusDown, results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_Empty тестирует пустой результат при фильтрации.
func TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_Empty(t *testing.T) {
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
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3 AND status = $4
		ORDER BY checked_at DESC
		LIMIT $5 OFFSET $6
	`)).WithArgs(monitorID, from, to, "DOWN", 10, 0).WillReturnRows(rows)

	results, err := repo.GetByMonitorIDAndPeriodAndStatus(context.Background(), monitorID, from, to, domain.StatusDown, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_Error тестирует ошибку при запросе.
func TestCheckResultRepository_GetByMonitorIDAndPeriodAndStatus_Error(t *testing.T) {
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
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3 AND status = $4
		ORDER BY checked_at DESC
		LIMIT $5 OFFSET $6
	`)).WithArgs(monitorID, from, to, "UP", 10, 0).WillReturnError(sql.ErrConnDone)

	results, err := repo.GetByMonitorIDAndPeriodAndStatus(context.Background(), monitorID, from, to, domain.StatusUp, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}
