package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/repository/interfaces"
)

// AuditRepository implements audit log repository using PostgreSQL
type AuditRepository struct {
	db *DB
}

// NewAuditRepository creates a new AuditRepository
func NewAuditRepository(db *DB) interfaces.AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error) {
	const query = `
		INSERT INTO auth_audit_log (id, user_id, event_type, provider, success, ip_address, user_agent, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, event_type, provider, success, ip_address, user_agent, error_message, created_at
	`

	// Postgres-тип inet не принимает пустые строки — превращаем "" в NULL.
	var ipArg any = log.IPAddress
	if log.IPAddress == "" {
		ipArg = nil
	}
	var ipScan sql.NullString
	err := r.db.QueryRowContext(ctx, query,
		log.ID, log.UserID, log.EventType, log.Provider, log.Success,
		ipArg, log.UserAgent, log.ErrorMessage, log.CreatedAt,
	).Scan(
		&log.ID, &log.UserID, &log.EventType, &log.Provider, &log.Success,
		&ipScan, &log.UserAgent, &log.ErrorMessage, &log.CreatedAt,
	)
	log.IPAddress = ipScan.String

	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %v", err)
	}

	return log, nil
}

// GetByID retrieves an audit log entry by ID
func (r *AuditRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AuditLog, error) {
	const query = `
		SELECT id, user_id, event_type, provider, success, ip_address, user_agent, error_message, created_at
		FROM auth_audit_log
		WHERE id = $1
	`

	log := &model.AuditLog{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID, &log.UserID, &log.EventType, &log.Provider, &log.Success,
		&log.IPAddress, &log.UserAgent, &log.ErrorMessage, &log.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %v", err)
	}

	return log, nil
}

// GetByUserID retrieves audit log entries for a user
func (r *AuditRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.AuditLog, int, error) {
	// Get total count
	const countQuery = `SELECT COUNT(*) FROM auth_audit_log WHERE user_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %v", err)
	}

	// Get logs
	const query = `
		SELECT id, user_id, event_type, provider, success, ip_address, user_agent, error_message, created_at
		FROM auth_audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			return
		}
	}()

	var logs []*model.AuditLog
	for rows.Next() {
		log := &model.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.EventType, &log.Provider, &log.Success,
			&log.IPAddress, &log.UserAgent, &log.ErrorMessage, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %v", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// GetByEventType retrieves audit log entries by event type
func (r *AuditRepository) GetByEventType(ctx context.Context, eventType string, limit, offset int) ([]*model.AuditLog, int, error) {
	// Get total count
	const countQuery = `SELECT COUNT(*) FROM auth_audit_log WHERE event_type = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, eventType).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %v", err)
	}

	// Get logs
	const query = `
		SELECT id, user_id, event_type, provider, success, ip_address, user_agent, error_message, created_at
		FROM auth_audit_log
		WHERE event_type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, eventType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			return
		}
	}()

	var logs []*model.AuditLog
	for rows.Next() {
		log := &model.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.EventType, &log.Provider, &log.Success,
			&log.IPAddress, &log.UserAgent, &log.ErrorMessage, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %v", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// DeleteOldLogs deletes audit logs older than specified duration in days
func (r *AuditRepository) DeleteOldLogs(ctx context.Context, olderThan int) error {
	const query = `DELETE FROM auth_audit_log WHERE created_at < NOW() - INTERVAL '1 day' * $1`

	_, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to delete old audit logs: %v", err)
	}

	return nil
}
