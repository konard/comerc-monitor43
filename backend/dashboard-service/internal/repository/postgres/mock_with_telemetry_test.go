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
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
	apptelemetry "github.com/raul/monitor/backend/dashboard-service/pkg/telemetry"
)

// newMockDBWithTelemetry создаёт DB с включёнными трейсером и метриками.
func newMockDBWithTelemetry(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := mockDB.Close(); err != nil {
			t.Logf("mock db close: %v", err)
		}
	})

	metrics, err := apptelemetry.NewMetrics("test")
	require.NoError(t, err)

	tracer := otel.GetTracerProvider().Tracer("test-tracer")

	db := &DB{
		DB:      sqlx.NewDb(mockDB, "sqlmock"),
		tracer:  tracer,
		metrics: metrics,
	}
	return db, mock
}

// ===================== MonitorStatusRepository с телеметрией =====================

func TestMonitorStatusRepository_Upsert_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectExec(`INSERT INTO monitor_statuses`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	view := &model.MonitorStatusView{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "test",
		URL:       "https://test.com",
		Status:    model.MonitorStatusUP,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Upsert(context.Background(), view)

	require.NoError(t, err)
}

func TestMonitorStatusRepository_Upsert_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestMonitorStatusRepository_Delete_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewMonitorStatusRepository(db)

	id := uuid.New().String()
	mock.ExpectExec(`DELETE FROM monitor_statuses`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), id)

	require.NoError(t, err)
}

func TestMonitorStatusRepository_Delete_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewMonitorStatusRepository(db)

	id := uuid.New().String()
	mock.ExpectExec(`DELETE FROM monitor_statuses`).
		WithArgs(id).
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(context.Background(), id)

	assert.Error(t, err)
}

func TestMonitorStatusRepository_GetByID_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestMonitorStatusRepository_GetByID_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT id, user_id`).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), uuid.New().String())

	assert.Error(t, err)
}

func TestMonitorStatusRepository_GetOverallUptime_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT COALESCE\(AVG`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(90.0))

	avg, err := repo.GetOverallUptime(context.Background(), uuid.New().String(), nil)

	require.NoError(t, err)
	assert.Equal(t, 90.0, avg)
}

func TestMonitorStatusRepository_GetOverallUptime_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewMonitorStatusRepository(db)

	mock.ExpectQuery(`SELECT COALESCE\(AVG`).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.GetOverallUptime(context.Background(), uuid.New().String(), nil)

	assert.Error(t, err)
}

// ===================== CheckHistoryRepository с телеметрией =====================

func TestCheckHistoryRepository_Create_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestCheckHistoryRepository_Create_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestCheckHistoryRepository_ListByMonitorID_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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

	_, _, err := repo.ListByMonitorID(context.Background(), monitorID, model.HistoryFilter{})

	require.NoError(t, err)
}

func TestCheckHistoryRepository_GetPeriodMetrics_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewCheckHistoryRepository(db)

	monitorID := uuid.New().String()
	now := time.Now()

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_checks", "success_count", "failed_count", "degraded_count",
		}).AddRow(10, 9, 1, 0))

	mock.ExpectQuery(`SELECT response_time_ms`).
		WillReturnRows(sqlmock.NewRows([]string{"response_time_ms"}).
			AddRow(10.0).AddRow(20.0))

	metrics, err := repo.GetPeriodMetrics(context.Background(), monitorID, now.Add(-time.Hour), now)

	require.NoError(t, err)
	assert.NotNil(t, metrics)
}

// ===================== IncidentRepository с телеметрией =====================

func TestIncidentRepository_Upsert_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestIncidentRepository_Upsert_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestIncidentRepository_ListByMonitorID_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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

	_, _, err := repo.ListByMonitorID(context.Background(), monitorID, model.IncidentFilter{})

	require.NoError(t, err)
}

func TestIncidentRepository_GetByID_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
}

func TestIncidentRepository_GetByID_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewIncidentRepository(db)

	mock.ExpectQuery(`SELECT id, monitor_id`).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), uuid.New().String())

	assert.Error(t, err)
}

func TestIncidentRepository_Update_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`UPDATE incidents`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	now := time.Now()
	duration := int64(3600)
	incident := &model.Incident{
		ID:              uuid.New(),
		EndedAt:         &now,
		DurationSeconds: &duration,
		Status:          model.IncidentStatusResolved,
		UpdatedAt:       now,
	}

	err := repo.Update(context.Background(), incident)

	require.NoError(t, err)
}

func TestIncidentRepository_Update_with_telemetry_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
	repo := NewIncidentRepository(db)

	mock.ExpectExec(`UPDATE incidents`).
		WillReturnError(sql.ErrConnDone)

	incident := &model.Incident{
		ID:        uuid.New(),
		Status:    model.IncidentStatusResolved,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), incident)

	assert.Error(t, err)
}

func TestMonitorStatusRepository_ListByUserID_with_telemetry(t *testing.T) {
	t.Parallel()

	db, mock := newMockDBWithTelemetry(t)
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
		}).AddRow(monitorID, ownerID, "test", "https://test.com", "UP", 99.5, nil, nil, "", now, now))

	_, _, err := repo.ListByUserID(context.Background(), userID, model.DashboardFilter{})

	require.NoError(t, err)
}
