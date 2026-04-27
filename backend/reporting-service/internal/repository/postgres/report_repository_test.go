package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/reporting-service/internal/model"
)

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	stdDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	return &DB{DB: sqlx.NewDb(stdDB, "postgres")}, mock
}

func TestReportRepository_GetByID(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		id := uuid.New()
		monitorID := uuid.New()
		userID := uuid.New()
		now := time.Now().Truncate(time.Microsecond)

		rows := sqlmock.NewRows([]string{
			"id", "monitor_id", "user_id", "monitor_name", "period_start", "period_end",
			"availability", "total_checks", "up_checks", "down_checks", "degraded_checks",
			"paused_checks", "total_downtime_seconds", "incidents_count", "created_at",
		}).AddRow(
			id, monitorID, userID, "test-monitor",
			now.AddDate(0, 0, -7), now,
			99.95, 10000, 9980, 15, 5, 0, 3600, 3, now,
		)

		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		report, err := repo.GetByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, id, report.ID)
		assert.Equal(t, monitorID, report.MonitorID)
		assert.Equal(t, userID, report.UserID)
		assert.Equal(t, "test-monitor", report.MonitorName)
		assert.Equal(t, 99.95, report.Availability)
		assert.Equal(t, 10000, report.TotalChecks)
		assert.Equal(t, 9980, report.UpChecks)
		assert.Equal(t, 15, report.DownChecks)
		assert.Equal(t, int64(3600), report.TotalDowntimeSeconds)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		id := uuid.New()

		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		report, err := repo.GetByID(context.Background(), id)
		require.Error(t, err)
		assert.Nil(t, report)
		assert.ErrorIs(t, err, model.ErrReportNotFound)
		assert.Contains(t, err.Error(), "report not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		id := uuid.New()

		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(fmt.Errorf("connection refused"))

		report, err := repo.GetByID(context.Background(), id)
		require.Error(t, err)
		assert.Nil(t, report)
		assert.Contains(t, err.Error(), "failed to get sla report by id")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReportRepository_ListByMonitorID(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		monitorID := uuid.New()
		now := time.Now().Truncate(time.Microsecond)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM sla_reports WHERE monitor_id = \\$1").
			WithArgs(monitorID).
			WillReturnRows(countRows)

		id1 := uuid.New()
		id2 := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "monitor_id", "user_id", "monitor_name", "period_start", "period_end",
			"availability", "total_checks", "up_checks", "down_checks", "degraded_checks",
			"paused_checks", "total_downtime_seconds", "incidents_count", "created_at",
		}).AddRow(
			id1, monitorID, uuid.New(), "test-monitor",
			now.AddDate(0, -1, 0), now.AddDate(0, 0, -30),
			99.90, 5000, 4990, 8, 2, 0, 1800, 2, now.AddDate(0, -1, 0),
		).AddRow(
			id2, monitorID, uuid.New(), "test-monitor",
			now.AddDate(0, 0, -30), now,
			99.95, 10000, 9980, 3, 1, 0, 900, 1, now,
		)

		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE monitor_id = \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").
			WithArgs(monitorID, 50, 0).
			WillReturnRows(rows)

		reports, total, err := repo.ListByMonitorID(context.Background(), monitorID, 0, 0)
		require.NoError(t, err)
		assert.Len(t, reports, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, id1, reports[0].ID)
		assert.Equal(t, id2, reports[1].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty_result", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		monitorID := uuid.New()

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM sla_reports WHERE monitor_id = \\$1").
			WithArgs(monitorID).
			WillReturnRows(countRows)

		columns := []string{
			"id", "monitor_id", "user_id", "monitor_name", "period_start", "period_end",
			"availability", "total_checks", "up_checks", "down_checks", "degraded_checks",
			"paused_checks", "total_downtime_seconds", "incidents_count", "created_at",
		}
		rows := sqlmock.NewRows(columns)
		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE monitor_id = \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").
			WithArgs(monitorID, 50, 0).
			WillReturnRows(rows)

		reports, total, err := repo.ListByMonitorID(context.Background(), monitorID, 0, 0)
		require.NoError(t, err)
		assert.Empty(t, reports)
		assert.Equal(t, 0, total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count_error", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		monitorID := uuid.New()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM sla_reports WHERE monitor_id = \\$1").
			WithArgs(monitorID).
			WillReturnError(fmt.Errorf("connection refused"))

		reports, total, err := repo.ListByMonitorID(context.Background(), monitorID, 0, 0)
		require.Error(t, err)
		assert.Nil(t, reports)
		assert.Equal(t, 0, total)
		assert.Contains(t, err.Error(), "failed to count sla reports")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query_error", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		monitorID := uuid.New()

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM sla_reports WHERE monitor_id = \\$1").
			WithArgs(monitorID).
			WillReturnRows(countRows)

		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE monitor_id = \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").
			WithArgs(monitorID, 50, 0).
			WillReturnError(fmt.Errorf("connection refused"))

		reports, total, err := repo.ListByMonitorID(context.Background(), monitorID, 0, 0)
		require.Error(t, err)
		assert.Nil(t, reports)
		assert.Equal(t, 0, total)
		assert.Contains(t, err.Error(), "failed to list sla reports")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("custom_limit_and_offset", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		monitorID := uuid.New()

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM sla_reports WHERE monitor_id = \\$1").
			WithArgs(monitorID).
			WillReturnRows(countRows)

		columns := []string{
			"id", "monitor_id", "user_id", "monitor_name", "period_start", "period_end",
			"availability", "total_checks", "up_checks", "down_checks", "degraded_checks",
			"paused_checks", "total_downtime_seconds", "incidents_count", "created_at",
		}
		rows := sqlmock.NewRows(columns)
		mock.ExpectQuery("SELECT .+ FROM sla_reports WHERE monitor_id = \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").
			WithArgs(monitorID, 10, 20).
			WillReturnRows(rows)

		reports, total, err := repo.ListByMonitorID(context.Background(), monitorID, 10, 20)
		require.NoError(t, err)
		assert.Empty(t, reports)
		assert.Equal(t, 100, total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReportRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		report := &model.SLAReport{
			ID:          uuid.New(),
			MonitorID:   uuid.New(),
			UserID:      uuid.New(),
			MonitorName: "test-monitor",
			PeriodStart: time.Now().AddDate(0, 0, -7),
			PeriodEnd:   time.Now(),
		}

		mock.ExpectExec("INSERT INTO sla_reports").
			WithArgs(
				report.ID, report.MonitorID, report.UserID, report.MonitorName,
				report.PeriodStart, report.PeriodEnd,
				report.Availability, report.TotalChecks, report.UpChecks,
				report.DownChecks, report.DegradedChecks, report.PausedChecks,
				report.TotalDowntimeSeconds, report.IncidentsCount, report.CreatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), report)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		pgDB, mock := newMockDB(t)
		repo := NewReportRepository(pgDB)

		report := &model.SLAReport{
			ID:          uuid.New(),
			MonitorID:   uuid.New(),
			UserID:      uuid.New(),
			MonitorName: "test",
		}

		mock.ExpectExec("INSERT INTO sla_reports").
			WithArgs(
				report.ID, report.MonitorID, report.UserID, report.MonitorName,
				report.PeriodStart, report.PeriodEnd,
				report.Availability, report.TotalChecks, report.UpChecks,
				report.DownChecks, report.DegradedChecks, report.PausedChecks,
				report.TotalDowntimeSeconds, report.IncidentsCount, report.CreatedAt,
			).
			WillReturnError(fmt.Errorf("connection refused"))

		err := repo.Create(context.Background(), report)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create sla report")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
