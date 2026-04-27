package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status  CheckStatus
		wantStr string
	}{
		{CheckStatusUP, "UP"},
		{CheckStatusDOWN, "DOWN"},
		{CheckStatusDEGRADED, "DEGRADED"},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantStr, tt.status.String())
		})
	}
}

func TestCheckHistoryEntry_Fields(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	monitorID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	statusCode := 503
	responseTime := 210.5
	errMsg := "connection refused"

	entry := CheckHistoryEntry{
		ID:             id,
		MonitorID:      monitorID,
		Status:         CheckStatusDOWN,
		StatusCode:     &statusCode,
		ResponseTimeMs: &responseTime,
		ErrorMessage:   &errMsg,
		CheckedAt:      now,
	}

	assert.Equal(t, id, entry.ID)
	assert.Equal(t, monitorID, entry.MonitorID)
	assert.Equal(t, CheckStatusDOWN, entry.Status)
	require.NotNil(t, entry.StatusCode)
	assert.Equal(t, 503, *entry.StatusCode)
	require.NotNil(t, entry.ResponseTimeMs)
	assert.Equal(t, 210.5, *entry.ResponseTimeMs)
	require.NotNil(t, entry.ErrorMessage)
	assert.Equal(t, "connection refused", *entry.ErrorMessage)
	assert.Equal(t, now, entry.CheckedAt)
}

func TestCheckHistoryEntry_NilPointers(t *testing.T) {
	t.Parallel()

	entry := CheckHistoryEntry{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		Status:    CheckStatusUP,
		CheckedAt: time.Now().UTC(),
	}

	assert.Nil(t, entry.StatusCode)
	assert.Nil(t, entry.ResponseTimeMs)
	assert.Nil(t, entry.ErrorMessage)
}
