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

// TestGetCheckCountByStatus_OtherStatusGroup проверяет статус-коды ниже 200 (группа "other").
func TestGetCheckCountByStatus_OtherStatusGroup(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor-id"
	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC()

	mc := new(MockMonitorClient)
	logger := testLogger()

	now := timestamppb.New(time.Now().UTC())
	results := []*monitov1.CheckResult{
		{StatusCode: 0, Status: "UNKNOWN", CheckedAt: now},   // other (< 200)
		{StatusCode: 100, Status: "UNKNOWN", CheckedAt: now}, // other (< 200)
		{StatusCode: 200, Status: "UP", CheckedAt: now},      // 2xx
	}

	mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
		Results: results,
		Total:   3,
	}, nil)

	svc := NewAnalyticsService(mc, logger, 100000)
	resp, err := svc.GetCheckCountByStatus(context.Background(), &dto.GetCheckCountByStatusRequest{
		MonitorID: monitorID,
		From:      from,
		To:        to,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)

	countMap := make(map[string]int)
	for _, sc := range resp.StatusCounts {
		countMap[sc.StatusGroup] = sc.Count
	}
	assert.Equal(t, 2, countMap["other"])
	assert.Equal(t, 1, countMap["2xx"])
	mc.AssertExpectations(t)
}

// TestCalculatePercentileIndex проверяет пограничный случай с единственным элементом.
func TestCalculatePercentileIndex_SingleElement(t *testing.T) {
	t.Parallel()

	// при length=1 idx будет 0, но проверка idx < 0 покрывается через float расчёт
	idx := calculatePercentileIndex(1, 99)
	assert.Equal(t, 0, idx)
}

// TestCalculatePercentileIndex_NegativeResult проверяет, что idx не уходит ниже 0.
func TestCalculatePercentileIndex_ZeroLength(t *testing.T) {
	t.Parallel()

	// при length=0 вычисление даёт -0.99, что должно быть скорректировано до 0
	idx := calculatePercentileIndex(0, 99)
	assert.Equal(t, 0, idx)
}

// TestGetResponseTimeMetrics_SingleResult проверяет метрики с единственным результатом.
func TestGetResponseTimeMetrics_SingleResult(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor"
	from := time.Now().UTC().AddDate(0, 0, -1)
	to := time.Now().UTC()

	mc := new(MockMonitorClient)
	logger := testLogger()

	now := timestamppb.New(time.Now().UTC())
	results := []*monitov1.CheckResult{
		{StatusCode: 200, ResponseTimeMs: 150, Status: "UP", CheckedAt: now},
	}

	mc.On("GetCheckResults", mock.Anything, monitorID, from, to, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
		Results: results,
		Total:   1,
	}, nil)

	svc := NewAnalyticsService(mc, logger, 100000)
	resp, err := svc.GetResponseTimeMetrics(context.Background(), &dto.GetResponseTimeMetricsRequest{
		MonitorID: monitorID,
		From:      from,
		To:        to,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalChecks)
	assert.Equal(t, 150, resp.Min)
	assert.Equal(t, 150, resp.Max)
	assert.Equal(t, 150, resp.Average)
	assert.Equal(t, 150, resp.P50)
	assert.Equal(t, 150, resp.P95)
	assert.Equal(t, 150, resp.P99)
	mc.AssertExpectations(t)
}
