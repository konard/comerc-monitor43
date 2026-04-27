package model

import (
	"time"

	"github.com/google/uuid"
)

type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "IDLE"
	WorkerStatusBusy    WorkerStatus = "BUSY"
	WorkerStatusOffline WorkerStatus = "OFFLINE"
)

func ParseWorkerStatus(s string) (WorkerStatus, error) {
	switch s {
	case string(WorkerStatusIdle):
		return WorkerStatusIdle, nil
	case string(WorkerStatusBusy):
		return WorkerStatusBusy, nil
	case string(WorkerStatusOffline):
		return WorkerStatusOffline, nil
	default:
		return "", ErrInvalidWorkerStatus
	}
}

type Worker struct {
	ID                 uuid.UUID
	Name               string
	Zone               string
	Status             WorkerStatus
	LastHeartbeat      time.Time
	ChecksCompleted    int
	ChecksFailed       int
	AvgCheckDurationMs float64
	Metadata           map[string]string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewWorker(name, zone string) *Worker {
	now := time.Now()
	return &Worker{
		ID:                 uuid.New(),
		Name:               name,
		Zone:               zone,
		Status:             WorkerStatusIdle,
		LastHeartbeat:      now,
		ChecksCompleted:    0,
		ChecksFailed:       0,
		AvgCheckDurationMs: 0,
		Metadata:           make(map[string]string),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func (w *Worker) UpdateHeartbeat(status WorkerStatus, checksCompleted, checksFailed int, avgDurationMs float64) {
	w.Status = status
	w.LastHeartbeat = time.Now()
	w.ChecksCompleted = checksCompleted
	w.ChecksFailed = checksFailed
	w.AvgCheckDurationMs = avgDurationMs
	w.UpdatedAt = time.Now()
}

func (w *Worker) MarkOffline() {
	w.Status = WorkerStatusOffline
	w.UpdatedAt = time.Now()
}

func (w *Worker) IsExpired(timeout time.Duration) bool {
	return time.Since(w.LastHeartbeat) > timeout
}

type CheckPriority string

const (
	CheckPriorityLow      CheckPriority = "LOW"
	CheckPriorityNormal   CheckPriority = "NORMAL"
	CheckPriorityHigh     CheckPriority = "HIGH"
	CheckPriorityCritical CheckPriority = "CRITICAL"
)

// Старые константы для обратной совместимости
const (
	PriorityLow      = CheckPriorityLow
	PriorityNormal   = CheckPriorityNormal
	PriorityHigh     = CheckPriorityHigh
	PriorityCritical = CheckPriorityCritical
)

type CheckStatus string

const (
	CheckStatusPending    CheckStatus = "PENDING"
	CheckStatusInProgress CheckStatus = "IN_PROGRESS"
	CheckStatusCompleted  CheckStatus = "COMPLETED"
	CheckStatusFailed     CheckStatus = "FAILED"
)

type ScheduledCheck struct {
	ID           uuid.UUID
	MonitorID    uuid.UUID
	WorkerID     uuid.UUID
	Priority     CheckPriority
	ScheduledAt  time.Time
	ExecutedAt   *time.Time
	CompletedAt  *time.Time
	Status       CheckStatus
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewScheduledCheck(monitorID uuid.UUID, priority CheckPriority, scheduledAt time.Time) *ScheduledCheck {
	now := time.Now()
	return &ScheduledCheck{
		ID:          uuid.New(),
		MonitorID:   monitorID,
		Priority:    priority,
		ScheduledAt: scheduledAt,
		Status:      CheckStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (c *ScheduledCheck) Assign(workerID uuid.UUID) {
	c.WorkerID = workerID
	c.Status = CheckStatusInProgress
	now := time.Now()
	c.ExecutedAt = &now
	c.UpdatedAt = now
}

func (c *ScheduledCheck) Complete() {
	c.Status = CheckStatusCompleted
	now := time.Now()
	c.CompletedAt = &now
	c.UpdatedAt = now
}

// MarkCompleted - псевдоним для Complete для API согласованности
func (c *ScheduledCheck) MarkCompleted() {
	c.Complete()
}

func (c *ScheduledCheck) Fail(errMsg string) {
	c.Status = CheckStatusFailed
	c.ErrorMessage = errMsg
	now := time.Now()
	c.CompletedAt = &now
	c.UpdatedAt = now
}

// MarkFailed - псевдоним для Fail для API согласованности
func (c *ScheduledCheck) MarkFailed(errMsg string) {
	c.Fail(errMsg)
}

func (c *ScheduledCheck) IsOverdue(threshold time.Duration) bool {
	if c.Status != CheckStatusPending {
		return false
	}
	return time.Since(c.ScheduledAt) > threshold
}
