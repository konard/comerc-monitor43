package model

import (
	"time"

	"github.com/google/uuid"
)

type MonitorStatus string

const (
	MonitorStatusUP       MonitorStatus = "UP"
	MonitorStatusDOWN     MonitorStatus = "DOWN"
	MonitorStatusDEGRADED MonitorStatus = "DEGRADED"
	MonitorStatusPAUSED   MonitorStatus = "PAUSED"
)

func (s MonitorStatus) String() string { return string(s) }

func MonitorStatusPriority(s MonitorStatus) int {
	switch s {
	case MonitorStatusDOWN:
		return 1
	case MonitorStatusDEGRADED:
		return 2
	case MonitorStatusUP:
		return 3
	case MonitorStatusPAUSED:
		return 4
	default:
		return 5
	}
}

type MonitorStatusView struct {
	ID                 uuid.UUID     `db:"id"`
	UserID             uuid.UUID     `db:"user_id"`
	Name               string        `db:"name"`
	URL                string        `db:"url"`
	Status             MonitorStatus `db:"status"`
	UptimePercentage   float64       `db:"uptime_percentage"`
	LastCheckedAt      *time.Time    `db:"last_checked_at"`
	LastResponseTimeMs *float64      `db:"last_response_time_ms"`
	Tags               string        `db:"tags"`
	CreatedAt          time.Time     `db:"created_at"`
	UpdatedAt          time.Time     `db:"updated_at"`
}
