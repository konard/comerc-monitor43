package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

// ScheduledCheckRepository определяет методы для хранения и поиска проверок.
type ScheduledCheckRepository interface {
	// Create сохраняет новую проверку.
	Create(ctx context.Context, check *model.ScheduledCheck) error

	// GetByID находит проверку по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.ScheduledCheck, error)

	// GetByMonitorID находит последнюю проверку монитора.
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID) (*model.ScheduledCheck, error)

	// Update обновляет проверку.
	Update(ctx context.Context, check *model.ScheduledCheck) error

	// ListPendingByWorkerID находит PENDING и IN_PROGRESS проверки воркера.
	ListPendingByWorkerID(ctx context.Context, workerID uuid.UUID) ([]*model.ScheduledCheck, error)

	// ListByTimeRange находит проверки в временном интервале с пагинацией.
	ListByTimeRange(ctx context.Context, from, to time.Time, status string, page, pageSize int) ([]*model.ScheduledCheck, int, error)

	// ListOverdue находит просроченные PENDING проверки.
	ListOverdue(ctx context.Context, threshold time.Duration) ([]*model.ScheduledCheck, error)

	// ReassignByWorkerID переназначает проверки одного воркера другим.
	ReassignByWorkerID(ctx context.Context, oldWorkerID uuid.UUID) (int, error)

	// DeleteCompletedBefore удаляет старые COMPLETED/FAILED проверки.
	DeleteCompletedBefore(ctx context.Context, before time.Time) error

	// HasPendingCheck проверяет наличие незавершённой проверки монитора.
	HasPendingCheck(ctx context.Context, monitorID uuid.UUID) (bool, error)
}
