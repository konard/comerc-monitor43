package dto

import "time"

// GetResponseTimeMetricsRequest содержит параметры запроса метрик времени ответа.
type GetResponseTimeMetricsRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
}

// ResponseTimeMetricsResponse содержит метрики времени ответа (перцентили, среднее, мин/макс).
type ResponseTimeMetricsResponse struct {
	MonitorID   string
	P50         int
	P95         int
	P99         int
	Average     int
	Min         int
	Max         int
	TotalChecks int
}

// GetIncidentsReportRequest содержит параметры запроса отчёта по инцидентам.
type GetIncidentsReportRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
	Status    string
	Limit     int
	Offset    int
}

// IncidentReportResponse содержит данные об инциденте.
type IncidentReportResponse struct {
	ID              string
	MonitorID       string
	StartTime       time.Time
	EndTime         *time.Time
	DurationSeconds int64
	Status          string
	FailedChecks    int
}

// GetIncidentsReportResponse содержит пагинированный список инцидентов.
type GetIncidentsReportResponse struct {
	Incidents []*IncidentReportResponse
	Total     int
	Limit     int
	Offset    int
}

// GetCheckCountByStatusRequest содержит параметры запроса количества проверок по статусу.
type GetCheckCountByStatusRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
}

// StatusCount содержит количество проверок для группы HTTP-статусов.
type StatusCount struct {
	StatusGroup string
	Count       int
}

// GetCheckCountByStatusResponse содержит распределение проверок по статус-кодам.
type GetCheckCountByStatusResponse struct {
	MonitorID    string
	StatusCounts []*StatusCount
	Total        int
}
