package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

// WorkerRepository определяет методы для хранения и поиска воркеров.
type WorkerRepository interface {
	// Create сохраняет нового воркера.
	Create(ctx context.Context, worker *model.Worker) error

	// GetByID находит воркера по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Worker, error)

	// GetByName находит воркера по имени (для проверки уникальности).
	GetByName(ctx context.Context, name string) (*model.Worker, error)

	// List возвращает воркеров с пагинацией и фильтрами.
	List(ctx context.Context, status, zone string, page, pageSize int) ([]*model.Worker, int, error)

	// Update обновляет воркера.
	Update(ctx context.Context, worker *model.Worker) error

	// Delete удаляет воркера.
	Delete(ctx context.Context, id uuid.UUID) error

	// ListExpired находит воркеров с просроченным heartbeat.
	ListExpired(ctx context.Context, timeout time.Duration) ([]*model.Worker, error)

	// ListIdleByZone находит IDLE воркеров в зоне (пустая зона = все).
	ListIdleByZone(ctx context.Context, zone string) ([]*model.Worker, error)

	// ListOfflineForCleanup находит OFFLINE воркеров для удаления.
	ListOfflineForCleanup(ctx context.Context, cutoffTime time.Time) ([]*model.Worker, error)
}
