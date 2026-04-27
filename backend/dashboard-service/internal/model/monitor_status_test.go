package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitorStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status  MonitorStatus
		wantStr string
	}{
		{MonitorStatusUP, "UP"},
		{MonitorStatusDOWN, "DOWN"},
		{MonitorStatusDEGRADED, "DEGRADED"},
		{MonitorStatusPAUSED, "PAUSED"},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantStr, tt.status.String())
		})
	}
}

func TestMonitorStatusPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input MonitorStatus
		want  int
	}{
		{"down", MonitorStatusDOWN, 1},
		{"degraded", MonitorStatusDEGRADED, 2},
		{"up", MonitorStatusUP, 3},
		{"paused", MonitorStatusPAUSED, 4},
		{"unknown", MonitorStatus("UNKNOWN"), 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, MonitorStatusPriority(tt.input))
		})
	}
}

func TestMonitorStatusView_Fields(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	responseTime := 123.45

	view := MonitorStatusView{
		ID:                 id,
		UserID:             userID,
		Name:               "example-monitor",
		URL:                "https://example.com",
		Status:             MonitorStatusUP,
		UptimePercentage:   99.5,
		LastCheckedAt:      &now,
		LastResponseTimeMs: &responseTime,
		Tags:               "prod,web",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	assert.Equal(t, id, view.ID)
	assert.Equal(t, userID, view.UserID)
	assert.Equal(t, "example-monitor", view.Name)
	assert.Equal(t, "https://example.com", view.URL)
	assert.Equal(t, MonitorStatusUP, view.Status)
	assert.Equal(t, 99.5, view.UptimePercentage)
	require.NotNil(t, view.LastCheckedAt)
	assert.Equal(t, now, *view.LastCheckedAt)
	require.NotNil(t, view.LastResponseTimeMs)
	assert.Equal(t, responseTime, *view.LastResponseTimeMs)
	assert.Equal(t, "prod,web", view.Tags)
	assert.Equal(t, now, view.CreatedAt)
	assert.Equal(t, now, view.UpdatedAt)
}
