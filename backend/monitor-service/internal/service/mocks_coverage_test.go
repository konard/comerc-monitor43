package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestMockCheckResultRepository_UnusedMethods тестирует методы MockCheckResultRepository,
// которые не покрыты другими тестами.
func TestMockCheckResultRepository_UnusedMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	monitorID := uuid.New()
	resultID := uuid.New()

	t.Run("GetByID returns result", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		expected := &domain.CheckResult{ID: resultID, MonitorID: monitorID}
		m.On("GetByID", ctx, resultID).Return(expected, nil)
		result, err := m.GetByID(ctx, resultID)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		m.AssertExpectations(t)
	})

	t.Run("GetByID returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("GetByID", ctx, resultID).Return(nil, assert.AnError)
		result, err := m.GetByID(ctx, resultID)
		assert.Error(t, err)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorID returns results", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		expected := []*domain.CheckResult{{ID: resultID, MonitorID: monitorID}}
		m.On("GetByMonitorID", ctx, monitorID, 10, 0).Return(expected, nil)
		results, err := m.GetByMonitorID(ctx, monitorID, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, expected, results)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorID returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("GetByMonitorID", ctx, monitorID, 10, 0).Return(nil, assert.AnError)
		results, err := m.GetByMonitorID(ctx, monitorID, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, results)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorIDAndPeriodAndStatus returns results", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()
		expected := []*domain.CheckResult{{ID: resultID}}
		m.On("GetByMonitorIDAndPeriodAndStatus", ctx, monitorID, from, to, domain.StatusUp, 10, 0).Return(expected, nil)
		results, err := m.GetByMonitorIDAndPeriodAndStatus(ctx, monitorID, from, to, domain.StatusUp, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, expected, results)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorIDAndPeriodAndStatus returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()
		m.On("GetByMonitorIDAndPeriodAndStatus", ctx, monitorID, from, to, domain.StatusDown, 10, 0).Return(nil, assert.AnError)
		results, err := m.GetByMonitorIDAndPeriodAndStatus(ctx, monitorID, from, to, domain.StatusDown, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, results)
		m.AssertExpectations(t)
	})

	t.Run("DeleteOld returns count", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		olderThan := time.Now().Add(-30 * 24 * time.Hour)
		m.On("DeleteOld", ctx, olderThan).Return(5, nil)
		count, err := m.DeleteOld(ctx, olderThan)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
		m.AssertExpectations(t)
	})

	t.Run("DeleteOld returns error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		olderThan := time.Now().Add(-30 * 24 * time.Hour)
		m.On("DeleteOld", ctx, olderThan).Return(0, assert.AnError)
		count, err := m.DeleteOld(ctx, olderThan)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		m.AssertExpectations(t)
	})
}

// TestMockCheckResultRepository_PeriodMethods тестирует GetByMonitorIDAndPeriod и другие методы.
func TestMockCheckResultRepository_PeriodMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	t.Run("GetByMonitorIDAndPeriod returns results", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		results := []*domain.CheckResult{{MonitorID: monitorID}}
		m.On("GetByMonitorIDAndPeriod", ctx, monitorID, from, to).Return(results, nil)
		got, err := m.GetByMonitorIDAndPeriod(ctx, monitorID, from, to)
		require.NoError(t, err)
		assert.Equal(t, results, got)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorIDAndPeriod returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("GetByMonitorIDAndPeriod", ctx, monitorID, from, to).Return(nil, assert.AnError)
		got, err := m.GetByMonitorIDAndPeriod(ctx, monitorID, from, to)
		assert.Error(t, err)
		assert.Nil(t, got)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorIDAndPeriodPaginated returns results", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		results := []*domain.CheckResult{{MonitorID: monitorID}}
		m.On("GetByMonitorIDAndPeriodPaginated", ctx, monitorID, from, to, 10, 0).Return(results, nil)
		got, err := m.GetByMonitorIDAndPeriodPaginated(ctx, monitorID, from, to, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, results, got)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorIDAndPeriodPaginated returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("GetByMonitorIDAndPeriodPaginated", ctx, monitorID, from, to, 10, 0).Return(nil, assert.AnError)
		got, err := m.GetByMonitorIDAndPeriodPaginated(ctx, monitorID, from, to, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, got)
		m.AssertExpectations(t)
	})

	t.Run("GetLatestByMonitorID returns results", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		results := []*domain.CheckResult{{MonitorID: monitorID}}
		m.On("GetLatestByMonitorID", ctx, monitorID, 5).Return(results, nil)
		got, err := m.GetLatestByMonitorID(ctx, monitorID, 5)
		require.NoError(t, err)
		assert.Equal(t, results, got)
		m.AssertExpectations(t)
	})

	t.Run("GetLatestByMonitorID returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("GetLatestByMonitorID", ctx, monitorID, 5).Return(nil, assert.AnError)
		got, err := m.GetLatestByMonitorID(ctx, monitorID, 5)
		assert.Error(t, err)
		assert.Nil(t, got)
		m.AssertExpectations(t)
	})

	t.Run("CountByMonitorID returns nil path", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("CountByMonitorID", ctx, monitorID).Return(nil, assert.AnError)
		count, err := m.CountByMonitorID(ctx, monitorID)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		m.AssertExpectations(t)
	})

	t.Run("CountByMonitorID returns count", func(t *testing.T) {
		t.Parallel()
		m := new(MockCheckResultRepository)
		m.On("CountByMonitorID", ctx, monitorID).Return(int64(42), nil)
		count, err := m.CountByMonitorID(ctx, monitorID)
		require.NoError(t, err)
		assert.Equal(t, int64(42), count)
		m.AssertExpectations(t)
	})
}

// TestMockIncidentRepository_UnusedMethods тестирует методы MockIncidentRepository,
// которые не покрыты другими тестами.
func TestMockIncidentRepository_UnusedMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	monitorID := uuid.New()
	incidentID := uuid.New()

	t.Run("GetByID returns incident", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		expected := &domain.Incident{ID: incidentID, MonitorID: monitorID}
		m.On("GetByID", ctx, incidentID).Return(expected, nil)
		result, err := m.GetByID(ctx, incidentID)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		m.AssertExpectations(t)
	})

	t.Run("GetByID returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		m.On("GetByID", ctx, incidentID).Return(nil, assert.AnError)
		result, err := m.GetByID(ctx, incidentID)
		assert.Error(t, err)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})

	t.Run("Update returns nil on success", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		incident := &domain.Incident{ID: incidentID, MonitorID: monitorID}
		m.On("Update", ctx, incident).Return(nil)
		err := m.Update(ctx, incident)
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("Update returns error", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		incident := &domain.Incident{ID: incidentID}
		m.On("Update", ctx, incident).Return(assert.AnError)
		err := m.Update(ctx, incident)
		assert.Error(t, err)
		m.AssertExpectations(t)
	})

	t.Run("DeleteOld returns count", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		olderThan := time.Now().Add(-30 * 24 * time.Hour)
		m.On("DeleteOld", ctx, olderThan).Return(3, nil)
		count, err := m.DeleteOld(ctx, olderThan)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
		m.AssertExpectations(t)
	})

	t.Run("DeleteOld returns error", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		olderThan := time.Now().Add(-30 * 24 * time.Hour)
		m.On("DeleteOld", ctx, olderThan).Return(0, assert.AnError)
		count, err := m.DeleteOld(ctx, olderThan)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorID returns incidents", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		expected := []*domain.Incident{{ID: incidentID, MonitorID: monitorID}}
		m.On("GetByMonitorID", ctx, monitorID, 10, 0).Return(expected, nil)
		results, err := m.GetByMonitorID(ctx, monitorID, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, expected, results)
		m.AssertExpectations(t)
	})

	t.Run("GetByMonitorID returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		m.On("GetByMonitorID", ctx, monitorID, 10, 0).Return(nil, assert.AnError)
		results, err := m.GetByMonitorID(ctx, monitorID, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, results)
		m.AssertExpectations(t)
	})

	t.Run("CountByMonitorID returns nil path", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		m.On("CountByMonitorID", ctx, monitorID).Return(nil, assert.AnError)
		count, err := m.CountByMonitorID(ctx, monitorID)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		m.AssertExpectations(t)
	})

	t.Run("CountByMonitorID returns count", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentRepository)
		m.On("CountByMonitorID", ctx, monitorID).Return(int64(7), nil)
		count, err := m.CountByMonitorID(ctx, monitorID)
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
		m.AssertExpectations(t)
	})
}

// TestMockMonitorService_Methods тестирует все методы MockMonitorService.
func TestMockMonitorService_Methods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New().String()

	t.Run("CreateMonitor returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.CreateMonitorRequest{Name: "Test", URL: "https://example.com", UserID: userID}
		expected := &dto.MonitorResponse{ID: uuid.New().String(), Name: "Test"}
		m.On("CreateMonitor", mock.Anything, req).Return(expected, nil)
		resp, err := m.CreateMonitor(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("CreateMonitor returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.CreateMonitorRequest{}
		m.On("CreateMonitor", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.CreateMonitor(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetMonitor returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		monitorID := uuid.New().String()
		expected := &dto.MonitorResponse{ID: monitorID}
		m.On("GetMonitor", mock.Anything, monitorID, userID).Return(expected, nil)
		resp, err := m.GetMonitor(ctx, monitorID, userID)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetMonitor returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		m.On("GetMonitor", mock.Anything, "bad-id", userID).Return(nil, assert.AnError)
		resp, err := m.GetMonitor(ctx, "bad-id", userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})

	t.Run("ListMonitors returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.ListMonitorsRequest{UserID: userID, Limit: 10}
		expected := &dto.ListMonitorsResponse{Monitors: []*dto.MonitorResponse{}}
		m.On("ListMonitors", mock.Anything, req).Return(expected, nil)
		resp, err := m.ListMonitors(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("ListMonitors returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.ListMonitorsRequest{}
		m.On("ListMonitors", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.ListMonitors(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})

	t.Run("UpdateMonitor returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.UpdateMonitorRequest{ID: uuid.New().String()}
		expected := &dto.MonitorResponse{ID: req.ID}
		m.On("UpdateMonitor", mock.Anything, req).Return(expected, nil)
		resp, err := m.UpdateMonitor(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("UpdateMonitor returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.UpdateMonitorRequest{}
		m.On("UpdateMonitor", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.UpdateMonitor(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})

	t.Run("DeleteMonitor returns nil on success", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		monitorID := uuid.New().String()
		m.On("DeleteMonitor", mock.Anything, monitorID, userID).Return(nil)
		err := m.DeleteMonitor(ctx, monitorID, userID)
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("DeleteMonitor returns error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		m.On("DeleteMonitor", mock.Anything, "bad-id", userID).Return(assert.AnError)
		err := m.DeleteMonitor(ctx, "bad-id", userID)
		assert.Error(t, err)
		m.AssertExpectations(t)
	})

	t.Run("PauseMonitor returns nil on success", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.PauseMonitorRequest{ID: uuid.New().String(), UserID: userID}
		m.On("PauseMonitor", mock.Anything, req).Return(nil)
		err := m.PauseMonitor(ctx, req)
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("PauseMonitor returns error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.PauseMonitorRequest{}
		m.On("PauseMonitor", mock.Anything, req).Return(assert.AnError)
		err := m.PauseMonitor(ctx, req)
		assert.Error(t, err)
		m.AssertExpectations(t)
	})

	t.Run("ResumeMonitor returns nil on success", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.ResumeMonitorRequest{ID: uuid.New().String(), UserID: userID}
		m.On("ResumeMonitor", mock.Anything, req).Return(nil)
		err := m.ResumeMonitor(ctx, req)
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("ResumeMonitor returns error", func(t *testing.T) {
		t.Parallel()
		m := new(MockMonitorService)
		req := &dto.ResumeMonitorRequest{}
		m.On("ResumeMonitor", mock.Anything, req).Return(assert.AnError)
		err := m.ResumeMonitor(ctx, req)
		assert.Error(t, err)
		m.AssertExpectations(t)
	})
}

// TestMockUptimeCalculator_UnusedMethods тестирует методы MockUptimeCalculator,
// которые не покрыты другими тестами.
func TestMockUptimeCalculator_UnusedMethods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("CalculateUptime returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockUptimeCalculator)
		req := &dto.GetUptimeStatsRequest{MonitorID: uuid.New().String()}
		expected := &dto.UptimeStatsResponse{Uptime: 99.5}
		m.On("CalculateUptime", mock.Anything, req).Return(expected, nil)
		resp, err := m.CalculateUptime(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("CalculateUptime returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockUptimeCalculator)
		req := &dto.GetUptimeStatsRequest{}
		m.On("CalculateUptime", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.CalculateUptime(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetMonitorHistory returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockUptimeCalculator)
		req := &dto.GetMonitorHistoryRequest{MonitorID: uuid.New().String()}
		expected := &dto.GetMonitorHistoryResponse{}
		m.On("GetMonitorHistory", mock.Anything, req).Return(expected, nil)
		resp, err := m.GetMonitorHistory(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetMonitorHistory returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockUptimeCalculator)
		req := &dto.GetMonitorHistoryRequest{}
		m.On("GetMonitorHistory", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.GetMonitorHistory(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetCheckResults returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockUptimeCalculator)
		req := &dto.GetCheckResultsRequest{MonitorID: uuid.New().String()}
		expected := &dto.GetCheckResultsResponse{}
		m.On("GetCheckResults", mock.Anything, req).Return(expected, nil)
		resp, err := m.GetCheckResults(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetCheckResults returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockUptimeCalculator)
		req := &dto.GetCheckResultsRequest{}
		m.On("GetCheckResults", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.GetCheckResults(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})
}

// TestMockIncidentDetector_Methods тестирует методы MockIncidentDetector.
func TestMockIncidentDetector_Methods(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("GetIncidents returns response", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentDetector)
		req := &dto.GetIncidentsRequest{MonitorID: uuid.New().String()}
		expected := &dto.GetIncidentsResponse{}
		m.On("GetIncidents", mock.Anything, req).Return(expected, nil)
		resp, err := m.GetIncidents(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, expected, resp)
		m.AssertExpectations(t)
	})

	t.Run("GetIncidents returns nil on error", func(t *testing.T) {
		t.Parallel()
		m := new(MockIncidentDetector)
		req := &dto.GetIncidentsRequest{}
		m.On("GetIncidents", mock.Anything, req).Return(nil, assert.AnError)
		resp, err := m.GetIncidents(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		m.AssertExpectations(t)
	})
}
