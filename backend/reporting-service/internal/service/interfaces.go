package service

import (
	"context"

	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

// SLAServiceInterface определяет интерфейс сервиса SLA-отчётов.
type SLAServiceInterface interface {
	GenerateSLAReport(ctx context.Context, req *dto.GenerateSLAReportRequest) (*dto.SLAReportResponse, error)
	GetSLAReport(ctx context.Context, reportID, userID string) (*dto.SLAReportResponse, error)
	ListSLAReports(ctx context.Context, req *dto.ListSLAReportsRequest) (*dto.ListSLAReportsResponse, error)
}

// AnalyticsServiceInterface определяет интерфейс аналитического сервиса.
type AnalyticsServiceInterface interface {
	GetResponseTimeMetrics(ctx context.Context, req *dto.GetResponseTimeMetricsRequest) (*dto.ResponseTimeMetricsResponse, error)
	GetIncidentsReport(ctx context.Context, req *dto.GetIncidentsReportRequest) (*dto.GetIncidentsReportResponse, error)
	GetCheckCountByStatus(ctx context.Context, req *dto.GetCheckCountByStatusRequest) (*dto.GetCheckCountByStatusResponse, error)
}

// ExportServiceInterface определяет интерфейс экспорта отчётов.
type ExportServiceInterface interface {
	ExportCSV(ctx context.Context, req *dto.ExportCSVRequest) (*dto.ExportCSVResponse, error)
	ExportPDF(ctx context.Context, req *dto.ExportPDFRequest) (*dto.ExportPDFResponse, error)
}
