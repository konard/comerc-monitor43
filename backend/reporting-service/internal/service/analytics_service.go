package service

import (
	"context"
	"log/slog"
	"math"
	"sort"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/monitor_client"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

// AnalyticsService реализует бизнес-логику аналитики мониторинга.
type AnalyticsService struct {
	monitorClient monitor_client.MonitorClient
	logger        *slog.Logger
	maxExportRows int
}

// NewAnalyticsService создаёт новый AnalyticsService.
func NewAnalyticsService(monitorClient monitor_client.MonitorClient, logger *slog.Logger, maxExportRows int) AnalyticsServiceInterface {
	return &AnalyticsService{
		monitorClient: monitorClient,
		logger:        logger,
		maxExportRows: maxExportRows,
	}
}

func (s *AnalyticsService) GetResponseTimeMetrics(ctx context.Context, req *dto.GetResponseTimeMetricsRequest) (*dto.ResponseTimeMetricsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "AnalyticsService.GetResponseTimeMetrics")
	defer span.End()

	s.logger.InfoContext(ctx, "getting response time metrics", "monitor_id", req.MonitorID)

	checkResultsResp, err := s.monitorClient.GetCheckResults(ctx, req.MonitorID, req.From, req.To, s.maxExportRows, 0)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get check results")
	}

	var responseTimes []float64
	minRT := math.MaxInt32
	maxRT := 0
	sum := 0
	count := 0

	for _, cr := range checkResultsResp.Results {
		if cr.StatusCode >= 200 && cr.StatusCode < 300 && cr.ResponseTimeMs > 0 {
			responseTimes = append(responseTimes, float64(cr.ResponseTimeMs))
			rt := int(cr.ResponseTimeMs)
			if rt < minRT {
				minRT = rt
			}
			if rt > maxRT {
				maxRT = rt
			}
			sum += rt
			count++
		}
	}

	p50, p95, p99 := 0, 0, 0
	if len(responseTimes) > 0 {
		sort.Float64s(responseTimes)
		p50 = int(responseTimes[calculatePercentileIndex(len(responseTimes), 50)])
		p95 = int(responseTimes[calculatePercentileIndex(len(responseTimes), 95)])
		p99 = int(responseTimes[calculatePercentileIndex(len(responseTimes), 99)])
	}

	avg := 0
	if count > 0 {
		avg = sum / count
	}
	if minRT == math.MaxInt32 {
		minRT = 0
	}

	tracing.SetSuccess(span)
	return &dto.ResponseTimeMetricsResponse{
		MonitorID:   req.MonitorID,
		P50:         p50,
		P95:         p95,
		P99:         p99,
		Average:     avg,
		Min:         minRT,
		Max:         maxRT,
		TotalChecks: count,
	}, nil
}

func (s *AnalyticsService) GetIncidentsReport(ctx context.Context, req *dto.GetIncidentsReportRequest) (*dto.GetIncidentsReportResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "AnalyticsService.GetIncidentsReport")
	defer span.End()

	s.logger.InfoContext(ctx, "getting incidents report", "monitor_id", req.MonitorID)

	resp, err := s.monitorClient.GetIncidents(ctx, req.MonitorID, req.From, req.To, req.Limit, req.Offset)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get incidents")
	}

	incidents := make([]*dto.IncidentReportResponse, 0, len(resp.Incidents))
	for _, inc := range resp.Incidents {
		if req.Status != "" && inc.Status != req.Status {
			continue
		}
		ir := &dto.IncidentReportResponse{
			ID:              inc.Id,
			MonitorID:       inc.MonitorId,
			StartTime:       inc.StartTime.AsTime(),
			DurationSeconds: int64(inc.DurationSeconds),
			Status:          inc.Status,
		}
		if inc.EndTime != nil {
			endTime := inc.EndTime.AsTime()
			ir.EndTime = &endTime
		}
		incidents = append(incidents, ir)
	}

	tracing.SetSuccess(span)
	return &dto.GetIncidentsReportResponse{
		Incidents: incidents,
		Total:     int(resp.Total),
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}

func (s *AnalyticsService) GetCheckCountByStatus(ctx context.Context, req *dto.GetCheckCountByStatusRequest) (*dto.GetCheckCountByStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "AnalyticsService.GetCheckCountByStatus")
	defer span.End()

	s.logger.InfoContext(ctx, "getting check count by status", "monitor_id", req.MonitorID)

	resp, err := s.monitorClient.GetCheckResults(ctx, req.MonitorID, req.From, req.To, s.maxExportRows, 0)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, errors.Wrap(err, "failed to get check results")
	}

	counts := map[string]int{}
	total := 0
	for _, cr := range resp.Results {
		code := cr.StatusCode
		switch {
		case code >= 200 && code < 300:
			counts["2xx"]++
		case code >= 300 && code < 400:
			counts["3xx"]++
		case code >= 400 && code < 500:
			counts["4xx"]++
		case code >= 500:
			counts["5xx"]++
		default:
			counts["other"]++
		}
		total++
	}

	statusCounts := make([]*dto.StatusCount, 0, len(counts))
	for _, group := range []string{"2xx", "3xx", "4xx", "5xx", "other"} {
		if c, ok := counts[group]; ok && c > 0 {
			statusCounts = append(statusCounts, &dto.StatusCount{
				StatusGroup: group,
				Count:       c,
			})
		}
	}

	tracing.SetSuccess(span)
	return &dto.GetCheckCountByStatusResponse{
		MonitorID:    req.MonitorID,
		StatusCounts: statusCounts,
		Total:        total,
	}, nil
}

func calculatePercentileIndex(length, percentile int) int {
	idx := float64(percentile) / 100.0 * float64(length-1)
	if idx < 0 {
		idx = 0
	}
	return int(idx)
}
