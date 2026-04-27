package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewMonitor(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	tests := []struct {
		name            string
		userID          uuid.UUID
		monitorName     string
		url             string
		intervalSeconds int
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "valid monitor",
			userID:          userID,
			monitorName:     "Test Monitor",
			url:             "https://example.com",
			intervalSeconds: 60,
			wantErr:         false,
		},
		{
			name:            "empty name",
			userID:          userID,
			monitorName:     "",
			url:             "https://example.com",
			intervalSeconds: 60,
			wantErr:         true,
			errMsg:          "cannot be empty",
		},
		{
			name:            "name too long",
			userID:          userID,
			monitorName:     string(make([]byte, 256)),
			url:             "https://example.com",
			intervalSeconds: 60,
			wantErr:         true,
			errMsg:          "too long",
		},
		{
			name:            "empty url",
			userID:          userID,
			monitorName:     "Test Monitor",
			url:             "",
			intervalSeconds: 60,
			wantErr:         true,
			errMsg:          "URL cannot be empty",
		},
		{
			name:            "invalid url scheme",
			userID:          userID,
			monitorName:     "Test Monitor",
			url:             "ftp://example.com",
			intervalSeconds: 60,
			wantErr:         true,
			errMsg:          "URL must start with http:// or https://",
		},
		{
			name:            "interval too small",
			userID:          userID,
			monitorName:     "Test Monitor",
			url:             "https://example.com",
			intervalSeconds: 10,
			wantErr:         true,
			errMsg:          "interval must be at least 30 seconds",
		},
		{
			name:            "interval too large",
			userID:          userID,
			monitorName:     "Test Monitor",
			url:             "https://example.com",
			intervalSeconds: 4000,
			wantErr:         true,
			errMsg:          "3600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMonitor(tt.userID, tt.monitorName, tt.url, tt.intervalSeconds)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMonitor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				// Check if error message contains expected substring
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewMonitor() error message = %v, want containing %v", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr {
				if got.ID == uuid.Nil {
					t.Error("NewMonitor() got nil ID")
				}
				if got.UserID != tt.userID {
					t.Errorf("NewMonitor() UserID = %v, want %v", got.UserID, tt.userID)
				}
				if got.Name != tt.monitorName {
					t.Errorf("NewMonitor() Name = %v, want %v", got.Name, tt.monitorName)
				}
				if got.URL != tt.url {
					t.Errorf("NewMonitor() URL = %v, want %v", got.URL, tt.url)
				}
				if got.IntervalSeconds != tt.intervalSeconds {
					t.Errorf("NewMonitor() IntervalSeconds = %v, want %v", got.IntervalSeconds, tt.intervalSeconds)
				}
				if got.Status != StatusPending {
					t.Errorf("NewMonitor() Status = %v, want %v", got.Status, StatusPending)
				}
			}
		})
	}
}

func TestMonitor_UpdateStatus(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	tests := []struct {
		name      string
		current   MonitorStatus
		newStatus MonitorStatus
		wantErr   bool
	}{
		{
			name:      "valid transition pending to up",
			current:   StatusPending,
			newStatus: StatusUp,
			wantErr:   false,
		},
		{
			name:      "valid transition up to down",
			current:   StatusUp,
			newStatus: StatusDown,
			wantErr:   false,
		},
		{
			name:      "valid transition down to up",
			current:   StatusDown,
			newStatus: StatusUp,
			wantErr:   false,
		},
		{
			name:      "invalid transition paused to down",
			current:   StatusPaused,
			newStatus: StatusDown,
			wantErr:   true,
		},
		{
			name:      "valid transition up to paused",
			current:   StatusUp,
			newStatus: StatusPaused,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor.Status = tt.current
			err := monitor.UpdateStatus(tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("Monitor.UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && monitor.Status != tt.newStatus {
				t.Errorf("Monitor.UpdateStatus() Status = %v, want %v", monitor.Status, tt.newStatus)
			}
		})
	}
}

func TestMonitor_ShouldCheckNow(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	t.Run("paused monitor should not check", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusPaused

		if monitor.ShouldCheckNow() {
			t.Error("Paused monitor should not check")
		}
	})

	t.Run("active monitor without restrictions should check", func(t *testing.T) {
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = StatusUp

		if !monitor.ShouldCheckNow() {
			t.Error("Active monitor should check")
		}
	})
}

func TestMonitor_UpdateLastCheck(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	before := monitor.LastCheckAt
	monitor.UpdateLastCheck()

	if monitor.LastCheckAt == nil {
		t.Error("LastCheckAt should be set")
	}

	if before != nil && monitor.LastCheckAt.Before(*before) {
		t.Error("LastCheckAt should be updated to later time")
	}
}
