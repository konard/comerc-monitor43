package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertRepository реализует интерфейс AlertRepository для PostgreSQL
type AlertRepository struct {
	db *DB
}

// NewAlertRepository создаёт новый экземпляр AlertRepository
func NewAlertRepository(db *DB) *AlertRepository {
	return &AlertRepository{
		db: db,
	}
}

// Create создаёт новый алерт
func (r *AlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("alert_id", alert.ID.String()),
			attribute.String("user_id", alert.UserID.String()),
			attribute.String("monitor_id", alert.MonitorID.String()),
			attribute.String("status", string(alert.Status)),
			attribute.String("type", string(alert.Type)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO alerts (
			id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		) VALUES (
			:id, :user_id, :monitor_id, :alert_rule_id, :status, :type, :enabled,
			:consecutive_failures, :threshold_ms, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, alert)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create alert: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "alerts", float64(duration))
	}

	return nil
}

// GetByID возвращает алерт по ID
func (r *AlertRepository) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.GetByID")
		defer span.End()

		span.SetAttributes(attribute.String("alert_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		FROM alerts
		WHERE id = $1
	`

	var alert model.Alert
	err := r.db.GetContext(ctx, &alert, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get alert by id: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyid", "alerts", float64(duration))
	}

	return &alert, nil
}

// GetLastAlertTimeAnyStatus возвращает время последнего алерта для монитора
func (r *AlertRepository) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.GetLastAlertTimeAnyStatus")
		defer span.End()

		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		FROM alerts
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var alert model.Alert
	err := r.db.GetContext(ctx, &alert, query, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get last alert: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getlastalert", "alerts", float64(duration))
	}

	return &alert, nil
}

// ListActiveByMonitorID возвращает активные алерты для монитора
func (r *AlertRepository) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.ListActiveByMonitorID")
		defer span.End()

		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		FROM alerts
		WHERE monitor_id = $1 AND status = 'triggered'
		ORDER BY created_at DESC
	`

	var alerts []*model.Alert
	err := r.db.SelectContext(ctx, &alerts, query, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to list active alerts: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "listactive", "alerts", float64(duration))
	}

	return alerts, nil
}

// List возвращает алерты с фильтрацией и пагинацией
func (r *AlertRepository) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.List")
		defer span.End()

		span.SetAttributes(
			attribute.String("user_id", userID),
			attribute.String("monitor_id", filter.MonitorID),
			attribute.String("status", string(filter.Status)),
			attribute.String("page", fmt.Sprintf("%d", filter.Page)),
			attribute.String("page_size", fmt.Sprintf("%d", filter.PageSize)),
		)
	}

	startTime := time.Now()

	// Базовый запрос
	baseQuery := `
		SELECT id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		FROM alerts
		WHERE user_id = $1
	`

	// Запрос для подсчёта общего количества
	countQuery := `SELECT COUNT(*) FROM alerts WHERE user_id = $1`

	args := []any{userID}
	argCount := 2

	// Добавляем фильтры
	if filter.MonitorID != "" {
		baseQuery += fmt.Sprintf(" AND monitor_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND monitor_id = $%d", argCount)
		args = append(args, filter.MonitorID)
		argCount++
	}

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	// Считаем общее количество
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, fmt.Errorf("failed to count alerts: %v", err)
	}

	// Добавляем сортировку и пагинацию
	baseQuery += " ORDER BY created_at DESC"
	if filter.PageSize > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.PageSize)
		argCount++

		if filter.Page > 0 {
			offset := (filter.Page - 1) * filter.PageSize
			baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, offset)
		}
	}

	var alerts []*model.Alert
	err = r.db.SelectContext(ctx, &alerts, baseQuery, args...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, 0, fmt.Errorf("failed to list alerts: %v", err)
	}

	duration := time.Since(startTime).Milliseconds()
	if r.db.metrics != nil {
		r.db.metrics.RecordDBQuery(ctx, "list", "alerts", float64(duration))
	}

	return alerts, total, nil
}

// Update обновляет алерт
func (r *AlertRepository) Update(ctx context.Context, alert *model.Alert) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.Update")
		defer span.End()

		span.SetAttributes(
			attribute.String("alert_id", alert.ID.String()),
			attribute.String("user_id", alert.UserID.String()),
			attribute.String("monitor_id", alert.MonitorID.String()),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE alerts SET
			status = :status,
			type = :type,
			enabled = :enabled,
			consecutive_failures = :consecutive_failures,
			threshold_ms = :threshold_ms,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, alert)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to update alert: %v", err)
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
			span.SetStatus(codes.Error, "alert not found")
		}
		return model.ErrAlertNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "update", "alerts", float64(duration))
	}

	return nil
}

// Delete удаляет алерт
func (r *AlertRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.Delete")
		defer span.End()

		span.SetAttributes(attribute.String("alert_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM alerts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete alert: %v", err)
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
			span.SetStatus(codes.Error, "alert not found")
		}
		return model.ErrAlertNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "delete", "alerts", float64(duration))
	}

	return nil
}

// CreateWithDeliveryAttempt создаёт алерт и попытку доставки в одной транзакции
func (r *AlertRepository) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.CreateWithDeliveryAttempt")
		defer span.End()

		span.SetAttributes(
			attribute.String("alert_id", alert.ID.String()),
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("channel_id", attempt.AlertChannelID.String()),
		)
	}

	startTime := time.Now()

	// Начинаем транзакцию
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	defer func() {
		if err != nil {
			// Откатываем транзакцию при ошибке
			if rbErr := r.db.Rollback(tx); rbErr != nil {
				if r.db.tracer != nil {
					span.RecordError(rbErr)
				}
			}
		}
	}()

	// Type assert для получения sqlx.Tx
	sqlxTx, ok := tx.(*sqlx.Tx)
	if !ok {
		if span != nil {
			span.SetStatus(codes.Error, "invalid transaction type")
		}
		return errors.New("invalid transaction type")
	}

	// Создаём алерт
	alertQuery := `
		INSERT INTO alerts (
			id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		) VALUES (
			:id, :user_id, :monitor_id, :alert_rule_id, :status, :type, :enabled,
			:consecutive_failures, :threshold_ms, :created_at, :updated_at
		)
	`

	_, err = sqlxTx.NamedExecContext(ctx, alertQuery, alert)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create alert in transaction: %v", err)
	}

	// Создаём попытку доставки
	attemptQuery := `
		INSERT INTO delivery_attempts (
			id, alert_id, alert_channel_id, status, error_message,
			retry_count, next_retry_at, created_at, updated_at
		) VALUES (
			:id, :alert_id, :alert_channel_id, :status, :error_message,
			:retry_count, :next_retry_at, :created_at, :updated_at
		)
	`

	_, err = sqlxTx.NamedExecContext(ctx, attemptQuery, attempt)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create delivery attempt in transaction: %v", err)
	}

	// Фиксируем транзакцию
	if err := r.db.Commit(tx); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create_with_delivery", "alerts,delivery_attempts", float64(duration))
	}

	return nil
}

// DeleteResolvedOlderThan удаляет алерты в статусе resolved старше указанного времени
func (r *AlertRepository) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.DeleteResolvedOlderThan")
		defer span.End()

		span.SetAttributes(
			attribute.String("older_than", olderThan.String()),
		)
	}

	startTime := time.Now()

	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM alerts WHERE status = 'resolved' AND updated_at < $1`

	_, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete resolved alerts older than: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "deleteresolved", "alerts", float64(duration))
	}

	return nil
}

// GetLastAlertByMonitorIDAndStatus возвращает последний алерт для монитора с указанным статусом
func (r *AlertRepository) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.GetLastAlertByMonitorIDAndStatus")
		defer span.End()

		span.SetAttributes(
			attribute.String("monitor_id", monitorID),
			attribute.String("status", string(status)),
		)
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		FROM alerts
		WHERE monitor_id = $1 AND status = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var alert model.Alert
	err := r.db.GetContext(ctx, &alert, query, monitorID, status)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get last alert by monitor and status: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getlastbymonitorstatus", "alerts", float64(duration))
	}

	return &alert, nil
}

// CountUniqueMonitorsWithAlertsSince возвращает количество уникальных мониторов с алертами с указанного времени
func (r *AlertRepository) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.CountUniqueMonitorsWithAlertsSince")
		defer span.End()

		span.SetAttributes(
			attribute.String("since", since.String()),
		)
	}

	startTime := time.Now()

	query := `SELECT COUNT(DISTINCT monitor_id) FROM alerts WHERE created_at > $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, since)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return 0, fmt.Errorf("failed to count unique monitors with alerts since: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "countunique", "alerts", float64(duration))
	}

	return count, nil
}

// AcknowledgeAlert переводит алерт в статус acknowledged и записывает кем и когда
func (r *AlertRepository) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.AcknowledgeAlert")
		defer span.End()

		span.SetAttributes(
			attribute.String("alert_id", alertID),
			attribute.String("user_id", userID),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE alerts
		SET status = 'acknowledged',
		    acknowledged_by = $1,
		    acknowledged_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2 AND status = 'triggered'
	`

	result, err := r.db.ExecContext(ctx, query, userID, alertID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to acknowledge alert: %v", err)
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
			span.SetStatus(codes.Error, "alert not found or not acknowledgeable")
		}
		return model.ErrAlertNotAcknowledgeable
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "acknowledge", "alerts", float64(duration))
	}

	return nil
}

// GetActiveAlertsForMonitor возвращает активные алерты (triggered, acknowledged) для монитора
func (r *AlertRepository) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRepository.GetActiveAlertsForMonitor")
		defer span.End()

		span.SetAttributes(attribute.String("monitor_id", monitorID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, alert_rule_id, status, type, enabled,
			consecutive_failures, threshold_ms, created_at, updated_at
		FROM alerts
		WHERE monitor_id = $1 AND status IN ('triggered', 'acknowledged')
		ORDER BY created_at DESC
	`

	var alerts []*model.Alert
	err := r.db.SelectContext(ctx, &alerts, query, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get active alerts for monitor: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getactiveformonitor", "alerts", float64(duration))
	}

	return alerts, nil
}
