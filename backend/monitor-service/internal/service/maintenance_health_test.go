package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHealthChecker тестирует создание нового HealthChecker.
func TestNewHealthChecker(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{}
	checker := NewHealthChecker(svc)

	assert.NotNil(t, checker)
	assert.Equal(t, svc, checker.service)
}

// TestHealthChecker_Check_NilService тестирует ошибку при nil сервисе.
func TestHealthChecker_Check_NilService(t *testing.T) {
	t.Parallel()

	checker := &HealthChecker{service: nil}

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestHealthChecker_Check_NilWindowRepo тестирует ошибку при nil репозитории окон.
func TestHealthChecker_Check_NilWindowRepo(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{
		windowRepo:  nil,
		monitorRepo: new(MockMonitorRepository),
		auditRepo:   new(MockAuditRepositoryForMaintenance),
		cfg:         DefaultMaintenanceWindowConfig(),
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maintenance window repository is not initialized")
}

// TestHealthChecker_Check_NilMonitorRepo тестирует ошибку при nil репозитории мониторов.
func TestHealthChecker_Check_NilMonitorRepo(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{
		windowRepo:  new(MockMaintenanceWindowRepository),
		monitorRepo: nil,
		auditRepo:   new(MockAuditRepositoryForMaintenance),
		cfg:         DefaultMaintenanceWindowConfig(),
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor repository is not initialized")
}

// TestHealthChecker_Check_NilAuditRepo тестирует ошибку при nil репозитории аудита.
func TestHealthChecker_Check_NilAuditRepo(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{
		windowRepo:  new(MockMaintenanceWindowRepository),
		monitorRepo: new(MockMonitorRepository),
		auditRepo:   nil,
		cfg:         DefaultMaintenanceWindowConfig(),
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "audit repository is not initialized")
}

// TestHealthChecker_Check_InvalidMaxDurationHours_Zero тестирует ошибку при нулевом MaxDurationHours.
func TestHealthChecker_Check_InvalidMaxDurationHours_Zero(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaintenanceWindowConfig()
	cfg.MaxDurationHours = 0

	svc := &MaintenanceService{
		windowRepo:  new(MockMaintenanceWindowRepository),
		monitorRepo: new(MockMonitorRepository),
		auditRepo:   new(MockAuditRepositoryForMaintenance),
		cfg:         cfg,
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxDurationHours out of range")
}

// TestHealthChecker_Check_InvalidMaxDurationHours_TooLarge тестирует ошибку при слишком большом MaxDurationHours.
func TestHealthChecker_Check_InvalidMaxDurationHours_TooLarge(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaintenanceWindowConfig()
	cfg.MaxDurationHours = 200 // > 168

	svc := &MaintenanceService{
		windowRepo:  new(MockMaintenanceWindowRepository),
		monitorRepo: new(MockMonitorRepository),
		auditRepo:   new(MockAuditRepositoryForMaintenance),
		cfg:         cfg,
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxDurationHours out of range")
}

// TestHealthChecker_Check_InvalidMinDurationMinutes тестирует ошибку при нулевом MinDurationMinutes.
func TestHealthChecker_Check_InvalidMinDurationMinutes(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaintenanceWindowConfig()
	cfg.MinDurationMinutes = 0

	svc := &MaintenanceService{
		windowRepo:  new(MockMaintenanceWindowRepository),
		monitorRepo: new(MockMonitorRepository),
		auditRepo:   new(MockAuditRepositoryForMaintenance),
		cfg:         cfg,
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MinDurationMinutes")
}

// TestHealthChecker_Check_ValidConfig тестирует успешную проверку здоровья.
func TestHealthChecker_Check_ValidConfig(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{
		windowRepo:  new(MockMaintenanceWindowRepository),
		monitorRepo: new(MockMonitorRepository),
		auditRepo:   new(MockAuditRepositoryForMaintenance),
		cfg:         DefaultMaintenanceWindowConfig(),
	}
	checker := NewHealthChecker(svc)

	err := checker.Check(context.Background())

	require.NoError(t, err)
}

// TestHealthChecker_GetMetrics_NilService тестирует GetMetrics при nil сервисе.
func TestHealthChecker_GetMetrics_NilService(t *testing.T) {
	t.Parallel()

	checker := &HealthChecker{service: nil}

	metrics := checker.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, "uninitialized", metrics["status"])
}

// TestHealthChecker_GetMetrics_NilMetrics тестирует GetMetrics при nil metrics в сервисе.
func TestHealthChecker_GetMetrics_NilMetrics(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{metrics: nil}
	checker := NewHealthChecker(svc)

	metrics := checker.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, "uninitialized", metrics["status"])
}

// TestHealthChecker_GetMetrics_WithMetrics тестирует GetMetrics при инициализированном сервисе.
func TestHealthChecker_GetMetrics_WithMetrics(t *testing.T) {
	t.Parallel()

	svc := &MaintenanceService{
		metrics: NewMaintenanceMetrics(),
	}
	checker := NewHealthChecker(svc)

	metrics := checker.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "created_total")
}
