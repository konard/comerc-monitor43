package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
)

// ScheduleCheckRequest - DTO для планирования проверки.
type ScheduleCheckRequest struct {
	MonitorID   uuid.UUID
	Priority    model.CheckPriority
	ScheduledAt time.Time
}

// GetScheduledChecksRequest - DTO для получения проверок за период.
type GetScheduledChecksRequest struct {
	FromTime time.Time
	ToTime   time.Time
	Status   string
	Page     int
	PageSize int
}

// GetNextCheckRequest - DTO для получения следующей проверки монитора.
type GetNextCheckRequest struct {
	MonitorID uuid.UUID
}

// TriggerScheduledCheckRequest - DTO для ручного запуска проверки.
type TriggerScheduledCheckRequest struct {
	MonitorID uuid.UUID
	Priority  model.CheckPriority
}

// CompleteCheckRequest - DTO для завершения проверки.
type CompleteCheckRequest struct {
	CheckID      uuid.UUID
	Failed       bool
	ErrorMessage string
}

// ReassignOverdueChecksRequest - DTO для переназначения просроченных проверок.
type ReassignOverdueChecksRequest struct {
	ThresholdTime time.Time
}

// ScheduledCheckResponse - DTO ответа с информацией о проверке.
type ScheduledCheckResponse struct {
	ID           uuid.UUID
	MonitorID    uuid.UUID
	WorkerID     uuid.UUID
	Priority     string
	ScheduledAt  time.Time
	ExecutedAt   *time.Time
	CompletedAt  *time.Time
	Status       string
	ErrorMessage string
}

// ScheduledChecksListResponse - DTO ответа со списком проверок.
type ScheduledChecksListResponse struct {
	Checks   []*ScheduledCheckResponse
	Total    int
	Page     int
	PageSize int
}
