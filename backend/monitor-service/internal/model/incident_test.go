package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewIncident(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	incident := NewIncident(monitorID)

	if incident.ID == uuid.Nil {
		t.Error("Incident ID should not be nil")
	}

	if incident.MonitorID != monitorID {
		t.Errorf("Incident MonitorID = %v, want %v", incident.MonitorID, monitorID)
	}

	if incident.Status != IncidentActive {
		t.Errorf("Incident Status = %v, want %v", incident.Status, IncidentActive)
	}

	if incident.StartTime.IsZero() {
		t.Error("Incident StartTime should be set")
	}

	if incident.EndTime != nil {
		t.Error("Incident EndTime should be nil for new incident")
	}

	if incident.DurationSeconds != nil {
		t.Error("Incident DurationSeconds should be nil for new incident")
	}
}

func TestIncident_Resolve(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	incident := NewIncident(monitorID)

	startTime := incident.StartTime
	time.Sleep(10 * time.Millisecond)
	endTime := time.Now()

	incident.Resolve(endTime)

	if incident.Status != IncidentResolved {
		t.Errorf("Incident Status = %v, want %v", incident.Status, IncidentResolved)
	}

	if incident.EndTime == nil {
		t.Error("Incident EndTime should be set after resolve")
	}

	if incident.DurationSeconds == nil {
		t.Error("Incident DurationSeconds should be calculated after resolve")
	}

	if !incident.StartTime.Equal(startTime) {
		t.Error("Incident StartTime should not change after resolve")
	}

	duration := endTime.Sub(startTime).Seconds()
	if incident.DurationSeconds != nil && *incident.DurationSeconds != int(duration) {
		t.Errorf("Incident DurationSeconds = %v, want %v", *incident.DurationSeconds, int(duration))
	}
}

func TestIncident_IsActive(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	t.Run("new incident is active", func(t *testing.T) {
		incident := NewIncident(monitorID)
		if !incident.IsActive() {
			t.Error("New incident should be active")
		}
	})

	t.Run("resolved incident is not active", func(t *testing.T) {
		incident := NewIncident(monitorID)
		incident.Resolve(time.Now())
		if incident.IsActive() {
			t.Error("Resolved incident should not be active")
		}
	})
}

func TestIncidentSlice_FilterActive(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	now := time.Now()

	active1 := NewIncident(monitorID)
	active2 := NewIncident(monitorID)

	resolved := NewIncident(monitorID)
	resolved.Resolve(now.Add(1 * time.Hour))

	incidents := IncidentSlice{*active1, *active2, *resolved}

	filtered := incidents.FilterActive()

	if len(filtered) != 2 {
		t.Errorf("FilterActive() returned %d incidents, want 2", len(filtered))
	}

	for _, inc := range filtered {
		if !inc.IsActive() {
			t.Error("Filtered incidents should all be active")
		}
	}
}

func TestIncident_GetDuration(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	incident := NewIncident(monitorID)

	// For active incident, duration is time since start
	duration := incident.GetDuration()
	if duration == 0 {
		t.Error("Active incident should have a duration")
	}

	// For resolved incident, duration is calculated
	endTime := time.Now().Add(1 * time.Hour)
	incident.Resolve(endTime)

	duration = incident.GetDuration()
	expected := endTime.Sub(incident.StartTime)

	// Allow small difference due to time calculations
	if duration < expected-100*time.Millisecond || duration > expected+100*time.Millisecond {
		t.Errorf("GetDuration() = %v, want %v", duration, expected)
	}
}

// TestIncident_IsResolved тестирует проверку на закрытый инцидент.
func TestIncident_IsResolved(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	t.Run("new incident is not resolved", func(t *testing.T) {
		incident := NewIncident(monitorID)
		if incident.IsResolved() {
			t.Error("New incident should not be resolved")
		}
	})

	t.Run("resolved incident is resolved", func(t *testing.T) {
		incident := NewIncident(monitorID)
		incident.Resolve(time.Now())
		if !incident.IsResolved() {
			t.Error("Resolved incident should be resolved")
		}
	})
}

// TestIncident_GetDuration_Variations тестирует вычисление длительности в различных сценариях.
func TestIncident_GetDuration_Variations(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	t.Run("active incident with no EndTime", func(t *testing.T) {
		incident := NewIncident(monitorID)
		time.Sleep(10 * time.Millisecond)
		duration := incident.GetDuration()

		if duration < 10*time.Millisecond {
			t.Errorf("Active incident duration should be >= 10ms, got %v", duration)
		}
	})

	t.Run("resolved incident with DurationSeconds set", func(t *testing.T) {
		incident := NewIncident(monitorID)
		endTime := incident.StartTime.Add(2 * time.Hour)
		incident.Resolve(endTime)

		duration := incident.GetDuration()
		expected := 2 * time.Hour

		// Allow small difference
		if duration < expected-100*time.Millisecond || duration > expected+100*time.Millisecond {
			t.Errorf("GetDuration() = %v, want %v", duration, expected)
		}
	})

	t.Run("active incident duration increases over time", func(t *testing.T) {
		incident := NewIncident(monitorID)
		duration1 := incident.GetDuration()

		time.Sleep(10 * time.Millisecond)
		duration2 := incident.GetDuration()

		if duration2 <= duration1 {
			t.Error("Active incident duration should increase over time")
		}
	})
}

// TestIncidentSlice_FilterByPeriod тестирует фильтрацию по периоду.
func TestIncidentSlice_FilterByPeriod(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	baseTime := time.Now()

	t.Run("filters incidents overlapping with period", func(t *testing.T) {
		// Incident 1: completely inside period
		incident1 := NewIncident(monitorID)
		incident1.StartTime = baseTime.Add(2 * time.Hour)
		endTime1 := baseTime.Add(3 * time.Hour)
		incident1.Resolve(endTime1)

		// Incident 2: starts before period, ends inside
		incident2 := NewIncident(monitorID)
		incident2.StartTime = baseTime.Add(30 * time.Minute)
		endTime2 := baseTime.Add(2 * time.Hour)
		incident2.Resolve(endTime2)

		// Incident 3: starts inside period, ends after
		incident3 := NewIncident(monitorID)
		incident3.StartTime = baseTime.Add(3 * time.Hour)
		endTime3 := baseTime.Add(6 * time.Hour)
		incident3.Resolve(endTime3)

		// Incident 4: completely outside period (after)
		incident4 := NewIncident(monitorID)
		incident4.StartTime = baseTime.Add(8 * time.Hour)
		endTime4 := baseTime.Add(9 * time.Hour)
		incident4.Resolve(endTime4)

		// Incident 5: completely outside period (before)
		incident5 := NewIncident(monitorID)
		incident5.StartTime = baseTime.Add(-2 * time.Hour)
		endTime5 := baseTime.Add(-1 * time.Hour)
		incident5.Resolve(endTime5)

		// Incident 6: active incident (no EndTime) inside period
		incident6 := NewIncident(monitorID)
		incident6.StartTime = baseTime.Add(4 * time.Hour)

		incidents := IncidentSlice{*incident1, *incident2, *incident3, *incident4, *incident5, *incident6}

		// Filter for period [baseTime+1h, baseTime+5h]
		from := baseTime.Add(1 * time.Hour)
		to := baseTime.Add(5 * time.Hour)
		filtered := incidents.FilterByPeriod(from, to)

		// Should include incidents 1, 2, 3, and 6 (all overlap with period)
		// Should NOT include incidents 4 and 5 (completely outside)
		if len(filtered) != 4 {
			t.Errorf("FilterByPeriod() returned %d incidents, want 4", len(filtered))
		}
	})

	t.Run("empty slice returns empty", func(t *testing.T) {
		var incidents IncidentSlice
		from := baseTime
		to := baseTime.Add(1 * time.Hour)

		filtered := incidents.FilterByPeriod(from, to)

		if len(filtered) != 0 {
			t.Errorf("FilterByPeriod() on empty slice returned %d incidents, want 0", len(filtered))
		}
	})

	t.Run("incident at exact boundary", func(t *testing.T) {
		// Incident starts exactly at period end
		incident1 := NewIncident(monitorID)
		incident1.StartTime = baseTime.Add(5 * time.Hour)
		endTime1 := baseTime.Add(6 * time.Hour)
		incident1.Resolve(endTime1)

		// Incident ends exactly at period start
		incident2 := NewIncident(monitorID)
		incident2.StartTime = baseTime.Add(-1 * time.Hour)
		endTime2 := baseTime.Add(1 * time.Hour)
		incident2.Resolve(endTime2)

		incidents := IncidentSlice{*incident1, *incident2}

		// Filter for period [baseTime+1h, baseTime+5h]
		from := baseTime.Add(1 * time.Hour)
		to := baseTime.Add(5 * time.Hour)
		filtered := incidents.FilterByPeriod(from, to)

		// Both should be included (boundaries are inclusive)
		if len(filtered) != 2 {
			t.Errorf("FilterByPeriod() with boundary conditions returned %d incidents, want 2", len(filtered))
		}
	})
}

// TestIncidentSlice_CalculateTotalDowntime тестирует вычисление общего времени простоя.
func TestIncidentSlice_CalculateTotalDowntime(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	baseTime := time.Now()

	t.Run("sums downtime within period", func(t *testing.T) {
		// Incident 1: 1 hour inside period
		incident1 := NewIncident(monitorID)
		incident1.StartTime = baseTime.Add(2 * time.Hour)
		endTime1 := baseTime.Add(3 * time.Hour)
		incident1.Resolve(endTime1)

		// Incident 2: 30 minutes inside period
		incident2 := NewIncident(monitorID)
		incident2.StartTime = baseTime.Add(4 * time.Hour)
		endTime2 := baseTime.Add(4*time.Hour + 30*time.Minute)
		incident2.Resolve(endTime2)

		incidents := IncidentSlice{*incident1, *incident2}

		// Calculate for full period
		from := baseTime
		to := baseTime.Add(24 * time.Hour)
		total := incidents.CalculateTotalDowntime(from, to)

		expected := 1*time.Hour + 30*time.Minute
		if total < expected-100*time.Millisecond || total > expected+100*time.Millisecond {
			t.Errorf("CalculateTotalDowntime() = %v, want %v", total, expected)
		}
	})

	t.Run("clips incidents to period boundaries", func(t *testing.T) {
		// Incident starts before period, ends during
		incident1 := NewIncident(monitorID)
		incident1.StartTime = baseTime.Add(30 * time.Minute)
		endTime1 := baseTime.Add(2 * time.Hour)
		incident1.Resolve(endTime1)

		// Incident starts during period, ends after
		incident2 := NewIncident(monitorID)
		incident2.StartTime = baseTime.Add(4 * time.Hour)
		endTime2 := baseTime.Add(6 * time.Hour)
		incident2.Resolve(endTime2)

		incidents := IncidentSlice{*incident1, *incident2}

		// Calculate for period [baseTime+1h, baseTime+5h]
		from := baseTime.Add(1 * time.Hour)
		to := baseTime.Add(5 * time.Hour)
		total := incidents.CalculateTotalDowntime(from, to)

		// Incident 1: clipped to [baseTime+1h, baseTime+2h] = 1 hour
		// Incident 2: clipped to [baseTime+4h, baseTime+5h] = 1 hour
		expected := 2 * time.Hour
		if total < expected-100*time.Millisecond || total > expected+100*time.Millisecond {
			t.Errorf("CalculateTotalDowntime() with clipping = %v, want %v", total, expected)
		}
	})

	t.Run("handles active incidents", func(t *testing.T) {
		// Active incident (no EndTime)
		incident := NewIncident(monitorID)
		incident.StartTime = baseTime.Add(2 * time.Hour)
		// No Resolve() call - incident is active

		incidents := IncidentSlice{*incident}

		// Calculate for period [baseTime+1h, baseTime+5h]
		from := baseTime.Add(1 * time.Hour)
		to := baseTime.Add(5 * time.Hour)
		total := incidents.CalculateTotalDowntime(from, to)

		// Should count from incident start to period end: [baseTime+2h, baseTime+5h] = 3 hours
		expected := 3 * time.Hour
		if total < expected-100*time.Millisecond || total > expected+100*time.Millisecond {
			t.Errorf("CalculateTotalDowntime() with active incident = %v, want %v", total, expected)
		}
	})

	t.Run("empty slice returns zero", func(t *testing.T) {
		var incidents IncidentSlice
		from := baseTime
		to := baseTime.Add(1 * time.Hour)

		total := incidents.CalculateTotalDowntime(from, to)

		if total != 0 {
			t.Errorf("CalculateTotalDowntime() on empty slice = %v, want 0", total)
		}
	})
}

// TestIncidentSlice_Count тестирует подсчёт инцидентов.
func TestIncidentSlice_Count(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	t.Run("counts all incidents", func(t *testing.T) {
		incident1 := NewIncident(monitorID)
		incident2 := NewIncident(monitorID)
		incident3 := NewIncident(monitorID)

		incidents := IncidentSlice{*incident1, *incident2, *incident3}

		if incidents.Count() != 3 {
			t.Errorf("Count() = %d, want 3", incidents.Count())
		}
	})

	t.Run("empty slice returns zero", func(t *testing.T) {
		var incidents IncidentSlice

		if incidents.Count() != 0 {
			t.Errorf("Count() on empty slice = %d, want 0", incidents.Count())
		}
	})
}

// TestIncidentSlice_CountActive тестирует подсчёт активных инцидентов.
func TestIncidentSlice_CountActive(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	t.Run("counts only active incidents", func(t *testing.T) {
		active1 := NewIncident(monitorID)
		active2 := NewIncident(monitorID)

		resolved := NewIncident(monitorID)
		resolved.Resolve(time.Now())

		resolved2 := NewIncident(monitorID)
		resolved2.Resolve(time.Now())

		incidents := IncidentSlice{*active1, *active2, *resolved, *resolved2}

		if incidents.CountActive() != 2 {
			t.Errorf("CountActive() = %d, want 2", incidents.CountActive())
		}
	})

	t.Run("all resolved returns zero", func(t *testing.T) {
		incident1 := NewIncident(monitorID)
		incident1.Resolve(time.Now())

		incident2 := NewIncident(monitorID)
		incident2.Resolve(time.Now())

		incidents := IncidentSlice{*incident1, *incident2}

		if incidents.CountActive() != 0 {
			t.Errorf("CountActive() with all resolved = %d, want 0", incidents.CountActive())
		}
	})

	t.Run("empty slice returns zero", func(t *testing.T) {
		var incidents IncidentSlice

		if incidents.CountActive() != 0 {
			t.Errorf("CountActive() on empty slice = %d, want 0", incidents.CountActive())
		}
	})
}
