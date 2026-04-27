package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMaintenanceWindow_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	cfg := DefaultMaintenanceWindowConfig()

	window, err := NewMaintenanceWindow(
		userID,
		"Test Maintenance",
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	require.NoError(t, err)
	assert.NotNil(t, window)
	assert.Equal(t, "Test Maintenance", window.Name)
	assert.Equal(t, userID, window.UserID)
	assert.Equal(t, WindowStatusScheduled, window.Status)
	assert.Equal(t, RecurrenceOnce, window.Recurrence)
	assert.Equal(t, 1, window.Version)
}

func TestNewMaintenanceWindow_EmptyName(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	cfg := DefaultMaintenanceWindowConfig()

	_, err := NewMaintenanceWindow(
		userID,
		"",
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyWindowName, err)
}

func TestNewMaintenanceWindow_NameTooLong(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	cfg := DefaultMaintenanceWindowConfig()

	longName := string(make([]byte, 256))
	_, err := NewMaintenanceWindow(
		userID,
		longName,
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrWindowNameTooLong, err)
}

func TestNewMaintenanceWindow_EndTimeBeforeStartTime(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(2 * time.Hour)
	endTime := startTime.Add(-1 * time.Hour)
	cfg := DefaultMaintenanceWindowConfig()

	_, err := NewMaintenanceWindow(
		userID,
		"Test",
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrEndTimeBeforeStartTime, err)
}

func TestNewMaintenanceWindow_DurationExceedsMaximum(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(25 * time.Hour) // 25 часов
	cfg := DefaultMaintenanceWindowConfig()

	_, err := NewMaintenanceWindow(
		userID,
		"Test",
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrDurationExceedsMaximum, err)
}

func TestNewMaintenanceWindow_DurationBelowMinimum(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(30 * time.Second) // 30 секунд
	cfg := MaintenanceWindowConfig{
		MinDurationMinutes:   1,
		MaxDurationHours:     24,
		MaxMonitorsPerWindow: 50,
	}

	_, err := NewMaintenanceWindow(
		userID,
		"Test",
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrDurationBelowMinimum, err)
}

func TestNewMaintenanceWindow_InvalidRecurrence(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	cfg := DefaultMaintenanceWindowConfig()

	_, err := NewMaintenanceWindow(
		userID,
		"Test",
		startTime,
		endTime,
		RecurrenceType("INVALID"),
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidRecurrence, err)
}

func TestNewMaintenanceWindow_InThePast(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := startTime.Add(1 * time.Hour)
	cfg := DefaultMaintenanceWindowConfig()

	_, err := NewMaintenanceWindow(
		userID,
		"Test",
		startTime,
		endTime,
		RecurrenceOnce,
		false,
		true,
		true,
		false,
		[]uuid.UUID{},
		cfg,
	)

	assert.Error(t, err)
	assert.Equal(t, ErrCannotCreateInPast, err)
}

func TestMaintenanceWindow_IsActive(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	assert.True(t, window.IsActive())
}

func TestMaintenanceWindow_IsActive_Scheduled(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	assert.False(t, window.IsActive())
}

func TestMaintenanceWindow_IsActive_OutsideTimeRange(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now().Add(-1 * time.Hour),
	}

	assert.False(t, window.IsActive())
}

func TestMaintenanceWindow_ShouldBeActive(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	assert.True(t, window.ShouldBeActive())
}

func TestMaintenanceWindow_ShouldBeCompleted(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now().Add(-1 * time.Minute),
	}

	assert.True(t, window.ShouldBeCompleted())
}

func TestMaintenanceWindow_OverlapsWith_SameMonitors(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	window1 := &MaintenanceWindow{
		MonitorIDs: []uuid.UUID{monitorID},
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(3 * time.Hour),
	}

	window2 := &MaintenanceWindow{
		MonitorIDs: []uuid.UUID{monitorID},
		StartTime:  time.Now().Add(2 * time.Hour),
		EndTime:    time.Now().Add(4 * time.Hour),
	}

	assert.True(t, window1.OverlapsWith(window2))
}

func TestMaintenanceWindow_OverlapsWith_DifferentMonitors(t *testing.T) {
	t.Parallel()
	window1 := &MaintenanceWindow{
		MonitorIDs: []uuid.UUID{uuid.New()},
		StartTime:  time.Now().Add(1 * time.Hour),
		EndTime:    time.Now().Add(3 * time.Hour),
	}

	window2 := &MaintenanceWindow{
		MonitorIDs: []uuid.UUID{uuid.New()},
		StartTime:  time.Now().Add(2 * time.Hour),
		EndTime:    time.Now().Add(4 * time.Hour),
	}

	assert.False(t, window1.OverlapsWith(window2))
}

func TestMaintenanceWindow_OverlapsWith_GlobalWindow(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()

	window1 := &MaintenanceWindow{
		IsGlobal:  true,
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(3 * time.Hour),
	}

	window2 := &MaintenanceWindow{
		MonitorIDs: []uuid.UUID{monitorID},
		StartTime:  time.Now().Add(2 * time.Hour),
		EndTime:    time.Now().Add(4 * time.Hour),
	}

	assert.True(t, window1.OverlapsWith(window2))
}

func TestMaintenanceWindow_Activate(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Hour),
		Version:   1,
	}

	err := window.Activate()
	require.NoError(t, err)
	assert.Equal(t, WindowStatusActive, window.Status)
	assert.NotNil(t, window.ActivatedAt)
	assert.Equal(t, 2, window.Version)
}

func TestMaintenanceWindow_Activate_NotScheduled(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	err := window.Activate()
	assert.Error(t, err)
}

func TestMaintenanceWindow_Complete(t *testing.T) {
	t.Parallel()
	now := time.Now()
	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: now.Add(-2 * time.Hour),
		EndTime:   now.Add(-1 * time.Minute),
		Version:   1,
	}

	err := window.Complete()
	require.NoError(t, err)
	assert.Equal(t, WindowStatusCompleted, window.Status)
	assert.NotNil(t, window.CompletedAt)
	assert.Equal(t, 2, window.Version)
}

func TestMaintenanceWindow_Complete_NotActive(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	err := window.Complete()
	assert.Error(t, err)
}

func TestMaintenanceWindow_Cancel_Scheduled(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusScheduled,
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Version:   1,
	}

	err := window.Cancel()
	require.NoError(t, err)
	assert.Equal(t, WindowStatusCancelled, window.Status)
	assert.Equal(t, 2, window.Version)
}

func TestMaintenanceWindow_Cancel_Active(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		Status:    WindowStatusActive,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
		Version:   1,
	}

	err := window.Cancel()
	require.NoError(t, err)
	assert.Equal(t, WindowStatusCancelled, window.Status)
	assert.Equal(t, 2, window.Version)
}

func TestMaintenanceWindowStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		from     MaintenanceWindowStatus
		to       MaintenanceWindowStatus
		expected bool
	}{
		{"Scheduled to Active", WindowStatusScheduled, WindowStatusActive, true},
		{"Scheduled to Cancelled", WindowStatusScheduled, WindowStatusCancelled, true},
		{"Active to Completed", WindowStatusActive, WindowStatusCompleted, true},
		{"Active to Cancelled", WindowStatusActive, WindowStatusCancelled, true},
		{"Completed to Active", WindowStatusCompleted, WindowStatusActive, false},
		{"Cancelled to Scheduled", WindowStatusCancelled, WindowStatusScheduled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaintenanceWindow_UpdateDetails(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Old Name",
		StartTime: time.Now().Add(2 * time.Hour),
		EndTime:   time.Now().Add(4 * time.Hour),
		Status:    WindowStatusScheduled,
		Version:   1,
	}

	newStartTime := time.Now().Add(3 * time.Hour)
	newEndTime := time.Now().Add(5 * time.Hour)

	err := window.UpdateDetails("New Name", newStartTime, newEndTime, 1)
	require.NoError(t, err)
	assert.Equal(t, "New Name", window.Name)
	assert.Equal(t, 2, window.Version)
}

func TestMaintenanceWindow_UpdateDetails_VersionConflict(t *testing.T) {
	t.Parallel()
	window := &MaintenanceWindow{
		ID:        uuid.New(),
		Name:      "Old Name",
		StartTime: time.Now().Add(2 * time.Hour),
		EndTime:   time.Now().Add(4 * time.Hour),
		Status:    WindowStatusScheduled,
		Version:   1,
	}

	err := window.UpdateDetails("New Name", window.StartTime, window.EndTime, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version conflict")
}

func TestMaintenanceWindow_Duration(t *testing.T) {
	t.Parallel()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)

	window := &MaintenanceWindow{
		StartTime: startTime,
		EndTime:   endTime,
	}

	duration := window.Duration()
	assert.Equal(t, 2*time.Hour, duration)
}
