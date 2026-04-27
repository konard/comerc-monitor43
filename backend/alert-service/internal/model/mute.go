package model

import (
	"time"

	"github.com/google/uuid"
)

// MuteScope представляет область действия заглушения
type MuteScope string

const (
	MuteScopeUser   MuteScope = "user"
	MuteScopeGlobal MuteScope = "global"
)

// String возвращает строковое представление MuteScope
func (m MuteScope) String() string { return string(m) }

// AlertMute представляет заглушение алертов
type AlertMute struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	MonitorID  *uuid.UUID `db:"monitor_id"`
	Scope      MuteScope  `db:"scope"`
	MutedUntil *time.Time `db:"muted_until"`
	CreatedAt  time.Time  `db:"created_at"`
	CreatedBy  uuid.UUID  `db:"created_by"`
}

// AlertEscalation представляет эскалацию алерта
type AlertEscalation struct {
	ID                   uuid.UUID  `db:"id"`
	AlertID              uuid.UUID  `db:"alert_id"`
	Level                int        `db:"level"`
	EscalatedToChannelID *uuid.UUID `db:"escalated_to_channel_id"`
	Reason               string     `db:"reason"`
	TimeoutMinutes       int        `db:"timeout_minutes"`
	CreatedAt            time.Time  `db:"created_at"`
}

// MaintenanceWindowStatus представляет статус окна обслуживания
type MaintenanceWindowStatus string

const (
	MaintenanceStatusScheduled MaintenanceWindowStatus = "scheduled"
	MaintenanceStatusActive    MaintenanceWindowStatus = "active"
	MaintenanceStatusCompleted MaintenanceWindowStatus = "completed"
	MaintenanceStatusCancelled MaintenanceWindowStatus = "cancelled"
	MaintenanceStatusOrphaned  MaintenanceWindowStatus = "orphaned"
)

// String возвращает строковое представление MaintenanceWindowStatus
func (s MaintenanceWindowStatus) String() string { return string(s) }

// RecurrenceType представляет тип повторения окна обслуживания
type RecurrenceType string

const (
	RecurrenceTypeOnce    RecurrenceType = "once"
	RecurrenceTypeDaily   RecurrenceType = "daily"
	RecurrenceTypeWeekly  RecurrenceType = "weekly"
	RecurrenceTypeMonthly RecurrenceType = "monthly"
)

// String возвращает строковое представление RecurrenceType
func (r RecurrenceType) String() string { return string(r) }

// MaintenanceWindowFilter представляет фильтр для поиска окон обслуживания
type MaintenanceWindowFilter struct {
	Status    MaintenanceWindowStatus
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	PageSize  int
}

// MaintenanceWindow представляет окно обслуживания
type MaintenanceWindow struct {
	ID              uuid.UUID               `db:"id"`
	UserID          uuid.UUID               `db:"user_id"`
	Name            string                  `db:"name"`
	Status          MaintenanceWindowStatus `db:"status"`
	Recurrence      RecurrenceType          `db:"recurrence"`
	IsGlobal        bool                    `db:"is_global"`
	PauseMonitoring bool                    `db:"pause_monitoring"`
	SuppressAlerts  bool                    `db:"suppress_alerts"`
	SafeMode        bool                    `db:"safe_mode"`
	MonitorIDs      []string                `db:"monitor_ids"`
	StartsAt        time.Time               `db:"starts_at"`
	EndsAt          time.Time               `db:"ends_at"`
	ActivatedAt     *time.Time              `db:"activated_at"`
	CompletedAt     *time.Time              `db:"completed_at"`
	Version         int                     `db:"version"`
	Reason          *string                 `db:"reason"`
	CreatedAt       time.Time               `db:"created_at"`
	UpdatedAt       time.Time               `db:"updated_at"`

	// Deprecated: используйте MonitorIDs. Оставлено для обратной совместимости.
	MonitorID *uuid.UUID `db:"monitor_id"`
}

// MonitorStatusChange представляет смену статуса монитора
type MonitorStatusChange struct {
	ID        uuid.UUID `db:"id"`
	MonitorID uuid.UUID `db:"monitor_id"`
	UserID    uuid.UUID `db:"user_id"`
	OldStatus string    `db:"old_status"`
	NewStatus string    `db:"new_status"`
	CreatedAt time.Time `db:"created_at"`
}
