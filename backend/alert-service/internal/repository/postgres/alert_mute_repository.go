package postgres

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertMuteRepository реализует интерфейс AlertMuteRepository для PostgreSQL
type AlertMuteRepository struct {
	db *DB
}

// NewAlertMuteRepository создаёт новый экземпляр AlertMuteRepository
func NewAlertMuteRepository(db *DB) *AlertMuteRepository {
	return &AlertMuteRepository{db: db}
}

// Create создаёт заглушение алертов
func (r *AlertMuteRepository) Create(ctx context.Context, mute *model.AlertMute) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("user_id", mute.UserID.String()),
			attribute.String("scope", string(mute.Scope)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO alert_mutes (
			id, user_id, monitor_id, scope, muted_until, created_at, created_by
		) VALUES (
			:id, :user_id, :monitor_id, :scope, :muted_until, :created_at, :created_by
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, mute)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create alert mute: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "alert_mutes", float64(duration))
	}

	return nil
}

// Delete удаляет заглушение алертов
func (r *AlertMuteRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.Delete")
		defer span.End()

		span.SetAttributes(attribute.String("mute_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM alert_mutes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete alert mute: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rows == 0 {
		return model.ErrAlertNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "delete", "alert_mutes", float64(duration))
	}

	return nil
}

// GetActiveByMonitorID возвращает активное заглушение для монитора
func (r *AlertMuteRepository) GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.AlertMute, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.GetActiveByMonitorID")
		defer span.End()

		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, scope, muted_until, created_at, created_by
		FROM alert_mutes
		WHERE monitor_id = $1 AND (muted_until IS NULL OR muted_until > NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`

	var mute model.AlertMute
	err := r.db.GetContext(ctx, &mute, query, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get active alert mute by monitor: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getactive", "alert_mutes", float64(duration))
	}

	return &mute, nil
}

// GetActiveByUserID возвращает активные заглушения для пользователя
func (r *AlertMuteRepository) GetActiveByUserID(ctx context.Context, userID string) ([]*model.AlertMute, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.GetActiveByUserID")
		defer span.End()

		span.SetAttributes(attribute.String("user_id", userID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, scope, muted_until, created_at, created_by
		FROM alert_mutes
		WHERE user_id = $1 AND (muted_until IS NULL OR muted_until > NOW())
		ORDER BY created_at DESC
	`

	var mutes []*model.AlertMute
	err := r.db.SelectContext(ctx, &mutes, query, userID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get active alert mutes by user: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getactive", "alert_mutes", float64(duration))
	}

	return mutes, nil
}

// DeleteByUserIDAndMonitorID удаляет все активные заглушения пользователя для монитора
func (r *AlertMuteRepository) DeleteByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.DeleteByUserIDAndMonitorID")
		defer span.End()

		span.SetAttributes(
			attribute.String("user_id", userID),
			attribute.String("monitor_id", monitorID),
		)
	}

	startTime := time.Now()

	query := `DELETE FROM alert_mutes WHERE user_id = $1 AND monitor_id = $2 AND scope = 'user'`

	_, err := r.db.ExecContext(ctx, query, userID, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete alert mutes by user and monitor: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "deletebyusermonitor", "alert_mutes", float64(duration))
	}

	return nil
}

// DeleteExpired удаляет все истёкшие заглушения из БД
func (r *AlertMuteRepository) DeleteExpired(ctx context.Context) (int64, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.DeleteExpired")
		defer span.End()
	}

	startTime := time.Now()

	query := `DELETE FROM alert_mutes WHERE muted_until IS NOT NULL AND muted_until <= NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return 0, fmt.Errorf("failed to delete expired alert mutes: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return 0, fmt.Errorf("failed to get rows affected: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "deleteexpired", "alert_mutes", float64(duration))
	}

	return rows, nil
}

// IsMuted проверяет, заглушены ли алерты для монитора и пользователя
func (r *AlertMuteRepository) IsMuted(ctx context.Context, userID, monitorID string) (bool, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertMuteRepository.IsMuted")
		defer span.End()

		span.SetAttributes(
			attribute.String("user_id", userID),
			attribute.String("monitor_id", monitorID),
		)
	}

	startTime := time.Now()

	query := `
		SELECT COUNT(*) FROM alert_mutes
		WHERE (user_id = $1 OR scope = 'global')
		AND (monitor_id = $2 OR monitor_id IS NULL)
		AND (muted_until IS NULL OR muted_until > NOW())
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return false, fmt.Errorf("failed to check alert mute status: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "ismuted", "alert_mutes", float64(duration))
	}

	return count > 0, nil
}
