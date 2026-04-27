package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
)

// CheckResultRepository определяет интерфейс для работы с результатами проверок.
type CheckResultRepository interface {
	// Create сохраняет результат проверки.
	Create(ctx context.Context, result *domain.CheckResult) error

	// GetByID возвращает результат по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CheckResult, error)

	// GetByMonitorID возвращает результаты проверки монитора.
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*domain.CheckResult, error)

	// GetByMonitorIDAndPeriod возвращает результаты за период.
	GetByMonitorIDAndPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.CheckResult, error)

	// GetByMonitorIDAndPeriodPaginated возвращает результаты за период с пагинацией.
	GetByMonitorIDAndPeriodPaginated(ctx context.Context, monitorID uuid.UUID, from, to time.Time, limit, offset int) ([]*domain.CheckResult, error)

	// GetByMonitorIDAndPeriodAndStatus возвращает результаты за период с фильтрацией по статусу.
	GetByMonitorIDAndPeriodAndStatus(ctx context.Context, monitorID uuid.UUID, from, to time.Time, status domain.MonitorStatus, limit, offset int) ([]*domain.CheckResult, error)

	// GetLatestByMonitorID возвращает последние N результатов.
	GetLatestByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.CheckResult, error)

	// DeleteOld удаляет старые результаты проверок.
	DeleteOld(ctx context.Context, olderThan time.Time) (int64, error)

	// CountByMonitorID возвращает количество результатов монитора.
	CountByMonitorID(ctx context.Context, monitorID uuid.UUID) (int64, error)
}

var (
	ErrCheckResultNotFound = errors.New("check result not found")
)
