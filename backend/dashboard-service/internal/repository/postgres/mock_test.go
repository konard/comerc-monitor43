package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
)

// newMockDB создаёт тестовую DB, обёрнутую в sqlx, используя sqlmock.
func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := mockDB.Close(); err != nil {
			t.Logf("mock db close: %v", err)
		}
	})

	db := &DB{
		DB: sqlx.NewDb(mockDB, "sqlmock"),
	}
	return db, mock
}

// ===================== MonitorStatusRepository =====================

func TestMonitorStatusRepository_Upsert_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectExec(`INSERT INTO monitor_statuses`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	view := &model.MonitorStatusView{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "test-monitor",
		URL:       "https://example.com",
		Status:    model.MonitorStatusUP,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Upsert(context.Background(), view)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitorStatusRepository_Upsert_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectExec(`INSERT INTO monitor_statuses`).
		WillReturnError(sql.ErrConnDone)

	view := &model.MonitorStatusView{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Upsert(context.Background(), view)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upsert monitor status")
}

func TestMonitorStatusRepository_Delete_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	id := uuid.New().String()
	mock.ExpectExec(`DELETE FROM monitor_statuses`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), id)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitorStatusRepository_Delete_not_found(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	id := uuid.New().String()
	mock.ExpectExec(`DELETE FROM monitor_statuses`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), id)

	assert.ErrorIs(t, err, model.ErrMonitorNotFound)
}

func TestMonitorStatusRepository_Delete_exec_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	id := uuid.New().String()
	mock.ExpectExec(`DELETE FROM monitor_statuses`).
		WithArgs(id).
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(context.Background(), id)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete monitor status")
}

func TestMonitorStatusRepository_ListByUserID_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	userID := uuid.New().String()
	monitorID := uuid.New()
	ownerID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM monitor_statuses`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "url", "status",
			"uptime_percentage", "last_checked_at", "last_response_time_ms",
			"tags", "created_at", "updated_at",
		}).AddRow(
			monitorID, ownerID, "test", "https://test.com", "UP",
			99.5, nil, nil, "", now, now,
		))

	statuses, total, err := repo.ListByUserID(context.Background(), userID, model.DashboardFilter{
		Page:     1,
		PageSize: 50,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, statuses, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitorStatusRepository_ListByUserID_with_filter(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	userID := uuid.New().String()
	now := time.Now()
	monitorID := uuid.New()
	ownerID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM monitor_statuses`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "url", "status",
			"uptime_percentage", "last_checked_at", "last_response_time_ms",
			"tags", "created_at", "updated_at",
		}).AddRow(
			monitorID, ownerID, "api", "https://api.com", "DOWN",
			50.0, nil, nil, "prod", now, now,
		))

	statuses, total, err := repo.ListByUserID(context.Background(), userID, model.DashboardFilter{
		Statuses:  []model.MonitorStatus{model.MonitorStatusDOWN},
		Search:    "api",
		Tags:      []string{"prod"},
		SortBy:    "name",
		SortOrder: "asc",
		Page:      1,
		PageSize:  10,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, statuses, 1)
}

func TestMonitorStatusRepository_ListByUserID_count_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM monitor_statuses`).
		WillReturnError(sql.ErrConnDone)

	_, _, err := repo.ListByUserID(context.Background(), uuid.New().String(), model.DashboardFilter{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count monitor statuses")
}

func TestMonitorStatusRepository_ListByUserID_data_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM monitor_statuses`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnError(sql.ErrConnDone)

	_, _, err := repo.ListByUserID(context.Background(), uuid.New().String(), model.DashboardFilter{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list monitor statuses")
}

func TestMonitorStatusRepository_GetByID_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT id, user_id`).
		WithArgs(id.String()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "url", "status",
			"uptime_percentage", "last_checked_at", "last_response_time_ms",
			"tags", "created_at", "updated_at",
		}).AddRow(id, userID, "test", "https://test.com", "UP", 99.5, nil, nil, "", now, now))

	result, err := repo.GetByID(context.Background(), id.String())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}

func TestMonitorStatusRepository_GetByID_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), uuid.New().String())

	assert.Error(t, err)
}

func TestMonitorStatusRepository_GetOverallUptime_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	userID := uuid.New().String()

	mock.ExpectQuery(`SELECT COALESCE\(AVG`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(97.5))

	avg, err := repo.GetOverallUptime(context.Background(), userID, nil)

	require.NoError(t, err)
	assert.Equal(t, 97.5, avg)
}

func TestMonitorStatusRepository_GetOverallUptime_with_statuses(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	userID := uuid.New().String()

	mock.ExpectQuery(`SELECT COALESCE\(AVG`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(85.0))

	avg, err := repo.GetOverallUptime(context.Background(), userID, []model.MonitorStatus{
		model.MonitorStatusUP, model.MonitorStatusDEGRADED,
	})

	require.NoError(t, err)
	assert.Equal(t, 85.0, avg)
}

func TestMonitorStatusRepository_GetOverallUptime_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT COALESCE\(AVG`).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.GetOverallUptime(context.Background(), uuid.New().String(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get overall uptime")
}

// ===================== CheckHistoryRepository =====================

func TestCheckHistoryRepository_Create_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	mock.ExpectExec(`INSERT INTO check_history`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	entry := &model.CheckHistoryEntry{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.CheckStatusUP,
		CheckedAt: time.Now(),
	}

	err := repo.Create(context.Background(), entry)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckHistoryRepository_Create_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	mock.ExpectExec(`INSERT INTO check_history`).
		WillReturnError(sql.ErrConnDone)

	entry := &model.CheckHistoryEntry{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.CheckStatusDOWN,
		CheckedAt: time.Now(),
	}

	err := repo.Create(context.Background(), entry)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create check history entry")
}

func TestCheckHistoryRepository_ListByMonitorID_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	entryID := uuid.New()
	entryMonitorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM check_history`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "monitor_id", "status", "status_code",
			"response_time_ms", "error_message", "checked_at",
		}).AddRow(entryID, entryMonitorID, "UP", nil, nil, nil, now))

	entries, total, err := repo.ListByMonitorID(context.Background(), monitorID, model.HistoryFilter{
		Page:     1,
		PageSize: 50,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, entries, 1)
}

func TestCheckHistoryRepository_ListByMonitorID_with_filters(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	start := now.Add(-time.Hour)
	end := now
	entryID := uuid.New()
	entryMonitorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM check_history`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "monitor_id", "status", "status_code",
			"response_time_ms", "error_message", "checked_at",
		}).AddRow(entryID, entryMonitorID, "DOWN", nil, nil, nil, now))

	entries, total, err := repo.ListByMonitorID(context.Background(), monitorID, model.HistoryFilter{
		Status:    model.CheckStatusDOWN,
		StartDate: &start,
		EndDate:   &end,
		SortBy:    "response_time_ms",
		SortOrder: "asc",
		Page:      1,
		PageSize:  10,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, entries, 1)
}

func TestCheckHistoryRepository_ListByMonitorID_default_sort(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	entryID := uuid.New()
	entryMonitorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM check_history`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "monitor_id", "status", "status_code",
			"response_time_ms", "error_message", "checked_at",
		}).AddRow(entryID, entryMonitorID, "UP", nil, nil, nil, now))

	entries, _, err := repo.ListByMonitorID(context.Background(), monitorID, model.HistoryFilter{
		SortBy:    "checked_at",
		SortOrder: "desc",
	})

	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestCheckHistoryRepository_ListByMonitorID_count_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM check_history`).
		WillReturnError(sql.ErrConnDone)

	_, _, err := repo.ListByMonitorID(context.Background(), uuid.New().String(), model.HistoryFilter{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count check history entries")
}

func TestCheckHistoryRepository_GetPeriodMetrics_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_checks", "success_count", "failed_count", "degraded_count",
		}).AddRow(100, 95, 3, 2))

	mock.ExpectQuery(`SELECT response_time_ms`).
		WillReturnRows(sqlmock.NewRows([]string{"response_time_ms"}).
			AddRow(10.0).AddRow(20.0).AddRow(30.0).AddRow(40.0).AddRow(50.0))

	metrics, err := repo.GetPeriodMetrics(context.Background(), monitorID, start, now)

	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, int64(100), metrics.TotalChecks)
	assert.Equal(t, int64(95), metrics.SuccessCount)
	assert.Equal(t, int64(3), metrics.FailedCount)
	assert.Equal(t, int64(2), metrics.DegradedCount)
	assert.Equal(t, 95.0, metrics.UptimePercentage)
	assert.Greater(t, metrics.P50ResponseMs, 0.0)
}

func TestCheckHistoryRepository_GetPeriodMetrics_no_response_times(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_checks", "success_count", "failed_count", "degraded_count",
		}).AddRow(0, 0, 0, 0))

	mock.ExpectQuery(`SELECT response_time_ms`).
		WillReturnRows(sqlmock.NewRows([]string{"response_time_ms"}))

	metrics, err := repo.GetPeriodMetrics(context.Background(), monitorID, start, now)

	require.NoError(t, err)
	assert.Equal(t, float64(0), metrics.P50ResponseMs)
	assert.Equal(t, float64(0), metrics.P95ResponseMs)
	assert.Equal(t, float64(0), metrics.P99ResponseMs)
}

func TestCheckHistoryRepository_GetPeriodMetrics_query_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	mock.ExpectQuery(`SELECT`).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.GetPeriodMetrics(context.Background(), uuid.New().String(), time.Now().Add(-time.Hour), time.Now())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get period metrics")
}

func TestCheckHistoryRepository_GetPeriodMetrics_percentile_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewCheckHistoryRepository(db)

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_checks", "success_count", "failed_count", "degraded_count",
		}).AddRow(10, 9, 1, 0))

	mock.ExpectQuery(`SELECT response_time_ms`).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.GetPeriodMetrics(context.Background(), uuid.New().String(), time.Now().Add(-time.Hour), time.Now())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get response times")
}

// ===================== IncidentRepository =====================

func TestIncidentRepository_Upsert_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`INSERT INTO incidents`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	incident := &model.Incident{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		StartedAt: time.Now(),
		Status:    model.IncidentStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Upsert(context.Background(), incident)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncidentRepository_Upsert_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`INSERT INTO incidents`).
		WillReturnError(sql.ErrConnDone)

	incident := &model.Incident{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		StartedAt: time.Now(),
		Status:    model.IncidentStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Upsert(context.Background(), incident)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upsert incident")
}

func TestIncidentRepository_ListByMonitorID_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	incID := uuid.New()
	incMonitorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM incidents`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "monitor_id", "started_at", "ended_at",
			"duration_seconds", "check_count", "status", "created_at", "updated_at",
		}).AddRow(incID, incMonitorID, now, nil, nil, 0, "ACTIVE", now, now))

	incidents, total, err := repo.ListByMonitorID(context.Background(), monitorID, model.IncidentFilter{
		Page:     1,
		PageSize: 50,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, incidents, 1)
}

func TestIncidentRepository_ListByMonitorID_with_dates(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()
	start := now.Add(-time.Hour)
	end := now
	incID := uuid.New()
	incMonitorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM incidents`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "monitor_id", "started_at", "ended_at",
			"duration_seconds", "check_count", "status", "created_at", "updated_at",
		}).AddRow(incID, incMonitorID, now, nil, nil, 5, "ACTIVE", now, now))

	_, _, err := repo.ListByMonitorID(context.Background(), monitorID, model.IncidentFilter{
		StartDate: &start,
		EndDate:   &end,
		Page:      1,
		PageSize:  10,
	})

	require.NoError(t, err)
}

func TestIncidentRepository_ListByMonitorID_count_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM incidents`).
		WillReturnError(sql.ErrConnDone)

	_, _, err := repo.ListByMonitorID(context.Background(), uuid.New().String(), model.IncidentFilter{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count incidents")
}

func TestIncidentRepository_GetByID_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	id := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WithArgs(id.String()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "monitor_id", "started_at", "ended_at",
			"duration_seconds", "check_count", "status", "created_at", "updated_at",
		}).AddRow(id, monitorID, now, nil, nil, 0, "ACTIVE", now, now))

	result, err := repo.GetByID(context.Background(), id.String())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}

func TestIncidentRepository_GetByID_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), uuid.New().String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get incident by id")
}

func TestIncidentRepository_Update_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`UPDATE incidents`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	now := time.Now()
	duration := int64(3600)
	incident := &model.Incident{
		ID:              uuid.New(),
		MonitorID:       uuid.New(),
		StartedAt:       now.Add(-time.Hour),
		EndedAt:         &now,
		DurationSeconds: &duration,
		Status:          model.IncidentStatusResolved,
		UpdatedAt:       now,
	}

	err := repo.Update(context.Background(), incident)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncidentRepository_Update_not_found(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`UPDATE incidents`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	incident := &model.Incident{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.IncidentStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), incident)

	assert.ErrorIs(t, err, model.ErrIncidentNotFound)
}

func TestIncidentRepository_Update_exec_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`UPDATE incidents`).
		WillReturnError(sql.ErrConnDone)

	incident := &model.Incident{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.IncidentStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), incident)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update incident")
}

// ===================== MonitorStatusRepository sortBy branches =====================

func TestMonitorStatusRepository_ListByUserID_sortby_branches(t *testing.T) {
	t.Parallel()

	for _, sortBy := range []string{"uptime_percentage", "last_checked_at", "status"} {
		t.Run(sortBy, func(t *testing.T) {
			t.Parallel()

			db, mock := newMockDB(t)
			repo := NewMonitorStatusRepository(db)

			userID := uuid.New().String()
			now := time.Now()
			monitorID := uuid.New()
			ownerID := uuid.New()

			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM monitor_statuses`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			mock.ExpectQuery(`SELECT id, user_id`).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "user_id", "name", "url", "status",
					"uptime_percentage", "last_checked_at", "last_response_time_ms",
					"tags", "created_at", "updated_at",
				}).AddRow(monitorID, ownerID, "test", "https://test.com", "UP", 99.0, nil, nil, "", now, now))

			_, _, err := repo.ListByUserID(context.Background(), userID, model.DashboardFilter{
				SortBy:    sortBy,
				SortOrder: "asc",
			})

			require.NoError(t, err)
		})
	}
}
