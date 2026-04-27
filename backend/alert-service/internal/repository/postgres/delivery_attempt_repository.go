package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// DeliveryAttemptRepository реализует интерфейс DeliveryAttemptRepository для PostgreSQL
type DeliveryAttemptRepository struct {
	db *DB
}

// NewDeliveryAttemptRepository создаёт новый экземпляр DeliveryAttemptRepository
func NewDeliveryAttemptRepository(db *DB) *DeliveryAttemptRepository {
	return &DeliveryAttemptRepository{db: db}
}

// Create создаёт новую попытку доставки
func (r *DeliveryAttemptRepository) Create(ctx context.Context, attempt *model.DeliveryAttempt) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("alert_id", attempt.AlertID.String()),
			attribute.String("channel_id", attempt.AlertChannelID.String()),
			attribute.String("status", string(attempt.Status)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO delivery_attempts (
			id, alert_id, alert_channel_id, status, error_message,
			retry_count, next_retry_at, created_at, updated_at
		) VALUES (
			:id, :alert_id, :alert_channel_id, :status, :error_message,
			:retry_count, :next_retry_at, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, attempt)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create delivery attempt: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "delivery_attempts", float64(duration))
	}

	return nil
}

// GetByID возвращает попытку доставки по ID
func (r *DeliveryAttemptRepository) GetByID(ctx context.Context, id string) (*model.DeliveryAttempt, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.GetByID")
		defer span.End()

		span.SetAttributes(attribute.String("attempt_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, alert_id, alert_channel_id, status, error_message,
			retry_count, next_retry_at, created_at, updated_at
		FROM delivery_attempts
		WHERE id = $1
	`

	var attempt model.DeliveryAttempt
	err := r.db.GetContext(ctx, &attempt, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get delivery attempt by id: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyid", "delivery_attempts", float64(duration))
	}

	return &attempt, nil
}

// List возвращает все попытки доставки для алерта
func (r *DeliveryAttemptRepository) List(ctx context.Context, alertID string) ([]*model.DeliveryAttempt, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.List")
		defer span.End()

		span.SetAttributes(attribute.String("alert_id", alertID))
	}

	startTime := time.Now()

	query := `
		SELECT id, alert_id, alert_channel_id, status, error_message,
			retry_count, next_retry_at, created_at, updated_at
		FROM delivery_attempts
		WHERE alert_id = $1
		ORDER BY created_at DESC
	`

	var attempts []*model.DeliveryAttempt
	err := r.db.SelectContext(ctx, &attempts, query, alertID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to list delivery attempts: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "list", "delivery_attempts", float64(duration))
	}

	return attempts, nil
}

// ListPending возвращает pending попытки доставки для retry
func (r *DeliveryAttemptRepository) ListPending(ctx context.Context, limit int) ([]*model.DeliveryAttempt, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.ListPending")
		defer span.End()

		span.SetAttributes(attribute.Int("limit", limit))
	}

	startTime := time.Now()

	query := `
		SELECT id, alert_id, alert_channel_id, status, error_message,
			retry_count, next_retry_at, created_at, updated_at
		FROM delivery_attempts
		WHERE status = 'pending'
			AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1
	`

	var attempts []*model.DeliveryAttempt
	err := r.db.SelectContext(ctx, &attempts, query, limit)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to list pending delivery attempts: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "listpending", "delivery_attempts", float64(duration))
	}

	return attempts, nil
}

// Update обновляет попытку доставки
func (r *DeliveryAttemptRepository) Update(ctx context.Context, attempt *model.DeliveryAttempt) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.Update")
		defer span.End()

		span.SetAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("status", string(attempt.Status)),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE delivery_attempts SET
			status = :status,
			error_message = :error_message,
			retry_count = :retry_count,
			next_retry_at = :next_retry_at,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, attempt)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to update delivery attempt: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rows == 0 {
		if span != nil {
			span.SetStatus(codes.Error, "delivery attempt not found")
		}
		return errors.New("delivery attempt not found")
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "update", "delivery_attempts", float64(duration))
	}

	return nil
}

// Delete удаляет попытку доставки
func (r *DeliveryAttemptRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.Delete")
		defer span.End()

		span.SetAttributes(attribute.String("attempt_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM delivery_attempts WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete delivery attempt: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "delete", "delivery_attempts", float64(duration))
	}

	return nil
}

// DeleteOldAttempts удаляет старые попытки доставки
func (r *DeliveryAttemptRepository) DeleteOldAttempts(ctx context.Context, olderThan int64) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.DeleteOldAttempts")
		defer span.End()

		span.SetAttributes(attribute.Int64("older_than", olderThan))
	}

	startTime := time.Now()

	query := `
		DELETE FROM delivery_attempts
		WHERE created_at < to_timestamp($1)
	`

	_, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete old delivery attempts: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "deleteold", "delivery_attempts", float64(duration))
	}

	return nil
}

// CleanupOldAttempts удаляет попытки старше указанного времени
func (r *DeliveryAttemptRepository) CleanupOldAttempts(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return r.DeleteOldAttempts(ctx, cutoff.Unix())
}

// CountRecentByMonitorAndChannel возвращает количество попыток доставки за указанный период
func (r *DeliveryAttemptRepository) CountRecentByMonitorAndChannel(
	ctx context.Context,
	monitorID string,
	channelType string,
	since time.Duration,
) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM delivery_attempts da
		JOIN alerts a ON da.alert_id = a.id
		JOIN alert_channels ac ON da.alert_channel_id = ac.id
		WHERE a.monitor_id = $1
			AND ac.type = $2
			AND da.created_at > NOW() - ($3 || ' seconds')::interval
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, monitorID, channelType, int(since.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("failed to count recent delivery attempts: %v", err)
	}

	return count, nil
}

// GetLastDeliveryTimeForMonitorAndStatus возвращает время последней доставки для монитора и статуса
func (r *DeliveryAttemptRepository) GetLastDeliveryTimeForMonitorAndStatus(
	ctx context.Context,
	monitorID string,
	status string,
) (*time.Time, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "DeliveryAttemptRepository.GetLastDeliveryTimeForMonitorAndStatus")
		defer span.End()

		span.SetAttributes(
			attribute.String("monitor_id", monitorID),
			attribute.String("status", status),
		)
	}

	startTime := time.Now()

	query := `
		SELECT MAX(da.created_at)
		FROM delivery_attempts da
		JOIN alerts a ON da.alert_id = a.id
		WHERE a.monitor_id = $1 AND a.status = $2
	`

	var lastDelivery *time.Time
	err := r.db.GetContext(ctx, &lastDelivery, query, monitorID, status)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get last delivery time for monitor and status: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getlastdelivery", "delivery_attempts,alerts", float64(duration))
	}

	return lastDelivery, nil
}
