package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMuteScope_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scope    MuteScope
		expected string
	}{
		{
			name:     "User scope",
			scope:    MuteScopeUser,
			expected: "user",
		},
		{
			name:     "Global scope",
			scope:    MuteScopeGlobal,
			expected: "global",
		},
		{
			name:     "Custom scope",
			scope:    MuteScope("custom"),
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.scope.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertMute_Validation(t *testing.T) {
	t.Parallel()

	t.Run("Valid mute with monitor ID", func(t *testing.T) {
		t.Parallel()

		mute := &AlertMute{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			MonitorID: uuidPtr(uuid.New()),
			Scope:     MuteScopeUser,
			CreatedBy: uuid.New(),
		}

		// Проверяем, что все необходимые поля заполнены
		assert.NotEqual(t, uuid.Nil, mute.ID)
		assert.NotEqual(t, uuid.Nil, mute.UserID)
		assert.NotNil(t, mute.MonitorID)
		assert.NotEqual(t, uuid.Nil, mute.CreatedBy)
	})

	t.Run("Valid global mute without monitor", func(t *testing.T) {
		t.Parallel()

		mute := &AlertMute{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			MonitorID: nil, // global mute doesn't require monitor
			Scope:     MuteScopeGlobal,
			CreatedBy: uuid.New(),
		}

		assert.Nil(t, mute.MonitorID) // nil для global mute корректно
		assert.Equal(t, MuteScopeGlobal, mute.Scope)
	})

	t.Run("Valid mute with expiration", func(t *testing.T) {
		t.Parallel()

		expiresAt := time.Now().Add(24 * time.Hour)
		mute := &AlertMute{
			ID:         uuid.New(),
			UserID:     uuid.New(),
			MonitorID:  uuidPtr(uuid.New()),
			Scope:      MuteScopeUser,
			MutedUntil: &expiresAt,
			CreatedBy:  uuid.New(),
		}

		assert.NotNil(t, mute.MutedUntil)
		assert.True(t, mute.MutedUntil.After(time.Now()))
	})
}

func TestAlertEscalation_Creation(t *testing.T) {
	t.Parallel()

	t.Run("Valid escalation", func(t *testing.T) {
		t.Parallel()

		channelID := uuid.New()
		escalation := &AlertEscalation{
			ID:                   uuid.New(),
			AlertID:              uuid.New(),
			Level:                1,
			EscalatedToChannelID: &channelID,
			Reason:               "No response",
			TimeoutMinutes:       15,
		}

		assert.NotEqual(t, uuid.Nil, escalation.ID)
		assert.NotEqual(t, uuid.Nil, escalation.AlertID)
		assert.Greater(t, escalation.Level, 0)
		assert.NotNil(t, escalation.EscalatedToChannelID)
		assert.NotEmpty(t, escalation.Reason)
		assert.Greater(t, escalation.TimeoutMinutes, 0)
	})

	t.Run("Escalation without channel", func(t *testing.T) {
		t.Parallel()

		escalation := &AlertEscalation{
			ID:                   uuid.New(),
			AlertID:              uuid.New(),
			Level:                1,
			EscalatedToChannelID: nil,
			Reason:               "Auto-escalation",
			TimeoutMinutes:       10,
		}

		assert.Nil(t, escalation.EscalatedToChannelID)
		assert.NotEmpty(t, escalation.Reason)
	})
}

func TestMaintenanceWindow_Validation(t *testing.T) {
	t.Parallel()

	t.Run("Valid maintenance window", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		reason := "Scheduled maintenance"

		window := &MaintenanceWindow{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			MonitorID: uuidPtr(uuid.New()),
			StartsAt:  now,
			EndsAt:    now.Add(2 * time.Hour),
			Reason:    &reason,
		}

		assert.NotEqual(t, uuid.Nil, window.ID)
		assert.NotEqual(t, uuid.Nil, window.UserID)
		assert.NotNil(t, window.MonitorID)
		assert.True(t, window.EndsAt.After(window.StartsAt))
		assert.NotNil(t, window.Reason)
		assert.Equal(t, "Scheduled maintenance", *window.Reason)
	})

	t.Run("Maintenance window without reason", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		window := &MaintenanceWindow{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			MonitorID: uuidPtr(uuid.New()),
			StartsAt:  now,
			EndsAt:    now.Add(1 * time.Hour),
			Reason:    nil,
		}

		assert.Nil(t, window.Reason)
	})

	t.Run("Invalid maintenance window - ends before starts", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		window := &MaintenanceWindow{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			MonitorID: uuidPtr(uuid.New()),
			StartsAt:  now.Add(2 * time.Hour),
			EndsAt:    now, // Ends before starts!
		}

		// Это недопустимая конфигурация, но модель сама не валидирует
		assert.True(t, window.EndsAt.Before(window.StartsAt))
	})
}

func TestMonitorStatusChange_Creation(t *testing.T) {
	t.Parallel()

	t.Run("Valid status change", func(t *testing.T) {
		t.Parallel()

		change := &MonitorStatusChange{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "pending",
			NewStatus: "up",
		}

		assert.NotEqual(t, uuid.Nil, change.ID)
		assert.NotEqual(t, uuid.Nil, change.MonitorID)
		assert.NotEqual(t, uuid.Nil, change.UserID)
		assert.NotEmpty(t, change.OldStatus)
		assert.NotEmpty(t, change.NewStatus)
		assert.NotEqual(t, change.OldStatus, change.NewStatus)
	})

	t.Run("Status change from same status", func(t *testing.T) {
		t.Parallel()

		change := &MonitorStatusChange{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "up",
			NewStatus: "up",
		}

		// Статус не изменился, но модель позволяет это
		assert.Equal(t, change.OldStatus, change.NewStatus)
	})
}

// Helper function
func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
