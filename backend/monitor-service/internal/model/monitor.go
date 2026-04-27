package domain

import (
	"time"

	"github.com/google/uuid"
)

// Monitor представляет HTTP/HTTPS монитор для проверки доступности сервиса.
type Monitor struct {
	// ID уникальный идентификатор монитора
	ID uuid.UUID
	// UserID идентификатор владельца монитора
	UserID uuid.UUID
	// Name уникальное имя монитора в пределах пользователя
	Name string
	// URL URL для проверки (http:// или https://)
	URL string
	// CheckType тип проверки (HTTP, HTTPS, PING)
	CheckType string
	// IntervalSeconds интервал между проверками в секундах (30-3600)
	IntervalSeconds int
	// TimeoutSeconds таймаут проверки в секундах (должен быть меньше интервала)
	TimeoutSeconds int
	// Status текущий статус монитора
	Status MonitorStatus
	// WorkingHoursStart начало рабочего времени (опционально)
	WorkingHoursStart *time.Time
	// WorkingHoursEnd конец рабочего времени (опционально)
	WorkingHoursEnd *time.Time
	// WorkingDays рабочие дни недели (опционально, []time.Weekday)
	WorkingDays []time.Weekday
	// DegradedResponseTimeThreshold порог response time для DEGRADED (мс, опционально)
	DegradedResponseTimeThreshold *int
	// DegradedFailureRateThreshold порог failure rate для DEGRADED % (опционально)
	DegradedFailureRateThreshold *int
	// LastCheckAt время последней проверки
	LastCheckAt *time.Time
	// CreatedAt время создания монитора
	CreatedAt time.Time
	// UpdatedAt время последнего обновления
	UpdatedAt time.Time
}

// NewMonitor создаёт новый монитор с валидацией.
func NewMonitor(userID uuid.UUID, name, url string, intervalSeconds int) (*Monitor, error) {
	if err := validateMonitorName(name); err != nil {
		return nil, err
	}
	if err := validateURL(url); err != nil {
		return nil, err
	}
	if err := validateInterval(intervalSeconds); err != nil {
		return nil, err
	}

	return &Monitor{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            name,
		URL:             url,
		CheckType:       "HTTP", // По умолчанию
		IntervalSeconds: intervalSeconds,
		TimeoutSeconds:  30, // По умолчанию
		Status:          StatusPending,
		WorkingDays:     defaultWorkingDays(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

// validateMonitorName проверяет валидность имени монитора.
func validateMonitorName(name string) error {
	if name == "" {
		return ErrEmptyMonitorName
	}
	if len(name) > 255 {
		return ErrMonitorNameTooLong
	}
	return nil
}

// validateURL проверяет валидность URL.
func validateURL(url string) error {
	if url == "" {
		return ErrEmptyURL
	}
	if len(url) > 2048 {
		return ErrURLTooLong
	}
	// Базовая проверка схемы
	if len(url) < 7 || (url[:7] != "http://" && url[:8] != "https://") {
		return ErrInvalidURLScheme
	}
	return nil
}

// validateInterval проверяет валидность интервала проверки.
func validateInterval(seconds int) error {
	if seconds < 30 {
		return ErrIntervalTooSmall
	}
	if seconds > 3600 {
		return ErrIntervalTooLarge
	}
	return nil
}

// defaultWorkingDays возвращает рабочие дни по умолчанию (все дни).
func defaultWorkingDays() []time.Weekday {
	return []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
		time.Friday, time.Saturday, time.Sunday,
	}
}

// UpdateStatus обновляет статус монитора с проверкой перехода.
func (m *Monitor) UpdateStatus(newStatus MonitorStatus) error {
	if !m.Status.CanTransitionTo(newStatus) {
		return &InvalidStatusTransitionError{
			From: m.Status,
			To:   newStatus,
		}
	}
	m.Status = newStatus
	m.UpdatedAt = time.Now()
	return nil
}

// ShouldCheckNow возвращает true, если монитор должен быть проверён сейчас
// с учётом рабочих часов и дней.
func (m *Monitor) ShouldCheckNow() bool {
	if m.Status == StatusPaused {
		return false
	}

	// Если не заданы рабочие часы - проверяем всегда
	if m.WorkingHoursStart == nil || m.WorkingHoursEnd == nil {
		return true
	}

	now := time.Now()
	currentTime := now

	// Проверяем, что текущее время в пределах рабочих часов
	if currentTime.Before(*m.WorkingHoursStart) || currentTime.After(*m.WorkingHoursEnd) {
		return false
	}

	// Проверяем, что сегодня рабочий день
	if len(m.WorkingDays) > 0 {
		isWorkingDay := false
		for _, wd := range m.WorkingDays {
			if now.Weekday() == wd {
				isWorkingDay = true
				break
			}
		}
		if !isWorkingDay {
			return false
		}
	}

	return true
}

// IsWithinWorkingHours возвращает true, если текущее время в рабочих часах.
func (m *Monitor) IsWithinWorkingHours() bool {
	return m.ShouldCheckNow()
}

// GetDegradedThresholds возвращает пороги для DEGRADED статуса.
func (m *Monitor) GetDegradedThresholds() (responseTimeThreshold int, failureRateThreshold int) {
	if m.DegradedResponseTimeThreshold != nil {
		responseTimeThreshold = *m.DegradedResponseTimeThreshold
	} else {
		responseTimeThreshold = 1000 // По умолчанию 1 секунда
	}

	if m.DegradedFailureRateThreshold != nil {
		failureRateThreshold = *m.DegradedFailureRateThreshold
	} else {
		failureRateThreshold = 50 // По умолчанию 50%
	}

	return responseTimeThreshold, failureRateThreshold
}

// UpdateLastCheck обновляет время последней проверки.
func (m *Monitor) UpdateLastCheck() {
	now := time.Now()
	m.LastCheckAt = &now
	m.UpdatedAt = now
}
