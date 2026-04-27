package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertRepository определяет интерфейс для работы с алертами
type AlertRepository interface {
	Create(ctx context.Context, alert *model.Alert) error
	GetByID(ctx context.Context, id string) (*model.Alert, error)
	GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error)
	ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error)
	GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error)
	List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error)
	Update(ctx context.Context, alert *model.Alert) error
	Delete(ctx context.Context, id string) error
	// CreateWithDeliveryAttempt создаёт алерт и попытку доставки в одной транзакции
	CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error
	DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error
	GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error)
	CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error)
	// AcknowledgeAlert подтверждает алерт и записывает кем и когда
	AcknowledgeAlert(ctx context.Context, alertID, userID string) error
}

// AlertRuleRepository определяет интерфейс для работы с правилами алертов
type AlertRuleRepository interface {
	Create(ctx context.Context, rule *model.AlertRule) error
	GetByID(ctx context.Context, id string) (*model.AlertRule, error)
	GetByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) (*model.AlertRule, error)
	List(ctx context.Context, userID string) ([]*model.AlertRule, error)
	Update(ctx context.Context, rule *model.AlertRule) error
	Delete(ctx context.Context, id string) error
}

// AlertChannelRepository определяет интерфейс для работы с каналами уведомлений
type AlertChannelRepository interface {
	Create(ctx context.Context, channel *model.AlertChannel) error
	GetByID(ctx context.Context, id string) (*model.AlertChannel, error)
	GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error)
	Update(ctx context.Context, channel *model.AlertChannel) error
	Delete(ctx context.Context, id string) error
	ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error)
	MarkAsFailed(ctx context.Context, id string, failureCount int) error
	ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error)
	CountByUserID(ctx context.Context, userID string) (int, error)
	IncrementFailureCount(ctx context.Context, id string) (int, error)
	DisableChannel(ctx context.Context, id string, reason string) error
	SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error
	GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error)
}

// DeliveryAttemptRepository определяет интерфейс для работы с попытками доставки
type DeliveryAttemptRepository interface {
	Create(ctx context.Context, attempt *model.DeliveryAttempt) error
	GetByID(ctx context.Context, id string) (*model.DeliveryAttempt, error)
	List(ctx context.Context, alertID string) ([]*model.DeliveryAttempt, error)
	ListPending(ctx context.Context, limit int) ([]*model.DeliveryAttempt, error)
	Update(ctx context.Context, attempt *model.DeliveryAttempt) error
	Delete(ctx context.Context, id string) error
	DeleteOldAttempts(ctx context.Context, olderThan int64) error
	CountRecentByMonitorAndChannel(ctx context.Context, monitorID string, channelType string, since time.Duration) (int, error)
	GetLastDeliveryTimeForMonitorAndStatus(ctx context.Context, monitorID string, status string) (*time.Time, error)
}

// AuditLogRepository определяет интерфейс для работы с журналом аудита
type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	List(ctx context.Context, resourceType string, resourceID string, limit int) ([]*model.AuditLog, error)
}

// AlertMuteRepository определяет интерфейс для работы с заглушением алертов
type AlertMuteRepository interface {
	Create(ctx context.Context, mute *model.AlertMute) error
	Delete(ctx context.Context, id string) error
	GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.AlertMute, error)
	GetActiveByUserID(ctx context.Context, userID string) ([]*model.AlertMute, error)
	IsMuted(ctx context.Context, userID, monitorID string) (bool, error)
	// DeleteByUserIDAndMonitorID удаляет все активные заглушения пользователя для монитора
	DeleteByUserIDAndMonitorID(ctx context.Context, userID, monitorID string) error
	// DeleteExpired удаляет все истёкшие заглушения из БД
	DeleteExpired(ctx context.Context) (int64, error)
}

// AlertEscalationRepository определяет интерфейс для работы с эскалациями алертов
type AlertEscalationRepository interface {
	Create(ctx context.Context, escalation *model.AlertEscalation) error
	GetLatestByAlertID(ctx context.Context, alertID string) (*model.AlertEscalation, error)
}

// MaintenanceWindowRepository определяет интерфейс для работы с окнами обслуживания
type MaintenanceWindowRepository interface {
	Create(ctx context.Context, mw *model.MaintenanceWindow) error
	GetByID(ctx context.Context, id string) (*model.MaintenanceWindow, error)
	GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.MaintenanceWindow, error)
	IsUnderMaintenance(ctx context.Context, monitorID string) (bool, error)
	Update(ctx context.Context, mw *model.MaintenanceWindow) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string, filter model.MaintenanceWindowFilter) ([]*model.MaintenanceWindow, int, error)
	// CheckOverlapping проверяет, пересекается ли новое окно с существующими для тех же мониторов.
	// excludeID позволяет исключить конкретное окно из проверки (при обновлении).
	CheckOverlapping(ctx context.Context, monitorIDs []string, startsAt, endsAt time.Time, excludeID string) (bool, *model.MaintenanceWindow, error)
}

// MonitorStatusChangeRepository определяет интерфейс для работы с изменениями статуса монитора
type MonitorStatusChangeRepository interface {
	Create(ctx context.Context, change *model.MonitorStatusChange) error
	CountInWindow(ctx context.Context, monitorID string, window time.Duration) (int, error)
	GetLatestByMonitorID(ctx context.Context, monitorID string) (*model.MonitorStatusChange, error)
}

// DB представляет интерфейс для работы с базой данных
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (any, error)
	QueryxContext(ctx context.Context, query string, args ...any) (any, error)
	QueryRowxContext(ctx context.Context, query string, args ...any) any
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (any, error)
	Commit(tx any) error
	Rollback(tx any) error
	Close() error
}
