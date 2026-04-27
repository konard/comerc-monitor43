package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/monitor-service/internal/infrastructure/tracing"
	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// UptimeCalculator рассчитывает статистику uptime.
type UptimeCalculator struct {
	monitorRepo     interfaces.MonitorRepository
	checkResultRepo interfaces.CheckResultRepository
	incidentRepo    interfaces.IncidentRepository
}

// NewUptimeCalculator создаёт новый UptimeCalculator.
func NewUptimeCalculator(
	monitorRepo interfaces.MonitorRepository,
	checkResultRepo interfaces.CheckResultRepository,
	incidentRepo interfaces.IncidentRepository,
) *UptimeCalculator {
	return &UptimeCalculator{
		monitorRepo:     monitorRepo,
		checkResultRepo: checkResultRepo,
		incidentRepo:    incidentRepo,
	}
}

// CalculateUptime рассчитывает uptime за период.
// PAUSED проверки исключаются из расчёта согласно требованиям.
func (c *UptimeCalculator) CalculateUptime(ctx context.Context, req *dto.GetUptimeStatsRequest) (*dto.UptimeStatsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "UptimeCalculator.CalculateUptime")
	defer span.End()

	tracing.AddEvent(ctx, "calculate_uptime.start", map[string]any{
		"monitor_id":  req.MonitorID,
		"period_from": req.From.String(),
		"period_to":   req.To.String(),
	})

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

	// Проверяем, что монитор принадлежит пользователю
	monitor, err := c.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get monitor")
	}
	if monitor.UserID != userID {
		tracing.RecordError(span, errors.New("monitor not found"))
		return nil, errors.New("monitor not found")
	}

	// Получаем результаты проверок за период
	results, err := c.checkResultRepo.GetByMonitorIDAndPeriod(ctx, monitorID, req.From, req.To)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get check results")
	}

	// Получаем инциденты за период
	incidents, err := c.incidentRepo.GetByMonitorIDAndPeriod(ctx, monitorID, req.From, req.To)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get incidents")
	}

	// Рассчитываем статистику
	stats := domain.NewUptimeStats()
	// Convert []*domain.CheckResult to []domain.CheckResult
	resultSlice := make([]domain.CheckResult, len(results))
	for i, r := range results {
		resultSlice[i] = *r
	}
	stats.Calculate(resultSlice)
	stats.CalculatePercentiles(resultSlice)
	// Convert []*domain.Incident to []domain.Incident
	incidentSlice := make([]domain.Incident, len(incidents))
	for i, inc := range incidents {
		incidentSlice[i] = *inc
	}
	stats.AddIncidents(incidentSlice)

	tracing.SetSuccess(span)
	return c.statsToResponse(stats), nil
}

// GetMonitorHistory возвращает историю проверок.
func (c *UptimeCalculator) GetMonitorHistory(ctx context.Context, req *dto.GetMonitorHistoryRequest) (*dto.GetMonitorHistoryResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "UptimeCalculator.GetMonitorHistory")
	defer span.End()

	tracing.AddEvent(ctx, "get_monitor_history.start", map[string]any{
		"monitor_id": req.MonitorID,
	})

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

	if err := validateDateRange(req.From, req.To); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	monitor, err := c.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get monitor")
	}
	if monitor.UserID != userID {
		tracing.RecordError(span, errors.New("monitor not found"))
		return nil, errors.New("monitor not found")
	}

	var results []*domain.CheckResult
	if req.Status != "" {
		status := domain.MonitorStatus(req.Status)
		if !status.IsValid() {
			return nil, errors.New("invalid status filter")
		}
		results, err = c.checkResultRepo.GetByMonitorIDAndPeriodAndStatus(ctx, monitorID, req.From, req.To, status, req.Limit, req.Offset)
	} else {
		results, err = c.checkResultRepo.GetByMonitorIDAndPeriod(ctx, monitorID, req.From, req.To)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get check results")
	}

	resultResponses := make([]*dto.CheckResultResponse, len(results))
	for i, r := range results {
		resultResponses[i] = c.checkResultToResponse(r)
	}

	total, err := c.checkResultRepo.CountByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count check results")
	}

	return &dto.GetMonitorHistoryResponse{
		Results: resultResponses,
		Total:   int(total),
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

func validateDateRange(from, to time.Time) error {
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return errors.New("invalid date range: from must be before to")
	}
	return nil
}

// GetCheckResults возвращает raw check results за период с пагинацией.
func (c *UptimeCalculator) GetCheckResults(ctx context.Context, req *dto.GetCheckResultsRequest) (*dto.GetCheckResultsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "UptimeCalculator.GetCheckResults")
	defer span.End()

	tracing.AddEvent(ctx, "get_check_results.start", map[string]any{
		"monitor_id": req.MonitorID,
	})

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

	if err := validateDateRange(req.From, req.To); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	monitor, err := c.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get monitor")
	}
	if monitor.UserID != userID {
		tracing.RecordError(span, errors.New("monitor not found"))
		return nil, errors.New("monitor not found")
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}

	results, err := c.checkResultRepo.GetByMonitorIDAndPeriodPaginated(ctx, monitorID, req.From, req.To, req.Limit, req.Offset)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get check results")
	}

	resultResponses := make([]*dto.CheckResultResponse, len(results))
	for i, r := range results {
		resultResponses[i] = c.checkResultToResponse(r)
	}

	total, err := c.checkResultRepo.CountByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count check results")
	}

	tracing.SetSuccess(span)
	return &dto.GetCheckResultsResponse{
		Results: resultResponses,
		Total:   int(total),
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// statsToResponse конвертирует domain.UptimeStats в dto.UptimeStatsResponse.
func (c *UptimeCalculator) statsToResponse(stats *domain.UptimeStats) *dto.UptimeStatsResponse {
	return &dto.UptimeStatsResponse{
		Uptime:              stats.Uptime,
		TotalChecks:         stats.TotalChecks,
		UpChecks:            stats.UpChecks,
		DegradedChecks:      stats.DegradedChecks,
		DownChecks:          stats.DownChecks,
		PausedChecks:        stats.PausedChecks,
		TotalDowntime:       int64(stats.TotalDowntime.Seconds()),
		AverageResponseTime: stats.AverageResponseTime,
		Incidents:           stats.Incidents,
		Note:                stats.Note,
		P50:                 stats.P50,
		P95:                 stats.P95,
		P99:                 stats.P99,
	}
}

// checkResultToResponse конвертирует domain.CheckResult в dto.CheckResultResponse.
func (c *UptimeCalculator) checkResultToResponse(r *domain.CheckResult) *dto.CheckResultResponse {
	return &dto.CheckResultResponse{
		ID:             r.ID.String(),
		MonitorID:      r.MonitorID.String(),
		Status:         string(r.Status),
		ResponseTimeMs: r.ResponseTimeMs,
		StatusCode:     r.StatusCode,
		ErrorMessage:   r.ErrorMessage,
		CheckedAt:      r.CheckedAt,
		CreatedAt:      r.CreatedAt,
	}
}
