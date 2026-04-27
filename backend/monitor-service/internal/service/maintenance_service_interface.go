package service

import (
	"context"

	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// MaintenanceServiceInterface определяет интерфейс для maintenance window service.
// Используется для инъекции зависимостей и тестирования.
type MaintenanceServiceInterface interface {
	// CreateMaintenanceWindow создаёт новое окно обслуживания.
	CreateMaintenanceWindow(ctx context.Context, req *dto.CreateMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error)

	// UpdateMaintenanceWindow обновляет окно обслуживания.
	UpdateMaintenanceWindow(ctx context.Context, req *dto.UpdateMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error)

	// ListMaintenanceWindows возвращает список окон.
	ListMaintenanceWindows(ctx context.Context, req *dto.ListMaintenanceWindowsRequest) (*dto.ListMaintenanceWindowsResponse, error)

	// CancelMaintenanceWindow отменяет окно.
	CancelMaintenanceWindow(ctx context.Context, req *dto.CancelMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error)

	// GetMaintenanceWindowHistory возвращает историю окон.
	GetMaintenanceWindowHistory(ctx context.Context, req *dto.GetMaintenanceWindowHistoryRequest) (*dto.ListMaintenanceWindowsResponse, error)

	// DeleteMaintenanceWindow удаляет окно.
	DeleteMaintenanceWindow(ctx context.Context, windowID, userID string) error

	// ActivateScheduledWindows активирует окна с наступившим start_time.
	ActivateScheduledWindows(ctx context.Context) error

	// CompleteActiveWindows завершает окна с истёкшим end_time.
	CompleteActiveWindows(ctx context.Context) error
}

// Проверка что MaintenanceService реализует интерфейс
var _ MaintenanceServiceInterface = (*MaintenanceService)(nil)
