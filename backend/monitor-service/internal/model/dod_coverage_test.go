package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInvalidStatusTransitionError_Methods тестирует методы InvalidStatusTransitionError.
func TestInvalidStatusTransitionError_Methods(t *testing.T) {
	t.Parallel()
	t.Run("Error() returns formatted error message", func(t *testing.T) {
		err := &InvalidStatusTransitionError{
			From: StatusUp,
			To:   StatusPaused,
		}

		msg := err.Error()
		assert.Contains(t, msg, "invalid status transition")
		assert.Contains(t, msg, "UP")
		assert.Contains(t, msg, "PAUSED")
	})

	t.Run("Error() formats all status types correctly", func(t *testing.T) {
		testCases := []struct {
			from MonitorStatus
			to   MonitorStatus
		}{
			{StatusPending, StatusPaused},
			{StatusUp, StatusDown},
			{StatusDown, StatusDegraded},
			{StatusDegraded, StatusUp},
			{StatusPaused, StatusUp},
		}

		for _, tc := range testCases {
			t.Run(tc.from.String()+"_to_"+tc.to.String(), func(t *testing.T) {
				err := &InvalidStatusTransitionError{
					From: tc.from,
					To:   tc.to,
				}

				msg := err.Error()
				assert.Contains(t, msg, tc.from.String())
				assert.Contains(t, msg, tc.to.String())
			})
		}
	})

	t.Run("Is() returns true for same error type", func(t *testing.T) {
		err1 := &InvalidStatusTransitionError{
			From: StatusUp,
			To:   StatusPaused,
		}
		err2 := &InvalidStatusTransitionError{
			From: StatusDown,
			To:   StatusPaused,
		}

		assert.True(t, errors.Is(err1, err2))
	})

	t.Run("Is() returns false for different error types", func(t *testing.T) {
		statusErr := &InvalidStatusTransitionError{
			From: StatusUp,
			To:   StatusPaused,
		}
		otherErr := errors.New("some other error")

		assert.False(t, errors.Is(statusErr, otherErr))
	})

	t.Run("Is() returns false for nil target", func(t *testing.T) {
		statusErr := &InvalidStatusTransitionError{
			From: StatusUp,
			To:   StatusPaused,
		}

		assert.False(t, errors.Is(statusErr, nil))
	})
}

// TestMonitorStatus_IsHealthy тестирует метод IsHealthy.
func TestMonitorStatus_IsHealthy(t *testing.T) {
	t.Parallel()
	t.Run("returns true for UP status", func(t *testing.T) {
		assert.True(t, StatusUp.IsHealthy())
	})

	t.Run("returns true for PENDING status", func(t *testing.T) {
		assert.True(t, StatusPending.IsHealthy())
	})

	t.Run("returns false for DOWN status", func(t *testing.T) {
		assert.False(t, StatusDown.IsHealthy())
	})

	t.Run("returns false for DEGRADED status", func(t *testing.T) {
		assert.False(t, StatusDegraded.IsHealthy())
	})

	t.Run("returns false for PAUSED status", func(t *testing.T) {
		assert.False(t, StatusPaused.IsHealthy())
	})
}

// TestMonitorStatus_IsUnhealthy тестирует метод IsUnhealthy.
func TestMonitorStatus_IsUnhealthy(t *testing.T) {
	t.Parallel()
	t.Run("returns true for DOWN status", func(t *testing.T) {
		assert.True(t, StatusDown.IsUnhealthy())
	})

	t.Run("returns true for DEGRADED status", func(t *testing.T) {
		assert.True(t, StatusDegraded.IsUnhealthy())
	})

	t.Run("returns false for UP status", func(t *testing.T) {
		assert.False(t, StatusUp.IsUnhealthy())
	})

	t.Run("returns false for PENDING status", func(t *testing.T) {
		assert.False(t, StatusPending.IsUnhealthy())
	})

	t.Run("returns false for PAUSED status", func(t *testing.T) {
		assert.False(t, StatusPaused.IsUnhealthy())
	})
}

// TestUptimeStats_AddIncidents тестирует метод AddIncidents.
func TestUptimeStats_AddIncidents(t *testing.T) {
	t.Parallel()
	t.Run("adds incidents with duration", func(t *testing.T) {
		stats := NewUptimeStats()

		duration1 := 3600
		duration2 := 7200
		incidents := []Incident{
			{
				ID:              uuid.New(),
				StartTime:       time.Now().Add(-2 * time.Hour),
				DurationSeconds: &duration1,
			},
			{
				ID:              uuid.New(),
				StartTime:       time.Now().Add(-1 * time.Hour),
				DurationSeconds: &duration2,
			},
		}

		stats.AddIncidents(incidents)

		assert.Equal(t, 2, stats.Incidents)
		assert.Equal(t, 3*time.Hour, stats.TotalDowntime)
	})

	t.Run("handles incidents without duration", func(t *testing.T) {
		stats := NewUptimeStats()

		incidents := []Incident{
			{
				ID:              uuid.New(),
				StartTime:       time.Now().Add(-1 * time.Hour),
				EndTime:         nil,
				DurationSeconds: nil,
			},
		}

		stats.AddIncidents(incidents)

		assert.Equal(t, 1, stats.Incidents)
		assert.Equal(t, time.Duration(0), stats.TotalDowntime)
	})

	t.Run("handles empty incidents slice", func(t *testing.T) {
		stats := NewUptimeStats()

		stats.AddIncidents([]Incident{})

		assert.Equal(t, 0, stats.Incidents)
		assert.Equal(t, time.Duration(0), stats.TotalDowntime)
	})

	t.Run("handles mixed incidents (with and without duration)", func(t *testing.T) {
		stats := NewUptimeStats()

		duration1 := 1800
		incidents := []Incident{
			{
				ID:              uuid.New(),
				StartTime:       time.Now().Add(-1 * time.Hour),
				DurationSeconds: &duration1,
			},
			{
				ID:              uuid.New(),
				StartTime:       time.Now().Add(-30 * time.Minute),
				DurationSeconds: nil,
			},
		}

		stats.AddIncidents(incidents)

		assert.Equal(t, 2, stats.Incidents)
		assert.Equal(t, 30*time.Minute, stats.TotalDowntime)
	})
}

// TestUptimeStats_GetAvailabilityString тестирует метод GetAvailabilityString.
func TestUptimeStats_GetAvailabilityString(t *testing.T) {
	t.Parallel()
	t.Run("returns note when note is set", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Note = "No checks in period"

		result := stats.GetAvailabilityString()
		assert.Equal(t, "No checks in period", result)
	})

	t.Run("returns Excellent for uptime >= 99.9%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 99.95
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Excellent", result)
	})

	t.Run("returns Good for uptime >= 99.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 99.5
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Good", result)
	})

	t.Run("returns Fair for uptime >= 95.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 97.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Fair", result)
	})

	t.Run("returns Poor for uptime >= 90.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 92.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Poor", result)
	})

	t.Run("returns Critical for uptime < 90.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 85.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Critical", result)
	})

	t.Run("returns Critical for very low uptime", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 0.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Critical", result)
	})

	t.Run("boundary test at 99.9%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 99.9
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Excellent", result)
	})

	t.Run("boundary test at 99.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 99.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Good", result)
	})

	t.Run("boundary test at 95.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 95.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Fair", result)
	})

	t.Run("boundary test at 90.0%", func(t *testing.T) {
		stats := NewUptimeStats()
		stats.Uptime = 90.0
		stats.Note = ""

		result := stats.GetAvailabilityString()
		assert.Equal(t, "Poor", result)
	})
}

// TestParseUptimePeriod тестирует функцию ParseUptimePeriod.
func TestParseUptimePeriod(t *testing.T) {
	t.Parallel()
	t.Run("parses 1h period", func(t *testing.T) {
		from, to, err := ParseUptimePeriod(PeriodLastHour, time.Time{}, time.Time{})

		require.NoError(t, err)
		assert.False(t, from.IsZero())
		assert.False(t, to.IsZero())
		assert.WithinDuration(t, time.Now().Add(-1*time.Hour), from, 1*time.Second)
		assert.WithinDuration(t, time.Now(), to, 1*time.Second)
	})

	t.Run("parses 24h period", func(t *testing.T) {
		from, to, err := ParseUptimePeriod(PeriodLast24h, time.Time{}, time.Time{})

		require.NoError(t, err)
		assert.False(t, from.IsZero())
		assert.False(t, to.IsZero())
		assert.WithinDuration(t, time.Now().Add(-24*time.Hour), from, 1*time.Second)
		assert.WithinDuration(t, time.Now(), to, 1*time.Second)
	})

	t.Run("parses 7d period", func(t *testing.T) {
		from, to, err := ParseUptimePeriod(PeriodLast7Days, time.Time{}, time.Time{})

		require.NoError(t, err)
		assert.False(t, from.IsZero())
		assert.False(t, to.IsZero())
		assert.WithinDuration(t, time.Now().Add(-7*24*time.Hour), from, 1*time.Second)
		assert.WithinDuration(t, time.Now(), to, 1*time.Second)
	})

	t.Run("parses 30d period", func(t *testing.T) {
		from, to, err := ParseUptimePeriod(PeriodLast30Days, time.Time{}, time.Time{})

		require.NoError(t, err)
		assert.False(t, from.IsZero())
		assert.False(t, to.IsZero())
		assert.WithinDuration(t, time.Now().Add(-30*24*time.Hour), from, 1*time.Second)
		assert.WithinDuration(t, time.Now(), to, 1*time.Second)
	})

	t.Run("parses custom period with valid from/to", func(t *testing.T) {
		customFrom := time.Now().Add(-48 * time.Hour)
		customTo := time.Now().Add(-24 * time.Hour)

		from, to, err := ParseUptimePeriod(PeriodCustom, customFrom, customTo)

		require.NoError(t, err)
		assert.Equal(t, customFrom, from)
		assert.Equal(t, customTo, to)
	})

	t.Run("returns error for custom period with zero from", func(t *testing.T) {
		customTo := time.Now()

		_, _, err := ParseUptimePeriod(PeriodCustom, time.Time{}, customTo)

		require.Error(t, err)
		assert.Equal(t, ErrInvalidCustomPeriod, err)
	})

	t.Run("returns error for custom period with zero to", func(t *testing.T) {
		customFrom := time.Now()

		_, _, err := ParseUptimePeriod(PeriodCustom, customFrom, time.Time{})

		require.Error(t, err)
		assert.Equal(t, ErrInvalidCustomPeriod, err)
	})

	t.Run("returns error for custom period with both zero", func(t *testing.T) {
		_, _, err := ParseUptimePeriod(PeriodCustom, time.Time{}, time.Time{})

		require.Error(t, err)
		assert.Equal(t, ErrInvalidCustomPeriod, err)
	})

	t.Run("returns error for invalid period", func(t *testing.T) {
		_, _, err := ParseUptimePeriod("invalid", time.Time{}, time.Time{})

		require.Error(t, err)
		assert.Equal(t, ErrInvalidPeriod, err)
	})
}

// TestCheckResult_CalculateFailureRate_EdgeCases тестирует граничные случаи CalculateFailureRate.
func TestCheckResult_CalculateFailureRate_EdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("zero total checks", func(t *testing.T) {
		results := CheckResultSlice{}
		rate := results.CalculateFailureRate()
		assert.Equal(t, 0.0, rate)
	})

	t.Run("all checks failed", func(t *testing.T) {
		results := CheckResultSlice{
			{Status: StatusDown},
			{Status: StatusDown},
			{Status: StatusDown},
		}
		rate := results.CalculateFailureRate()
		assert.Equal(t, 100.0, rate)
	})

	t.Run("all checks passed", func(t *testing.T) {
		results := CheckResultSlice{
			{Status: StatusUp},
			{Status: StatusUp},
			{Status: StatusUp},
		}
		rate := results.CalculateFailureRate()
		assert.Equal(t, 0.0, rate)
	})

	t.Run("mix of passed and failed", func(t *testing.T) {
		results := CheckResultSlice{
			{Status: StatusUp},
			{Status: StatusDown},
			{Status: StatusUp},
			{Status: StatusDown},
		}
		rate := results.CalculateFailureRate()
		assert.Equal(t, 50.0, rate)
	})

	t.Run("includes DEGRADED as passed", func(t *testing.T) {
		results := CheckResultSlice{
			{Status: StatusUp},
			{Status: StatusDegraded},
			{Status: StatusDegraded},
			{Status: StatusDown},
		}
		rate := results.CalculateFailureRate()
		assert.Equal(t, 25.0, rate)
	})

	t.Run("PAUSED counts as not failed", func(t *testing.T) {
		results := CheckResultSlice{
			{Status: StatusUp},
			{Status: StatusPaused},
			{Status: StatusPaused},
			{Status: StatusDown},
		}
		rate := results.CalculateFailureRate()
		assert.Equal(t, 25.0, rate) // 1 failure out of 4 total checks
	})
}

// TestIncident_GetDuration_EdgeCases тестирует граничные случаи GetDuration.
func TestIncident_GetDuration_EdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("returns duration when DurationSeconds is set", func(t *testing.T) {
		duration := 3600
		incident := Incident{
			ID:              uuid.New(),
			StartTime:       time.Now().Add(-2 * time.Hour),
			DurationSeconds: &duration,
		}

		result := incident.GetDuration()
		assert.Equal(t, 1*time.Hour, result)
	})

	t.Run("calculates duration from EndTime when DurationSeconds is nil", func(t *testing.T) {
		startTime := time.Now().Add(-2 * time.Hour)
		endTime := time.Now().Add(-1 * time.Hour)
		incident := Incident{
			ID:        uuid.New(),
			StartTime: startTime,
			EndTime:   &endTime,
		}

		result := incident.GetDuration()
		assert.InDelta(t, float64(1*time.Hour), float64(result), float64(1*time.Second))
	})

	t.Run("calculates duration from now for active incident", func(t *testing.T) {
		startTime := time.Now().Add(-30 * time.Minute)
		incident := Incident{
			ID:              uuid.New(),
			StartTime:       startTime,
			EndTime:         nil,
			DurationSeconds: nil,
		}

		result := incident.GetDuration()
		assert.InDelta(t, float64(30*time.Minute), float64(result), float64(1*time.Second))
	})

	t.Run("handles zero start time", func(t *testing.T) {
		incident := Incident{
			ID:              uuid.New(),
			StartTime:       time.Time{},
			EndTime:         nil,
			DurationSeconds: nil,
		}

		result := incident.GetDuration()
		assert.True(t, result > 0) // time.Since would return positive duration
	})
}

// TestMonitorStatus_CanTransitionTo_AllCombinations тестирует все комбинации переходов.
func TestMonitorStatus_CanTransitionTo_AllCombinations(t *testing.T) {
	t.Parallel()
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

	for _, from := range allStatuses {
		for _, to := range allStatuses {
			t.Run(from.String()+"_to_"+to.String(), func(t *testing.T) {
				result := from.CanTransitionTo(to)

				validTargets := validTransitions[from]
				isValid := false
				for _, valid := range validTargets {
					if valid == to {
						isValid = true
						break
					}
				}

				assert.Equal(t, isValid, result,
					"Transition %s -> %s should be %v", from, to, isValid)
			})
		}
	}
}
