package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dashboardproto "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/dashboard-service/internal/infrastructure/auth"
	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
)

type DashboardServiceServer struct {
	dashboardService DashboardService
	historyService   HistoryService
	authMiddleware   *auth.AuthMiddleware
	dashboardproto.UnimplementedDashboardServiceServer
}

type DashboardService interface {
	GetDashboard(ctx context.Context, userID string, filter domain.DashboardFilter) ([]*domain.MonitorStatusView, int, float64, error)
	ExportDashboard(ctx context.Context, userID string, filter domain.DashboardFilter, format string) (string, error)
}

type HistoryService interface {
	GetCheckHistory(ctx context.Context, monitorID string, filter domain.HistoryFilter) ([]*domain.CheckHistoryEntry, int, error)
	GetIncidents(ctx context.Context, monitorID string, filter domain.IncidentFilter) ([]*domain.Incident, int, error)
	GetIncidentDetails(ctx context.Context, incidentID string) (*domain.IncidentDetail, error)
	GetPeriodMetrics(ctx context.Context, monitorID string, start, end time.Time) (*domain.PeriodMetrics, error)
	ExportHistory(ctx context.Context, monitorID string, filter domain.HistoryFilter, format string, fields []string) (string, error)
}

func NewDashboardServiceServer(
	dashboardService DashboardService,
	historyService HistoryService,
	authMiddleware *auth.AuthMiddleware,
) *DashboardServiceServer {
	return &DashboardServiceServer{
		dashboardService: dashboardService,
		historyService:   historyService,
		authMiddleware:   authMiddleware,
	}
}

func validateUUID(s string) error {
	if _, err := uuid.Parse(s); err != nil {
		return status.Error(codes.InvalidArgument, "invalid UUID format")
	}
	return nil
}

func validatePagination(page, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func validateDateRange(start, end *timestamppb.Timestamp) error {
	if start != nil && end != nil {
		if start.AsTime().After(end.AsTime()) {
			return status.Error(codes.InvalidArgument, "start_date must be before end_date")
		}
	}
	return nil
}

// uc_03_01_05: Get dashboard with monitor statuses
func (s *DashboardServiceServer) GetDashboard(ctx context.Context, req *dashboardproto.GetDashboardRequest) (*dashboardproto.GetDashboardResponse, error) {
	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	if req.PageSize > 200 {
		return nil, status.Error(codes.InvalidArgument, "page_size exceeds maximum of 200")
	}
	if req.Search != "" && len(req.Search) > 200 {
		return nil, status.Error(codes.InvalidArgument, "search query exceeds maximum length")
	}

	page, pageSize := validatePagination(req.Page, req.PageSize)

	filter := protoToDashboardFilter(req)
	filter.Page = int(page)
	filter.PageSize = int(pageSize)

	views, total, uptime, err := s.dashboardService.GetDashboard(ctx, userID.String(), filter)
	if err != nil {
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}
	totalCount, err := safeInt32(total, "total")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoViews := make([]*dashboardproto.MonitorStatusView, 0, len(views))
	for _, v := range views {
		pv, err := monitorStatusToProto(v)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert monitor status")
		}
		protoViews = append(protoViews, pv)
	}

	return &dashboardproto.GetDashboardResponse{
		Monitors:                protoViews,
		Total:                   totalCount,
		Page:                    page,
		PageSize:                pageSize,
		OverallUptimePercentage: uptime,
	}, nil
}

// uc_03_01_17d: Export dashboard data
func (s *DashboardServiceServer) ExportDashboard(ctx context.Context, req *dashboardproto.ExportDashboardRequest) (*dashboardproto.ExportResponse, error) {
	userID := auth.ExtractUserID(ctx)
	if userID == uuid.Nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	format := req.Format
	if format == "" {
		format = "csv"
	}
	switch format {
	case "csv", "json":
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported export format")
	}

	filter := protoToExportDashboardFilter(req)

	downloadURL, err := s.dashboardService.ExportDashboard(ctx, userID.String(), filter, format)
	if err != nil {
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}

	return &dashboardproto.ExportResponse{
		DownloadUrl:   downloadURL,
		Format:        format,
		ExpiresAtUnix: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// uc_03_02_01: Get check history for a monitor
func (s *DashboardServiceServer) GetCheckHistory(ctx context.Context, req *dashboardproto.GetCheckHistoryRequest) (*dashboardproto.GetCheckHistoryResponse, error) {
	if req.MonitorId == "" {
		return nil, status.Error(codes.InvalidArgument, "monitor_id is required")
	}
	if err := validateUUID(req.MonitorId); err != nil {
		return nil, err
	}

	if err := validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	page, pageSize := validatePagination(req.Page, req.PageSize)

	filter := protoToHistoryFilter(req)
	filter.Page = int(page)
	filter.PageSize = int(pageSize)

	entries, total, err := s.historyService.GetCheckHistory(ctx, req.MonitorId, filter)
	if err != nil {
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}
	totalCount, err := safeInt32(total, "total")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoEntries := make([]*dashboardproto.CheckHistoryEntry, 0, len(entries))
	for _, e := range entries {
		pe, err := checkHistoryToProto(e)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert check history")
		}
		protoEntries = append(protoEntries, pe)
	}

	return &dashboardproto.GetCheckHistoryResponse{
		Checks:   protoEntries,
		Total:    totalCount,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// uc_03_02_06: Get incidents for a monitor
func (s *DashboardServiceServer) GetIncidents(ctx context.Context, req *dashboardproto.GetIncidentsRequest) (*dashboardproto.GetIncidentsResponse, error) {
	if req.MonitorId == "" {
		return nil, status.Error(codes.InvalidArgument, "monitor_id is required")
	}
	if err := validateUUID(req.MonitorId); err != nil {
		return nil, err
	}

	page, pageSize := validatePagination(req.Page, req.PageSize)

	filter := protoToIncidentFilter(req)
	filter.Page = int(page)
	filter.PageSize = int(pageSize)

	incidents, total, err := s.historyService.GetIncidents(ctx, req.MonitorId, filter)
	if err != nil {
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}
	totalCount, err := safeInt32(total, "total")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoIncidents := make([]*dashboardproto.Incident, 0, len(incidents))
	for _, inc := range incidents {
		pi, err := incidentToProto(inc)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert incident")
		}
		protoIncidents = append(protoIncidents, pi)
	}

	return &dashboardproto.GetIncidentsResponse{
		Incidents: protoIncidents,
		Total:     totalCount,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// uc_03_02_07: Get incident details
func (s *DashboardServiceServer) GetIncidentDetails(ctx context.Context, req *dashboardproto.GetIncidentDetailsRequest) (*dashboardproto.IncidentDetail, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := validateUUID(req.Id); err != nil {
		return nil, err
	}

	detail, err := s.historyService.GetIncidentDetails(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return nil, status.Error(codes.NotFound, "incident not found")
		}
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}

	protoIncident, err := incidentToProto(&detail.Incident)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert incident")
	}

	protoTimeline := make([]*dashboardproto.CheckHistoryEntry, 0, len(detail.Timeline))
	for i := range detail.Timeline {
		pe, err := checkHistoryToProto(&detail.Timeline[i])
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert timeline entry")
		}
		protoTimeline = append(protoTimeline, pe)
	}

	return &dashboardproto.IncidentDetail{
		Incident: protoIncident,
		Timeline: protoTimeline,
	}, nil
}

// uc_03_02_10: Get period metrics
func (s *DashboardServiceServer) GetPeriodMetrics(ctx context.Context, req *dashboardproto.GetPeriodMetricsRequest) (*dashboardproto.PeriodMetrics, error) {
	if req.MonitorId == "" {
		return nil, status.Error(codes.InvalidArgument, "monitor_id is required")
	}
	if err := validateUUID(req.MonitorId); err != nil {
		return nil, err
	}
	if req.StartDate == nil || req.EndDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}
	if err := validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	start := req.StartDate.AsTime()
	end := req.EndDate.AsTime()

	metrics, err := s.historyService.GetPeriodMetrics(ctx, req.MonitorId, start, end)
	if err != nil {
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}

	return periodMetricsToProto(metrics), nil
}

// uc_03_02_09: Export history
func (s *DashboardServiceServer) ExportHistory(ctx context.Context, req *dashboardproto.ExportHistoryRequest) (*dashboardproto.ExportResponse, error) {
	if req.MonitorId == "" {
		return nil, status.Error(codes.InvalidArgument, "monitor_id is required")
	}
	if err := validateUUID(req.MonitorId); err != nil {
		return nil, err
	}

	allowedFields := map[string]bool{
		"timestamp": true, "status": true, "status_code": true,
		"response_time_ms": true, "error_message": true,
	}
	if len(req.Fields) > 0 {
		for _, f := range req.Fields {
			if !allowedFields[f] {
				return nil, status.Errorf(codes.InvalidArgument, "invalid field: %s", f)
			}
		}
	}

	format := req.Format
	if format == "" {
		format = "csv"
	}
	switch format {
	case "csv", "json":
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported export format")
	}

	page, pageSize := validatePagination(1, 50)

	filter := protoToExportHistoryFilter(req)
	filter.Page = int(page)
	filter.PageSize = int(pageSize)

	downloadURL, err := s.historyService.ExportHistory(ctx, req.MonitorId, filter, format, req.Fields)
	if err != nil {
		return nil, status.Error(domainErrorToGRPCCode(err), err.Error())
	}

	return &dashboardproto.ExportResponse{
		DownloadUrl:   downloadURL,
		Format:        format,
		ExpiresAtUnix: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

func domainErrorToGRPCCode(err error) codes.Code {
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Code() {
		case "INVALID_FILTER_PARAMETER", "INVALID_PAGE_SIZE", "INVALID_DATE_RANGE",
			"INVALID_SEARCH_QUERY", "SEARCH_QUERY_TOO_LONG", "FILTER_TOO_LONG":
			return codes.InvalidArgument
		case "MONITOR_NOT_FOUND", "INCIDENT_NOT_FOUND":
			return codes.NotFound
		case "EXPORT_FAILED":
			return codes.Internal
		}
	}
	return codes.Internal
}
