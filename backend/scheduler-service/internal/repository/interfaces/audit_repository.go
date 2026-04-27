package interfaces

import (
	"context"

	"github.com/google/uuid"
)

// AuditRepository определяет методы для аудита операций.
type AuditRepository interface {
	// Create записывает действие в аудит-лог.
	Create(ctx context.Context, action string, workerID, checkID, monitorID uuid.UUID, details map[string]any) error
}
