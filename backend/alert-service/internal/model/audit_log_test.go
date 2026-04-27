package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditAction_Constants(t *testing.T) {
	t.Parallel()

	// Проверяем, что все константы имеют значения
	constants := []AuditAction{
		AuditActionChannelCreated,
		AuditActionChannelUpdated,
		AuditActionChannelDeleted,
		AuditActionChannelVerified,
		AuditActionChannelVerifyFailed,
		AuditActionChannelAutoDisabled,
		AuditActionAlertRuleCreated,
		AuditActionAlertTriggered,
		AuditActionAlertAcknowledged,
		AuditActionAlertResolved,
		AuditActionAlertRetriggered,
		AuditActionAlertMutedGlobal,
		AuditActionFlappingDetected,
		AuditActionAlertDelivered,
		AuditActionAlertRateLimited,
		AuditActionAlertStormDetected,
		AuditActionDeliveryRetry,
		AuditActionDeliveryNoRetry,
		AuditActionUnauthorizedAttempt,
	}

	for _, action := range constants {
		assert.NotEmpty(t, string(action), "AuditAction constant should not be empty")
	}
}

func TestAuditAction_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		action   AuditAction
		expected string
	}{
		{
			name:     "Channel created",
			action:   AuditActionChannelCreated,
			expected: "channel_created",
		},
		{
			name:     "Alert triggered",
			action:   AuditActionAlertTriggered,
			expected: "alert_triggered",
		},
		{
			name:     "Alert acknowledged",
			action:   AuditActionAlertAcknowledged,
			expected: "alert_acknowledged",
		},
		{
			name:     "Custom action",
			action:   AuditAction("custom_action"),
			expected: "custom_action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.action.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditLog_Creation(t *testing.T) {
	t.Parallel()

	t.Run("Valid audit log with user", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		log := &AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       AuditActionAlertTriggered,
			ResourceType: "alert",
			ResourceID:   strPtr("alert-123"),
			Fields:       map[string]any{"severity": "critical"},
			IPAddress:    strPtr("192.168.1.1"),
			CreatedAt:    time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, log.ID)
		assert.NotNil(t, log.UserID)
		assert.NotEmpty(t, log.Action)
		assert.NotEmpty(t, log.ResourceType)
		assert.NotNil(t, log.ResourceID)
		assert.NotNil(t, log.Fields)
		assert.NotNil(t, log.IPAddress)
		assert.NotEqual(t, time.Time{}, log.CreatedAt)
	})

	t.Run("Valid audit log without user", func(t *testing.T) {
		t.Parallel()

		log := &AuditLog{
			ID:           uuid.New(),
			UserID:       nil, // system action
			Action:       AuditActionAlertStormDetected,
			ResourceType: "system",
			ResourceID:   nil,
			Fields:       map[string]any{"alerts_count": 100},
			IPAddress:    nil,
			CreatedAt:    time.Now(),
		}

		assert.Nil(t, log.UserID)
		assert.Nil(t, log.ResourceID)
		assert.Nil(t, log.IPAddress)
		assert.NotEmpty(t, log.Fields)
	})
}

func TestAuditLog_ChannelActions(t *testing.T) {
	t.Parallel()

	channelActions := []AuditAction{
		AuditActionChannelCreated,
		AuditActionChannelUpdated,
		AuditActionChannelDeleted,
		AuditActionChannelVerified,
		AuditActionChannelVerifyFailed,
		AuditActionChannelAutoDisabled,
	}

	for _, action := range channelActions {
		t.Run(action.String(), func(t *testing.T) {
			t.Parallel()

			log := &AuditLog{
				ID:           uuid.New(),
				Action:       action,
				ResourceType: "channel",
				ResourceID:   strPtr("channel-123"),
				CreatedAt:    time.Now(),
			}

			assert.Equal(t, "channel", log.ResourceType)
			assert.Contains(t, string(action), "channel")
		})
	}
}

func TestAuditLog_AlertActions(t *testing.T) {
	t.Parallel()

	alertActions := []struct {
		action       AuditAction
		requireAlert bool
	}{
		{AuditActionAlertRuleCreated, true},
		{AuditActionAlertTriggered, true},
		{AuditActionAlertAcknowledged, true},
		{AuditActionAlertResolved, true},
		{AuditActionAlertRetriggered, true},
		{AuditActionAlertMutedGlobal, false}, // global action
	}

	for _, tt := range alertActions {
		t.Run(tt.action.String(), func(t *testing.T) {
			t.Parallel()

			log := &AuditLog{
				ID:           uuid.New(),
				Action:       tt.action,
				ResourceType: "alert",
				CreatedAt:    time.Now(),
			}

			assert.Equal(t, "alert", log.ResourceType)
			assert.Contains(t, string(tt.action), "alert")
		})
	}
}

func TestAuditLog_DeliveryActions(t *testing.T) {
	t.Parallel()

	deliveryActions := []AuditAction{
		AuditActionFlappingDetected,
		AuditActionAlertDelivered,
		AuditActionAlertRateLimited,
		AuditActionAlertStormDetected,
		AuditActionDeliveryRetry,
		AuditActionDeliveryNoRetry,
	}

	for _, action := range deliveryActions {
		t.Run(action.String(), func(t *testing.T) {
			t.Parallel()

			log := &AuditLog{
				ID:        uuid.New(),
				Action:    action,
				Fields:    map[string]any{"attempt": 1},
				CreatedAt: time.Now(),
			}

			assert.NotEmpty(t, log.Action)
			assert.NotEmpty(t, log.Fields)
		})
	}
}

func TestAuditLog_Fields(t *testing.T) {
	t.Parallel()

	t.Run("Audit log with complex fields", func(t *testing.T) {
		t.Parallel()

		log := &AuditLog{
			ID:     uuid.New(),
			Action: AuditActionChannelVerified,
			Fields: map[string]any{
				"channel_id":        "chan-123",
				"verification_time": 1234500,
				"success":           true,
				"attempts":          3,
				"metadata": map[string]string{
					"region": "us-east-1",
				},
			},
			CreatedAt: time.Now(),
		}

		assert.Equal(t, "chan-123", log.Fields["channel_id"])
		assert.Equal(t, int64(1234500), int64(log.Fields["verification_time"].(int)))
		assert.Equal(t, true, log.Fields["success"])
		assert.Equal(t, 3, log.Fields["attempts"])
	})

	t.Run("Audit log with empty fields", func(t *testing.T) {
		t.Parallel()

		log := &AuditLog{
			ID:        uuid.New(),
			Action:    AuditActionUnauthorizedAttempt,
			Fields:    map[string]any{},
			CreatedAt: time.Now(),
		}

		assert.Empty(t, log.Fields)
		assert.NotNil(t, log.Fields)
	})
}

func TestAuditLog_ResourceTracking(t *testing.T) {
	t.Parallel()

	t.Run("Track channel resource", func(t *testing.T) {
		t.Parallel()

		log := &AuditLog{
			ID:           uuid.New(),
			Action:       AuditActionChannelCreated,
			ResourceType: "channel",
			ResourceID:   strPtr("channel-xyz"),
			CreatedAt:    time.Now(),
		}

		assert.Equal(t, "channel", log.ResourceType)
		assert.Equal(t, "channel-xyz", *log.ResourceID)
	})

	t.Run("Track alert resource", func(t *testing.T) {
		t.Parallel()

		log := &AuditLog{
			ID:           uuid.New(),
			Action:       AuditActionAlertResolved,
			ResourceType: "alert",
			ResourceID:   strPtr("alert-abc"),
			CreatedAt:    time.Now(),
		}

		assert.Equal(t, "alert", log.ResourceType)
		assert.Equal(t, "alert-abc", *log.ResourceID)
	})

	t.Run("Track system resource", func(t *testing.T) {
		t.Parallel()

		log := &AuditLog{
			ID:           uuid.New(),
			Action:       AuditActionAlertStormDetected,
			ResourceType: "system",
			ResourceID:   nil, // system-wide event
			CreatedAt:    time.Now(),
		}

		assert.Equal(t, "system", log.ResourceType)
		assert.Nil(t, log.ResourceID)
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}
