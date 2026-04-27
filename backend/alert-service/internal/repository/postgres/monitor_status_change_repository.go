package postgres

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MonitorStatusChangeRepository реализует интерфейс MonitorStatusChangeRepository для PostgreSQL
type MonitorStatusChangeRepository struct {
	db *DB
}

// NewMonitorStatusChangeRepository создаёт новый экземпляр MonitorStatusChangeRepository
func NewMonitorStatusChangeRepository(db *DB) *MonitorStatusChangeRepository {
	return &MonitorStatusChangeRepository{db: db}
}

// Create создаёт запись об изменении статуса монитора
func (r *MonitorStatusChangeRepository) Create(ctx context.Context, change *model.MonitorStatusChange) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusChangeRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("monitor_id", change.MonitorID.String()),
			attribute.String("new_status", change.NewStatus),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO monitor_status_changes (
			id, monitor_id, user_id, old_status, new_status, created_at
		) VALUES (
			:id, :monitor_id, :user_id, :old_status, :new_status, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, change)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create monitor status change: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "monitor_status_changes", float64(duration))
	}

	return nil
}

// CountInWindow возвращает количество изменений статуса за указанное время
func (r *MonitorStatusChangeRepository) CountInWindow(ctx context.Context, monitorID string, window time.Duration) (int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusChangeRepository.CountInWindow")
		defer span.End()

		span.SetAttributes(
			attribute.String("monitor_id", monitorID),
			attribute.String("window", window.String()),
		)
	}

	startTime := time.Now()

	cutoff := time.Now().Add(-window)
	query := `
		SELECT COUNT(*) FROM monitor_status_changes
		WHERE monitor_id = $1 AND created_at > $2
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, monitorID, cutoff)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return 0, fmt.Errorf("failed to count monitor status changes: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "countinwindow", "monitor_status_changes", float64(duration))
	}

	return count, nil
}

// GetLatestByMonitorID возвращает последнее изменение статуса монитора
func (r *MonitorStatusChangeRepository) GetLatestByMonitorID(ctx context.Context, monitorID string) (*model.MonitorStatusChange, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "MonitorStatusChangeRepository.GetLatestByMonitorID")
		defer span.End()

		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT id, monitor_id, user_id, old_status, new_status, created_at
		FROM monitor_status_changes
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var change model.MonitorStatusChange
	err := r.db.GetContext(ctx, &change, query, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get latest monitor status change: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getlatest", "monitor_status_changes", float64(duration))
	}

	return &change, nil
}
