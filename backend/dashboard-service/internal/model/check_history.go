package model

import (
	"time"

	"github.com/google/uuid"
)

type CheckStatus string

const (
	CheckStatusUP       CheckStatus = "UP"
	CheckStatusDOWN     CheckStatus = "DOWN"
	CheckStatusDEGRADED CheckStatus = "DEGRADED"
)

func (s CheckStatus) String() string { return string(s) }

type CheckHistoryEntry struct {
	ID             uuid.UUID   `db:"id"`
	MonitorID      uuid.UUID   `db:"monitor_id"`
	Status         CheckStatus `db:"status"`
	StatusCode     *int        `db:"status_code"`
	ResponseTimeMs *float64    `db:"response_time_ms"`
	ErrorMessage   *string     `db:"error_message"`
	CheckedAt      time.Time   `db:"checked_at"`
}

type HistoryFilter struct {
	Status    CheckStatus
	StartDate *time.Time
	EndDate   *time.Time
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}
