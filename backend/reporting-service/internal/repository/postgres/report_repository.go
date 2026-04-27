package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/repository/interfaces"
)

type reportRepository struct {
	db *DB
}

func NewReportRepository(db *DB) interfaces.ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) Create(ctx context.Context, report *model.SLAReport) error {
	query := `
		INSERT INTO sla_reports (
			id, monitor_id, user_id, monitor_name, period_start, period_end,
			availability, total_checks, up_checks, down_checks, degraded_checks,
			paused_checks, total_downtime_seconds, incidents_count, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		report.ID, report.MonitorID, report.UserID, report.MonitorName,
		report.PeriodStart, report.PeriodEnd,
		report.Availability, report.TotalChecks, report.UpChecks,
		report.DownChecks, report.DegradedChecks, report.PausedChecks,
		report.TotalDowntimeSeconds, report.IncidentsCount, report.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create sla report")
	}

	return nil
}

func (r *reportRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SLAReport, error) {
	query := `
		SELECT id, monitor_id, user_id, monitor_name, period_start, period_end,
			availability, total_checks, up_checks, down_checks, degraded_checks,
			paused_checks, total_downtime_seconds, incidents_count, created_at
		FROM sla_reports
		WHERE id = $1
	`

	var report model.SLAReport
	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(model.ErrReportNotFound, "report not found")
		}
		return nil, errors.Wrap(err, "failed to get sla report by id")
	}

	return &report, nil
}

func (r *reportRepository) ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*model.SLAReport, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM sla_reports WHERE monitor_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, monitorID); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count sla reports")
	}

	query := `
		SELECT id, monitor_id, user_id, monitor_name, period_start, period_end,
			availability, total_checks, up_checks, down_checks, degraded_checks,
			paused_checks, total_downtime_seconds, incidents_count, created_at
		FROM sla_reports
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var reports []model.SLAReport
	if err := r.db.SelectContext(ctx, &reports, query, monitorID, limit, offset); err != nil {
		return nil, 0, errors.Wrap(err, "failed to list sla reports")
	}

	result := make([]*model.SLAReport, len(reports))
	for i := range reports {
		result[i] = &reports[i]
	}

	return result, total, nil
}

// ListByUserID возвращает SLA-отчёты пользователя без фильтра по monitor_id.
func (r *reportRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.SLAReport, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM sla_reports WHERE user_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count sla reports by user")
	}

	query := `
		SELECT id, monitor_id, user_id, monitor_name, period_start, period_end,
			availability, total_checks, up_checks, down_checks, degraded_checks,
			paused_checks, total_downtime_seconds, incidents_count, created_at
		FROM sla_reports
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var reports []model.SLAReport
	if err := r.db.SelectContext(ctx, &reports, query, userID, limit, offset); err != nil {
		return nil, 0, errors.Wrap(err, "failed to list sla reports by user")
	}

	result := make([]*model.SLAReport, len(reports))
	for i := range reports {
		result[i] = &reports[i]
	}

	return result, total, nil
}
