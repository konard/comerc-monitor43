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

// TestMonitorHandler_GetCheckResults_Success тестирует успешный вызов GetCheckResults.
func TestMonitorHandler_GetCheckResults_Success(t *testing.T) {
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

	monitorID := uuid.New().String()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	req := &monitov1.GetCheckResultsRequest{
		MonitorId: monitorID,
		From:      timestamppb.New(from),
		To:        timestamppb.New(to),
		Limit:     50,
		Offset:    0,
	}

	expectedResp := &dto.GetCheckResultsResponse{
		Results: []*dto.CheckResultResponse{
			{
				ID:        uuid.New().String(),
				MonitorID: monitorID,
				Status:    "UP",
				CheckedAt: time.Now(),
				CreatedAt: time.Now(),
			},
		},
		Total:  1,
		Limit:  50,
		Offset: 0,
	}

	mockUptimeCalc.On("GetCheckResults", mock.Anything, mock.AnythingOfType("*dto.GetCheckResultsRequest")).Return(expectedResp, nil)

	resp, err := handler.GetCheckResults(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, int32(50), resp.Limit)
	assert.Equal(t, int32(0), resp.Offset)

	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetCheckResults_DefaultLimit тестирует применение дефолтного лимита.
func TestMonitorHandler_GetCheckResults_DefaultLimit(t *testing.T) {
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

	monitorID := uuid.New().String()

	req := &monitov1.GetCheckResultsRequest{
		MonitorId: monitorID,
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     0, // Дефолтный лимит должен быть применён
	}

	mockUptimeCalc.On("GetCheckResults", mock.Anything, mock.MatchedBy(func(r *dto.GetCheckResultsRequest) bool {
		return r.Limit == 100
	})).Return(&dto.GetCheckResultsResponse{
		Results: []*dto.CheckResultResponse{},
		Total:   0,
		Limit:   100,
		Offset:  0,
	}, nil)

	resp, err := handler.GetCheckResults(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetCheckResults_FallbackUserID тестирует что handler использует fallback userID при отсутствии аутентификации.
func TestMonitorHandler_GetCheckResults_FallbackUserID(t *testing.T) {
	t.Parallel()

	mockUptimeCalc := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalc,
		validator:        validator.New(),
	}

	req := &monitov1.GetCheckResultsRequest{
		MonitorId: uuid.New().String(),
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     50,
	}

	mockUptimeCalc.On("GetCheckResults", mock.Anything, mock.MatchedBy(func(r *dto.GetCheckResultsRequest) bool {
		return r.UserID == "00000000-0000-0000-0000-000000000001"
	})).Return(&dto.GetCheckResultsResponse{
		Results: []*dto.CheckResultResponse{},
		Total:   0,
		Limit:   50,
		Offset:  0,
	}, nil)

	resp, err := handler.GetCheckResults(createTestContext(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)

	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_GetCheckResults_ServiceError тестирует ошибку сервиса.
func TestMonitorHandler_GetCheckResults_ServiceError(t *testing.T) {
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
		From:      timestamppb.New(time.Now().Add(-24 * time.Hour)),
		To:        timestamppb.New(time.Now()),
		Limit:     50,
	}

	mockUptimeCalc.On("GetCheckResults", mock.Anything, mock.Anything).Return((*dto.GetCheckResultsResponse)(nil), errors.New("service error"))

	resp, err := handler.GetCheckResults(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	mockUptimeCalc.AssertExpectations(t)
}

// TestMonitorHandler_PauseMonitor_Success тестирует успешную паузу монитора.
func TestMonitorHandler_PauseMonitor_Success(t *testing.T) {
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

	req := &monitov1.PauseMonitorRequest{
		Id: uuid.New().String(),
	}

	mockMonitorService.On("PauseMonitor", mock.Anything, mock.AnythingOfType("*dto.PauseMonitorRequest")).Return(nil)

	resp, err := handler.PauseMonitor(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_PauseMonitor_ServiceError тестирует ошибку сервиса при паузе.
func TestMonitorHandler_PauseMonitor_ServiceError(t *testing.T) {
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

	req := &monitov1.PauseMonitorRequest{
		Id: uuid.New().String(),
	}

	mockMonitorService.On("PauseMonitor", mock.Anything, mock.Anything).Return(errors.New("service error"))

	resp, err := handler.PauseMonitor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_ResumeMonitor_Success тестирует успешное возобновление монитора.
func TestMonitorHandler_ResumeMonitor_Success(t *testing.T) {
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

	req := &monitov1.ResumeMonitorRequest{
		Id: uuid.New().String(),
	}

	mockMonitorService.On("ResumeMonitor", mock.Anything, mock.AnythingOfType("*dto.ResumeMonitorRequest")).Return(nil)

	resp, err := handler.ResumeMonitor(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_ResumeMonitor_ServiceError тестирует ошибку сервиса при возобновлении.
func TestMonitorHandler_ResumeMonitor_ServiceError(t *testing.T) {
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

	req := &monitov1.ResumeMonitorRequest{
		Id: uuid.New().String(),
	}

	mockMonitorService.On("ResumeMonitor", mock.Anything, mock.Anything).Return(errors.New("service error"))

	resp, err := handler.ResumeMonitor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_DeleteMonitor_Success тестирует успешное удаление монитора.
func TestMonitorHandler_DeleteMonitor_Success(t *testing.T) {
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

	req := &monitov1.DeleteMonitorRequest{
		Id: monitorID,
	}

	mockMonitorService.On("DeleteMonitor", mock.Anything, monitorID, mock.AnythingOfType("string")).Return(nil)

	resp, err := handler.DeleteMonitor(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_DeleteMonitor_ServiceError тестирует ошибку сервиса при удалении.
func TestMonitorHandler_DeleteMonitor_ServiceError(t *testing.T) {
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

	req := &monitov1.DeleteMonitorRequest{
		Id: uuid.New().String(),
	}

	mockMonitorService.On("DeleteMonitor", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("service error"))

	resp, err := handler.DeleteMonitor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_UpdateMonitor_Success тестирует успешное обновление монитора.
func TestMonitorHandler_UpdateMonitor_Success(t *testing.T) {
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

	req := &monitov1.UpdateMonitorRequest{
		Id:              monitorID,
		Name:            "Updated Monitor",
		Url:             "https://updated.com",
		IntervalSeconds: 120,
		TimeoutSeconds:  60,
	}

	expectedResp := &dto.MonitorResponse{
		ID:              monitorID,
		Name:            "Updated Monitor",
		URL:             "https://updated.com",
		IntervalSeconds: 120,
		TimeoutSeconds:  60,
		Status:          "UP",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mockMonitorService.On("UpdateMonitor", mock.Anything, mock.AnythingOfType("*dto.UpdateMonitorRequest")).Return(expectedResp, nil)

	resp, err := handler.UpdateMonitor(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Monitor)

	mockMonitorService.AssertExpectations(t)
}

// TestMonitorHandler_UpdateMonitor_ServiceError тестирует ошибку сервиса при обновлении.
func TestMonitorHandler_UpdateMonitor_ServiceError(t *testing.T) {
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

	req := &monitov1.UpdateMonitorRequest{
		Id:              uuid.New().String(),
		Name:            "Updated Monitor",
		Url:             "https://updated.com",
		IntervalSeconds: 120,
		TimeoutSeconds:  60,
	}

	mockMonitorService.On("UpdateMonitor", mock.Anything, mock.Anything).Return((*dto.MonitorResponse)(nil), errors.New("service error"))

	resp, err := handler.UpdateMonitor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	mockMonitorService.AssertExpectations(t)
}
