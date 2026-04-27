package model

import (
	"time"

	"github.com/google/uuid"
)

type IncidentStatus string

const (
	IncidentStatusActive   IncidentStatus = "ACTIVE"
	IncidentStatusResolved IncidentStatus = "RESOLVED"
)

func (s IncidentStatus) String() string { return string(s) }

type Incident struct {
	ID              uuid.UUID      `db:"id"`
	MonitorID       uuid.UUID      `db:"monitor_id"`
	StartedAt       time.Time      `db:"started_at"`
	EndedAt         *time.Time     `db:"ended_at"`
	DurationSeconds *int64         `db:"duration_seconds"`
	CheckCount      int            `db:"check_count"`
	Status          IncidentStatus `db:"status"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

type IncidentDetail struct {
	Incident Incident
	Timeline []CheckHistoryEntry
}

type IncidentFilter struct {
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	PageSize  int
}
