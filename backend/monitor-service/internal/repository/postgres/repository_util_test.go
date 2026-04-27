// Package postgres предоставляет реализацию репозиториев для работы с PostgreSQL.
package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkingDaysToString тестирует конвертацию []time.Weekday в []string.
func TestWorkingDaysToString(t *testing.T) {
	tests := []struct {
		name     string
		days     []time.Weekday
		expected []string
	}{
		{
			name:     "all weekdays",
			days:     []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			expected: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
		},
		{
			name:     "weekend",
			days:     []time.Weekday{time.Saturday, time.Sunday},
			expected: []string{"Saturday", "Sunday"},
		},
		{
			name:     "all days",
			days:     []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday},
			expected: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		},
		{
			name:     "empty days",
			days:     []time.Weekday{},
			expected: nil,
		},
		{
			name:     "nil days",
			days:     nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workingDaysToString(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStringToWorkingDays тестирует конвертацию []string в []time.Weekday.
func TestStringToWorkingDays(t *testing.T) {
	tests := []struct {
		name     string
		days     []string
		expected []time.Weekday
	}{
		{
			name:     "all weekdays",
			days:     []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			expected: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		},
		{
			name:     "weekend",
			days:     []string{"Saturday", "Sunday"},
			expected: []time.Weekday{time.Saturday, time.Sunday},
		},
		{
			name:     "all days",
			days:     []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
			expected: []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday},
		},
		{
			name:     "empty days",
			days:     []string{},
			expected: nil,
		},
		{
			name:     "nil days",
			days:     nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringToWorkingDays(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWorkingDaysConversion_Roundtrip тестирует двустороннюю конвертацию рабочих дней.
func TestWorkingDaysConversion_Roundtrip(t *testing.T) {
	originalDays := []time.Weekday{time.Monday, time.Wednesday, time.Friday}

	// Конвертируем в строки и обратно
	strings := workingDaysToString(originalDays)
	require.NotNil(t, strings)

	convertedBack := stringToWorkingDays(strings)
	require.NotNil(t, convertedBack)

	assert.Equal(t, originalDays, convertedBack)
}

// TestMonitorRepositoryHelpers тестирует helper функции monitor repository.
func TestMonitorRepositoryHelpers(t *testing.T) {
	t.Run("workingDaysToString with single day", func(t *testing.T) {
		days := []time.Weekday{time.Monday}
		result := workingDaysToString(days)
		assert.Equal(t, []string{"Monday"}, result)
	})

	t.Run("stringToWorkingDays with single day", func(t *testing.T) {
		days := []string{"Monday"}
		result := stringToWorkingDays(days)
		assert.Equal(t, []time.Weekday{time.Monday}, result)
	})

	t.Run("workingDaysToString preserves order", func(t *testing.T) {
		days := []time.Weekday{time.Friday, time.Monday, time.Wednesday}
		result := workingDaysToString(days)
		assert.Equal(t, []string{"Friday", "Monday", "Wednesday"}, result)
	})

	t.Run("stringToWorkingDays preserves order", func(t *testing.T) {
		days := []string{"Friday", "Monday", "Wednesday"}
		result := stringToWorkingDays(days)
		assert.Equal(t, []time.Weekday{time.Friday, time.Monday, time.Wednesday}, result)
	})
}

// TestNewMonitorRepository тестирует создание monitor repository.
func TestNewMonitorRepository_Constructor(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		repo := NewMonitorRepository(nil)
		assert.NotNil(t, repo)
	})
}

// TestNewCheckResultRepository_Constructor тестирует создание check result repository.
func TestNewCheckResultRepository_Constructor(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		repo := NewCheckResultRepository(nil)
		assert.NotNil(t, repo)
	})
}

// TestNewIncidentRepository_Constructor тестирует создание incident repository.
func TestNewIncidentRepository_Constructor(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		repo := NewIncidentRepository(nil)
		assert.NotNil(t, repo)
	})
}

// TestNewAuditRepository_Constructor тестирует создание audit repository.
func TestNewAuditRepository_Constructor(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		repo := NewAuditRepository(nil)
		assert.NotNil(t, repo)
	})
}
