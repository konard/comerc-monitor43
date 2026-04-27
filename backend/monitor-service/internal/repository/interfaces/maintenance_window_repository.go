package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MaintenanceWindowRepository определяет интерфейс для работы с окнами обслуживания.
type MaintenanceWindowRepository interface {
	// Create создаёт новое окно обслуживания.
	Create(ctx context.Context, window *MaintenanceWindow) error

	// GetByID возвращает окно по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*MaintenanceWindow, error)

	// GetByUserID возвращает окна пользователя с пагинацией.
	GetByUserID(ctx context.Context, userID uuid.UUID, status string, limit, offset int) ([]*MaintenanceWindow, error)

	// GetActiveWindowsForMonitor возвращает активные окна для монитора.
	GetActiveWindowsForMonitor(ctx context.Context, monitorID uuid.UUID, at time.Time) ([]*MaintenanceWindow, error)

	// GetActiveWindowsForUser возвращает все активные окна пользователя.
	GetActiveWindowsForUser(ctx context.Context, userID uuid.UUID, at time.Time) ([]*MaintenanceWindow, error)

	// Update обновляет окно обслуживания.
	Update(ctx context.Context, window *MaintenanceWindow) error

	// Delete удаляет окно обслуживания.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByUserID возвращает количество окон пользователя.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// CheckOverlap проверяет пересечение окон для мониторов.
	CheckOverlap(ctx context.Context, userID uuid.UUID, monitorIDs []uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error)

	// GetWindowsRequiringActivation возвращает окна, которые нужно активировать.
	GetWindowsRequiringActivation(ctx context.Context, before time.Time) ([]*MaintenanceWindow, error)

	// GetWindowsRequiringCompletion возвращает окна, которые нужно завершить.
	GetWindowsRequiringCompletion(ctx context.Context, before time.Time) ([]*MaintenanceWindow, error)

	// AddMonitorsToWindow добавляет мониторы к окну.
	AddMonitorsToWindow(ctx context.Context, windowID uuid.UUID, monitorIDs []uuid.UUID) error

	// RemoveMonitorsFromWindow удаляет мониторы из окна.
	RemoveMonitorsFromWindow(ctx context.Context, windowID uuid.UUID, monitorIDs []uuid.UUID) error

	// GetWindowMonitors возвращает мониторы окна.
	GetWindowMonitors(ctx context.Context, windowID uuid.UUID) ([]uuid.UUID, error)

	// GetHistory возвращает историю окон пользователя за период.
	GetHistory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, monitorIDs []uuid.UUID, limit, offset int) ([]*MaintenanceWindow, error)
}

// MaintenanceWindow представляет окно технического обслуживания.
type MaintenanceWindow struct {
	ID              uuid.UUID   `db:"id"`
	UserID          uuid.UUID   `db:"user_id"`
	Name            string      `db:"name"`
	StartTime       time.Time   `db:"start_time"`
	EndTime         time.Time   `db:"end_time"`
	Status          string      `db:"status"`
	Recurrence      string      `db:"recurrence"`
	IsGlobal        bool        `db:"is_global"`
	PauseMonitoring bool        `db:"pause_monitoring"`
	SuppressAlerts  bool        `db:"suppress_alerts"`
	SafeMode        bool        `db:"safe_mode"`
	MonitorIDs      []uuid.UUID `db:"-"` // Заполняется при загрузке через GetWindowMonitors
	CreatedAt       time.Time   `db:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at"`
	ActivatedAt     *time.Time  `db:"activated_at"`
	CompletedAt     *time.Time  `db:"completed_at"`
	Version         int         `db:"version"`
}

// MaintenanceWindowFilter представляет фильтр для поиска окон.
type MaintenanceWindowFilter struct {
	UserID     *uuid.UUID
	Status     *string
	StartDate  *time.Time
	EndDate    *time.Time
	MonitorIDs []uuid.UUID
	Limit      int
	Offset     int
}
