package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AuditLogRepository реализует интерфейс AuditLogRepository для PostgreSQL
type AuditLogRepository struct {
	db *DB
}

// NewAuditLogRepository создаёт новый экземпляр AuditLogRepository
func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create создаёт запись в журнале аудита
func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AuditLogRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("action", string(log.Action)),
			attribute.String("resource_type", log.ResourceType),
		)
	}

	startTime := time.Now()

	fieldsJSON, err := json.Marshal(log.Fields)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to marshal audit log fields: %v", err)
	}

	query := `
		INSERT INTO audit_logs (
			id, user_id, action, resource_type, resource_id, fields, ip_address, created_at
		) VALUES (
			:id, :user_id, :action, :resource_type, :resource_id, :fields, :ip_address, :created_at
		)
	`

	params := map[string]any{
		"id":            log.ID,
		"user_id":       log.UserID,
		"action":        log.Action,
		"resource_type": log.ResourceType,
		"resource_id":   log.ResourceID,
		"fields":        fieldsJSON,
		"ip_address":    log.IPAddress,
		"created_at":    log.CreatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to create audit log: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "audit_logs", float64(duration))
	}

	return nil
}

// List возвращает записи журнала аудита с фильтрацией
func (r *AuditLogRepository) List(ctx context.Context, resourceType, resourceID string, limit int) ([]*model.AuditLog, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AuditLogRepository.List")
		defer span.End()

		span.SetAttributes(
			attribute.String("resource_type", resourceType),
			attribute.String("resource_id", resourceID),
		)
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, action, resource_type, resource_id, fields, ip_address, created_at
		FROM audit_logs
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	type auditRow struct {
		ID           string            `db:"id"`
		UserID       *string           `db:"user_id"`
		Action       model.AuditAction `db:"action"`
		ResourceType string            `db:"resource_type"`
		ResourceID   *string           `db:"resource_id"`
		Fields       []byte            `db:"fields"`
		IPAddress    *string           `db:"ip_address"`
		CreatedAt    time.Time         `db:"created_at"`
	}

	var rows []auditRow
	err := r.db.SelectContext(ctx, &rows, query, resourceType, resourceID, limit)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to list audit logs: %v", err)
	}

	logs := make([]*model.AuditLog, len(rows))
	for i, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse audit log id: %v", err)
		}

		log := &model.AuditLog{
			ID:           id,
			Action:       row.Action,
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			IPAddress:    row.IPAddress,
			CreatedAt:    row.CreatedAt,
		}

		if row.UserID != nil {
			userID, err := uuid.Parse(*row.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to parse audit log user_id: %v", err)
			}
			log.UserID = &userID
		}

		if len(row.Fields) > 0 {
			var fields map[string]any
			if err := json.Unmarshal(row.Fields, &fields); err != nil {
				return nil, fmt.Errorf("failed to unmarshal audit log fields: %v", err)
			}
			log.Fields = fields
		}

		logs[i] = log
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "list", "audit_logs", float64(duration))
	}

	return logs, nil
}
