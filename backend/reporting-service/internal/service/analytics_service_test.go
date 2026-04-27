package service

import (
	"context"
	"testing"
	"time"

	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

func TestNewAnalyticsService(t *testing.T) {
	t.Parallel()

	mc := new(MockMonitorClient)
	logger := testLogger()

	svc := NewAnalyticsService(mc, logger, 100000)

	assert.NotNil(t, svc)
}

func TestGetResponseTimeMetrics(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor-id"
	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC()

	t.Run("success_with_results", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		now := timestamppb.New(time.Now().UTC())
		results := []*monitov1.CheckResult{
			{Id: "1", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 100, Status: "UP", CheckedAt: now},
			{Id: "2", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 200, Status: "UP", CheckedAt: now},
			{Id: "3", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 300, Status: "UP", CheckedAt: now},
			{Id: "4", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 400, Status: "UP", CheckedAt: now},
			{Id: "5", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 500, Status: "UP", CheckedAt: now},
			{Id: "6", MonitorId: monitorID, StatusCode: 503, ResponseTimeMs: 50, Status: "DOWN", CheckedAt: now},
			{Id: "7", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 0, Status: "UP", CheckedAt: now},
		}

		mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
			Results: results,
			Total:   7,
		}, nil)

		svc := NewAnalyticsService(mc, logger, 100000)
		resp, err := svc.GetResponseTimeMetrics(context.Background(), &dto.GetResponseTimeMetricsRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		require.NoError(t, err)
		assert.Equal(t, monitorID, resp.MonitorID)
		assert.Equal(t, 5, resp.TotalChecks)
		assert.Equal(t, 100, resp.Min)
		assert.Equal(t, 500, resp.Max)
		assert.Equal(t, 300, resp.Average)
		assert.Greater(t, resp.P50, 0)
		assert.Greater(t, resp.P95, 0)
		assert.Greater(t, resp.P99, 0)
		mc.AssertExpectations(t)
	})

	t.Run("success_empty_results", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
			Results: nil,
			Total:   0,
		}, nil)

		svc := NewAnalyticsService(mc, logger, 100000)
		resp, err := svc.GetResponseTimeMetrics(context.Background(), &dto.GetResponseTimeMetricsRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, resp.TotalChecks)
		assert.Equal(t, 0, resp.Min)
		assert.Equal(t, 0, resp.Max)
		assert.Equal(t, 0, resp.Average)
		mc.AssertExpectations(t)
	})

	t.Run("error-monitor-client-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(nil, assert.AnError)

		svc := NewAnalyticsService(mc, logger, 100000)
		_, err := svc.GetResponseTimeMetrics(context.Background(), &dto.GetResponseTimeMetricsRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get check results")
		mc.AssertExpectations(t)
	})
}

func TestGetIncidentsReport(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor-id"
	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		startTime := timestamppb.New(time.Now().UTC().Add(-2 * time.Hour))
		endTime := timestamppb.New(time.Now().UTC().Add(-1 * time.Hour))

		incidents := []*monitov1.Incident{
			{Id: "inc-1", MonitorId: monitorID, StartTime: startTime, EndTime: endTime, DurationSeconds: 3600, Status: "resolved"},
			{Id: "inc-2", MonitorId: monitorID, StartTime: startTime, EndTime: nil, DurationSeconds: 7200, Status: "active"},
		}

		mc.On("GetIncidents", mock.Anything, monitorID, from, to, 10, 0).Once().Return(&monitov1.GetIncidentsResponse{
			Incidents: incidents,
			Total:     2,
		}, nil)

		svc := NewAnalyticsService(mc, logger, 100000)
		resp, err := svc.GetIncidentsReport(context.Background(), &dto.GetIncidentsReportRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
			Limit:     10,
			Offset:    0,
		})

		require.NoError(t, err)
		assert.Len(t, resp.Incidents, 2)
		assert.Equal(t, 2, resp.Total)
		assert.NotNil(t, resp.Incidents[0].EndTime)
		assert.Nil(t, resp.Incidents[1].EndTime)
		mc.AssertExpectations(t)
	})

	t.Run("success_with_status_filter", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		startTime := timestamppb.New(time.Now().UTC().Add(-2 * time.Hour))

		incidents := []*monitov1.Incident{
			{Id: "inc-1", MonitorId: monitorID, StartTime: startTime, DurationSeconds: 3600, Status: "resolved"},
			{Id: "inc-2", MonitorId: monitorID, StartTime: startTime, DurationSeconds: 7200, Status: "active"},
		}

		mc.On("GetIncidents", mock.Anything, monitorID, from, to, 10, 0).Once().Return(&monitov1.GetIncidentsResponse{
			Incidents: incidents,
			Total:     2,
		}, nil)

		svc := NewAnalyticsService(mc, logger, 100000)
		resp, err := svc.GetIncidentsReport(context.Background(), &dto.GetIncidentsReportRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
			Status:    "active",
			Limit:     10,
			Offset:    0,
		})

		require.NoError(t, err)
		assert.Len(t, resp.Incidents, 1)
		assert.Equal(t, "active", resp.Incidents[0].Status)
		mc.AssertExpectations(t)
	})

	t.Run("error-monitor-client-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		mc.On("GetIncidents", mock.Anything, monitorID, from, to, 10, 0).Once().Return(nil, assert.AnError)

		svc := NewAnalyticsService(mc, logger, 100000)
		_, err := svc.GetIncidentsReport(context.Background(), &dto.GetIncidentsReportRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
			Limit:     10,
			Offset:    0,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get incidents")
		mc.AssertExpectations(t)
	})
}

func TestGetCheckCountByStatus(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor-id"
	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC()

	t.Run("success_mixed_status_codes", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		now := timestamppb.New(time.Now().UTC())
		results := []*monitov1.CheckResult{
			{StatusCode: 200, Status: "UP", CheckedAt: now},
			{StatusCode: 201, Status: "UP", CheckedAt: now},
			{StatusCode: 301, Status: "UP", CheckedAt: now},
			{StatusCode: 404, Status: "DOWN", CheckedAt: now},
			{StatusCode: 500, Status: "DOWN", CheckedAt: now},
			{StatusCode: 503, Status: "DOWN", CheckedAt: now},
		}

		mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
			Results: results,
			Total:   6,
		}, nil)

		svc := NewAnalyticsService(mc, logger, 100000)
		resp, err := svc.GetCheckCountByStatus(context.Background(), &dto.GetCheckCountByStatusRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		require.NoError(t, err)
		assert.Equal(t, 6, resp.Total)
		countMap := make(map[string]int)
		for _, sc := range resp.StatusCounts {
			countMap[sc.StatusGroup] = sc.Count
		}
		assert.Equal(t, 2, countMap["2xx"])
		assert.Equal(t, 1, countMap["3xx"])
		assert.Equal(t, 1, countMap["4xx"])
		assert.Equal(t, 2, countMap["5xx"])
		mc.AssertExpectations(t)
	})

	t.Run("success_empty_results", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
			Results: nil,
			Total:   0,
		}, nil)

		svc := NewAnalyticsService(mc, logger, 100000)
		resp, err := svc.GetCheckCountByStatus(context.Background(), &dto.GetCheckCountByStatusRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.StatusCounts)
		mc.AssertExpectations(t)
	})

	t.Run("error-monitor-client-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		logger := testLogger()

		mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(nil, assert.AnError)

		svc := NewAnalyticsService(mc, logger, 100000)
		_, err := svc.GetCheckCountByStatus(context.Background(), &dto.GetCheckCountByStatusRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get check results")
		mc.AssertExpectations(t)
	})
}
