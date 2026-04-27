package handler

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/reporting-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/service"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

// epic=06_reporting, us=06_01_sla_reports
//
// epic=06_reporting, us=06_02_analytics
type ReportingHandler struct {
	reportingv1.UnimplementedReportingServiceServer

	slaService       service.SLAServiceInterface
	analyticsService service.AnalyticsServiceInterface
	exportService    service.ExportServiceInterface
	logger           *slog.Logger
}

// NewReportingHandler создаёт новый ReportingHandler.
func NewReportingHandler(
	slaService service.SLAServiceInterface,
	analyticsService service.AnalyticsServiceInterface,
	exportService service.ExportServiceInterface,
	logger *slog.Logger,
) *ReportingHandler {
	return &ReportingHandler{
		slaService:       slaService,
		analyticsService: analyticsService,
		exportService:    exportService,
		logger:           logger,
	}
}

// uc_06_01_01: Generate SLA report
func (h *ReportingHandler) GenerateSLAReport(ctx context.Context, req *reportingv1.GenerateSLAReportRequest) (*reportingv1.SLAReport, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.GenerateSLAReport")
	defer span.End()

	h.logger.InfoContext(ctx, "generating sla report", "monitor_id", req.MonitorId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	slaReq := &dto.GenerateSLAReportRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
	}

	resp, err := h.slaService.GenerateSLAReport(ctx, slaReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return h.convertSLAReportToProto(resp), nil
}

// uc_06_01_04: List SLA reports
func (h *ReportingHandler) ListSLAReports(ctx context.Context, req *reportingv1.ListSLAReportsRequest) (*reportingv1.ListSLAReportsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.ListSLAReports")
	defer span.End()

	h.logger.InfoContext(ctx, "listing sla reports", "monitor_id", req.MonitorId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	listReq := &dto.ListSLAReportsRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}
	if listReq.Limit <= 0 {
		listReq.Limit = 50
	}

	resp, err := h.slaService.ListSLAReports(ctx, listReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list sla reports", "error", err, "monitor_id", req.MonitorId, "user_id", userID)
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	reports := make([]*reportingv1.SLAReport, len(resp.Reports))
	for i, r := range resp.Reports {
		reports[i] = h.convertSLAReportToProto(r)
	}

	tracing.SetSuccess(span)
	return &reportingv1.ListSLAReportsResponse{
		Reports: reports,
		Total:   int32(resp.Total),  //nolint:gosec // G115: значения из БД в допустимом диапазоне
		Limit:   int32(resp.Limit),  //nolint:gosec // G115: значения из БД в допустимом диапазоне
		Offset:  int32(resp.Offset), //nolint:gosec // G115: значения из БД в допустимом диапазоне
	}, nil
}

// uc_06_01_05: Get SLA report details
func (h *ReportingHandler) GetSLAReport(ctx context.Context, req *reportingv1.GetSLAReportRequest) (*reportingv1.SLAReport, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.GetSLAReport")
	defer span.End()

	h.logger.InfoContext(ctx, "getting sla report", "report_id", req.ReportId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	resp, err := h.slaService.GetSLAReport(ctx, req.ReportId, userID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return h.convertSLAReportToProto(resp), nil
}

// uc_06_02_03: Get response time metrics
func (h *ReportingHandler) GetResponseTimeMetrics(ctx context.Context, req *reportingv1.GetResponseTimeMetricsRequest) (*reportingv1.ResponseTimeMetricsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.GetResponseTimeMetrics")
	defer span.End()

	h.logger.InfoContext(ctx, "getting response time metrics", "monitor_id", req.MonitorId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	metricsReq := &dto.GetResponseTimeMetricsRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
	}

	resp, err := h.analyticsService.GetResponseTimeMetrics(ctx, metricsReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &reportingv1.ResponseTimeMetricsResponse{
		MonitorId:   resp.MonitorID,
		P50:         int32(resp.P50),         //nolint:gosec // G115: значения в допустимом диапазоне
		P95:         int32(resp.P95),         //nolint:gosec // G115: значения в допустимом диапазоне
		P99:         int32(resp.P99),         //nolint:gosec // G115: значения в допустимом диапазоне
		Average:     int32(resp.Average),     //nolint:gosec // G115: значения в допустимом диапазоне
		Min:         int32(resp.Min),         //nolint:gosec // G115: значения в допустимом диапазоне
		Max:         int32(resp.Max),         //nolint:gosec // G115: значения в допустимом диапазоне
		TotalChecks: int32(resp.TotalChecks), //nolint:gosec // G115: значения в допустимом диапазоне
	}, nil
}

// uc_06_02_04: Get incidents report
func (h *ReportingHandler) GetIncidentsReport(ctx context.Context, req *reportingv1.GetIncidentsReportRequest) (*reportingv1.GetIncidentsReportResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.GetIncidentsReport")
	defer span.End()

	h.logger.InfoContext(ctx, "getting incidents report", "monitor_id", req.MonitorId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	incReq := &dto.GetIncidentsReportRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
		Status:    req.Status,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}
	if incReq.Limit <= 0 {
		incReq.Limit = 100
	}

	resp, err := h.analyticsService.GetIncidentsReport(ctx, incReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	incidents := make([]*reportingv1.IncidentReport, len(resp.Incidents))
	for i, inc := range resp.Incidents {
		incidents[i] = h.convertIncidentReportToProto(inc)
	}

	tracing.SetSuccess(span)
	return &reportingv1.GetIncidentsReportResponse{
		Incidents: incidents,
		Total:     int32(resp.Total),  //nolint:gosec // G115: значения из БД в допустимом диапазоне
		Limit:     int32(resp.Limit),  //nolint:gosec // G115: значения из БД в допустимом диапазоне
		Offset:    int32(resp.Offset), //nolint:gosec // G115: значения из БД в допустимом диапазоне
	}, nil
}

// uc_06_02_01: Get check count by status
func (h *ReportingHandler) GetCheckCountByStatus(ctx context.Context, req *reportingv1.GetCheckCountByStatusRequest) (*reportingv1.GetCheckCountByStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.GetCheckCountByStatus")
	defer span.End()

	h.logger.InfoContext(ctx, "getting check count by status", "monitor_id", req.MonitorId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	countReq := &dto.GetCheckCountByStatusRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime(),
		To:        req.To.AsTime(),
	}

	resp, err := h.analyticsService.GetCheckCountByStatus(ctx, countReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	statusCounts := make([]*reportingv1.StatusCount, len(resp.StatusCounts))
	for i, sc := range resp.StatusCounts {
		statusCounts[i] = &reportingv1.StatusCount{
			StatusGroup: sc.StatusGroup,
			Count:       int32(sc.Count), //nolint:gosec // G115: значения в допустимом диапазоне
		}
	}

	tracing.SetSuccess(span)
	return &reportingv1.GetCheckCountByStatusResponse{
		MonitorId:    resp.MonitorID,
		StatusCounts: statusCounts,
		Total:        int32(resp.Total), //nolint:gosec // G115: значения в допустимом диапазоне
	}, nil
}

// uc_06_02_07: Export report to CSV
func (h *ReportingHandler) ExportReportCSV(ctx context.Context, req *reportingv1.ExportReportCSVRequest) (*reportingv1.ExportReportCSVResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.ExportReportCSV")
	defer span.End()

	h.logger.InfoContext(ctx, "exporting report csv", "monitor_id", req.MonitorId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	csvReq := &dto.ExportCSVRequest{
		MonitorID: req.MonitorId,
		UserID:    userID,
		From:      req.From.AsTime().Format("2006-01-02T15:04:05Z"),
		To:        req.To.AsTime().Format("2006-01-02T15:04:05Z"),
	}

	resp, err := h.exportService.ExportCSV(ctx, csvReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &reportingv1.ExportReportCSVResponse{
		Content:  resp.Content,
		Filename: resp.Filename,
		Rows:     int32(resp.Rows), //nolint:gosec // G115: значения в допустимом диапазоне
	}, nil
}

// uc_06_01_05: Export SLA report to PDF
func (h *ReportingHandler) ExportReportPDF(ctx context.Context, req *reportingv1.ExportReportPDFRequest) (*reportingv1.ExportReportPDFResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ReportingHandler.ExportReportPDF")
	defer span.End()

	h.logger.InfoContext(ctx, "exporting report pdf", "report_id", req.ReportId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	pdfReq := &dto.ExportPDFRequest{
		ReportID: req.ReportId,
		UserID:   userID,
	}

	resp, err := h.exportService.ExportPDF(ctx, pdfReq)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, h.handleError(err)
	}

	tracing.SetSuccess(span)
	return &reportingv1.ExportReportPDFResponse{
		Content:  resp.Content,
		Filename: resp.Filename,
	}, nil
}

func (h *ReportingHandler) handleError(err error) error {
	var reportErr *model.ReportError
	if errors.As(err, &reportErr) {
		switch reportErr.Code {
		case "INVALID_PERIOD_FORMAT", "INVALID_DATE_RANGE", "INVALID_TIMEZONE":
			return status.Error(codes.InvalidArgument, reportErr.Message)
		case "PERIOD_EXCEEDS_RETENTION":
			return status.Error(codes.InvalidArgument, reportErr.Message)
		case "REPORT_NOT_FOUND", "MONITOR_NOT_FOUND":
			return status.Error(codes.NotFound, reportErr.Message)
		case "UNAUTHORIZED_ACCESS":
			return status.Error(codes.PermissionDenied, reportErr.Message)
		case "EXPORT_SERVICE_UNAVAILABLE", "TIME_SERIES_SERVICE_UNAVAILABLE", "DATABASE_UNAVAILABLE":
			return status.Error(codes.Unavailable, reportErr.Message)
		case "QUERY_TIMEOUT", "PDF_GENERATION_TIMEOUT":
			return status.Error(codes.DeadlineExceeded, reportErr.Message)
		case "REPORT_GENERATION_IN_PROGRESS":
			return status.Error(codes.Aborted, reportErr.Message)
		}
	}
	return status.Error(codes.Internal, "internal server error")
}

func (h *ReportingHandler) convertSLAReportToProto(r *dto.SLAReportResponse) *reportingv1.SLAReport {
	return &reportingv1.SLAReport{
		Id:                   r.ID,
		MonitorId:            r.MonitorID,
		MonitorName:          r.MonitorName,
		PeriodStart:          timestamppb.New(r.PeriodStart),
		PeriodEnd:            timestamppb.New(r.PeriodEnd),
		Availability:         r.Availability,
		TotalChecks:          int32(r.TotalChecks),    //nolint:gosec // G115: значения в допустимом диапазоне
		UpChecks:             int32(r.UpChecks),       //nolint:gosec // G115: значения в допустимом диапазоне
		DownChecks:           int32(r.DownChecks),     //nolint:gosec // G115: значения в допустимом диапазоне
		DegradedChecks:       int32(r.DegradedChecks), //nolint:gosec // G115: значения в допустимом диапазоне
		PausedChecks:         int32(r.PausedChecks),   //nolint:gosec // G115: значения в допустимом диапазоне
		TotalDowntimeSeconds: r.TotalDowntimeSeconds,
		IncidentsCount:       int32(r.IncidentsCount), //nolint:gosec // G115: значения в допустимом диапазоне
		CreatedAt:            timestamppb.New(r.CreatedAt),
	}
}

func (h *ReportingHandler) convertIncidentReportToProto(r *dto.IncidentReportResponse) *reportingv1.IncidentReport {
	proto := &reportingv1.IncidentReport{
		Id:              r.ID,
		MonitorId:       r.MonitorID,
		StartTime:       timestamppb.New(r.StartTime),
		DurationSeconds: r.DurationSeconds,
		Status:          r.Status,
		FailedChecks:    int32(r.FailedChecks), //nolint:gosec // G115: значения в допустимом диапазоне
	}
	if r.EndTime != nil {
		proto.EndTime = timestamppb.New(*r.EndTime)
	}
	return proto
}

func (h *ReportingHandler) extractUserID(ctx context.Context) (string, error) {
	userID, err := middleware.ExtractUserID(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to extract user_id")
	}
	return userID, nil
}
