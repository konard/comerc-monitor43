package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/export"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/monitor_client"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

// ExportService реализует экспорт отчётов в форматы CSV и PDF.
type ExportService struct {
	monitorClient monitor_client.MonitorClient
	reportRepo    interfaces.ReportRepository
	csvExporter   export.CSVExporter
	pdfExporter   export.PDFExporter
	logger        *slog.Logger
	maxExportRows int
}

// NewExportService создаёт новый ExportService.
func NewExportService(
	monitorClient monitor_client.MonitorClient,
	reportRepo interfaces.ReportRepository,
	csvExporter export.CSVExporter,
	pdfExporter export.PDFExporter,
	logger *slog.Logger,
	maxExportRows int,
) ExportServiceInterface {
	return &ExportService{
		monitorClient: monitorClient,
		reportRepo:    reportRepo,
		csvExporter:   csvExporter,
		pdfExporter:   pdfExporter,
		logger:        logger,
		maxExportRows: maxExportRows,
	}
}

func (s *ExportService) ExportCSV(ctx context.Context, req *dto.ExportCSVRequest) (*dto.ExportCSVResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ExportService.ExportCSV")
	defer span.End()

	s.logger.InfoContext(ctx, "exporting csv", "monitor_id", req.MonitorID)

	from, err := time.Parse(time.RFC3339, req.From)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid from time format")
	}
	to, err := time.Parse(time.RFC3339, req.To)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid to time format")
	}

	checkResultsResp, err := s.monitorClient.GetCheckResults(ctx, req.MonitorID, from, to, s.maxExportRows, 0)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get check results for csv export")
	}

	monitorResp, err := s.monitorClient.GetMonitor(ctx, req.MonitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get monitor for csv export")
	}

	content, filename, err := s.csvExporter.ExportCheckResults(checkResultsResp.Results, monitorResp.Monitor.Name)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to generate csv")
	}

	tracing.SetSuccess(span)
	return &dto.ExportCSVResponse{
		Content:  content,
		Filename: filename,
		Rows:     len(checkResultsResp.Results),
	}, nil
}

func (s *ExportService) ExportPDF(ctx context.Context, req *dto.ExportPDFRequest) (*dto.ExportPDFResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "ExportService.ExportPDF")
	defer span.End()

	s.logger.InfoContext(ctx, "exporting pdf", "report_id", req.ReportID)

	reportID, err := uuid.Parse(req.ReportID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid report_id")
	}

	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	if report.UserID.String() != req.UserID {
		tracing.RecordError(span, model.ErrUnauthorizedAccess)
		return nil, model.ErrUnauthorizedAccess
	}

	protoReport := s.convertModelToProtoReport(report)
	content, filename, err := s.pdfExporter.ExportSLAReport(protoReport)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to generate pdf")
	}

	tracing.SetSuccess(span)
	return &dto.ExportPDFResponse{
		Content:  content,
		Filename: filename,
	}, nil
}

func (s *ExportService) convertModelToProtoReport(r *model.SLAReport) *reportingv1.SLAReport {
	return &reportingv1.SLAReport{
		Id:                   r.ID.String(),
		MonitorId:            r.MonitorID.String(),
		MonitorName:          r.MonitorName,
		PeriodStart:          timestamppb.New(r.PeriodStart),
		PeriodEnd:            timestamppb.New(r.PeriodEnd),
		Availability:         r.Availability,
		TotalChecks:          int32(r.TotalChecks),    //nolint:gosec // G115: значения из БД в допустимом диапазоне
		UpChecks:             int32(r.UpChecks),       //nolint:gosec // G115: значения из БД в допустимом диапазоне
		DownChecks:           int32(r.DownChecks),     //nolint:gosec // G115: значения из БД в допустимом диапазоне
		DegradedChecks:       int32(r.DegradedChecks), //nolint:gosec // G115: значения из БД в допустимом диапазоне
		PausedChecks:         int32(r.PausedChecks),   //nolint:gosec // G115: значения из БД в допустимом диапазоне
		TotalDowntimeSeconds: r.TotalDowntimeSeconds,
		IncidentsCount:       int32(r.IncidentsCount), //nolint:gosec // G115: значения из БД в допустимом диапазоне
		CreatedAt:            timestamppb.New(r.CreatedAt),
	}
}
