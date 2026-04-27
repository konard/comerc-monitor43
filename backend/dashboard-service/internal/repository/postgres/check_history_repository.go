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

type CheckHistoryRepository struct {
	db *DB
}

func NewCheckHistoryRepository(db *DB) *CheckHistoryRepository {
	return &CheckHistoryRepository{db: db}
}

func (r *CheckHistoryRepository) Create(ctx context.Context, entry *model.CheckHistoryEntry) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "CheckHistoryRepository.Create")
		defer span.End()
		span.SetAttributes(
			attribute.String("check_id", entry.ID.String()),
			attribute.String("monitor_id", entry.MonitorID.String()),
			attribute.String("status", string(entry.Status)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO check_history (id, monitor_id, status, status_code, response_time_ms, error_message, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.MonitorID, entry.Status, entry.StatusCode,
		entry.ResponseTimeMs, entry.ErrorMessage, entry.CheckedAt,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return errors.Wrap(err, "failed to create check history entry")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "check_history", float64(duration))
	}

	return nil
}

func (r *CheckHistoryRepository) ListByMonitorID(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "CheckHistoryRepository.ListByMonitorID")
		defer span.End()
		span.SetAttributes(
			attribute.String("monitor_id", monitorID),
			attribute.Int("page", filter.Page),
			attribute.Int("page_size", filter.PageSize),
		)
	}

	startTime := time.Now()

	args := []any{monitorID}
	argNum := 2

	var whereParts []string

	if filter.Status != "" {
		whereParts = append(whereParts, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(filter.Status))
		argNum++
	}

	if filter.StartDate != nil {
		whereParts = append(whereParts, fmt.Sprintf("checked_at >= $%d", argNum))
		args = append(args, *filter.StartDate)
		argNum++
	}

	if filter.EndDate != nil {
		whereParts = append(whereParts, fmt.Sprintf("checked_at <= $%d", argNum))
		args = append(args, *filter.EndDate)
		argNum++
	}

	whereClause := "WHERE monitor_id = $1"
	if len(whereParts) > 0 {
		whereClause += " AND " + strings.Join(whereParts, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM check_history " + whereClause
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, errors.Wrap(err, "failed to count check history entries")
	}

	dataQuery := `
		SELECT id, monitor_id, status, status_code, response_time_ms, error_message, checked_at
		FROM check_history
	` + whereClause

	switch filter.SortBy {
	case "checked_at":
		if strings.EqualFold(filter.SortOrder, "asc") {
			dataQuery += " ORDER BY checked_at ASC"
		} else {
			dataQuery += " ORDER BY checked_at DESC"
		}
	case "response_time_ms":
		if strings.EqualFold(filter.SortOrder, "asc") {
			dataQuery += " ORDER BY response_time_ms ASC"
		} else {
			dataQuery += " ORDER BY response_time_ms DESC"
		}
	default:
		dataQuery += " ORDER BY checked_at DESC"
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

	var entries []*model.CheckHistoryEntry
	err = r.db.SelectContext(ctx, &entries, dataQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, errors.Wrap(err, "failed to list check history entries")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "list", "check_history", float64(duration))
	}

	return entries, total, nil
}

func (r *CheckHistoryRepository) GetPeriodMetrics(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "CheckHistoryRepository.GetPeriodMetrics")
		defer span.End()
		span.SetAttributes(
			attribute.String("monitor_id", monitorID),
		)
	}

	startTime := time.Now()

	query := `
		SELECT
			COUNT(*) as total_checks,
			COUNT(*) FILTER (WHERE status = 'UP') as success_count,
			COUNT(*) FILTER (WHERE status = 'DOWN') as failed_count,
			COUNT(*) FILTER (WHERE status = 'DEGRADED') as degraded_count
		FROM check_history
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
	`

	var totalChecks, successCount, failedCount, degradedCount int64

	err := r.db.QueryRowxContext(ctx, query, monitorID, start, end).Scan(
		&totalChecks, &successCount, &failedCount, &degradedCount,
	)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, errors.Wrap(err, "failed to get period metrics")
	}

	var uptimePercentage float64
	if totalChecks > 0 {
		uptimePercentage = float64(successCount) / float64(totalChecks) * 100
	}

	percentileQuery := `
		SELECT response_time_ms FROM check_history
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3 AND response_time_ms IS NOT NULL
		ORDER BY response_time_ms ASC
	`

	var responseTimes []float64
	err = r.db.SelectContext(ctx, &responseTimes, percentileQuery, monitorID, start, end)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, errors.Wrap(err, "failed to get response times for percentiles")
	}

	var p50, p95, p99 float64
	if len(responseTimes) > 0 {
		p50 = responseTimes[percentileIndex(len(responseTimes), 50)]
		p95 = responseTimes[percentileIndex(len(responseTimes), 95)]
		p99 = responseTimes[percentileIndex(len(responseTimes), 99)]
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "get_period_metrics", "check_history", float64(duration))
	}

	return &model.PeriodMetrics{
		TotalChecks:      totalChecks,
		SuccessCount:     successCount,
		FailedCount:      failedCount,
		DegradedCount:    degradedCount,
		UptimePercentage: uptimePercentage,
		P50ResponseMs:    p50,
		P95ResponseMs:    p95,
		P99ResponseMs:    p99,
	}, nil
}

func percentileIndex(count, percentile int) int {
	idx := count * percentile / 100
	if idx >= count {
		idx = count - 1
	}
	return idx
}
