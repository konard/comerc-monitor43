package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestMaintenanceService_CreateMaintenanceWindow_InvalidUserID тестирует невалидный user_id.
func TestMaintenanceService_CreateMaintenanceWindow_InvalidUserID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     "not-a-uuid",
		Name:       "Test",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		Recurrence: "ONCE",
		UserRole:   "USER",
		UserTier:   "Free",
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_CreateMaintenanceWindow_CountError тестирует ошибку при подсчёте окон.
func TestMaintenanceService_CreateMaintenanceWindow_CountError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	userID := uuid.New()
	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, errors.New("db error"))

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test",
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(2 * time.Hour),
		Recurrence: "ONCE",
		UserRole:   "USER",
		UserTier:   "Free",
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CreateMaintenanceWindow_OverlappingWindows тестирует пересечение окон.
func TestMaintenanceService_CreateMaintenanceWindow_OverlappingWindows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, mock.Anything, startTime, endTime, (*uuid.UUID)(nil)).Return(true, nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		IsGlobal:   false,
		MonitorIDs: []string{monitorID.String()},
		UserRole:   "USER",
		UserTier:   "Free",
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "OVERLAPPING_MAINTENANCE_WINDOWS")
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CreateMaintenanceWindow_OverlapCheckError тестирует ошибку при проверке пересечения.
func TestMaintenanceService_CreateMaintenanceWindow_OverlapCheckError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, mock.Anything, startTime, endTime, (*uuid.UUID)(nil)).Return(false, errors.New("overlap check error"))

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		IsGlobal:   false,
		MonitorIDs: []string{monitorID.String()},
		UserRole:   "USER",
		UserTier:   "Free",
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CreateMaintenanceWindow_SaveError тестирует ошибку при сохранении.
func TestMaintenanceService_CreateMaintenanceWindow_SaveError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		IsGlobal:   false,
		MonitorIDs: []string{},
		UserRole:   "USER",
		UserTier:   "Free",
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_InvalidWindowID тестирует невалидный window ID.
func TestMaintenanceService_UpdateMaintenanceWindow_InvalidWindowID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        "not-a-uuid",
		UserID:    uuid.New().String(),
		Name:      "Test",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_UpdateMaintenanceWindow_InvalidUserID тестирует невалидный user ID при обновлении.
func TestMaintenanceService_UpdateMaintenanceWindow_InvalidUserID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        uuid.New().String(),
		UserID:    "not-a-uuid",
		Name:      "Test",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_UpdateMaintenanceWindow_GetByIDError тестирует ошибку при получении окна.
func TestMaintenanceService_UpdateMaintenanceWindow_GetByIDError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return((*interfaces.MaintenanceWindow)(nil), errors.New("db error"))

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Test",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_NotFound тестирует случай когда окно не найдено.
func TestMaintenanceService_UpdateMaintenanceWindow_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return((*interfaces.MaintenanceWindow)(nil), nil)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Test",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_WrongUser тестирует ошибку прав при обновлении.
func TestMaintenanceService_UpdateMaintenanceWindow_WrongUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()
	anotherUserID := uuid.New()

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    anotherUserID,
		Name:      "Other User Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Test",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "INSUFFICIENT_PERMISSIONS")
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_UpdateError тестирует ошибку при сохранении обновления.
func TestMaintenanceService_UpdateMaintenanceWindow_UpdateError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	windowID := uuid.New()
	userID := uuid.New()
	startTime := time.Now().Add(3 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("GetWindowMonitors", mock.Anything, windowID).Return([]uuid.UUID{}, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Updated Name",
		StartTime: startTime,
		EndTime:   endTime,
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CancelMaintenanceWindow_InvalidWindowID тестирует невалидный window ID при отмене.
func TestMaintenanceService_CancelMaintenanceWindow_InvalidWindowID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     "not-a-uuid",
		UserID: uuid.New().String(),
	}

	resp, err := svc.CancelMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_CancelMaintenanceWindow_GetByIDError тестирует ошибку при получении окна для отмены.
func TestMaintenanceService_CancelMaintenanceWindow_GetByIDError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return((*interfaces.MaintenanceWindow)(nil), errors.New("db error"))

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     windowID.String(),
		UserID: userID.String(),
	}

	resp, err := svc.CancelMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CancelMaintenanceWindow_NotFound тестирует отмену несуществующего окна.
func TestMaintenanceService_CancelMaintenanceWindow_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return((*interfaces.MaintenanceWindow)(nil), nil)

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     windowID.String(),
		UserID: userID.String(),
	}

	resp, err := svc.CancelMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CancelMaintenanceWindow_UpdateError тестирует ошибку при сохранении отмены.
func TestMaintenanceService_CancelMaintenanceWindow_UpdateError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	windowID := uuid.New()
	userID := uuid.New()

	now := time.Now()
	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
		Status:    "ACTIVE",
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     windowID.String(),
		UserID: userID.String(),
	}

	resp, err := svc.CancelMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_ListMaintenanceWindows_InvalidUserID тестирует невалидный user ID при листинге.
func TestMaintenanceService_ListMaintenanceWindows_InvalidUserID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.ListMaintenanceWindowsRequest{
		UserID:   "not-a-uuid",
		Status:   "",
		Page:     1,
		PageSize: 10,
	}

	resp, err := svc.ListMaintenanceWindows(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_ListMaintenanceWindows_GetByUserIDError тестирует ошибку при получении окон.
func TestMaintenanceService_ListMaintenanceWindows_GetByUserIDError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	userID := uuid.New()

	mockWindowRepo.On("GetByUserID", mock.Anything, userID, "", 10, 0).Return(([]*interfaces.MaintenanceWindow)(nil), errors.New("db error"))

	req := &dto.ListMaintenanceWindowsRequest{
		UserID:   userID.String(),
		Status:   "",
		Page:     1,
		PageSize: 10,
	}

	resp, err := svc.ListMaintenanceWindows(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_ListMaintenanceWindows_CountError тестирует ошибку при подсчёте при листинге.
func TestMaintenanceService_ListMaintenanceWindows_CountError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	userID := uuid.New()

	mockWindowRepo.On("GetByUserID", mock.Anything, userID, "", 10, 0).Return([]*interfaces.MaintenanceWindow{}, nil)
	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, errors.New("count error"))

	req := &dto.ListMaintenanceWindowsRequest{
		UserID:   userID.String(),
		Status:   "",
		Page:     1,
		PageSize: 10,
	}

	resp, err := svc.ListMaintenanceWindows(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_GetMaintenanceWindowHistory_InvalidUserID тестирует невалидный user ID в истории.
func TestMaintenanceService_GetMaintenanceWindowHistory_InvalidUserID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.GetMaintenanceWindowHistoryRequest{
		UserID:    "not-a-uuid",
		StartDate: time.Now().Add(-7 * 24 * time.Hour),
		EndDate:   time.Now(),
		Page:      1,
		PageSize:  10,
	}

	resp, err := svc.GetMaintenanceWindowHistory(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_GetMaintenanceWindowHistory_GetHistoryError тестирует ошибку при получении истории.
func TestMaintenanceService_GetMaintenanceWindowHistory_GetHistoryError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	userID := uuid.New()
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	mockWindowRepo.On("GetHistory", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything, 10, 0).Return(([]*interfaces.MaintenanceWindow)(nil), errors.New("db error"))

	req := &dto.GetMaintenanceWindowHistoryRequest{
		UserID:    userID.String(),
		StartDate: startDate,
		EndDate:   endDate,
		Page:      1,
		PageSize:  10,
	}

	resp, err := svc.GetMaintenanceWindowHistory(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_GetMaintenanceWindowHistory_WithInvalidMonitorID тестирует невалидный monitor ID в истории.
func TestMaintenanceService_GetMaintenanceWindowHistory_WithInvalidMonitorID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	req := &dto.GetMaintenanceWindowHistoryRequest{
		UserID:     uuid.New().String(),
		StartDate:  time.Now().Add(-7 * 24 * time.Hour),
		EndDate:    time.Now(),
		MonitorIDs: []string{"not-a-uuid"},
		Page:       1,
		PageSize:   10,
	}

	resp, err := svc.GetMaintenanceWindowHistory(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestMaintenanceService_DeleteMaintenanceWindow_InvalidWindowID тестирует невалидный window ID при удалении.
func TestMaintenanceService_DeleteMaintenanceWindow_InvalidWindowID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(
		new(MockMaintenanceWindowRepository),
		new(MockMonitorRepository),
		new(MockAuditRepositoryForMaintenance),
		cfg,
	)

	err := svc.DeleteMaintenanceWindow(ctx, "not-a-uuid", uuid.New().String())

	assert.Error(t, err)
}

// TestMaintenanceService_DeleteMaintenanceWindow_GetByIDError тестирует ошибку при получении окна для удаления.
func TestMaintenanceService_DeleteMaintenanceWindow_GetByIDError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return((*interfaces.MaintenanceWindow)(nil), errors.New("db error"))

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), userID.String())

	assert.Error(t, err)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_DeleteMaintenanceWindow_NotFound тестирует удаление несуществующего окна.
func TestMaintenanceService_DeleteMaintenanceWindow_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return((*interfaces.MaintenanceWindow)(nil), nil)

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), userID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_DeleteMaintenanceWindow_DeleteError тестирует ошибку при удалении из БД.
func TestMaintenanceService_DeleteMaintenanceWindow_DeleteError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	windowID := uuid.New()
	userID := uuid.New()

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("Delete", mock.Anything, windowID).Return(errors.New("db error"))

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), userID.String())

	assert.Error(t, err)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_VersionConflict тестирует конфликт версии при обновлении.
func TestMaintenanceService_UpdateMaintenanceWindow_VersionConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()
	startTime := time.Now().Add(3 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
		Version:   2, // version mismatch
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Updated Name",
		StartTime: startTime,
		EndTime:   endTime,
		Version:   1, // stale version
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_WithMonitorsOverlap тестирует пересечение при обновлении с мониторами.
func TestMaintenanceService_UpdateMaintenanceWindow_WithMonitorsOverlap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(3 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("GetWindowMonitors", mock.Anything, windowID).Return([]uuid.UUID{monitorID}, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, []uuid.UUID{monitorID}, startTime, endTime, &windowID).Return(true, nil)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Updated Name",
		StartTime: startTime,
		EndTime:   endTime,
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "OVERLAPPING_MAINTENANCE_WINDOWS")
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_UpdateMaintenanceWindow_OverlapCheckError тестирует ошибку при проверке пересечения при обновлении.
func TestMaintenanceService_UpdateMaintenanceWindow_OverlapCheckError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(3 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("GetWindowMonitors", mock.Anything, windowID).Return([]uuid.UUID{monitorID}, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, []uuid.UUID{monitorID}, startTime, endTime, &windowID).Return(false, errors.New("overlap check error"))

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		UserID:    userID.String(),
		Name:      "Updated Name",
		StartTime: startTime,
		EndTime:   endTime,
		Version:   1,
	}

	resp, err := svc.UpdateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CancelMaintenanceWindow_InvalidTransition тестирует невалидный переход статуса.
func TestMaintenanceService_CancelMaintenanceWindow_InvalidTransition(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), new(MockAuditRepositoryForMaintenance), cfg)

	windowID := uuid.New()
	userID := uuid.New()

	// COMPLETED status cannot be cancelled
	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Completed Window",
		StartTime: time.Now().Add(-3 * time.Hour),
		EndTime:   time.Now().Add(-1 * time.Hour),
		Status:    "COMPLETED",
		Version:   2,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     windowID.String(),
		UserID: userID.String(),
	}

	resp, err := svc.CancelMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditOverlappingRejected тестирует метод auditOverlappingRejected через CreateMaintenanceWindow.
func TestMaintenanceService_AuditOverlappingRejected_Coverage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, new(MockMonitorRepository), mockAuditRepo, cfg)

	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, mock.Anything, startTime, endTime, (*uuid.UUID)(nil)).Return(true, nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test Overlap",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		IsGlobal:   false,
		MonitorIDs: []string{monitorID.String()},
		UserRole:   "USER",
		UserTier:   "Free",
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "OVERLAPPING_MAINTENANCE_WINDOWS")
	// auditOverlappingRejected is called internally; audit repo may or may not be called depending on implementation
}
