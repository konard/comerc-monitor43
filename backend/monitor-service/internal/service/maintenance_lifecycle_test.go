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
)

// TestMaintenanceService_ActivateScheduledWindows_Success тестирует успешную активацию окон.
func TestMaintenanceService_ActivateScheduledWindows_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	now := time.Now()

	scheduledWindows := []*interfaces.MaintenanceWindow{
		{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Scheduled Window",
			StartTime:  now.Add(-5 * time.Minute),
			EndTime:    now.Add(2 * time.Hour),
			Status:     "SCHEDULED",
			Recurrence: "ONCE",
			Version:    1,
		},
	}

	mockWindowRepo.On("GetWindowsRequiringActivation", mock.Anything, mock.Anything).Return(scheduledWindows, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.ActivateScheduledWindows(ctx)

	require.NoError(t, err)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_ActivateScheduledWindows_NoWindows тестирует активацию при отсутствии окон.
func TestMaintenanceService_ActivateScheduledWindows_NoWindows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	mockWindowRepo.On("GetWindowsRequiringActivation", mock.Anything, mock.Anything).Return([]*interfaces.MaintenanceWindow{}, nil)

	err := svc.ActivateScheduledWindows(ctx)

	require.NoError(t, err)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_ActivateScheduledWindows_RepoError тестирует ошибку репозитория.
func TestMaintenanceService_ActivateScheduledWindows_RepoError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	mockWindowRepo.On("GetWindowsRequiringActivation", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	err := svc.ActivateScheduledWindows(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get windows requiring activation")
}

// TestMaintenanceService_ActivateScheduledWindows_InvalidTransition тестирует пропуск окна с невалидным переходом статуса.
func TestMaintenanceService_ActivateScheduledWindows_InvalidTransition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	now := time.Now()

	// Окно уже в статусе ACTIVE — нельзя активировать ещё раз
	alreadyActiveWindows := []*interfaces.MaintenanceWindow{
		{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Already Active Window",
			StartTime:  now.Add(-1 * time.Hour),
			EndTime:    now.Add(1 * time.Hour),
			Status:     "ACTIVE",
			Recurrence: "ONCE",
			Version:    1,
		},
	}

	mockWindowRepo.On("GetWindowsRequiringActivation", mock.Anything, mock.Anything).Return(alreadyActiveWindows, nil)

	// Ошибка активации — окно пропускается, Update не вызывается
	err := svc.ActivateScheduledWindows(ctx)

	require.NoError(t, err)
	// Update не должен вызываться для окна с невалидным переходом
	mockWindowRepo.AssertNotCalled(t, "Update")
}

// TestMaintenanceService_CompleteActiveWindows_Success тестирует успешное завершение окон.
func TestMaintenanceService_CompleteActiveWindows_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	now := time.Now()

	activeWindows := []*interfaces.MaintenanceWindow{
		{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Active Window",
			StartTime:  now.Add(-2 * time.Hour),
			EndTime:    now.Add(-5 * time.Minute),
			Status:     "ACTIVE",
			Recurrence: "ONCE",
			Version:    1,
		},
	}

	mockWindowRepo.On("GetWindowsRequiringCompletion", mock.Anything, mock.Anything).Return(activeWindows, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.CompleteActiveWindows(ctx)

	require.NoError(t, err)
	mockWindowRepo.AssertExpectations(t)
}

// TestMaintenanceService_CompleteActiveWindows_NoWindows тестирует завершение при отсутствии окон.
func TestMaintenanceService_CompleteActiveWindows_NoWindows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	mockWindowRepo.On("GetWindowsRequiringCompletion", mock.Anything, mock.Anything).Return([]*interfaces.MaintenanceWindow{}, nil)

	err := svc.CompleteActiveWindows(ctx)

	require.NoError(t, err)
}

// TestMaintenanceService_CompleteActiveWindows_RepoError тестирует ошибку репозитория.
func TestMaintenanceService_CompleteActiveWindows_RepoError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	mockWindowRepo.On("GetWindowsRequiringCompletion", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	err := svc.CompleteActiveWindows(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get windows requiring completion")
}

// TestMaintenanceService_CompleteActiveWindows_InvalidTransition тестирует пропуск окна с невалидным переходом.
func TestMaintenanceService_CompleteActiveWindows_InvalidTransition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	now := time.Now()

	// Окно в статусе CANCELLED — нельзя завершить
	cancelledWindows := []*interfaces.MaintenanceWindow{
		{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Cancelled Window",
			StartTime:  now.Add(-2 * time.Hour),
			EndTime:    now.Add(-5 * time.Minute),
			Status:     "CANCELLED",
			Recurrence: "ONCE",
			Version:    1,
		},
	}

	mockWindowRepo.On("GetWindowsRequiringCompletion", mock.Anything, mock.Anything).Return(cancelledWindows, nil)

	err := svc.CompleteActiveWindows(ctx)

	require.NoError(t, err)
	mockWindowRepo.AssertNotCalled(t, "Update")
}

// TestMaintenanceService_CompleteActiveWindows_UpdateError тестирует обработку ошибки обновления.
func TestMaintenanceService_CompleteActiveWindows_UpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockWindowRepo := new(MockMaintenanceWindowRepository)
	mockMonitorRepo := new(MockMonitorRepository)
	mockAuditRepo := new(MockAuditRepositoryForMaintenance)

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(mockWindowRepo, mockMonitorRepo, mockAuditRepo, cfg)

	userID := uuid.New()
	now := time.Now()

	activeWindows := []*interfaces.MaintenanceWindow{
		{
			ID:         uuid.New(),
			UserID:     userID,
			Name:       "Active Window",
			StartTime:  now.Add(-2 * time.Hour),
			EndTime:    now.Add(-5 * time.Minute),
			Status:     "ACTIVE",
			Recurrence: "ONCE",
			Version:    1,
		},
	}

	mockWindowRepo.On("GetWindowsRequiringCompletion", mock.Anything, mock.Anything).Return(activeWindows, nil)
	mockWindowRepo.On("Update", mock.Anything, mock.Anything).Return(assert.AnError)

	// Ошибка Update логируется, но не возвращается — метод продолжает работу
	err := svc.CompleteActiveWindows(ctx)

	require.NoError(t, err)
}

// TestMaintenanceService_GetMaxWindowsForTier тестирует лимиты для разных тиров.
func TestMaintenanceService_GetMaxWindowsForTier(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaintenanceWindowConfig()
	svc := NewMaintenanceService(nil, nil, nil, cfg)

	assert.Equal(t, cfg.MaxWindowsFreeTier, svc.getMaxWindowsForTier("Free"))
	assert.Equal(t, cfg.MaxWindowsFreeTier, svc.getMaxWindowsForTier(""))
	assert.Equal(t, cfg.MaxWindowsProTier, svc.getMaxWindowsForTier("Pro"))
	assert.Equal(t, cfg.MaxWindowsEnterprise, svc.getMaxWindowsForTier("Enterprise"))
	// Неизвестный тир → Free лимит
	assert.Equal(t, cfg.MaxWindowsFreeTier, svc.getMaxWindowsForTier("Unknown"))
}
