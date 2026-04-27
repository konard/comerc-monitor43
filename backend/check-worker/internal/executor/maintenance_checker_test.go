package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMaintenanceClient реализует заглушку для MaintenanceClient
type MockMaintenanceClient struct {
	windows []MaintenanceWindow
	err     error
	callCnt int
}

func (m *MockMaintenanceClient) GetActiveMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error) {
	m.callCnt++
	if m.err != nil {
		return nil, m.err
	}
	return m.windows, nil
}

func TestMaintenanceWindow_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		startTime time.Time
		endTime   time.Time
		want      bool
	}{
		{
			name:      "active window",
			startTime: now.Add(-1 * time.Hour),
			endTime:   now.Add(1 * time.Hour),
			want:      true,
		},
		{
			name:      "window starting now",
			startTime: now,
			endTime:   now.Add(1 * time.Hour),
			want:      true,
		},
		{
			name:      "window ending now",
			startTime: now.Add(-1 * time.Hour),
			endTime:   now,
			want:      false, // Окно закрылось в эту секунду
		},
		{
			name:      "future window",
			startTime: now.Add(1 * time.Hour),
			endTime:   now.Add(2 * time.Hour),
			want:      false,
		},
		{
			name:      "past window",
			startTime: now.Add(-2 * time.Hour),
			endTime:   now.Add(-1 * time.Hour),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &MaintenanceWindow{
				StartTime: tt.startTime,
				EndTime:   tt.endTime,
			}
			got := w.IsActive()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMaintenanceWindow_AffectsMonitor(t *testing.T) {
	now := time.Now()

	t.Run("global window affects all monitors", func(t *testing.T) {
		w := &MaintenanceWindow{
			ID:              "global-1",
			IsGlobal:        true,
			StartTime:       now.Add(-1 * time.Hour),
			EndTime:         now.Add(1 * time.Hour),
			PauseMonitoring: true,
		}

		assert.True(t, w.AffectsMonitor("monitor-1"))
		assert.True(t, w.AffectsMonitor("monitor-2"))
	})

	t.Run("specific window affects only its monitor", func(t *testing.T) {
		w := &MaintenanceWindow{
			ID:              "specific-1",
			MonitorID:       "monitor-1",
			IsGlobal:        false,
			StartTime:       now.Add(-1 * time.Hour),
			EndTime:         now.Add(1 * time.Hour),
			PauseMonitoring: true,
		}

		assert.True(t, w.AffectsMonitor("monitor-1"))
		assert.False(t, w.AffectsMonitor("monitor-2"))
	})

	t.Run("inactive window does not affect monitors", func(t *testing.T) {
		w := &MaintenanceWindow{
			ID:              "future-1",
			MonitorID:       "monitor-1",
			IsGlobal:        false,
			StartTime:       now.Add(1 * time.Hour),
			EndTime:         now.Add(2 * time.Hour),
			PauseMonitoring: true,
		}

		assert.False(t, w.AffectsMonitor("monitor-1"))
	})
}

func TestMaintenanceCache_ShouldRefresh(t *testing.T) {
	t.Run("empty cache should refresh", func(t *testing.T) {
		cache := NewMaintenanceCache(5 * time.Minute)
		assert.True(t, cache.ShouldRefresh())
	})

	t.Run("fresh cache should not refresh", func(t *testing.T) {
		cache := NewMaintenanceCache(5 * time.Minute)
		cache.Update([]MaintenanceWindow{})
		assert.False(t, cache.ShouldRefresh())
	})

	t.Run("expired cache should refresh", func(t *testing.T) {
		cache := NewMaintenanceCache(10 * time.Millisecond)
		cache.Update([]MaintenanceWindow{})
		time.Sleep(20 * time.Millisecond)
		assert.True(t, cache.ShouldRefresh())
	})
}

func TestMaintenanceCache_IsMonitorInMaintenance(t *testing.T) {
	now := time.Now()

	cache := NewMaintenanceCache(5 * time.Minute)
	windows := []MaintenanceWindow{
		{
			ID:              "global-1",
			IsGlobal:        true,
			StartTime:       now.Add(-1 * time.Hour),
			EndTime:         now.Add(1 * time.Hour),
			PauseMonitoring: true,
		},
		{
			ID:              "specific-1",
			MonitorID:       "monitor-1",
			IsGlobal:        false,
			StartTime:       now.Add(-1 * time.Hour),
			EndTime:         now.Add(1 * time.Hour),
			PauseMonitoring: true,
		},
	}
	cache.Update(windows)

	assert.True(t, cache.IsMonitorInMaintenance("monitor-1"))
	assert.True(t, cache.IsMonitorInMaintenance("monitor-2")) // Global window
}

func TestMaintenanceChecker_ShouldSkipCheck(t *testing.T) {
	t.Run("skip check when monitor in maintenance", func(t *testing.T) {
		now := time.Now()
		mockClient := &MockMaintenanceClient{
			windows: []MaintenanceWindow{
				{
					ID:              "window-1",
					MonitorID:       "monitor-1",
					StartTime:       now.Add(-1 * time.Hour),
					EndTime:         now.Add(1 * time.Hour),
					PauseMonitoring: true,
				},
			},
		}

		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		ctx := context.Background()

		shouldSkip, window, err := checker.ShouldSkipCheck(ctx, "monitor-1")
		assert.NoError(t, err)
		assert.True(t, shouldSkip)
		assert.NotNil(t, window)
		assert.Equal(t, "window-1", window.ID)
		assert.Equal(t, 1, mockClient.callCnt) // Cache was loaded
	})

	t.Run("do not skip check when monitor not in maintenance", func(t *testing.T) {
		mockClient := &MockMaintenanceClient{
			windows: []MaintenanceWindow{}, // No active windows
		}

		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		ctx := context.Background()

		shouldSkip, window, err := checker.ShouldSkipCheck(ctx, "monitor-1")
		assert.NoError(t, err)
		assert.False(t, shouldSkip)
		assert.Nil(t, window)
		assert.Equal(t, 1, mockClient.callCnt)
	})

	t.Run("use cache on subsequent calls", func(t *testing.T) {
		now := time.Now()
		mockClient := &MockMaintenanceClient{
			windows: []MaintenanceWindow{
				{
					ID:              "window-1",
					MonitorID:       "monitor-1",
					StartTime:       now.Add(-1 * time.Hour),
					EndTime:         now.Add(1 * time.Hour),
					PauseMonitoring: true,
				},
			},
		}

		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		ctx := context.Background()

		// First call
		_, _, err := checker.ShouldSkipCheck(ctx, "monitor-1")
		require.NoError(t, err)
		firstCallCnt := mockClient.callCnt

		// Second call - should use cache
		_, _, err = checker.ShouldSkipCheck(ctx, "monitor-1")
		require.NoError(t, err)
		assert.Equal(t, firstCallCnt, mockClient.callCnt) // No new API call
	})

	t.Run("global window affects all monitors", func(t *testing.T) {
		now := time.Now()
		mockClient := &MockMaintenanceClient{
			windows: []MaintenanceWindow{
				{
					ID:              "global-1",
					IsGlobal:        true,
					StartTime:       now.Add(-1 * time.Hour),
					EndTime:         now.Add(1 * time.Hour),
					PauseMonitoring: true,
				},
			},
		}

		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		ctx := context.Background()

		shouldSkip1, _, err := checker.ShouldSkipCheck(ctx, "monitor-1")
		require.NoError(t, err)
		shouldSkip2, _, err := checker.ShouldSkipCheck(ctx, "monitor-2")
		require.NoError(t, err)

		assert.True(t, shouldSkip1)
		assert.True(t, shouldSkip2)
	})

	t.Run("handle client error gracefully", func(t *testing.T) {
		mockClient := &MockMaintenanceClient{
			windows: []MaintenanceWindow{},
			err:     assert.AnError,
		}

		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		ctx := context.Background()

		// Should not return error - error is ignored and check proceeds
		shouldSkip, window, err := checker.ShouldSkipCheck(ctx, "monitor-1")
		assert.NoError(t, err)
		assert.False(t, shouldSkip) // No maintenance windows
		assert.Nil(t, window)
	})
}

func TestMaintenanceChecker_ForceRefresh(t *testing.T) {
	mockClient := &MockMaintenanceClient{
		windows: []MaintenanceWindow{
			{
				ID: "window-1",
			},
		},
	}

	checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
	ctx := context.Background()

	// Force refresh
	err := checker.ForceRefresh(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mockClient.callCnt)

	// Should not refresh again immediately
	err = checker.ForceRefresh(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, mockClient.callCnt) // Called again because force
}
