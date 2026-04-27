package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditAction представляет тип действия в audit log.
type AuditAction string

const (
	// ActionMonitorCreate создание монитора
	ActionMonitorCreate AuditAction = "monitor.create"
	// ActionMonitorUpdate обновление монитора
	ActionMonitorUpdate AuditAction = "monitor.update"
	// ActionMonitorDelete удаление монитора
	ActionMonitorDelete AuditAction = "monitor.delete"
	// ActionMonitorPause приостановка монитора
	ActionMonitorPause AuditAction = "monitor.pause"
	// ActionMonitorResume возобновление монитора
	ActionMonitorResume AuditAction = "monitor.resume"
	// ActionMonitorStatusChange изменение статуса монитора
	ActionMonitorStatusChange AuditAction = "monitor.status_change"
	// ActionCheckCompleted проверка выполнена успешно
	ActionCheckCompleted AuditAction = "check_completed"
	// ActionCheckFailed проверка завершилась с ошибкой
	ActionCheckFailed AuditAction = "check_failed"
	// ActionCheckTimeout проверка превысила таймаут
	ActionCheckTimeout AuditAction = "check_timeout"
)

// AuditLogEntry представляет запись в audit log.
type AuditLogEntry struct {
	ID        uuid.UUID
	MonitorID uuid.UUID
	UserID    uuid.UUID
	Action    AuditAction
	OldValues map[string]any
	NewValues map[string]any
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

// AuditRepository определяет интерфейс для работы с audit log.
type AuditRepository interface {
	// Create записывает новую ауди запись.
	Create(ctx context.Context, entry *AuditLogEntry) error

	// GetByMonitorID возвращает audit log монитора.
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*AuditLogEntry, error)

	// GetByUserID возвращает audit log пользователя.
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*AuditLogEntry, error)

	// DeleteOld удаляет старые записи.
	DeleteOld(ctx context.Context, olderThan time.Time) (int64, error)
}
