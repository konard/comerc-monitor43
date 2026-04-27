package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MaintenanceWindowErrors - ошибки, связанные с окнами обслуживания.
var (
	ErrEmptyWindowName        = errors.New("window name cannot be empty")
	ErrWindowNameTooLong      = errors.New("window name too long (max 255 chars)")
	ErrEndTimeBeforeStartTime = errors.New("END_TIME_BEFORE_START_TIME: end time must be after start time")
	ErrDurationExceedsMaximum = errors.New("DURATION_EXCEEDS_MAXIMUM: duration exceeds maximum allowed")
	ErrDurationBelowMinimum   = errors.New("DURATION_BELOW_MINIMUM: duration below minimum allowed")
	ErrInvalidRecurrence      = errors.New("invalid recurrence type")
	ErrCannotCreateInPast     = errors.New("CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST: cannot create maintenance window in past")
	ErrOverlappingWindows     = errors.New("OVERLAPPING_MAINTENANCE_WINDOWS: maintenance windows overlap for the same monitor")
	ErrMonitorLimitPerWindow  = errors.New("MONITOR_LIMIT_PER_WINDOW_REACHED: monitor limit per window exceeded")
)

// MaintenanceWindowStatus представляет статус окна обслуживания.
type MaintenanceWindowStatus string

const (
	WindowStatusScheduled MaintenanceWindowStatus = "SCHEDULED"
	WindowStatusActive    MaintenanceWindowStatus = "ACTIVE"
	WindowStatusCompleted MaintenanceWindowStatus = "COMPLETED"
	WindowStatusCancelled MaintenanceWindowStatus = "CANCELLED"
	WindowStatusOrphaned  MaintenanceWindowStatus = "ORPHANED"
)

// RecurrenceType представляет тип повторения окна обслуживания.
type RecurrenceType string

const (
	RecurrenceOnce    RecurrenceType = "ONCE"
	RecurrenceDaily   RecurrenceType = "DAILY"
	RecurrenceWeekly  RecurrenceType = "WEEKLY"
	RecurrenceMonthly RecurrenceType = "MONTHLY"
)

// MaintenanceWindow представляет окно технического обслуживания.
type MaintenanceWindow struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Name            string
	StartTime       time.Time
	EndTime         time.Time
	Status          MaintenanceWindowStatus
	Recurrence      RecurrenceType
	IsGlobal        bool
	PauseMonitoring bool
	SuppressAlerts  bool
	SafeMode        bool
	MonitorIDs      []uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ActivatedAt     *time.Time
	CompletedAt     *time.Time
	Version         int
}

// MaintenanceWindowConfig содержит конфигурационные лимиты для окон обслуживания.
type MaintenanceWindowConfig struct {
	MinDurationMinutes   int
	MaxDurationHours     int
	MaxWindowsPerAccount int
	MaxMonitorsPerWindow int
}

// DefaultMaintenanceWindowConfig возвращает конфигурацию по умолчанию.
func DefaultMaintenanceWindowConfig() MaintenanceWindowConfig {
	return MaintenanceWindowConfig{
		MinDurationMinutes:   1,
		MaxDurationHours:     24,
		MaxWindowsPerAccount: 5, // Для Free tier
		MaxMonitorsPerWindow: 50,
	}
}

// NewMaintenanceWindow создаёт новое окно обслуживания с валидацией.
func NewMaintenanceWindow(
	userID uuid.UUID,
	name string,
	startTime, endTime time.Time,
	recurrence RecurrenceType,
	isGlobal, pauseMonitoring, suppressAlerts, safeMode bool,
	monitorIDs []uuid.UUID,
	cfg MaintenanceWindowConfig,
) (*MaintenanceWindow, error) {
	if err := validateWindowName(name); err != nil {
		return nil, err
	}
	if err := validateTimeRange(startTime, endTime); err != nil {
		return nil, err
	}
	if err := validateDuration(startTime, endTime, cfg); err != nil {
		return nil, err
	}
	if err := validateRecurrence(recurrence); err != nil {
		return nil, err
	}
	if err := validateNotInPast(startTime); err != nil {
		return nil, err
	}
	if err := validateMonitorsPerWindow(monitorIDs, cfg); err != nil {
		return nil, err
	}

	now := time.Now()

	return &MaintenanceWindow{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            name,
		StartTime:       startTime,
		EndTime:         endTime,
		Status:          WindowStatusScheduled,
		Recurrence:      recurrence,
		IsGlobal:        isGlobal,
		PauseMonitoring: pauseMonitoring,
		SuppressAlerts:  suppressAlerts,
		SafeMode:        safeMode,
		MonitorIDs:      monitorIDs,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}, nil
}

// validateWindowName проверяет валидность имени окна.
func validateWindowName(name string) error {
	if name == "" {
		return ErrEmptyWindowName
	}
	if len(name) > 255 {
		return ErrWindowNameTooLong
	}
	return nil
}

// validateTimeRange проверяет, что endTime после startTime.
func validateTimeRange(startTime, endTime time.Time) error {
	if !endTime.After(startTime) {
		return ErrEndTimeBeforeStartTime
	}
	return nil
}

// validateDuration проверяет, что длительность в допустимых пределах.
func validateDuration(startTime, endTime time.Time, cfg MaintenanceWindowConfig) error {
	duration := endTime.Sub(startTime)

	minDuration := time.Duration(cfg.MinDurationMinutes) * time.Minute
	if duration < minDuration {
		return ErrDurationBelowMinimum
	}

	maxDuration := time.Duration(cfg.MaxDurationHours) * time.Hour
	if duration > maxDuration {
		return ErrDurationExceedsMaximum
	}

	return nil
}

// validateRecurrence проверяет валидность типа повторения.
func validateRecurrence(recurrence RecurrenceType) error {
	switch recurrence {
	case RecurrenceOnce, RecurrenceDaily, RecurrenceWeekly, RecurrenceMonthly:
		return nil
	default:
		return ErrInvalidRecurrence
	}
}

// validateNotInPast проверяет, что окно не создаётся в прошлом.
func validateNotInPast(startTime time.Time) error {
	// Допускаем создание окна, если оно начинается не раньше текущего времени
	if startTime.Before(time.Now().Add(-time.Second)) {
		return ErrCannotCreateInPast
	}
	return nil
}

// validateMonitorsPerWindow проверяет лимит мониторов в окне.
func validateMonitorsPerWindow(monitorIDs []uuid.UUID, cfg MaintenanceWindowConfig) error {
	if len(monitorIDs) > cfg.MaxMonitorsPerWindow {
		return ErrMonitorLimitPerWindow
	}
	return nil
}

// IsActive возвращает true, если окно активно сейчас.
func (w *MaintenanceWindow) IsActive() bool {
	if w.Status != WindowStatusActive {
		return false
	}
	now := time.Now()
	return now.Equal(w.StartTime) || (now.After(w.StartTime) && now.Before(w.EndTime))
}

// ShouldBeActive возвращает true, если окно должно стать активным.
func (w *MaintenanceWindow) ShouldBeActive() bool {
	if w.Status != WindowStatusScheduled {
		return false
	}
	now := time.Now()
	return !now.Before(w.StartTime)
}

// ShouldBeCompleted возвращает true, если окно должно завершиться.
func (w *MaintenanceWindow) ShouldBeCompleted() bool {
	if w.Status != WindowStatusActive {
		return false
	}
	return time.Now().After(w.EndTime)
}

// OverlapsWith проверяет, пересекается ли это окно с другим для мониторов.
func (w *MaintenanceWindow) OverlapsWith(other *MaintenanceWindow) bool {
	// Если окна не для общих мониторов - не пересекаются
	if !w.sharesMonitorsWith(other) {
		return false
	}

	// Проверяем пересечение по времени
	return w.StartTime.Before(other.EndTime) && w.EndTime.After(other.StartTime)
}

// sharesMonitorsWith проверяет, есть ли общие мониторы у двух окон.
func (w *MaintenanceWindow) sharesMonitorsWith(other *MaintenanceWindow) bool {
	// Глобальные окна пересекаются со всеми
	if w.IsGlobal || other.IsGlobal {
		return true
	}

	// Проверяем пересечение множеств мониторов
	monitorSet := make(map[uuid.UUID]bool)
	for _, id := range w.MonitorIDs {
		monitorSet[id] = true
	}

	for _, id := range other.MonitorIDs {
		if monitorSet[id] {
			return true
		}
	}

	return false
}

// Duration возвращает длительность окна.
func (w *MaintenanceWindow) Duration() time.Duration {
	return w.EndTime.Sub(w.StartTime)
}

// Activate переводит окно в активный статус.
func (w *MaintenanceWindow) Activate() error {
	if w.Status != WindowStatusScheduled {
		return errors.New("only scheduled windows can be activated")
	}

	now := time.Now()
	w.Status = WindowStatusActive
	w.ActivatedAt = &now
	w.UpdatedAt = now
	w.Version++
	return nil
}

// Complete завершает окно.
func (w *MaintenanceWindow) Complete() error {
	if w.Status != WindowStatusActive {
		return errors.New("only active windows can be completed")
	}

	now := time.Now()
	w.Status = WindowStatusCompleted
	w.CompletedAt = &now
	w.UpdatedAt = now
	w.Version++
	return nil
}

// Cancel отменяет окно.
func (w *MaintenanceWindow) Cancel() error {
	if w.Status != WindowStatusScheduled && w.Status != WindowStatusActive {
		return errors.New("only scheduled or active windows can be cancelled")
	}

	w.Status = WindowStatusCancelled
	w.UpdatedAt = time.Now()
	w.Version++
	return nil
}

// CanTransitionTo проверяет возможность перехода статуса.
func (w MaintenanceWindowStatus) CanTransitionTo(newStatus MaintenanceWindowStatus) bool {
	transitions := map[MaintenanceWindowStatus][]MaintenanceWindowStatus{
		WindowStatusScheduled: {WindowStatusActive, WindowStatusCancelled},
		WindowStatusActive:    {WindowStatusCompleted, WindowStatusCancelled},
		WindowStatusCompleted: {}, // Финальный статус
		WindowStatusCancelled: {}, // Финальный статус
		WindowStatusOrphaned:  {}, // Финальный статус
	}

	allowed, exists := transitions[w]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}

	return false
}

// UpdateDetails обновляет параметры окна с проверкой версии.
func (w *MaintenanceWindow) UpdateDetails(name string, startTime, endTime time.Time, version int) error {
	if w.Version != version {
		return errors.New("version conflict: window was modified by another user")
	}

	if w.Status != WindowStatusScheduled {
		return errors.New("only scheduled windows can be updated")
	}

	if err := validateWindowName(name); err != nil {
		return err
	}
	if err := validateTimeRange(startTime, endTime); err != nil {
		return err
	}

	cfg := DefaultMaintenanceWindowConfig()
	if err := validateDuration(startTime, endTime, cfg); err != nil {
		return err
	}

	w.Name = name
	w.StartTime = startTime
	w.EndTime = endTime
	w.Version++
	w.UpdatedAt = time.Now()

	return nil
}
