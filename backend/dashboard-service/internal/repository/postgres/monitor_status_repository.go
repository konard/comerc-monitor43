package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
)

type MonitorStatusRepository struct {
	db *DB
}

func NewMonitorStatusRepository(db *DB) *MonitorStatusRepository {
	return &MonitorStatusRepository{db: db}
}

func (r *MonitorStatusRepository) Upsert(ctx context.Context, status *model.MonitorStatusView) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusRepository.Upsert")
		defer span.End()
		span.SetAttributes(
			attribute.String("monitor_id", status.ID.String()),
			attribute.String("user_id", status.UserID.String()),
			attribute.String("status", string(status.Status)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO monitor_statuses (id, user_id, name, url, status, uptime_percentage, last_checked_at, last_response_time_ms, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, url = EXCLUDED.url, status = EXCLUDED.status,
			uptime_percentage = EXCLUDED.uptime_percentage, last_checked_at = EXCLUDED.last_checked_at,
			last_response_time_ms = EXCLUDED.last_response_time_ms, tags = EXCLUDED.tags, updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		status.ID, status.UserID, status.Name, status.URL, status.Status,
		status.UptimePercentage, status.LastCheckedAt, status.LastResponseTimeMs,
		status.Tags, status.CreatedAt, status.UpdatedAt,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to upsert monitor status")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "upsert", "monitor_statuses", float64(duration))
	}

	return nil
}

func (r *MonitorStatusRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusRepository.Delete")
		defer span.End()
		span.SetAttributes(attribute.String("monitor_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM monitor_statuses WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to delete monitor status")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return model.ErrMonitorNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "delete", "monitor_statuses", float64(duration))
	}

	return nil
}

func (r *MonitorStatusRepository) ListByUserID(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusRepository.ListByUserID")
		defer span.End()
		span.SetAttributes(
			attribute.String("user_id", userID),
			attribute.String("sort_by", filter.SortBy),
			attribute.Int("page", filter.Page),
			attribute.Int("page_size", filter.PageSize),
		)
	}

	startTime := time.Now()

	args := []any{userID}
	argNum := 2

	var whereParts []string

	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, string(s))
			argNum++
		}
		whereParts = append(whereParts, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	if filter.Search != "" {
		whereParts = append(whereParts, fmt.Sprintf("name ILIKE $%d", argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	for _, tag := range filter.Tags {
		whereParts = append(whereParts, fmt.Sprintf("tags LIKE $%d", argNum))
		args = append(args, "%"+tag+"%")
		argNum++
	}

	whereClause := "WHERE user_id = $1"
	if len(whereParts) > 0 {
		whereClause += " AND " + strings.Join(whereParts, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM monitor_statuses " + whereClause
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, errors.Wrap(err, "failed to count monitor statuses")
	}

	dataQuery := `
		SELECT id, user_id, name, url, status, uptime_percentage, last_checked_at,
			last_response_time_ms, tags, created_at, updated_at
		FROM monitor_statuses
	` + whereClause

	switch filter.SortBy {
	case "name":
		dataQuery += " ORDER BY name"
	case "uptime_percentage":
		dataQuery += " ORDER BY uptime_percentage"
	case "last_checked_at":
		dataQuery += " ORDER BY last_checked_at"
	default:
		dataQuery += " ORDER BY (CASE status WHEN 'DOWN' THEN 1 WHEN 'DEGRADED' THEN 2 WHEN 'UP' THEN 3 WHEN 'PAUSED' THEN 4 ELSE 5 END)"
	}

	if strings.EqualFold(filter.SortOrder, "asc") || (filter.SortBy == "" && filter.SortOrder == "") {
		dataQuery += " ASC"
	} else {
		dataQuery += " DESC"
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	page := filter.Page
	if page <= 0 {
		page = model.DefaultPage
	}

	dataQuery += fmt.Sprintf(" LIMIT $%d", argNum)
	args = append(args, pageSize)
	argNum++

	offset := (page - 1) * pageSize
	dataQuery += fmt.Sprintf(" OFFSET $%d", argNum)
	args = append(args, offset)

	var statuses []*model.MonitorStatusView
	err = r.db.SelectContext(ctx, &statuses, dataQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, errors.Wrap(err, "failed to list monitor statuses")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "list", "monitor_statuses", float64(duration))
	}

	return statuses, total, nil
}

func (r *MonitorStatusRepository) GetByID(ctx context.Context, id string) (*model.MonitorStatusView, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusRepository.GetByID")
		defer span.End()
		span.SetAttributes(attribute.String("monitor_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, name, url, status, uptime_percentage, last_checked_at,
			last_response_time_ms, tags, created_at, updated_at
		FROM monitor_statuses
		WHERE id = $1
	`

	var status model.MonitorStatusView
	err := r.db.GetContext(ctx, &status, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, errors.Wrap(err, "failed to get monitor status by id")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyid", "monitor_statuses", float64(duration))
	}

	return &status, nil
}

func (r *MonitorStatusRepository) GetOverallUptime(ctx context.Context, userID string, statuses []model.MonitorStatus) (float64, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusRepository.GetOverallUptime")
		defer span.End()
		span.SetAttributes(attribute.String("user_id", userID))
	}

	startTime := time.Now()

	args := []any{userID}
	argNum := 2

	whereClause := "WHERE user_id = $1"

	if len(statuses) > 0 {
		placeholders := make([]string, len(statuses))
		for i, s := range statuses {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, string(s))
			argNum++
		}
		whereClause += fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ", "))
	}

	query := "SELECT COALESCE(AVG(uptime_percentage), 0) FROM monitor_statuses " + whereClause

	var avg float64
	err := r.db.GetContext(ctx, &avg, query, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return 0, errors.Wrap(err, "failed to get overall uptime")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getoveralluptime", "monitor_statuses", float64(duration))
	}

	return avg, nil
}
