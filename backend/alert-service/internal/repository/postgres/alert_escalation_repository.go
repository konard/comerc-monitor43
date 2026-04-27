package postgres

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertEscalationRepository реализует интерфейс AlertEscalationRepository для PostgreSQL
type AlertEscalationRepository struct {
	db *DB
}

// NewAlertEscalationRepository создаёт новый экземпляр AlertEscalationRepository
func NewAlertEscalationRepository(db *DB) *AlertEscalationRepository {
	return &AlertEscalationRepository{db: db}
}

// Create создаёт эскалацию алерта
func (r *AlertEscalationRepository) Create(ctx context.Context, escalation *model.AlertEscalation) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertEscalationRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("alert_id", escalation.AlertID.String()),
			attribute.Int("level", escalation.Level),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO alert_escalations (
			id, alert_id, level, escalated_to_channel_id, reason, timeout_minutes, created_at
		) VALUES (
			:id, :alert_id, :level, :escalated_to_channel_id, :reason, :timeout_minutes, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, escalation)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create alert escalation: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "alert_escalations", float64(duration))
	}

	return nil
}

// GetLatestByAlertID возвращает последнюю эскалацию для алерта
func (r *AlertEscalationRepository) GetLatestByAlertID(ctx context.Context, alertID string) (*model.AlertEscalation, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertEscalationRepository.GetLatestByAlertID")
		defer span.End()

		span.SetAttributes(attribute.String("alert_id", alertID))
	}

	startTime := time.Now()

	query := `
		SELECT id, alert_id, level, escalated_to_channel_id, reason, timeout_minutes, created_at
		FROM alert_escalations
		WHERE alert_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var escalation model.AlertEscalation
	err := r.db.GetContext(ctx, &escalation, query, alertID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get latest alert escalation: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getlatest", "alert_escalations", float64(duration))
	}

	return &escalation, nil
}
