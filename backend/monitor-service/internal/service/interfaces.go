package service

import (
	"context"

	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// MonitorServiceInterface описывает интерфейс сервиса мониторов.
type MonitorServiceInterface interface {
	CreateMonitor(ctx context.Context, req *dto.CreateMonitorRequest) (*dto.MonitorResponse, error)
	GetMonitor(ctx context.Context, id, userID string) (*dto.MonitorResponse, error)
	ListMonitors(ctx context.Context, req *dto.ListMonitorsRequest) (*dto.ListMonitorsResponse, error)
	UpdateMonitor(ctx context.Context, req *dto.UpdateMonitorRequest) (*dto.MonitorResponse, error)
	DeleteMonitor(ctx context.Context, id, userID string) error
	PauseMonitor(ctx context.Context, req *dto.PauseMonitorRequest) error
	ResumeMonitor(ctx context.Context, req *dto.ResumeMonitorRequest) error
}

// UptimeCalculatorInterface описывает интерфейс калькулятора uptime.
type UptimeCalculatorInterface interface {
	CalculateUptime(ctx context.Context, req *dto.GetUptimeStatsRequest) (*dto.UptimeStatsResponse, error)
	GetMonitorHistory(ctx context.Context, req *dto.GetMonitorHistoryRequest) (*dto.GetMonitorHistoryResponse, error)
	GetCheckResults(ctx context.Context, req *dto.GetCheckResultsRequest) (*dto.GetCheckResultsResponse, error)
}

// IncidentDetectorInterface описывает интерфейс детектора инцидентов.
type IncidentDetectorInterface interface {
	GetIncidents(ctx context.Context, req *dto.GetIncidentsRequest) (*dto.GetIncidentsResponse, error)
}
