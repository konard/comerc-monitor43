package model

import (
	"time"

	"github.com/google/uuid"
)

// SLAReport представляет отчёт SLA (Service Level Agreement).
type SLAReport struct {
	ID                   uuid.UUID `db:"id"`
	MonitorID            uuid.UUID `db:"monitor_id"`
	UserID               uuid.UUID `db:"user_id"`
	MonitorName          string    `db:"monitor_name"`
	PeriodStart          time.Time `db:"period_start"`
	PeriodEnd            time.Time `db:"period_end"`
	Availability         float64   `db:"availability"`
	TotalChecks          int       `db:"total_checks"`
	UpChecks             int       `db:"up_checks"`
	DownChecks           int       `db:"down_checks"`
	DegradedChecks       int       `db:"degraded_checks"`
	PausedChecks         int       `db:"paused_checks"`
	TotalDowntimeSeconds int64     `db:"total_downtime_seconds"`
	IncidentsCount       int       `db:"incidents_count"`
	CreatedAt            time.Time `db:"created_at"`
}

// NewSLAReport создаёт новый экземпляр SLAReport.
func NewSLAReport(
	monitorID, userID uuid.UUID,
	monitorName string,
	periodStart, periodEnd time.Time,
) *SLAReport {
	return &SLAReport{
		ID:          uuid.New(),
		MonitorID:   monitorID,
		UserID:      userID,
		MonitorName: monitorName,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   time.Now(),
	}
}
