package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	maintenancepb "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestMaintenanceHandler_HandleError_AllCases тестирует все ветки handleError.
func TestMaintenanceHandler_HandleError_AllCases(t *testing.T) {
	t.Parallel()

	handler := NewMaintenanceHandler(nil)

	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: codes.OK,
		},
		{
			name:         "insufficient permissions",
			err:          errors.New("INSUFFICIENT_PERMISSIONS: user is not admin"),
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "duration exceeds maximum",
			err:          errors.New("DURATION_EXCEEDS_MAXIMUM: window too long"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "duration below minimum",
			err:          errors.New("DURATION_BELOW_MINIMUM: window too short"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "end time before start time",
			err:          errors.New("END_TIME_BEFORE_START_TIME: invalid period"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "cannot create in past",
			err:          errors.New("CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST: start time is past"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "overlapping windows",
			err:          errors.New("OVERLAPPING_MAINTENANCE_WINDOWS: conflicts detected"),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "window limit reached",
			err:          errors.New("MAINTENANCE_WINDOW_LIMIT_REACHED: quota exceeded"),
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "monitor limit per window",
			err:          errors.New("MONITOR_LIMIT_PER_WINDOW_REACHED: too many monitors"),
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "not found",
			err:          errors.New("window not found in repository"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "conflict",
			err:          errors.New("version CONFLICT detected"),
			expectedCode: codes.Aborted,
		},
		{
			name:         "internal error",
			err:          errors.New("unexpected internal error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.handleError(tt.err)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedCode, status.Code(result))
		})
	}
}

// TestMapStatusToProto_AllStatuses тестирует все значения статусов.
func TestMapStatusToProto_AllStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected maintenancepb.MaintenanceWindowStatus
	}{
		{"SCHEDULED", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED},
		{"ACTIVE", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE},
		{"COMPLETED", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED},
		{"CANCELLED", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED},
		{"ORPHANED", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED},
		{"UNKNOWN", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED},
		{"", maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run("status_"+tt.input, func(t *testing.T) {
			result := mapStatusToProto(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMapRecurrenceToProto_AllTypes тестирует все типы повторяемости.
func TestMapRecurrenceToProto_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected maintenancepb.RecurrenceType
	}{
		{"ONCE", maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE},
		{"DAILY", maintenancepb.RecurrenceType_RECURRENCE_TYPE_DAILY},
		{"WEEKLY", maintenancepb.RecurrenceType_RECURRENCE_TYPE_WEEKLY},
		{"MONTHLY", maintenancepb.RecurrenceType_RECURRENCE_TYPE_MONTHLY},
		{"UNKNOWN", maintenancepb.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED},
		{"", maintenancepb.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run("recurrence_"+tt.input, func(t *testing.T) {
			result := mapRecurrenceToProto(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestContainsHelper тестирует вспомогательную функцию containsHelper.
func TestContainsHelper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "CONFLICT", "CONFLICT", true},
		{"substring at start", "CONFLICT detected", "CONFLICT", true},
		{"substring at end", "version CONFLICT", "CONFLICT", true},
		{"substring in middle", "has CONFLICT here", "CONFLICT", true},
		{"no match", "everything is fine", "CONFLICT", false},
		{"empty substr in non-empty", "test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHelper(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMaintenanceHandler_DtoToProto_WithDates тестирует конвертацию dto с датами activated/completed.
func TestMaintenanceHandler_DtoToProto_WithDates(t *testing.T) {
	t.Parallel()

	handler := NewMaintenanceHandler(nil)

	activatedAt := time.Now().Add(-1 * time.Hour)
	completedAt := time.Now()

	dtoResp := &dto.MaintenanceWindowResponse{
		ID:          uuid.New().String(),
		UserID:      uuid.New().String(),
		Name:        "Completed Window",
		StartTime:   time.Now().Add(-2 * time.Hour),
		EndTime:     time.Now().Add(-30 * time.Minute),
		Status:      "COMPLETED",
		Recurrence:  "ONCE",
		ActivatedAt: &activatedAt,
		CompletedAt: &completedAt,
		Version:     2,
	}

	proto := handler.dtoToProto(dtoResp)

	assert.NotNil(t, proto)
	assert.NotNil(t, proto.ActivatedAt)
	assert.NotNil(t, proto.CompletedAt)
	assert.Equal(t, int32(2), proto.Version)
	assert.Equal(t, maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED, proto.Status)
}

// TestMaintenanceHandler_DtoToProto_WithoutDates тестирует конвертацию dto без дат activated/completed.
func TestMaintenanceHandler_DtoToProto_WithoutDates(t *testing.T) {
	t.Parallel()

	handler := NewMaintenanceHandler(nil)

	dtoResp := &dto.MaintenanceWindowResponse{
		ID:          uuid.New().String(),
		UserID:      uuid.New().String(),
		Name:        "Scheduled Window",
		StartTime:   time.Now().Add(1 * time.Hour),
		EndTime:     time.Now().Add(2 * time.Hour),
		Status:      "SCHEDULED",
		Recurrence:  "WEEKLY",
		ActivatedAt: nil,
		CompletedAt: nil,
		Version:     1,
	}

	proto := handler.dtoToProto(dtoResp)

	assert.NotNil(t, proto)
	assert.Nil(t, proto.ActivatedAt)
	assert.Nil(t, proto.CompletedAt)
	assert.Equal(t, maintenancepb.RecurrenceType_RECURRENCE_TYPE_WEEKLY, proto.Recurrence)
}

// TestMaintenanceHandler_CreateMaintenanceWindow_ServiceError тестирует ошибку сервиса при создании.
func TestMaintenanceHandler_CreateMaintenanceWindow_ServiceError(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:      "Test Maintenance",
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	mockService.On("CreateMaintenanceWindow", mock.Anything, mock.Anything).Return((*dto.MaintenanceWindowResponse)(nil), errors.New("DURATION_EXCEEDS_MAXIMUM"))

	resp, err := handler.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_UpdateMaintenanceWindow_ServiceError тестирует ошибку сервиса при обновлении.
func TestMaintenanceHandler_UpdateMaintenanceWindow_ServiceError(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &maintenancepb.UpdateMaintenanceWindowRequest{
		Id:        uuid.New().String(),
		Name:      "Updated Maintenance",
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
		Version:   1,
	}

	mockService.On("UpdateMaintenanceWindow", mock.Anything, mock.Anything).Return((*dto.MaintenanceWindowResponse)(nil), errors.New("OVERLAPPING_MAINTENANCE_WINDOWS"))

	resp, err := handler.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_ListMaintenanceWindows_ServiceError тестирует ошибку сервиса при листинге.
func TestMaintenanceHandler_ListMaintenanceWindows_ServiceError(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.ListMaintenanceWindowsRequest{
		Page:     1,
		PageSize: 10,
	}

	mockService.On("ListMaintenanceWindows", mock.Anything, mock.Anything).Return((*dto.ListMaintenanceWindowsResponse)(nil), errors.New("internal error"))

	resp, err := handler.ListMaintenanceWindows(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_DeleteMaintenanceWindow_ServiceError тестирует ошибку сервиса при удалении.
func TestMaintenanceHandler_DeleteMaintenanceWindow_ServiceError(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	windowID := uuid.New()

	req := &maintenancepb.DeleteMaintenanceWindowRequest{
		Id: windowID.String(),
	}

	mockService.On("DeleteMaintenanceWindow", mock.Anything, windowID.String(), mock.AnythingOfType("string")).Return(errors.New("not found"))

	_, err := handler.DeleteMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_CancelMaintenanceWindow_ServiceError тестирует ошибку сервиса при отмене.
func TestMaintenanceHandler_CancelMaintenanceWindow_ServiceError(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	windowID := uuid.New()

	req := &maintenancepb.CancelMaintenanceWindowRequest{
		Id:                 windowID.String(),
		CancellationReason: "Test reason",
	}

	mockService.On("CancelMaintenanceWindow", mock.Anything, mock.Anything).Return((*dto.MaintenanceWindowResponse)(nil), errors.New("INSUFFICIENT_PERMISSIONS"))

	resp, err := handler.CancelMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_GetMaintenanceWindowHistory_ServiceError тестирует ошибку сервиса при получении истории.
func TestMaintenanceHandler_GetMaintenanceWindowHistory_ServiceError(t *testing.T) {
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

	mockService.On("GetMaintenanceWindowHistory", mock.Anything, mock.Anything).Return((*dto.ListMaintenanceWindowsResponse)(nil), errors.New("MAINTENANCE_WINDOW_LIMIT_REACHED"))

	resp, err := handler.GetMaintenanceWindowHistory(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_ListMaintenanceWindows_WithStatusFilter тестирует фильтрацию по статусу.
func TestMaintenanceHandler_ListMaintenanceWindows_WithStatusFilter(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	req := &maintenancepb.ListMaintenanceWindowsRequest{
		Status:   maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE,
		Page:     1,
		PageSize: 10,
	}

	expectedResp := &dto.ListMaintenanceWindowsResponse{
		Windows: []*dto.MaintenanceWindowResponse{
			{
				ID:     uuid.New().String(),
				Name:   "Active Window",
				Status: "ACTIVE",
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 10,
	}

	mockService.On("ListMaintenanceWindows", mock.Anything, mock.MatchedBy(func(req *dto.ListMaintenanceWindowsRequest) bool {
		return req.UserID == userID && req.Status == "ACTIVE" && req.Page == 1 && req.PageSize == 10
	})).Return(expectedResp, nil)

	resp, err := handler.ListMaintenanceWindows(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Windows, 1)

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_CreateMaintenanceWindow_ExtractRoleError тестирует ошибку при извлечении role.
func TestMaintenanceHandler_CreateMaintenanceWindow_ExtractRoleError(t *testing.T) {
	t.Parallel()

	// Контекст с userID но без role
	ctx := middleware.ContextWithAuth(context.Background(), &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "", // пустая role — ExtractUserRole вернёт ошибку
		Tier:   "Free",
	})
	handler := NewMaintenanceHandler(nil)

	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:      "Test",
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	resp, err := handler.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceHandler_CreateMaintenanceWindow_ExtractTierError тестирует ошибку при извлечении tier.
func TestMaintenanceHandler_CreateMaintenanceWindow_ExtractTierError(t *testing.T) {
	t.Parallel()

	// Контекст с userID и role, но без tier
	ctx := middleware.ContextWithAuth(context.Background(), &middleware.AuthClaims{
		UserID: uuid.New().String(),
		Role:   "USER",
		Tier:   "", // пустой tier — ExtractUserTier вернёт ошибку
	})
	handler := NewMaintenanceHandler(nil)

	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:      "Test",
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	resp, err := handler.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceHandler_ListMaintenanceWindows_WithDateFilters тестирует листинг с датами.
func TestMaintenanceHandler_ListMaintenanceWindows_WithDateFilters(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	req := &maintenancepb.ListMaintenanceWindowsRequest{
		Page:      1,
		PageSize:  10,
		StartDate: timestamppb.New(startDate),
		EndDate:   timestamppb.New(endDate),
	}

	expectedResp := &dto.ListMaintenanceWindowsResponse{
		Windows:  []*dto.MaintenanceWindowResponse{},
		Total:    0,
		Page:     1,
		PageSize: 10,
	}

	mockService.On("ListMaintenanceWindows", mock.Anything, mock.AnythingOfType("*dto.ListMaintenanceWindowsRequest")).Return(expectedResp, nil)

	resp, err := handler.ListMaintenanceWindows(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(0), resp.Total)

	mockService.AssertExpectations(t)
}

// TestMaintenanceHandler_CreateMaintenanceWindow_WithActivatedAt тестирует создание с ActivatedAt.
func TestMaintenanceHandler_CreateMaintenanceWindow_WithActivatedAt(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	ctx := contextWithAuth(userID, "USER", "Free")
	mockService := new(MockMaintenanceService)
	handler := NewMaintenanceHandler(mockService)

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	activatedAt := time.Now().Add(-1 * time.Hour)

	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:      "Active Window",
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	expectedResp := &dto.MaintenanceWindowResponse{
		ID:          uuid.New().String(),
		Name:        "Active Window",
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      "ACTIVE",
		Recurrence:  "ONCE",
		ActivatedAt: &activatedAt,
		Version:     1,
	}

	mockService.On("CreateMaintenanceWindow", mock.Anything, mock.AnythingOfType("*dto.CreateMaintenanceWindowRequest")).Return(expectedResp, nil)

	resp, err := handler.CreateMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.ActivatedAt)
	assert.Nil(t, resp.CompletedAt)

	mockService.AssertExpectations(t)
}
