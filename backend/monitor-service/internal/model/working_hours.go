package domain

import (
	"errors"
	"fmt"
	"time"
)

// WorkingHours представляет рабочие часы для проверок монитора.
type WorkingHours struct {
	// Start время начала рабочих часов (например, 09:00)
	Start time.Time
	// End время окончания рабочих часов (например, 18:00)
	End time.Time
	// Days рабочие дни недели
	Days []time.Weekday
}

// NewWorkingHours создаёт новые рабочие часы.
func NewWorkingHours(start, end string, days []time.Weekday) (*WorkingHours, error) {
	startTime, err := parseTime(start)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}

	endTime, err := parseTime(end)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}

	if endTime.Before(startTime) || endTime.Equal(startTime) {
		return nil, ErrInvalidWorkingHours
	}

	return &WorkingHours{
		Start: startTime,
		End:   endTime,
		Days:  days,
	}, nil
}

// parseTime парсит время в формате "15:04".
func parseTime(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}

// IsWithinHours возвращает true, если указанное время в рабочих часах.
func (wh *WorkingHours) IsWithinHours(t time.Time) bool {
	if wh == nil {
		return true // Нет ограничений
	}

	// Проверяем время
	currentTime := time.Date(0, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
	startTime := time.Date(0, 1, 1, wh.Start.Hour(), wh.Start.Minute(), 0, 0, time.UTC)
	endTime := time.Date(0, 1, 1, wh.End.Hour(), wh.End.Minute(), 0, 0, time.UTC)

	if currentTime.Before(startTime) || currentTime.After(endTime) {
		return false
	}

	// Проверяем день недели
	if len(wh.Days) > 0 {
		isWorkingDay := false
		for _, d := range wh.Days {
			if t.Weekday() == d {
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

// NextCheckTime возвращает ближайшее время для следующей проверки
// с учётом рабочих часов.
func (wh *WorkingHours) NextCheckTime(after time.Time) time.Time {
	if wh == nil {
		return after // Нет ограничений
	}

	// Если текущее время в рабочих часах - проверяем сейчас
	if wh.IsWithinHours(after) {
		return after
	}

	// Иначе находим ближайший рабочий час
	candidate := after

	// Проверяем следующие 7 дней
	for i := 0; i < 7; i++ {
		dayTime := candidate.Add(time.Duration(i) * 24 * time.Hour)

		// Устанавливаем время на начало рабочих часов
		checkTime := time.Date(
			dayTime.Year(), dayTime.Month(), dayTime.Day(),
			wh.Start.Hour(), wh.Start.Minute(), 0, 0,
			dayTime.Location(),
		)

		if wh.IsWithinHours(checkTime) {
			return checkTime
		}
	}

	// Если не нашли (не должно происходить) - возвращаем исходное время
	return after
}

// GetStartEndTime возвращает время начала и конца для хранения в Monitor.
func (wh *WorkingHours) GetStartEndTime() (*time.Time, *time.Time) {
	if wh == nil {
		return nil, nil
	}

	start := time.Date(0, 1, 1, wh.Start.Hour(), wh.Start.Minute(), 0, 0, time.UTC)
	end := time.Date(0, 1, 1, wh.End.Hour(), wh.End.Minute(), 0, 0, time.UTC)

	return &start, &end
}

var (
	ErrInvalidWorkingHours = errors.New("end time must be after start time")
	ErrInvalidTimeFormat   = errors.New("time must be in format '15:04'")
)

// WorkingDaysFromString парсит строку с рабочими днями.
// Например: "mon,tue,wed,thu,fri"
func WorkingDaysFromString(s string) ([]time.Weekday, error) {
	if s == "" {
		return nil, nil // Нет ограничений
	}

	dayMap := map[string]time.Weekday{
		"sun": time.Sunday,
		"mon": time.Monday,
		"tue": time.Tuesday,
		"wed": time.Wednesday,
		"thu": time.Thursday,
		"fri": time.Friday,
		"sat": time.Saturday,
	}

	days := make([]time.Weekday, 0)
	for _, part := range []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"} {
		if contains(s, part) {
			days = append(days, dayMap[part])
		}
	}

	if len(days) == 0 {
		return nil, ErrInvalidWorkingDays
	}

	return days, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// WorkingDaysToString конвертирует рабочие дни в строку.
func WorkingDaysToString(days []time.Weekday) string {
	if len(days) == 0 {
		return ""
	}

	dayNames := map[time.Weekday]string{
		time.Sunday:    "sun",
		time.Monday:    "mon",
		time.Tuesday:   "tue",
		time.Wednesday: "wed",
		time.Thursday:  "thu",
		time.Friday:    "fri",
		time.Saturday:  "sat",
	}

	result := ""
	for _, d := range days {
		result += dayNames[d] + ","
	}

	// Удаляем последнюю запятую
	if len(result) > 0 {
		result = result[:len(result)-1]
	}

	return result
}

var (
	ErrInvalidWorkingDays = errors.New("invalid working days format")
)
