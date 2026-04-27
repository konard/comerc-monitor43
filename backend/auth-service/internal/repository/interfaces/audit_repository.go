package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// AuditRepository defines interface for audit log operations
type AuditRepository interface {
	// Create creates a new audit log entry
	Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error)

	// GetByID retrieves an audit log entry by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.AuditLog, error)

	// GetByUserID retrieves audit log entries for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.AuditLog, int, error)

	// GetByEventType retrieves audit log entries by event type
	GetByEventType(ctx context.Context, eventType string, limit, offset int) ([]*model.AuditLog, int, error)

	// DeleteOldLogs deletes audit logs older than specified duration
	DeleteOldLogs(ctx context.Context, olderThan int) error
}
