package postgres

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertRuleRepository реализует интерфейс AlertRuleRepository для PostgreSQL
type AlertRuleRepository struct {
	db *DB
}

// NewAlertRuleRepository создаёт новый экземпляр AlertRuleRepository
func NewAlertRuleRepository(db *DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

// Create создаёт новое правило алерта
func (r *AlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRuleRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("rule_id", rule.ID.String()),
			attribute.String("user_id", rule.UserID.String()),
			attribute.String("monitor_id", rule.MonitorID.String()),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO alert_rules (
			id, user_id, monitor_id, enabled, consecutive_failures, created_at, updated_at
		) VALUES (
			:id, :user_id, :monitor_id, :enabled, :consecutive_failures, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, rule)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create alert rule: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "alert_rules", float64(duration))
	}

	return nil
}

// GetByID возвращает правило алерта по ID
func (r *AlertRuleRepository) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRuleRepository.GetByID")
		defer span.End()

		span.SetAttributes(attribute.String("rule_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, enabled, consecutive_failures, created_at, updated_at
		FROM alert_rules
		WHERE id = $1
	`

	var rule model.AlertRule
	err := r.db.GetContext(ctx, &rule, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get alert rule by id: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyid", "alert_rules", float64(duration))
	}

	return &rule, nil
}

// GetByUserIDAndMonitorID возвращает правило алерта для пользователя и монитора
func (r *AlertRuleRepository) GetByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) (*model.AlertRule, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRuleRepository.GetByUserIDAndMonitorID")
		defer span.End()

		span.SetAttributes(
			attribute.String("user_id", userID),
			attribute.String("monitor_id", monitorID),
		)
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, enabled, consecutive_failures, created_at, updated_at
		FROM alert_rules
		WHERE user_id = $1 AND monitor_id = $2
	`

	var rule model.AlertRule
	err := r.db.GetContext(ctx, &rule, query, userID, monitorID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get alert rule by user and monitor: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyusermonitor", "alert_rules", float64(duration))
	}

	return &rule, nil
}

// List возвращает все правила алертов пользователя
func (r *AlertRuleRepository) List(ctx context.Context, userID string) ([]*model.AlertRule, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRuleRepository.List")
		defer span.End()

		span.SetAttributes(attribute.String("user_id", userID))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, monitor_id, enabled, consecutive_failures, created_at, updated_at
		FROM alert_rules
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var rules []*model.AlertRule
	err := r.db.SelectContext(ctx, &rules, query, userID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to list alert rules: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "list", "alert_rules", float64(duration))
	}

	return rules, nil
}

// Update обновляет правило алерта
func (r *AlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRuleRepository.Update")
		defer span.End()

		span.SetAttributes(
			attribute.String("rule_id", rule.ID.String()),
			attribute.String("user_id", rule.UserID.String()),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE alert_rules SET
			enabled = :enabled,
			consecutive_failures = :consecutive_failures,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, rule)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to update alert rule: %v", err)
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
			span.SetStatus(codes.Error, "alert rule not found")
		}
		return model.ErrAlertRuleNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "update", "alert_rules", float64(duration))
	}

	return nil
}

// Delete удаляет правило алерта
func (r *AlertRuleRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertRuleRepository.Delete")
		defer span.End()

		span.SetAttributes(attribute.String("rule_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM alert_rules WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete alert rule: %v", err)
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
			span.SetStatus(codes.Error, "alert rule not found")
		}
		return model.ErrAlertRuleNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "delete", "alert_rules", float64(duration))
	}

	return nil
}
