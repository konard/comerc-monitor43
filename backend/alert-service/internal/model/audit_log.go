package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction представляет тип действия в журнале аудита
type AuditAction string

const (
	AuditActionChannelCreated      AuditAction = "channel_created"
	AuditActionChannelUpdated      AuditAction = "channel_updated"
	AuditActionChannelDeleted      AuditAction = "channel_deleted"
	AuditActionChannelVerified     AuditAction = "channel_verified"
	AuditActionChannelVerifyFailed AuditAction = "channel_verification_failed"
	AuditActionChannelAutoDisabled AuditAction = "channel_auto_disabled"
	AuditActionAlertRuleCreated    AuditAction = "alert_rule_created"
	AuditActionAlertTriggered      AuditAction = "alert_triggered"
	AuditActionAlertCreated        AuditAction = "alert_created"
	AuditActionAlertUpdated        AuditAction = "alert_updated"
	AuditActionAlertDeleted        AuditAction = "alert_deleted"
	AuditActionAlertEnabled        AuditAction = "alert_enabled"
	AuditActionAlertDisabled       AuditAction = "alert_disabled"
	AuditActionAlertAcknowledged   AuditAction = "alert_acknowledged"
	AuditActionAlertResolved       AuditAction = "alert_resolved"
	AuditActionAlertRetriggered    AuditAction = "alert_retriggered"
	AuditActionAlertMutedGlobal    AuditAction = "alerts_muted_global"
	AuditActionFlappingDetected    AuditAction = "flapping_detected"
	AuditActionAlertDelivered      AuditAction = "alert_delivered"
	AuditActionAlertRateLimited    AuditAction = "alert_rate_limited"
	AuditActionAlertStormDetected  AuditAction = "alert_storm_detected"
	AuditActionDeliveryRetry       AuditAction = "alert_delivery_retry"
	AuditActionDeliveryNoRetry     AuditAction = "delivery_no_retry_permanent_error"
	AuditActionUnauthorizedAttempt AuditAction = "unauthorized_access_attempt"

	// Действия с окнами обслуживания
	AuditActionMaintenanceWindowCreated        AuditAction = "maintenance_window.created"
	AuditActionMaintenanceWindowUpdated        AuditAction = "maintenance_window.updated"
	AuditActionMaintenanceWindowDeleted        AuditAction = "maintenance_window.deleted"
	AuditActionMaintenanceWindowCancelled      AuditAction = "maintenance_window.cancelled"
	AuditActionMaintenanceWindowDurationExceed AuditAction = "maintenance_window_duration_exceeded"
	AuditActionMaintenanceWindowOverlapping    AuditAction = "overlapping_maintenance_windows_rejected"
	AuditActionGlobalMaintenanceWindowDenied   AuditAction = "global_maintenance_window_creation_denied"
	AuditActionGlobalMaintenanceWindowCreated  AuditAction = "global_maintenance_window_created"
)

// String возвращает строковое представление AuditAction
func (a AuditAction) String() string { return string(a) }

// AuditLog представляет запись в журнале аудита
type AuditLog struct {
	ID           uuid.UUID      `db:"id"`
	UserID       *uuid.UUID     `db:"user_id"`
	Action       AuditAction    `db:"action"`
	ResourceType string         `db:"resource_type"`
	ResourceID   *string        `db:"resource_id"`
	Fields       map[string]any `db:"fields"`
	IPAddress    *string        `db:"ip_address"`
	CreatedAt    time.Time      `db:"created_at"`
}
