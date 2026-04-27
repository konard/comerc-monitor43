package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

// MonitorRepository определяет интерфейс для работы с мониторами в БД.
type MonitorRepository interface {
	// Create создаёт новый монитор.
	Create(ctx context.Context, monitor *domain.Monitor) error

	// GetByID возвращает монитор по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error)

	// GetByUserIDAndName возвращает монитор по userID и name.
	GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*domain.Monitor, error)

	// ListByUserID возвращает список мониторов пользователя.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Monitor, error)

	// ListByUserIDAndStatus возвращает мониторы пользователя с указанным статусом.
	ListByUserIDAndStatus(ctx context.Context, userID uuid.UUID, status domain.MonitorStatus) ([]*domain.Monitor, error)

	// ListActive возвращает все активные мониторы (не PAUSED).
	ListActive(ctx context.Context) ([]*domain.Monitor, error)

	// ListDueForCheck возвращает мониторы, которые нужно проверить.
	ListDueForCheck(ctx context.Context, limit int) ([]*domain.Monitor, error)

	// Update обновляет монитор.
	Update(ctx context.Context, monitor *domain.Monitor) error

	// UpdateStatus обновляет статус монитора.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error

	// UpdateLastCheck обновляет время последней проверки.
	UpdateLastCheck(ctx context.Context, id uuid.UUID, lastCheckAt time.Time) error

	// Delete удаляет монитор.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByUserID возвращает количество мониторов пользователя.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// ExistsByName проверяет существование монитора с именем.
	ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error)
}

var (
	ErrMonitorNotFound = errors.New("monitor not found")
)
