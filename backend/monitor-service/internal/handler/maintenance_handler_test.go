package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	maintenancepb "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// MockMaintenanceService для тестов
type MockMaintenanceService struct {
	mock.Mock
}

func (m *MockMaintenanceService) CreateMaintenanceWindow(ctx context.Context, req *dto.CreateMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MaintenanceWindowResponse), args.Error(1)
}

func (m *MockMaintenanceService) UpdateMaintenanceWindow(ctx context.Context, req *dto.UpdateMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MaintenanceWindowResponse), args.Error(1)
}

func (m *MockMaintenanceService) ListMaintenanceWindows(ctx context.Context, req *dto.ListMaintenanceWindowsRequest) (*dto.ListMaintenanceWindowsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ListMaintenanceWindowsResponse), args.Error(1)
}

func (m *MockMaintenanceService) CancelMaintenanceWindow(ctx context.Context, req *dto.CancelMaintenanceWindowRequest) (*dto.MaintenanceWindowResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MaintenanceWindowResponse), args.Error(1)
}

func (m *MockMaintenanceService) GetMaintenanceWindowHistory(ctx context.Context, req *dto.GetMaintenanceWindowHistoryRequest) (*dto.ListMaintenanceWindowsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ListMaintenanceWindowsResponse), args.Error(1)
}

func (m *MockMaintenanceService) DeleteMaintenanceWindow(ctx context.Context, windowID, userID string) error {
	args := m.Called(ctx, windowID, userID)
	return args.Error(0)
}

func (m *MockMaintenanceService) ActivateScheduledWindows(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMaintenanceService) CompleteActiveWindows(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Helper function to create context with auth data
func contextWithAuth(userID, role, tier string) context.Context {
	ctx := context.Background()
	claims := &middleware.AuthClaims{
		UserID: userID,
		Role:   role,
		Tier:   tier,
	}
	return middleware.ContextWithAuth(ctx, claims)
}

// Tests

func TestMaintenanceHandler_CreateMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:            "Test Maintenance",
		StartTime:       timestamppb.New(startTime),
		EndTime:         timestamppb.New(endTime),
		Recurrence:      maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIds:      []string{},
	}

	expectedResp := &dto.MaintenanceWindowResponse{
		ID:              uuid.New().String(),
		Name:            "Test Maintenance",
		StartTime:       startTime,
		EndTime:         endTime,
		Status:          "SCHEDULED",
		Recurrence:      "ONCE",
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{},
		Version:         1,
	}

	mockService.On("CreateMaintenanceWindow", mock.Anything, mock.AnythingOfType("*dto.CreateMaintenanceWindowRequest")).Return(expectedResp, nil)

	resp, err := handler.CreateMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test Maintenance", resp.Name)
	assert.Equal(t, maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED, resp.Status)

	mockService.AssertExpectations(t)
}

func TestMaintenanceHandler_UpdateMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startTime := time.Now().Add(3 * time.Hour)
	endTime := startTime.Add(3 * time.Hour)

	req := &maintenancepb.UpdateMaintenanceWindowRequest{
		Id:        uuid.New().String(),
		Name:      "Updated Name",
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
		Version:   1,
	}

	expectedResp := &dto.MaintenanceWindowResponse{
		ID:        req.Id,
		Name:      "Updated Name",
		StartTime: startTime,
		EndTime:   endTime,
		Status:    "SCHEDULED",
		Version:   2,
	}

	mockService.On("UpdateMaintenanceWindow", mock.Anything, mock.AnythingOfType("*dto.UpdateMaintenanceWindowRequest")).Return(expectedResp, nil)

	resp, err := handler.UpdateMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Updated Name", resp.Name)
	assert.Equal(t, int32(2), resp.Version)

	mockService.AssertExpectations(t)
}

func TestMaintenanceHandler_ListMaintenanceWindows_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.ListMaintenanceWindowsRequest{
		Page:     1,
		PageSize: 10,
	}

	expectedResp := &dto.ListMaintenanceWindowsResponse{
		Windows: []*dto.MaintenanceWindowResponse{
			{
				ID:     uuid.New().String(),
				Name:   "Window 1",
				Status: "SCHEDULED",
			},
			{
				ID:     uuid.New().String(),
				Name:   "Window 2",
				Status: "ACTIVE",
			},
		},
		Total:    2,
		Page:     1,
		PageSize: 10,
	}

	mockService.On("ListMaintenanceWindows", mock.Anything, mock.MatchedBy(func(req *dto.ListMaintenanceWindowsRequest) bool {
		return req.UserID == userID && req.Status == "" && req.Page == 1 && req.PageSize == 10
	})).Return(expectedResp, nil)

	resp, err := handler.ListMaintenanceWindows(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Windows, 2)
	assert.Equal(t, int32(2), resp.Total)

	mockService.AssertExpectations(t)
}

func TestMaintenanceHandler_CancelMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	windowID := uuid.New()

	req := &maintenancepb.CancelMaintenanceWindowRequest{
		Id:                 windowID.String(),
		CancellationReason: "Test cancellation",
	}

	expectedResp := &dto.MaintenanceWindowResponse{
		ID:     windowID.String(),
		Name:   "Test Window",
		Status: "CANCELLED",
	}

	mockService.On("CancelMaintenanceWindow", mock.Anything, mock.AnythingOfType("*dto.CancelMaintenanceWindowRequest")).Return(expectedResp, nil)

	resp, err := handler.CancelMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED, resp.Status)

	mockService.AssertExpectations(t)
}

func TestMaintenanceHandler_DeleteMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	windowID := uuid.New()

	req := &maintenancepb.DeleteMaintenanceWindowRequest{
		Id: windowID.String(),
	}

	mockService.On("DeleteMaintenanceWindow", mock.Anything, windowID.String(), mock.AnythingOfType("string")).Return(nil)

	_, err := handler.DeleteMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestMaintenanceHandler_GetMaintenanceWindowHistory_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	req := &maintenancepb.GetMaintenanceWindowHistoryRequest{
		StartDate: timestamppb.New(startDate),
		EndDate:   timestamppb.New(endDate),
		Page:      1,
		PageSize:  10,
	}

	expectedResp := &dto.ListMaintenanceWindowsResponse{
		Windows: []*dto.MaintenanceWindowResponse{
			{
				ID:     uuid.New().String(),
				Name:   "Completed Window",
				Status: "COMPLETED",
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 10,
	}

	mockService.On("GetMaintenanceWindowHistory", mock.Anything, mock.AnythingOfType("*dto.GetMaintenanceWindowHistoryRequest")).Return(expectedResp, nil)

	resp, err := handler.GetMaintenanceWindowHistory(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Windows, 1)
	assert.Equal(t, int32(1), resp.Total)

	mockService.AssertExpectations(t)
}

func TestMaintenanceHandler_HandleError_InsufficientPermissions(t *testing.T) {
	t.Parallel()
	handler := NewMaintenanceHandler(nil)

	err := handler.handleError(assert.AnError)
	assert.Contains(t, err.Error(), "internal server error")
}
