package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/service"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestMonitorHandler_ListMonitors_DefaultLimit тестирует дефолтный лимит при Limit=0.
func TestMonitorHandler_ListMonitors_DefaultLimit(t *testing.T) {
	t.Parallel()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.ListMonitorsRequest{
		Limit:  0, // должен применить дефолтный лимит 100
		Offset: 0,
	}

	mockMonitorService.On("ListMonitors", mock.Anything, mock.MatchedBy(func(r *dto.ListMonitorsRequest) bool {
		return r.Limit == 100
	})).Return(&dto.ListMonitorsResponse{
		Monitors: []*dto.MonitorResponse{},
		Total:    0,
		Limit:    100,
		Offset:   0,
	}, nil)

	resp, err := handler.ListMonitors(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_ListMonitors_ServiceError тестирует ошибку сервиса при листинге.
func TestMonitorHandler_ListMonitors_ServiceError(t *testing.T) {
	t.Parallel()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.ListMonitorsRequest{
		Limit:  10,
		Offset: 0,
	}

	mockMonitorService.On("ListMonitors", mock.Anything, mock.Anything).Return((*dto.ListMonitorsResponse)(nil), errors.New("service error"))

	resp, err := handler.ListMonitors(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_GetMonitorHistory_DefaultLimit тестирует дефолтный лимит в GetMonitorHistory.
func TestMonitorHandler_GetMonitorHistory_DefaultLimit(t *testing.T) {
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

	req := &monitov1.GetMonitorHistoryRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     0, // дефолтный лимит
	}

	mockUptimeCalc.On("GetMonitorHistory", mock.Anything, mock.MatchedBy(func(r *dto.GetMonitorHistoryRequest) bool {
		return r.Limit == 100
	})).Return(&dto.GetMonitorHistoryResponse{
		Results: []*dto.CheckResultResponse{},
		Total:   0,
		Limit:   100,
		Offset:  0,
	}, nil)

	resp, err := handler.GetMonitorHistory(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetMonitorHistory_ServiceError тестирует ошибку сервиса при GetMonitorHistory.
func TestMonitorHandler_GetMonitorHistory_ServiceError(t *testing.T) {
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

	req := &monitov1.GetMonitorHistoryRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     50,
	}

	mockUptimeCalc.On("GetMonitorHistory", mock.Anything, mock.Anything).Return((*dto.GetMonitorHistoryResponse)(nil), errors.New("service error"))

	resp, err := handler.GetMonitorHistory(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetUptimeStats_Success тестирует успешный вызов GetUptimeStats.
func TestMonitorHandler_GetUptimeStats_Success(t *testing.T) {
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

	req := &monitov1.GetUptimeStatsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
	}

	mockUptimeCalc.On("CalculateUptime", mock.Anything, mock.AnythingOfType("*dto.GetUptimeStatsRequest")).Return(&dto.UptimeStatsResponse{
		Uptime:      99.5,
		TotalChecks: 100,
		UpChecks:    99,
	}, nil)

	resp, err := handler.GetUptimeStats(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 99.5, resp.Uptime)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetUptimeStats_ServiceError тестирует ошибку сервиса при GetUptimeStats.
func TestMonitorHandler_GetUptimeStats_ServiceError(t *testing.T) {
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

	req := &monitov1.GetUptimeStatsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
	}

	mockUptimeCalc.On("CalculateUptime", mock.Anything, mock.Anything).Return((*dto.UptimeStatsResponse)(nil), errors.New("service error"))

	resp, err := handler.GetUptimeStats(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetIncidents_Success тестирует успешный вызов GetIncidents.
func TestMonitorHandler_GetIncidents_Success(t *testing.T) {
	t.Parallel()

	mockIncidentDetector := new(service.MockIncidentDetector)
	handler := &MonitorHandler{
		incidentDetector: mockIncidentDetector,
		validator:        validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	monitorID := uuid.New().String()

	req := &monitov1.GetIncidentsRequest{
		MonitorId: monitorID,
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     10,
		Offset:    0,
	}

	mockIncidentDetector.On("GetIncidents", mock.Anything, mock.AnythingOfType("*dto.GetIncidentsRequest")).Return(&dto.GetIncidentsResponse{
		Incidents: []*dto.IncidentResponse{},
		Total:     0,
		Limit:     10,
		Offset:    0,
	}, nil)

	resp, err := handler.GetIncidents(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Incidents)
	mockIncidentDetector.AssertExpectations(t)
}

// TestMonitorHandler_GetIncidents_DefaultLimit тестирует дефолтный лимит в GetIncidents.
func TestMonitorHandler_GetIncidents_DefaultLimit(t *testing.T) {
	t.Parallel()

	mockIncidentDetector := new(service.MockIncidentDetector)
	handler := &MonitorHandler{
		incidentDetector: mockIncidentDetector,
		validator:        validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.GetIncidentsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     0, // дефолтный лимит 100
	}

	mockIncidentDetector.On("GetIncidents", mock.Anything, mock.MatchedBy(func(r *dto.GetIncidentsRequest) bool {
		return r.Limit == 100
	})).Return(&dto.GetIncidentsResponse{
		Incidents: []*dto.IncidentResponse{},
		Total:     0,
		Limit:     100,
		Offset:    0,
	}, nil)

	resp, err := handler.GetIncidents(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	mockIncidentDetector.AssertExpectations(t)
}

// TestMonitorHandler_GetIncidents_ServiceError тестирует ошибку сервиса при GetIncidents.
func TestMonitorHandler_GetIncidents_ServiceError(t *testing.T) {
	t.Parallel()

	mockIncidentDetector := new(service.MockIncidentDetector)
	handler := &MonitorHandler{
		incidentDetector: mockIncidentDetector,
		validator:        validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.GetIncidentsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     10,
	}

	mockIncidentDetector.On("GetIncidents", mock.Anything, mock.Anything).Return((*dto.GetIncidentsResponse)(nil), errors.New("service error"))

	resp, err := handler.GetIncidents(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockIncidentDetector.AssertExpectations(t)
}

// TestMonitorHandler_CreateMonitor_Success тестирует успешное создание монитора.
func TestMonitorHandler_CreateMonitor_Success(t *testing.T) {
	t.Parallel()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.CreateMonitorRequest{
		Name:            "Test Monitor",
		Url:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutSeconds:  30,
	}

	expectedResp := &dto.MonitorResponse{
		ID:              uuid.New().String(),
		Name:            "Test Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutSeconds:  30,
		Status:          "UP",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mockMonitorService.On("CreateMonitor", mock.Anything, mock.AnythingOfType("*dto.CreateMonitorRequest")).Return(expectedResp, nil)

	resp, err := handler.CreateMonitor(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Monitor)
	assert.Equal(t, "Test Monitor", resp.Monitor.Name)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_CreateMonitor_ServiceError тестирует ошибку сервиса при создании.
func TestMonitorHandler_CreateMonitor_ServiceError(t *testing.T) {
	t.Parallel()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.CreateMonitorRequest{
		Name:            "Test Monitor",
		Url:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutSeconds:  30,
	}

	mockMonitorService.On("CreateMonitor", mock.Anything, mock.Anything).Return((*dto.MonitorResponse)(nil), errors.New("service error"))

	resp, err := handler.CreateMonitor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_GetMonitor_Success тестирует успешное получение монитора.
func TestMonitorHandler_GetMonitor_Success(t *testing.T) {
	t.Parallel()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	monitorID := uuid.New().String()

	req := &monitov1.GetMonitorRequest{
		Id: monitorID,
	}

	expectedResp := &dto.MonitorResponse{
		ID:              monitorID,
		Name:            "Test Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
		TimeoutSeconds:  30,
		Status:          "UP",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mockMonitorService.On("GetMonitor", mock.Anything, monitorID, mock.AnythingOfType("string")).Return(expectedResp, nil)

	resp, err := handler.GetMonitor(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Monitor)
	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_GetMonitor_ServiceError тестирует ошибку сервиса при получении монитора.
func TestMonitorHandler_GetMonitor_ServiceError(t *testing.T) {
	t.Parallel()

	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	claims := &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "Free",
	}
	ctx := middleware.ContextWithAuth(context.Background(), claims)

	req := &monitov1.GetMonitorRequest{
		Id: uuid.New().String(),
	}

	mockMonitorService.On("GetMonitor", mock.Anything, mock.Anything, mock.Anything).Return((*dto.MonitorResponse)(nil), errors.New("service error"))

	resp, err := handler.GetMonitor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockMonitorService.AssertExpectations(t)
}
