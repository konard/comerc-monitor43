package dto

import "time"

// GenerateSLAReportRequest содержит параметры запроса на генерацию SLA-отчёта.
type GenerateSLAReportRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
}

// SLAReportResponse содержит данные сгенерированного SLA-отчёта.
type SLAReportResponse struct {
	ID                   string
	MonitorID            string
	MonitorName          string
	PeriodStart          time.Time
	PeriodEnd            time.Time
	Availability         float64
	TotalChecks          int
	UpChecks             int
	DownChecks           int
	DegradedChecks       int
	PausedChecks         int
	TotalDowntimeSeconds int64
	IncidentsCount       int
	CreatedAt            time.Time
}

// ListSLAReportsRequest содержит параметры запроса списка SLA-отчётов.
type ListSLAReportsRequest struct {
	MonitorID string
	UserID    string
	Limit     int
	Offset    int
}

// ListSLAReportsResponse содержит пагинированный список SLA-отчётов.
type ListSLAReportsResponse struct {
	Reports []*SLAReportResponse
	Total   int
	Limit   int
	Offset  int
}
