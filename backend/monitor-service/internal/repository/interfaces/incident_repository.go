package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
)

// IncidentRepository определяет интерфейс для работы с инцидентами.
type IncidentRepository interface {
	// Create создаёт новый инцидент.
	Create(ctx context.Context, incident *domain.Incident) error

	// GetByID возвращает инцидент по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error)

	// GetByMonitorID возвращает инциденты монитора.
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*domain.Incident, error)

	// GetByMonitorIDAndPeriod возвращает инциденты за период.
	GetByMonitorIDAndPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Incident, error)

	// GetActiveByMonitorID возвращает активные инциденты монитора.
	GetActiveByMonitorID(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error)

	// Update обновляет инцидент.
	Update(ctx context.Context, incident *domain.Incident) error

	// Resolve закрывает инцидент.
	Resolve(ctx context.Context, id uuid.UUID, endTime time.Time) error

	// DeleteOld удаляет старые инциденты.
	DeleteOld(ctx context.Context, olderThan time.Time) (int64, error)

	// CountByMonitorID возвращает количество инцидентов монитора.
	CountByMonitorID(ctx context.Context, monitorID uuid.UUID) (int64, error)
}

var (
	ErrIncidentNotFound = errors.New("incident not found")
)
