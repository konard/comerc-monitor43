package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// ImportHistoryRepository определяет интерфейс для работы с историей импорта.
type ImportHistoryRepository interface {
	// Create создаёт новую запись истории импорта.
	Create(ctx context.Context, history *model.ImportHistory) error

	// GetByID получает историю импорта по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.ImportHistory, error)

	// ListByUserID получает список истории импорта для пользователя.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.ImportHistory, error)

	// Update обновляет историю импорта.
	Update(ctx context.Context, history *model.ImportHistory) error

	// UpdateStatus обновляет статус импорта.
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.ImportStatus) error

	// Delete удаляет историю импорта.
	Delete(ctx context.Context, id uuid.UUID) error
}
