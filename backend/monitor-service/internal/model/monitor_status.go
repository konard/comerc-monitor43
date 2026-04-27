package domain

import (
	"fmt"
)

// MonitorStatus представляет текущий статус монитора.
type MonitorStatus string

const (
	// StatusPending - начальный статус после создания монитора
	StatusPending MonitorStatus = "PENDING"
	// StatusUp - монитор доступен и работает нормально
	StatusUp MonitorStatus = "UP"
	// StatusDown - монитор недоступен или возвращает ошибки
	StatusDown MonitorStatus = "DOWN"
	// StatusDegraded - монитор доступен, но с ухудшением производительности
	StatusDegraded MonitorStatus = "DEGRADED"
	// StatusPaused - монитор приостановлен пользователем
	StatusPaused MonitorStatus = "PAUSED"
)

// String возвращает строковое представление статуса.
func (s MonitorStatus) String() string {
	return string(s)
}

// IsValid проверяет, что статус является допустимым значением.
func (s MonitorStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusUp, StatusDown, StatusDegraded, StatusPaused:
		return true
	default:
		return false
	}
}

// CanTransitionTo проверяет, возможен ли переход из текущего статуса в целевой.
func (s MonitorStatus) CanTransitionTo(to MonitorStatus) bool {
	// Разрешённые переходы:
	// PENDING -> UP, DOWN, DEGRADED, PAUSED
	// UP -> DOWN, DEGRADED, PAUSED
	// DOWN -> UP, DEGRADED, PAUSED
	// DEGRADED -> UP, DOWN, PAUSED
	// PAUSED -> UP (при восстановлении)

	switch s {
	case StatusPending:
		return to == StatusUp || to == StatusDown || to == StatusDegraded || to == StatusPaused
	case StatusUp:
		return to == StatusDown || to == StatusDegraded || to == StatusPaused
	case StatusDown:
		return to == StatusUp || to == StatusDegraded || to == StatusPaused
	case StatusDegraded:
		return to == StatusUp || to == StatusDown || to == StatusPaused
	case StatusPaused:
		return to == StatusUp
	default:
		return false
	}
}

// ParseMonitorStatus парсит строку в MonitorStatus.
func ParseMonitorStatus(s string) (MonitorStatus, error) {
	status := MonitorStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid monitor status: %s", s)
	}
	return status, nil
}

// IsActive возвращает true, если монитор активен (не приостановлен).
func (s MonitorStatus) IsActive() bool {
	return s != StatusPaused
}

// IsHealthy возвращает true, если монитор здоров (UP или PENDING).
func (s MonitorStatus) IsHealthy() bool {
	return s == StatusUp || s == StatusPending
}

// IsUnhealthy возвращает true, если монитор нездоров (DOWN или DEGRADED).
func (s MonitorStatus) IsUnhealthy() bool {
	return s == StatusDown || s == StatusDegraded
}
