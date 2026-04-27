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

// MockMaintenanceWindowRepository для тестов maintenance service
type MockMaintenanceWindowRepository struct {
	mock.Mock
}

func (m *MockMaintenanceWindowRepository) Create(ctx context.Context, window *interfaces.MaintenanceWindow) error {
	args := m.Called(mock.Anything, window)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) GetByID(ctx context.Context, id uuid.UUID) (*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetByUserID(ctx context.Context, userID uuid.UUID, status string, limit, offset int) ([]*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, userID, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetActiveWindowsForMonitor(ctx context.Context, monitorID uuid.UUID, at time.Time) ([]*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, monitorID, at)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetActiveWindowsForUser(ctx context.Context, userID uuid.UUID, at time.Time) ([]*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, userID, at)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) Update(ctx context.Context, window *interfaces.MaintenanceWindow) error {
	args := m.Called(mock.Anything, window)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(mock.Anything, id)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(mock.Anything, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) CheckOverlap(ctx context.Context, userID uuid.UUID, monitorIDs []uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(mock.Anything, userID, monitorIDs, startTime, endTime, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetWindowsRequiringActivation(ctx context.Context, before time.Time) ([]*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetWindowsRequiringCompletion(ctx context.Context, before time.Time) ([]*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) AddMonitorsToWindow(ctx context.Context, windowID uuid.UUID, monitorIDs []uuid.UUID) error {
	args := m.Called(mock.Anything, windowID, monitorIDs)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) RemoveMonitorsFromWindow(ctx context.Context, windowID uuid.UUID, monitorIDs []uuid.UUID) error {
	args := m.Called(mock.Anything, windowID, monitorIDs)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) GetWindowMonitors(ctx context.Context, windowID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(mock.Anything, windowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetHistory(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, monitorIDs []uuid.UUID, limit, offset int) ([]*interfaces.MaintenanceWindow, error) {
	args := m.Called(mock.Anything, userID, startDate, endDate, monitorIDs, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.MaintenanceWindow), args.Error(1)
}

// MockAuditRepositoryForMaintenance для тестов maintenance service (отдельный от существующего)
type MockAuditRepositoryForMaintenance struct {
	mock.Mock
}

func (m *MockAuditRepositoryForMaintenance) Create(ctx context.Context, entry *interfaces.AuditLogEntry) error {
	args := m.Called(mock.Anything, entry)
	return args.Error(0)
}

func (m *MockAuditRepositoryForMaintenance) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*interfaces.AuditLogEntry, error) {
	args := m.Called(mock.Anything, monitorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.AuditLogEntry), args.Error(1)
}

func (m *MockAuditRepositoryForMaintenance) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*interfaces.AuditLogEntry, error) {
	args := m.Called(mock.Anything, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*interfaces.AuditLogEntry), args.Error(1)
}

func (m *MockAuditRepositoryForMaintenance) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	args := m.Called(mock.Anything, olderThan)
	return int64(args.Int(0)), args.Error(1)
}

// Tests

func TestMaintenanceService_CreateMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(0, nil)
	mockWindowRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:          userID.String(),
		Name:            "Test Maintenance",
		StartTime:       startTime,
		EndTime:         endTime,
		Recurrence:      "ONCE",
		IsGlobal:        true,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{},
		UserRole:        "ADMIN",
		UserTier:        "Free",
	}

	resp, err := service.CreateMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test Maintenance", resp.Name)
	assert.True(t, resp.IsGlobal)

	mockWindowRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

func TestMaintenanceService_CreateMaintenanceWindow_GlobalWindowByNonAdmin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:          userID.String(),
		Name:            "Global Maintenance",
		StartTime:       startTime,
		EndTime:         endTime,
		Recurrence:      "ONCE",
		IsGlobal:        true,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{},
		UserRole:        "USER", // Not ADMIN
		UserTier:        "Free",
	}

	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := service.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "INSUFFICIENT_PERMISSIONS")
}

func TestMaintenanceService_CreateMaintenanceWindow_LimitReached(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(5, nil) // Already at limit
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.CreateMaintenanceWindowRequest{
		UserID:          userID.String(),
		Name:            "Test Maintenance",
		StartTime:       startTime,
		EndTime:         endTime,
		Recurrence:      "ONCE",
		IsGlobal:        false,
		PauseMonitoring: true,
		SuppressAlerts:  true,
		SafeMode:        false,
		MonitorIDs:      []string{},
		UserRole:        "USER",
		UserTier:        "Free",
	}

	resp, err := service.CreateMaintenanceWindow(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "MAINTENANCE_WINDOW_LIMIT_REACHED")
}

func TestMaintenanceService_UpdateMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	windowID := uuid.New()
	startTime := time.Now().Add(3 * time.Hour)
	endTime := startTime.Add(3 * time.Hour)

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Old Name",
		StartTime: time.Now().Add(2 * time.Hour),
		EndTime:   time.Now().Add(4 * time.Hour),
		Status:    "SCHEDULED",
		Version:   1,
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("GetWindowMonitors", mock.Anything, windowID).Return([]uuid.UUID{}, nil)
	mockWindowRepo.On("CheckOverlap", mock.Anything, userID, []uuid.UUID{}, startTime, endTime, &windowID).Return(false, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.UpdateMaintenanceWindowRequest{
		ID:        windowID.String(),
		Name:      "New Name",
		StartTime: startTime,
		EndTime:   endTime,
		Version:   1,
		UserID:    userID.String(),
	}

	resp, err := service.UpdateMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "New Name", resp.Name)
}

func TestMaintenanceService_CancelMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	windowID := uuid.New()

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
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.CancelMaintenanceWindowRequest{
		ID:     windowID.String(),
		UserID: userID.String(),
	}

	resp, err := service.CancelMaintenanceWindow(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "CANCELLED", resp.Status)
}

func TestMaintenanceService_ListMaintenanceWindows_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()

	windows := []*interfaces.MaintenanceWindow{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Window 1",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
			Status:    "SCHEDULED",
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Window 2",
			StartTime: time.Now().Add(3 * time.Hour),
			EndTime:   time.Now().Add(4 * time.Hour),
			Status:    "ACTIVE",
		},
	}

	mockWindowRepo.On("GetByUserID", mock.Anything, userID, "", 10, 0).Return(windows, nil)
	mockWindowRepo.On("CountByUserID", mock.Anything, userID).Return(2, nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := &dto.ListMaintenanceWindowsRequest{
		UserID:   userID.String(),
		Status:   "",
		Page:     1,
		PageSize: 10,
	}

	resp, err := service.ListMaintenanceWindows(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Windows, 2)
	assert.Equal(t, 2, resp.Total)
}

func TestMaintenanceService_DeleteMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	windowID := uuid.New()

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    userID,
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)
	mockWindowRepo.On("Delete", mock.Anything, windowID).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.DeleteMaintenanceWindow(ctx, windowID.String(), userID.String())

	require.NoError(t, err)
}

func TestMaintenanceService_DeleteMaintenanceWindow_WrongUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	anotherUserID := uuid.New()
	windowID := uuid.New()

	existingWindow := &interfaces.MaintenanceWindow{
		ID:        windowID,
		UserID:    anotherUserID, // Different user
		Name:      "Test Window",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    "SCHEDULED",
	}

	mockWindowRepo.On("GetByID", mock.Anything, windowID).Return(existingWindow, nil)

	err := service.DeleteMaintenanceWindow(ctx, windowID.String(), userID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INSUFFICIENT_PERMISSIONS")
}

func TestMaintenanceService_GetMaintenanceWindowHistory_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	service := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()

	windows := []*interfaces.MaintenanceWindow{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Completed Window",
			StartTime: startDate,
			EndTime:   startDate.Add(2 * time.Hour),
			Status:    "COMPLETED",
		},
	}

	mockWindowRepo.On("GetHistory", mock.Anything, userID, mock.Anything, mock.Anything, mock.Anything, 10, 0).Return(windows, nil)

	req := &dto.GetMaintenanceWindowHistoryRequest{
		UserID:    userID.String(),
		StartDate: startDate,
		EndDate:   endDate,
		Page:      1,
		PageSize:  10,
	}

	resp, err := service.GetMaintenanceWindowHistory(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Windows, 1)
}
