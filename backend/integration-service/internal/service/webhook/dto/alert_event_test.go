package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAlertEventFromJSON(t *testing.T) {
	t.Parallel()

	alertID := uuid.New()
	monitorID := uuid.New()
	userID := uuid.New()
	triggeredAt := time.Now().UTC().Truncate(time.Second)

	t.Run("valid event", func(t *testing.T) {
		t.Parallel()

		event := AlertEvent{
			UserID:              userID,
			AlertID:             alertID,
			MonitorID:           monitorID,
			MonitorName:         "My Monitor",
			MonitorStatus:       "down",
			Severity:            "critical",
			ConsecutiveFailures: 3,
			IsFlapping:          false,
			FlapCount:           0,
			TriggeredAt:         triggeredAt,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		parsed, err := ParseAlertEventFromJSON(data)
		require.NoError(t, err)
		require.NotNil(t, parsed)
		assert.Equal(t, alertID, parsed.AlertID)
		assert.Equal(t, monitorID, parsed.MonitorID)
		assert.Equal(t, "My Monitor", parsed.MonitorName)
		assert.Equal(t, "critical", parsed.Severity)
		assert.Equal(t, 3, parsed.ConsecutiveFailures)
	})

	t.Run("missing alert_id returns error", func(t *testing.T) {
		t.Parallel()

		data, err := json.Marshal(map[string]any{
			"monitor_id":   monitorID.String(),
			"triggered_at": triggeredAt,
		})
		require.NoError(t, err)

		_, err = ParseAlertEventFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "alert_id")
	})

	t.Run("missing monitor_id returns error", func(t *testing.T) {
		t.Parallel()

		// When MonitorID is uuid.Nil, must get an error about monitor_id
		event := AlertEvent{
			AlertID:     alertID,
			MonitorID:   uuid.Nil, // missing
			TriggeredAt: triggeredAt,
		}
		data, err := json.Marshal(event)
		require.NoError(t, err)

		_, err = ParseAlertEventFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "monitor_id")
	})

	t.Run("zero triggered_at gets filled with current time", func(t *testing.T) {
		t.Parallel()

		event := AlertEvent{
			AlertID:   alertID,
			MonitorID: monitorID,
			// TriggeredAt is zero - will be set by ParseAlertEventFromJSON
		}
		data, err := json.Marshal(event)
		require.NoError(t, err)

		parsed, err := ParseAlertEventFromJSON(data)
		require.NoError(t, err)
		assert.False(t, parsed.TriggeredAt.IsZero())
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAlertEventFromJSON([]byte("not-json"))
		require.Error(t, err)
	})

	t.Run("flapping event", func(t *testing.T) {
		t.Parallel()

		event := AlertEvent{
			UserID:      userID,
			AlertID:     alertID,
			MonitorID:   monitorID,
			IsFlapping:  true,
			FlapCount:   5,
			TriggeredAt: triggeredAt,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		parsed, err := ParseAlertEventFromJSON(data)
		require.NoError(t, err)
		assert.True(t, parsed.IsFlapping)
		assert.Equal(t, 5, parsed.FlapCount)
	})
}
