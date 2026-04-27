package grpc

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/service"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestMonitorHandler_UpdateMonitor_WithFallbackUserID тестирует UpdateMonitor с fallback userID.
// extractUserID имеет fallback, поэтому пустой контекст не дает ошибку.
func TestMonitorHandler_UpdateMonitor_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.UpdateMonitorRequest{Id: uuid.New().String()}
	mockMonitorService.On("UpdateMonitor", mock.Anything, mock.Anything).Return((*dto.MonitorResponse)(nil), assert.AnError)

	resp, err := handler.UpdateMonitor(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_DeleteMonitor_WithFallbackUserID тестирует DeleteMonitor с fallback userID.
func TestMonitorHandler_DeleteMonitor_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.DeleteMonitorRequest{Id: uuid.New().String()}
	mockMonitorService.On("DeleteMonitor", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

	resp, err := handler.DeleteMonitor(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_PauseMonitor_WithFallbackUserID тестирует PauseMonitor с fallback userID.
func TestMonitorHandler_PauseMonitor_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.PauseMonitorRequest{Id: uuid.New().String()}
	mockMonitorService.On("PauseMonitor", mock.Anything, mock.Anything).Return(assert.AnError)

	resp, err := handler.PauseMonitor(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_ResumeMonitor_WithFallbackUserID тестирует ResumeMonitor с fallback userID.
func TestMonitorHandler_ResumeMonitor_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.ResumeMonitorRequest{Id: uuid.New().String()}
	mockMonitorService.On("ResumeMonitor", mock.Anything, mock.Anything).Return(assert.AnError)

	resp, err := handler.ResumeMonitor(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_GetMonitorHistory_WithFallbackUserID тестирует GetMonitorHistory с fallback userID.
func TestMonitorHandler_GetMonitorHistory_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockUptimeCalc := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalc,
		validator:        validator.New(),
	}

	req := &monitov1.GetMonitorHistoryRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.Now(),
		To:        timestamppb.Now(),
	}
	mockUptimeCalc.On("GetMonitorHistory", mock.Anything, mock.Anything).Return((*dto.GetMonitorHistoryResponse)(nil), assert.AnError)

	resp, err := handler.GetMonitorHistory(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetUptimeStats_WithFallbackUserID тестирует GetUptimeStats с fallback userID.
func TestMonitorHandler_GetUptimeStats_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockUptimeCalc := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalc,
		validator:        validator.New(),
	}

	req := &monitov1.GetUptimeStatsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.Now(),
		To:        timestamppb.Now(),
	}
	mockUptimeCalc.On("CalculateUptime", mock.Anything, mock.Anything).Return((*dto.UptimeStatsResponse)(nil), assert.AnError)

	resp, err := handler.GetUptimeStats(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetIncidents_WithFallbackUserID тестирует GetIncidents с fallback userID.
func TestMonitorHandler_GetIncidents_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockIncidentDetector := new(service.MockIncidentDetector)
	handler := &MonitorHandler{
		incidentDetector: mockIncidentDetector,
		validator:        validator.New(),
	}

	req := &monitov1.GetIncidentsRequest{MonitorId: uuid.New().String()}
	mockIncidentDetector.On("GetIncidents", mock.Anything, mock.Anything).Return((*dto.GetIncidentsResponse)(nil), assert.AnError)

	resp, err := handler.GetIncidents(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockIncidentDetector.AssertExpectations(t)
}

// TestMonitorHandler_GetCheckResults_WithFallbackUserID тестирует GetCheckResults с fallback userID и дефолтным лимитом.
func TestMonitorHandler_GetCheckResults_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockUptimeCalc := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalc,
		validator:        validator.New(),
	}

	req := &monitov1.GetCheckResultsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.Now(),
		To:        timestamppb.Now(),
		Limit:     0, // должен стать 100
	}

	mockUptimeCalc.On("GetCheckResults", mock.Anything, mock.MatchedBy(func(r *dto.GetCheckResultsRequest) bool {
		return r.Limit == 100
	})).Return(&dto.GetCheckResultsResponse{}, nil)

	resp, err := handler.GetCheckResults(authedCtx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_ListMonitors_WithFallbackUserID тестирует ListMonitors с fallback userID.
func TestMonitorHandler_ListMonitors_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.ListMonitorsRequest{Limit: 10}
	mockMonitorService.On("ListMonitors", mock.Anything, mock.Anything).Return((*dto.ListMonitorsResponse)(nil), assert.AnError)

	resp, err := handler.ListMonitors(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_GetMonitor_WithFallbackUserID тестирует GetMonitor с fallback userID.
func TestMonitorHandler_GetMonitor_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.GetMonitorRequest{Id: uuid.New().String()}
	mockMonitorService.On("GetMonitor", mock.Anything, mock.Anything, mock.Anything).Return((*dto.MonitorResponse)(nil), assert.AnError)

	resp, err := handler.GetMonitor(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_CreateMonitor_WithFallbackUserID тестирует CreateMonitor с fallback userID.
func TestMonitorHandler_CreateMonitor_WithFallbackUserID(t *testing.T) {
	t.Parallel()

	authedCtx := createTestContext()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	req := &monitov1.CreateMonitorRequest{
		Name:      "test",
		Url:       "https://example.com",
		CheckType: "http",
	}
	mockMonitorService.On("CreateMonitor", mock.Anything, mock.Anything).Return((*dto.MonitorResponse)(nil), assert.AnError)

	resp, err := handler.CreateMonitor(authedCtx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_GetCheckResults_ServiceError тестирует GetCheckResults с ошибкой сервиса.
func TestMonitorHandler_GetCheckResults_ServiceErr(t *testing.T) {
	t.Parallel()

	mockUptimeCalc := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalc,
		validator:        validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.GetCheckResultsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.Now(),
		To:        timestamppb.Now(),
		Limit:     10,
	}

	mockUptimeCalc.On("GetCheckResults", mock.Anything, mock.Anything).Return((*dto.GetCheckResultsResponse)(nil), assert.AnError)

	resp, err := handler.GetCheckResults(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}
