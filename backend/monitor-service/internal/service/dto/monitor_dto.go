package dto

import "time"

// CreateMonitorRequest запрос на создание монитора.
type CreateMonitorRequest struct {
	UserID                        string
	Tier                          string // Subscription tier (Free, Basic, Pro, Enterprise)
	Name                          string
	URL                           string
	CheckType                     string
	IntervalSeconds               int
	TimeoutSeconds                int
	WorkingHoursStart             string
	WorkingHoursEnd               string
	WorkingDays                   []string
	DegradedResponseTimeThreshold *int
	DegradedFailureRateThreshold  *int
}

// UpdateMonitorRequest запрос на обновление монитора.
type UpdateMonitorRequest struct {
	ID                            string
	UserID                        string
	Name                          *string
	URL                           *string
	IntervalSeconds               *int
	TimeoutSeconds                *int
	WorkingHoursStart             *string
	WorkingHoursEnd               *string
	WorkingDays                   []string
	DegradedResponseTimeThreshold *int
	DegradedFailureRateThreshold  *int
}

// MonitorResponse ответ с информацией о мониторе.
type MonitorResponse struct {
	ID                            string
	UserID                        string
	Name                          string
	URL                           string
	CheckType                     string
	IntervalSeconds               int
	TimeoutSeconds                int
	Status                        string
	WorkingHoursStart             *string
	WorkingHoursEnd               *string
	WorkingDays                   []string
	DegradedResponseTimeThreshold *int
	DegradedFailureRateThreshold  *int
	LastCheckAt                   *time.Time
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

// ListMonitorsRequest запрос на список мониторов.
type ListMonitorsRequest struct {
	UserID string
	Status string
	Limit  int
	Offset int
	// SkipUserFilter отключает фильтрацию по UserID и возвращает все активные мониторы.
	// Используется сервисными вызовами (например, scheduler-service) с Role=SERVICE.
	SkipUserFilter bool
}

// ListMonitorsResponse ответ со списком мониторов.
type ListMonitorsResponse struct {
	Monitors []*MonitorResponse
	Total    int
	Limit    int
	Offset   int
}

// PauseMonitorRequest запрос на приостановку монитора.
type PauseMonitorRequest struct {
	ID     string
	UserID string
}

// ResumeMonitorRequest запрос на возобновление монитора.
type ResumeMonitorRequest struct {
	ID     string
	UserID string
}

// GetMonitorHistoryRequest запрос на историю проверок.
type GetMonitorHistoryRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
	Limit     int
	Offset    int
	Status    string
}

// GetMonitorHistoryResponse ответ с историей проверок.
type GetMonitorHistoryResponse struct {
	Results []*CheckResultResponse
	Total   int
	Limit   int
	Offset  int
}

// CheckResultResponse ответ с информацией о результате проверки.
type CheckResultResponse struct {
	ID             string
	MonitorID      string
	Status         string
	ResponseTimeMs *int
	StatusCode     *int
	ErrorMessage   *string
	CheckedAt      time.Time
	CreatedAt      time.Time
}

// GetUptimeStatsRequest запрос на статистику uptime.
type GetUptimeStatsRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
}

// UptimeStatsResponse ответ со статистикой uptime.
type UptimeStatsResponse struct {
	Uptime              float64
	TotalChecks         int
	UpChecks            int
	DegradedChecks      int
	DownChecks          int
	PausedChecks        int
	TotalDowntime       int64
	AverageResponseTime int
	Incidents           int
	Note                string
	P50                 int
	P95                 int
	P99                 int
}

// GetIncidentsRequest запрос на список инцидентов.
type GetIncidentsRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
	Limit     int
	Offset    int
}

// IncidentResponse ответ с информацией об инциденте.
type IncidentResponse struct {
	ID              string
	MonitorID       string
	StartTime       time.Time
	EndTime         *time.Time
	DurationSeconds *int
	Status          string
	CreatedAt       time.Time
}

// GetIncidentsResponse ответ со списком инцидентов.
type GetIncidentsResponse struct {
	Incidents []*IncidentResponse
	Total     int
	Limit     int
	Offset    int
}

// GetCheckResultsRequest запрос на raw check results за период.
type GetCheckResultsRequest struct {
	MonitorID string
	UserID    string
	From      time.Time
	To        time.Time
	Limit     int
	Offset    int
}

// GetCheckResultsResponse ответ с raw check results.
type GetCheckResultsResponse struct {
	Results []*CheckResultResponse
	Total   int
	Limit   int
	Offset  int
}
