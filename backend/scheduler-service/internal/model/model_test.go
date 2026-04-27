package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorker(t *testing.T) {
	t.Parallel()

	// Act
	w := NewWorker("worker-msk-01", "moscow")

	// Assert
	assert.Equal(t, "worker-msk-01", w.Name)
	assert.Equal(t, "moscow", w.Zone)
	assert.Equal(t, WorkerStatusIdle, w.Status)
	assert.Equal(t, 0, w.ChecksCompleted)
	assert.Equal(t, 0, w.ChecksFailed)
	assert.Equal(t, 0.0, w.AvgCheckDurationMs)
	assert.NotEmpty(t, w.ID)
	assert.False(t, w.LastHeartbeat.IsZero())
	assert.False(t, w.CreatedAt.IsZero())
	assert.NotNil(t, w.Metadata)
}

func TestWorker_UpdateHeartbeat(t *testing.T) {
	t.Parallel()

	// Arrange
	w := NewWorker("worker-01", "msk")
	oldHeartbeat := w.LastHeartbeat

	// Act
	w.UpdateHeartbeat(WorkerStatusBusy, 100, 5, 245.5)

	// Assert
	assert.Equal(t, WorkerStatusBusy, w.Status)
	assert.True(t, w.LastHeartbeat.After(oldHeartbeat))
	assert.Equal(t, 100, w.ChecksCompleted)
	assert.Equal(t, 5, w.ChecksFailed)
	assert.Equal(t, 245.5, w.AvgCheckDurationMs)
}

func TestWorker_MarkOffline(t *testing.T) {
	t.Parallel()

	// Arrange
	w := NewWorker("worker-01", "msk")
	require.Equal(t, WorkerStatusIdle, w.Status)

	// Act
	w.MarkOffline()

	// Assert
	assert.Equal(t, WorkerStatusOffline, w.Status)
}

func TestWorker_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		lastHeartbeat time.Time
		timeout       time.Duration
		wantExpired   bool
	}{
		{
			name:          "not_expired",
			lastHeartbeat: time.Now().Add(-10 * time.Second),
			timeout:       30 * time.Second,
			wantExpired:   false,
		},
		{
			name:          "exactly_at_threshold",
			lastHeartbeat: time.Now().Add(-30 * time.Second),
			timeout:       30 * time.Second,
			wantExpired:   true,
		},
		{
			name:          "expired",
			lastHeartbeat: time.Now().Add(-31 * time.Second),
			timeout:       30 * time.Second,
			wantExpired:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := NewWorker("worker-01", "msk")
			w.LastHeartbeat = tc.lastHeartbeat
			assert.Equal(t, tc.wantExpired, w.IsExpired(tc.timeout))
		})
	}
}

func TestParseWorkerStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected WorkerStatus
		wantErr  bool
	}{
		{"IDLE", WorkerStatusIdle, false},
		{"BUSY", WorkerStatusBusy, false},
		{"OFFLINE", WorkerStatusOffline, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			result, err := ParseWorkerStatus(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidWorkerStatus)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestNewScheduledCheck(t *testing.T) {
	t.Parallel()

	// Act
	monitorID := mustParseUUID("550e8400-e29b-41d4-a716-446655440000")
	check := NewScheduledCheck(monitorID, PriorityNormal, time.Now())

	// Assert
	assert.Equal(t, monitorID, check.MonitorID)
	assert.Equal(t, PriorityNormal, check.Priority)
	assert.Equal(t, CheckStatusPending, check.Status)
	assert.True(t, check.WorkerID == uuid.Nil)
	assert.Nil(t, check.ExecutedAt)
	assert.Nil(t, check.CompletedAt)
	assert.NotEmpty(t, check.ID)
	assert.False(t, check.ScheduledAt.IsZero())
}

func TestScheduledCheck_Assign(t *testing.T) {
	t.Parallel()

	// Arrange
	check := NewScheduledCheck(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), PriorityNormal, time.Now())
	workerID := mustParseUUID("660e8400-e29b-41d4-a716-446655440001")
	require.Equal(t, CheckStatusPending, check.Status)
	require.Nil(t, check.ExecutedAt)

	// Act
	check.Assign(workerID)

	// Assert
	assert.Equal(t, workerID, check.WorkerID)
	assert.Equal(t, CheckStatusInProgress, check.Status)
	assert.NotNil(t, check.ExecutedAt)
}

func TestScheduledCheck_Complete(t *testing.T) {
	t.Parallel()

	// Arrange
	check := NewScheduledCheck(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), PriorityNormal, time.Now())
	check.Assign(mustParseUUID("660e8400-e29b-41d4-a716-446655440001"))

	// Act
	check.Complete()

	// Assert
	assert.Equal(t, CheckStatusCompleted, check.Status)
	assert.NotNil(t, check.CompletedAt)
}

func TestScheduledCheck_Fail(t *testing.T) {
	t.Parallel()

	// Arrange
	check := NewScheduledCheck(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), PriorityNormal, time.Now())
	check.Assign(mustParseUUID("660e8400-e29b-41d4-a716-446655440001"))

	// Act
	check.Fail("connection refused")

	// Assert
	assert.Equal(t, CheckStatusFailed, check.Status)
	assert.Equal(t, "connection refused", check.ErrorMessage)
	assert.NotNil(t, check.CompletedAt)
}

func TestScheduledCheck_IsOverdue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    CheckStatus
		schedule  time.Time
		threshold time.Duration
		want      bool
	}{
		{
			name:      "pending_overdue",
			status:    CheckStatusPending,
			schedule:  time.Now().Add(-5 * time.Minute),
			threshold: 2 * time.Minute,
			want:      true,
		},
		{
			name:      "pending_not_overdue",
			status:    CheckStatusPending,
			schedule:  time.Now().Add(-1 * time.Minute),
			threshold: 2 * time.Minute,
			want:      false,
		},
		{
			name:      "in_progress_not_overdue",
			status:    CheckStatusInProgress,
			schedule:  time.Now().Add(-5 * time.Minute),
			threshold: 2 * time.Minute,
			want:      false,
		},
		{
			name:      "completed_not_overdue",
			status:    CheckStatusCompleted,
			schedule:  time.Now().Add(-5 * time.Minute),
			threshold: 2 * time.Minute,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			check := NewScheduledCheck(mustParseUUID("550e8400-e29b-41d4-a716-446655440000"), PriorityNormal, tc.schedule)
			check.Status = tc.status
			assert.Equal(t, tc.want, check.IsOverdue(tc.threshold))
		})
	}
}

func TestSchedulerError_Error(t *testing.T) {
	t.Parallel()

	err := ErrWorkerNotFound
	assert.Equal(t, "WORKER_NOT_FOUND: worker not found", err.Error())
}

func TestIsSchedulerError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"scheduler_error", ErrWorkerNotFound, true},
		{"wrapped", fmt.Errorf("wrap: %w", ErrMonitorNotFound), true},
		{"other_error", fmt.Errorf("something"), false},
		{"nil", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsSchedulerError(tc.err))
		})
	}
}

func TestScheduledCheck_MarkCompleted(t *testing.T) {
	t.Parallel()

	check := NewScheduledCheck(uuid.New(), PriorityNormal, time.Now())
	check.Assign(uuid.New())

	check.MarkCompleted()

	assert.Equal(t, CheckStatusCompleted, check.Status)
	assert.NotNil(t, check.CompletedAt)
}

func TestScheduledCheck_MarkFailed(t *testing.T) {
	t.Parallel()

	check := NewScheduledCheck(uuid.New(), PriorityNormal, time.Now())
	check.Assign(uuid.New())

	check.MarkFailed("timeout")

	assert.Equal(t, CheckStatusFailed, check.Status)
	assert.Equal(t, "timeout", check.ErrorMessage)
	assert.NotNil(t, check.CompletedAt)
}

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
