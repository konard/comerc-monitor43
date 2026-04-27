package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestMaintenanceService_AuditCreate_ErrorLogged тестирует, что ошибка auditRepo.Create при создании
// окна обслуживания только логируется, а операция всё равно проходит успешно.
func TestMaintenanceService_AuditCreate_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	// auditRepo.Create возвращает ошибку — должна только логироваться
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test Window",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		UserRole:   "ADMIN",
		UserTier:   "Free",
		IsGlobal:   true,
		MonitorIDs: []string{},
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	// Операция успешна, несмотря на ошибку audit
	require.NoError(t, err)
	assert.NotNil(t, resp)
	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditCreateInPastDenied_ErrorLogged тестирует auditCreateInPastDenied ошибку.
func TestMaintenanceService_AuditCreateInPastDenied_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	// auditRepo.Create возвращает ошибку — должна только логироваться
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Global Window",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		UserRole:   "USER", // не ADMIN → триггерит auditCreateInPastDenied
		IsGlobal:   true,
		MonitorIDs: []string{},
	}

	// Операция завершится ошибкой (INSUFFICIENT_PERMISSIONS), но audit лог тоже запишется
	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "INSUFFICIENT_PERMISSIONS")
	mockAuditRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditLimitReached_ErrorLogged тестирует auditLimitReached ошибку.
func TestMaintenanceService_AuditLimitReached_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	// Лимит превышен: текущее кол-во = maxWindowsFreeTier (5)
	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(cfg.MaxWindowsFreeTier, nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test Window",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		UserRole:   "USER",
		UserTier:   "Free",
		IsGlobal:   false,
		MonitorIDs: []string{},
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "MAINTENANCE_WINDOW_LIMIT_REACHED")
	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditList_ErrorLogged тестирует auditList ошибку.
func TestMaintenanceService_AuditList_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()

	mockWindowRepo.On("GetByUserID", mock.Anything, userID, "", 10, 0).Return([]*interfaces.MaintenanceWindow{}, nil)
	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	req := &dto.ListMaintenanceWindowsRequest{
		UserID:   userID.String(),
		Page:     1,
		PageSize: 10,
	}

	resp, err := svc.ListMaintenanceWindows(ctx, req)

	// Операция успешна, несмотря на ошибку audit
	require.NoError(t, err)
	assert.NotNil(t, resp)
	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditCancel_ErrorLogged тестирует auditCancel ошибку.
func TestMaintenanceService_AuditCancel_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	windowID := uuid.New()

	repoWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test",
		Status:    "SCHEDULED",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(3 * time.Hour),
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(repoWindow, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     windowID.String(),
		UserID: userID.String(),
	}

	resp, err := svc.CancelMaintenanceWindow(ctx, req)

	// Операция успешна, несмотря на ошибку audit
	require.NoError(t, err)
	assert.NotNil(t, resp)
	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditDelete_ErrorLogged тестирует auditDelete ошибку.
func TestMaintenanceService_AuditDelete_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	windowID := uuid.New()

	repoWindow := &interfaces.MaintenanceWindow{
		ID:     windowID,
		UserID: userID,
		Name:   "Test",
		Status: "SCHEDULED",
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(repoWindow, nil)
	mockWindowRepo.On("Delete", mock.Anything, windowID).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	err := svc.DeleteMaintenanceWindow(ctx, windowID.String(), userID.String())

	// Операция успешна, несмотря на ошибку audit
	require.NoError(t, err)
	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

// TestMaintenanceService_AuditOverlapping_ErrorLogged тестирует auditOverlappingRejected ошибку.
func TestMaintenanceService_AuditOverlapping_ErrorLogged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	monitorID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, []uuid.UUID{monitorID}, startTime, endTime, (*uuid.UUID)(nil)).Return(true, nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:     userID.String(),
		Name:       "Test Window",
		StartTime:  startTime,
		EndTime:    endTime,
		Recurrence: "ONCE",
		UserRole:   "USER",
		UserTier:   "Free",
		IsGlobal:   false,
		MonitorIDs: []string{monitorID.String()},
	}

	resp, err := svc.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "OVERLAPPING_MAINTENANCE_WINDOWS")
	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}
