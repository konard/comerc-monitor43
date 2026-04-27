package service

import (
	"context"
	"fmt"
)

// HealthChecker проверяет здоровье MaintenanceService.
type HealthChecker struct {
	service *MaintenanceService
}

// NewHealthChecker создаёт новый HealthChecker.
func NewHealthChecker(service *MaintenanceService) *HealthChecker {
	return &HealthChecker{
		service: service,
	}
}

// Check проверяет здоровье maintenance service.
func (h *HealthChecker) Check(ctx context.Context) error {
	// Проверка: Сервис должен быть инициализирован
	if h.service == nil {
		return fmt.Errorf("maintenance service is not initialized")
	}

	// Проверка: Репозитории должны быть инициализированы
	if h.service.windowRepo == nil {
		return fmt.Errorf("maintenance window repository is not initialized")
	}
	if h.service.monitorRepo == nil {
		return fmt.Errorf("monitor repository is not initialized")
	}
	if h.service.auditRepo == nil {
		return fmt.Errorf("audit repository is not initialized")
	}

	// Проверка: Конфигурация должна быть валидной
	if h.service.cfg.MaxDurationHours < 1 || h.service.cfg.MaxDurationHours > 168 {
		return fmt.Errorf("invalid configuration: MaxDurationHours out of range [1, 168]")
	}
	if h.service.cfg.MinDurationMinutes < 1 {
		return fmt.Errorf("invalid configuration: MinDurationMinutes must be >= 1")
	}

	return nil
}

// GetMetrics возвращает текущие метрики сервиса.
func (h *HealthChecker) GetMetrics() map[string]any {
	if h.service == nil || h.service.metrics == nil {
		return map[string]any{
			"status": "uninitialized",
		}
	}

	return h.service.metrics.GetMetrics()
}
