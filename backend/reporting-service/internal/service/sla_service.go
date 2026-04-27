package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/monitor_client"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

// SLAService реализует бизнес-логику генерации и получения SLA-отчётов.
type SLAService struct {
	monitorClient monitor_client.MonitorClient
	reportRepo    interfaces.ReportRepository
	logger        *slog.Logger
	retentionDays int
}

// NewSLAService создаёт новый SLAService.
func NewSLAService(
	monitorClient monitor_client.MonitorClient,
	reportRepo interfaces.ReportRepository,
	logger *slog.Logger,
	retentionDays int,
) SLAServiceInterface {
	return &SLAService{
		monitorClient: monitorClient,
		reportRepo:    reportRepo,
		logger:        logger,
		retentionDays: retentionDays,
	}
}

func (s *SLAService) GenerateSLAReport(ctx context.Context, req *dto.GenerateSLAReportRequest) (*dto.SLAReportResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "SLAService.GenerateSLAReport")
	defer span.End()

	s.logger.InfoContext(ctx, "generating sla report", "monitor_id", req.MonitorID)

	if err := s.validatePeriod(req.From, req.To, s.retentionDays); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	monitorID, err := uuid.Parse(req.MonitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid monitor_id")
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid user_id")
	}

	monitorResp, err := s.monitorClient.GetMonitor(ctx, req.MonitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get monitor")
	}

	uptimeResp, err := s.monitorClient.GetUptimeStats(ctx, req.MonitorID, req.From, req.To)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get uptime stats")
	}

	report := model.NewSLAReport(monitorID, userID, monitorResp.Monitor.Name, req.From, req.To)
	report.Availability = uptimeResp.Uptime
	report.TotalChecks = int(uptimeResp.TotalChecks)
	report.UpChecks = int(uptimeResp.UpChecks)
	report.DownChecks = int(uptimeResp.DownChecks)
	report.DegradedChecks = int(uptimeResp.DegradedChecks)
	report.PausedChecks = int(uptimeResp.PausedChecks)
	report.TotalDowntimeSeconds = uptimeResp.TotalDowntime
	report.IncidentsCount = int(uptimeResp.Incidents)

	if err := s.reportRepo.Create(ctx, report); err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to save sla report")
	}

	tracing.SetSuccess(span)
	return s.convertModelToResponse(report), nil
}

func (s *SLAService) GetSLAReport(ctx context.Context, reportID, userID string) (*dto.SLAReportResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "SLAService.GetSLAReport")
	defer span.End()

	s.logger.InfoContext(ctx, "getting sla report", "report_id", reportID)

	id, err := uuid.Parse(reportID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "invalid report_id")
	}

	report, err := s.reportRepo.GetByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	if report.UserID.String() != userID {
		tracing.RecordError(span, model.ErrUnauthorizedAccess)
		return nil, model.ErrUnauthorizedAccess
	}

	tracing.SetSuccess(span)
	return s.convertModelToResponse(report), nil
}

func (s *SLAService) ListSLAReports(ctx context.Context, req *dto.ListSLAReportsRequest) (*dto.ListSLAReportsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "SLAService.ListSLAReports")
	defer span.End()

	s.logger.InfoContext(ctx, "listing sla reports", "monitor_id", req.MonitorID, "user_id", req.UserID)

	// Если monitor_id не задан, возвращаем отчёты пользователя без фильтра по монитору
	var (
		reports []*model.SLAReport
		total   int
		err     error
	)
	if req.MonitorID == "" {
		userID, parseErr := uuid.Parse(req.UserID)
		if parseErr != nil {
			tracing.RecordError(span, parseErr)
			return nil, errors.Wrap(parseErr, "invalid user_id")
		}
		reports, total, err = s.reportRepo.ListByUserID(ctx, userID, req.Limit, req.Offset)
	} else {
		monitorID, parseErr := uuid.Parse(req.MonitorID)
		if parseErr != nil {
			tracing.RecordError(span, parseErr)
			return nil, errors.Wrap(parseErr, "invalid monitor_id")
		}
		reports, total, err = s.reportRepo.ListByMonitorID(ctx, monitorID, req.Limit, req.Offset)
	}
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to list sla reports")
	}

	respReports := make([]*dto.SLAReportResponse, len(reports))
	for i, r := range reports {
		respReports[i] = s.convertModelToResponse(r)
	}

	tracing.SetSuccess(span)
	return &dto.ListSLAReportsResponse{
		Reports: respReports,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

func (s *SLAService) validatePeriod(from, to time.Time, retentionDays int) error {
	if from.IsZero() || to.IsZero() {
		return model.ErrInvalidDateRange
	}
	if to.Before(from) {
		return model.ErrInvalidDateRange
	}
	now := time.Now().UTC()
	if from.After(now) {
		return model.ErrFuturePeriod
	}
	maxDate := from.AddDate(0, 0, retentionDays)
	if to.After(maxDate) {
		return model.ErrPeriodExceedsRetention
	}
	return nil
}

func (s *SLAService) convertModelToResponse(r *model.SLAReport) *dto.SLAReportResponse {
	return &dto.SLAReportResponse{
		ID:                   r.ID.String(),
		MonitorID:            r.MonitorID.String(),
		MonitorName:          r.MonitorName,
		PeriodStart:          r.PeriodStart,
		PeriodEnd:            r.PeriodEnd,
		Availability:         r.Availability,
		TotalChecks:          r.TotalChecks,
		UpChecks:             r.UpChecks,
		DownChecks:           r.DownChecks,
		DegradedChecks:       r.DegradedChecks,
		PausedChecks:         r.PausedChecks,
		TotalDowntimeSeconds: r.TotalDowntimeSeconds,
		IncidentsCount:       r.IncidentsCount,
		CreatedAt:            r.CreatedAt,
	}
}
