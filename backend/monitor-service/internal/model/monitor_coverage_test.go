package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMonitor_ShouldCheckNow_WorkingHours тестирует логику рабочих часов.
func TestMonitor_ShouldCheckNow_WorkingHours(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	t.Run("always check when no working hours set", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusUp
		monitor.WorkingHoursStart = nil
		monitor.WorkingHoursEnd = nil

		assert.True(t, monitor.ShouldCheckNow())
	})

	t.Run("respect working hours when set", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusUp

		// Set working hours: 9 AM to 6 PM
		now := time.Now()
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC)
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.UTC)

		monitor.WorkingHoursStart = &startTime
		monitor.WorkingHoursEnd = &endTime

		// Should check only during working hours
		currentHour := time.Now().UTC().Hour()
		isWorkingHours := currentHour >= 9 && currentHour < 18
		assert.Equal(t, isWorkingHours, monitor.ShouldCheckNow())
	})

	t.Run("check all days when no working days set", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusUp

		now := time.Now()
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

		monitor.WorkingHoursStart = &startTime
		monitor.WorkingHoursEnd = &endTime
		monitor.WorkingDays = nil // No working days restriction

		assert.True(t, monitor.ShouldCheckNow())
	})

	t.Run("respect working days when set", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusUp

		now := time.Now()
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

		monitor.WorkingHoursStart = &startTime
		monitor.WorkingHoursEnd = &endTime

		// Set only weekdays as working days
		monitor.WorkingDays = []time.Weekday{
			time.Monday, time.Tuesday, time.Wednesday,
			time.Thursday, time.Friday,
		}

		currentDay := time.Now().Weekday()
		isWeekday := currentDay >= time.Monday && currentDay <= time.Friday
		assert.Equal(t, isWeekday, monitor.ShouldCheckNow())
	})

	t.Run("paused monitor never checks", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusPaused

		now := time.Now()
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)

		monitor.WorkingHoursStart = &startTime
		monitor.WorkingHoursEnd = &endTime
		monitor.WorkingDays = []time.Weekday{time.Now().Weekday()}

		assert.False(t, monitor.ShouldCheckNow())
	})
}

// TestMonitor_IsWithinWorkingHours тестирует проверку рабочих часов.
func TestMonitor_IsWithinWorkingHours(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	t.Run("delegates to ShouldCheckNow", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusUp

		// Test with no restrictions
		monitor.WorkingHoursStart = nil
		monitor.WorkingHoursEnd = nil

		assert.True(t, monitor.IsWithinWorkingHours())
		assert.Equal(t, monitor.IsWithinWorkingHours(), monitor.ShouldCheckNow())
	})
}

// TestMonitor_GetDegradedThresholds тестирует получение порогов DEGRADED.
func TestMonitor_GetDegradedThresholds(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	t.Run("returns custom thresholds when set", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		customResponseTime := 2000
		customFailureRate := 75
		monitor.DegradedResponseTimeThreshold = &customResponseTime
		monitor.DegradedFailureRateThreshold = &customFailureRate

		rt, fr := monitor.GetDegradedThresholds()
		assert.Equal(t, 2000, rt)
		assert.Equal(t, 75, fr)
	})

	t.Run("returns default thresholds when not set", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.DegradedResponseTimeThreshold = nil
		monitor.DegradedFailureRateThreshold = nil

		rt, fr := monitor.GetDegradedThresholds()
		assert.Equal(t, 1000, rt) // Default 1 second
		assert.Equal(t, 50, fr)   // Default 50%
	})

	t.Run("returns mixed custom and default thresholds", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		customResponseTime := 3000
		monitor.DegradedResponseTimeThreshold = &customResponseTime
		monitor.DegradedFailureRateThreshold = nil

		rt, fr := monitor.GetDegradedThresholds()
		assert.Equal(t, 3000, rt) // Custom
		assert.Equal(t, 50, fr)   // Default

		monitor.DegradedResponseTimeThreshold = nil
		customFailureRate := 80
		monitor.DegradedFailureRateThreshold = &customFailureRate

		rt, fr = monitor.GetDegradedThresholds()
		assert.Equal(t, 1000, rt) // Default
		assert.Equal(t, 80, fr)   // Custom
	})

	t.Run("zero threshold is valid", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

		zeroValue := 0
		monitor.DegradedResponseTimeThreshold = &zeroValue
		monitor.DegradedFailureRateThreshold = &zeroValue

		rt, fr := monitor.GetDegradedThresholds()
		assert.Equal(t, 0, rt)
		assert.Equal(t, 0, fr)
	})
}

// TestMonitor_UpdateStatus_AllTransitions тестирует все переходы статусов.
func TestMonitor_UpdateStatus_AllTransitions(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	allStatuses := []MonitorStatus{
		StatusPending,
		StatusUp,
		StatusDown,
		StatusDegraded,
		StatusPaused,
	}

	validTransitions := map[MonitorStatus][]MonitorStatus{
		StatusPending:  {StatusUp, StatusDown, StatusDegraded, StatusPaused},
		StatusUp:       {StatusDown, StatusDegraded, StatusPaused},
		StatusDown:     {StatusUp, StatusDegraded, StatusPaused},
		StatusDegraded: {StatusUp, StatusDown, StatusPaused},
		StatusPaused:   {StatusUp},
	}

	for _, fromStatus := range allStatuses {
		for _, toStatus := range allStatuses {
			t.Run(fromStatus.String()+"_to_"+toStatus.String(), func(t *testing.T) {
				monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
				monitor.Status = fromStatus

				err := monitor.UpdateStatus(toStatus)

				// Check if transition is valid
				validTargets := validTransitions[fromStatus]
				isValid := false
				for _, valid := range validTargets {
					if valid == toStatus {
						isValid = true
						break
					}
				}

				if isValid {
					assert.NoError(t, err)
					assert.Equal(t, toStatus, monitor.Status)
				} else {
					assert.Error(t, err)
					assert.Equal(t, fromStatus, monitor.Status)
				}
			})
		}
	}
}

// TestMonitor_UpdateStatus_UpdatedAt тестирует обновление времени.
func TestMonitor_UpdateStatus_UpdatedAt(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	oldUpdatedAt := monitor.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	err := monitor.UpdateStatus(StatusUp)

	require.NoError(t, err)
	assert.True(t, monitor.UpdatedAt.After(oldUpdatedAt))
}

// TestNewMonitor_DefaultValues тестирует значения по умолчанию.
func TestNewMonitor_DefaultValues(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	monitor, err := NewMonitor(userID, "Test Monitor", "https://example.com", 60)

	require.NoError(t, err)

	assert.Equal(t, "HTTP", monitor.CheckType)
	assert.Equal(t, 30, monitor.TimeoutSeconds)
	assert.Equal(t, StatusPending, monitor.Status)
	assert.NotNil(t, monitor.WorkingDays)
	assert.Equal(t, 7, len(monitor.WorkingDays)) // All 7 days
	assert.NotEqual(t, uuid.Nil, monitor.ID)
	assert.False(t, monitor.CreatedAt.IsZero())
	assert.False(t, monitor.UpdatedAt.IsZero())
}

// TestNewMonitor_URLValidation тестирует валидацию URL.
func TestNewMonitor_URLValidation(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	t.Run("rejects URLs that are too long", func(t *testing.T) {
		longURL := "https://example.com/" + string(make([]byte, 2048))
		_, err := NewMonitor(userID, "Test", longURL, 60)
		assert.Error(t, err)
	})

	t.Run("accepts URLs with port", func(t *testing.T) {
		monitor, err := NewMonitor(userID, "Test", "https://example.com:8080", 60)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com:8080", monitor.URL)
	})

	t.Run("accepts URLs with path", func(t *testing.T) {
		monitor, err := NewMonitor(userID, "Test", "https://example.com/api/v1", 60)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/api/v1", monitor.URL)
	})

	t.Run("accepts URLs with query", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "https://example.com?query=value", 60)
		assert.NoError(t, err)
	})

	t.Run("rejects URL without protocol", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "example.com", 60)
		assert.Error(t, err)
	})

	t.Run("rejects URL with wrong protocol", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "ftp://example.com", 60)
		assert.Error(t, err)
	})

	t.Run("accepts http and https protocols", func(t *testing.T) {
		monitor1, err1 := NewMonitor(userID, "Test1", "http://example.com", 60)
		assert.NoError(t, err1)
		assert.Equal(t, "http://example.com", monitor1.URL)

		monitor2, err2 := NewMonitor(userID, "Test2", "https://example.com", 60)
		assert.NoError(t, err2)
		assert.Equal(t, "https://example.com", monitor2.URL)
	})
}

// TestNewMonitor_IntervalValidation тестирует валидацию интервала.
func TestNewMonitor_IntervalValidation(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	t.Run("rejects interval below minimum", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "https://example.com", 29)
		assert.Error(t, err)
	})

	t.Run("accepts minimum interval", func(t *testing.T) {
		monitor, err := NewMonitor(userID, "Test", "https://example.com", 30)
		assert.NoError(t, err)
		assert.Equal(t, 30, monitor.IntervalSeconds)
	})

	t.Run("accepts maximum interval", func(t *testing.T) {
		monitor, err := NewMonitor(userID, "Test", "https://example.com", 3600)
		assert.NoError(t, err)
		assert.Equal(t, 3600, monitor.IntervalSeconds)
	})

	t.Run("rejects interval above maximum", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "https://example.com", 3601)
		assert.Error(t, err)
	})

	t.Run("rejects negative interval", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "https://example.com", -100)
		assert.Error(t, err)
	})

	t.Run("rejects zero interval", func(t *testing.T) {
		_, err := NewMonitor(userID, "Test", "https://example.com", 0)
		assert.Error(t, err)
	})
}
