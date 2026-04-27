package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestMaintenanceWindow_ShouldBeActive_FutureStartTime тестирует ShouldBeActive когда StartTime в будущем.
func TestMaintenanceWindow_ShouldBeActive_FutureStartTime(t *testing.T) {
	t.Parallel()

	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(1 * time.Hour), // в будущем
		EndTime:   time.Now().Add(2 * time.Hour),
	}

	assert.False(t, window.ShouldBeActive())
}

// TestMaintenanceWindow_ShouldBeActive_WrongStatus тестирует ShouldBeActive когда статус не SCHEDULED.
func TestMaintenanceWindow_ShouldBeActive_WrongStatus(t *testing.T) {
	t.Parallel()

	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	assert.False(t, window.ShouldBeActive())
}

// TestMaintenanceWindow_ShouldBeCompleted_WrongStatus тестирует ShouldBeCompleted когда статус не ACTIVE.
func TestMaintenanceWindow_ShouldBeCompleted_WrongStatus(t *testing.T) {
	t.Parallel()

	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now().Add(-1 * time.Minute),
	}

	assert.False(t, window.ShouldBeCompleted())
}

// TestMaintenanceWindow_ShouldBeCompleted_EndTimeInFuture тестирует ShouldBeCompleted когда EndTime в будущем.
func TestMaintenanceWindow_ShouldBeCompleted_EndTimeInFuture(t *testing.T) {
	t.Parallel()

	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour), // в будущем
	}

	assert.False(t, window.ShouldBeCompleted())
}

// TestValidateMonitorsPerWindow_ExceedsLimit тестирует validateMonitorsPerWindow при превышении лимита.
func TestValidateMonitorsPerWindow_ExceedsLimit(t *testing.T) {
	t.Parallel()

	cfg := MaintenanceWindowConfig{MaxMonitorsPerWindow: 2}
	monitorIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()} // 3 > 2

	err := validateMonitorsPerWindow(monitorIDs, cfg)

	assert.Error(t, err)
}

// TestValidateMonitorsPerWindow_WithinLimit тестирует validateMonitorsPerWindow при соответствии лимиту.
func TestValidateMonitorsPerWindow_WithinLimit(t *testing.T) {
	t.Parallel()

	cfg := MaintenanceWindowConfig{MaxMonitorsPerWindow: 5}
	monitorIDs := []uuid.UUID{uuid.New(), uuid.New()}

	err := validateMonitorsPerWindow(monitorIDs, cfg)

	assert.NoError(t, err)
}

// TestMaintenanceWindow_Cancel_CompletedWindow тестирует Cancel для завершённого окна.
func TestMaintenanceWindow_Cancel_CompletedWindow(t *testing.T) {
	t.Parallel()

	window := &MaintenanceWindow{
		Status: WindowStatusCompleted,
	}

	err := window.Cancel()

	assert.Error(t, err)
}

// TestMaintenanceWindow_CanTransitionTo_UnknownStatus тестирует переход из неизвестного статуса.
func TestMaintenanceWindow_CanTransitionTo_UnknownStatus(t *testing.T) {
	t.Parallel()

	// WindowStatusOrphaned не имеет разрешённых переходов
	result := WindowStatusOrphaned.CanTransitionTo(WindowStatusActive)
	assert.False(t, result)

	// Статус "UNKNOWN" не существует в карте transitions
	unknownStatus := MaintenanceWindowStatus("UNKNOWN_STATUS")
	result = unknownStatus.CanTransitionTo(WindowStatusActive)
	assert.False(t, result)
}

// TestMaintenanceWindow_UpdateDetails_WrongStatus тестирует UpdateDetails для неподходящего статуса.
func TestMaintenanceWindow_UpdateDetails_WrongStatus(t *testing.T) {
	t.Parallel()

	window := &MaintenanceWindow{
		Status:  WindowStatusActive,
		Version: 1,
	}

	newStart := time.Now().Add(1 * time.Hour)
	newEnd := newStart.Add(2 * time.Hour)
	err := window.UpdateDetails("New Name", newStart, newEnd, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheduled windows")
}

// TestMaintenanceWindow_UpdateDetails_ValidationErrors тестирует UpdateDetails с невалидными данными.
func TestMaintenanceWindow_UpdateDetails_ValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		window := &MaintenanceWindow{
			Status:  WindowStatusScheduled,
			Version: 1,
		}
		newStart := time.Now().Add(1 * time.Hour)
		newEnd := newStart.Add(2 * time.Hour)
		err := window.UpdateDetails("", newStart, newEnd, 1)
		assert.Error(t, err)
	})
}

// TestNewMaintenanceWindow_TooManyMonitors тестирует NewMaintenanceWindow с превышением лимита.
func TestNewMaintenanceWindow_TooManyMonitors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cfg := MaintenanceWindowConfig{
		MaxWindowsPerAccount: 10,
		MaxDurationHours:     24,
		MinDurationMinutes:   1,
		MaxMonitorsPerWindow: 2,
	}

	// Создаём 3 мониторов, превышая лимит 2
	monitorIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	_, err := NewMaintenanceWindow(userID, "Test", startTime, endTime, RecurrenceOnce, false, false, false, false, monitorIDs, cfg)
	assert.Error(t, err)
}
