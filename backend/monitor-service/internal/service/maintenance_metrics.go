package service

import (
	"context"
	"time"

	"github.com/raul/monitor/backend/monitor-service/pkg/logger"
)

// Определяем ключи для метрик
const (
	metricMaintenanceWindowsCreatedTotal     = "maintenance.windows.created.total"
	metricMaintenanceWindowsUpdatedTotal     = "maintenance.windows.updated.total"
	metricMaintenanceWindowsDeletedTotal     = "maintenance.windows.deleted.total"
	metricMaintenanceWindowsCancelledTotal   = "maintenance.windows.cancelled.total"
	metricMaintenanceWindowsActiveCurrent    = "maintenance.windows.active"
	metricMaintenanceWindowsCreationDuration = "maintenance.windows.creation.duration"
	metricMaintenanceWindowsUpdateDuration   = "maintenance.windows.update.duration"
)

// MaintenanceMetrics собирает метрики для maintenance windows
type MaintenanceMetrics struct {
	// Для простоты используем счетчики без реального Meter provider
	// В production нужно использовать OpenTelemetry Meter
	createdTotal     int64
	updatedTotal     int64
	deletedTotal     int64
	cancelledTotal   int64
	activeCurrent    int64
	creationDuration time.Duration
	updateDuration   time.Duration
}

// NewMaintenanceMetrics создаёт новый коллектор метрик
func NewMaintenanceMetrics() *MaintenanceMetrics {
	return &MaintenanceMetrics{}
}

// RecordCreation записывает метрику создания окна
func (m *MaintenanceMetrics) RecordCreation(ctx context.Context, duration time.Duration, success bool) {
	m.createdTotal++
	if success {
		m.creationDuration += duration
	}

	// NOTE: Метрики записываются в debug лог.
	// Для production необходимо настроить OpenTelemetry MeterProvider в cmd/service/main.go
	// и раскомментировать запись метрик:
	// meter.RecordCounter(ctx, metricMaintenanceWindowsCreatedTotal,...)
	logger.DefaultLogger.DebugContext(ctx, "maintenance window created",
		"duration_ms", duration.Milliseconds(),
		"total_created", m.createdTotal,
	)
}

// RecordUpdate записывает метрику обновления окна
func (m *MaintenanceMetrics) RecordUpdate(ctx context.Context, duration time.Duration, success bool) {
	m.updatedTotal++
	if success {
		m.updateDuration += duration
	}

	logger.DefaultLogger.DebugContext(ctx, "maintenance window updated",
		"duration_ms", duration.Milliseconds(),
		"total_updated", m.updatedTotal,
	)
}

// RecordDeletion записывает метрику удаления окна
func (m *MaintenanceMetrics) RecordDeletion(ctx context.Context) {
	m.deletedTotal++
	logger.DefaultLogger.DebugContext(ctx, "maintenance window deleted",
		"total_deleted", m.deletedTotal,
	)
}

// RecordCancellation записывает метрику отмены окна
func (m *MaintenanceMetrics) RecordCancellation(ctx context.Context) {
	m.cancelledTotal++
	logger.DefaultLogger.DebugContext(ctx, "maintenance window cancelled",
		"total_cancelled", m.cancelledTotal,
	)
}

// SetActiveWindows устанавливает количество активных окон
func (m *MaintenanceMetrics) SetActiveWindows(count int64) {
	m.activeCurrent = count
}

// GetMetrics возвращает текущие значения метрик
func (m *MaintenanceMetrics) GetMetrics() map[string]any {
	return map[string]any{
		"created_total":        m.createdTotal,
		"updated_total":        m.updatedTotal,
		"deleted_total":        m.deletedTotal,
		"cancelled_total":      m.cancelledTotal,
		"active_current":       m.activeCurrent,
		"creation_duration_ms": m.creationDuration.Milliseconds(),
		"update_duration_ms":   m.updateDuration.Milliseconds(),
	}
}
