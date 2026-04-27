package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncidentStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status  IncidentStatus
		wantStr string
	}{
		{IncidentStatusActive, "ACTIVE"},
		{IncidentStatusResolved, "RESOLVED"},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantStr, tt.status.String())
		})
	}
}

func TestIncident_Fields(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	monitorID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	endTime := now.Add(30 * time.Minute)
	duration := int64(1800)

	incident := Incident{
		ID:              id,
		MonitorID:       monitorID,
		StartedAt:       now,
		EndedAt:         &endTime,
		DurationSeconds: &duration,
		CheckCount:      15,
		Status:          IncidentStatusResolved,
		CreatedAt:       now,
		UpdatedAt:       endTime,
	}

	assert.Equal(t, id, incident.ID)
	assert.Equal(t, monitorID, incident.MonitorID)
	assert.Equal(t, now, incident.StartedAt)
	require.NotNil(t, incident.EndedAt)
	assert.Equal(t, endTime, *incident.EndedAt)
	require.NotNil(t, incident.DurationSeconds)
	assert.Equal(t, int64(1800), *incident.DurationSeconds)
	assert.Equal(t, 15, incident.CheckCount)
	assert.Equal(t, IncidentStatusResolved, incident.Status)
	assert.Equal(t, now, incident.CreatedAt)
	assert.Equal(t, endTime, incident.UpdatedAt)
}

func TestIncidentDetail_Fields(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	incident := Incident{
		ID:        uuid.New(),
		MonitorID: monitorID,
		StartedAt: now,
		Status:    IncidentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	timeline := []CheckHistoryEntry{
		{
			ID:        uuid.New(),
			MonitorID: monitorID,
			Status:    CheckStatusDOWN,
			CheckedAt: now,
		},
		{
			ID:        uuid.New(),
			MonitorID: monitorID,
			Status:    CheckStatusUP,
			CheckedAt: now.Add(1 * time.Minute),
		},
	}

	detail := IncidentDetail{
		Incident: incident,
		Timeline: timeline,
	}

	assert.Equal(t, incident.ID, detail.Incident.ID)
	assert.Equal(t, monitorID, detail.Incident.MonitorID)
	assert.Equal(t, IncidentStatusActive, detail.Incident.Status)
	require.Len(t, detail.Timeline, 2)
	assert.Equal(t, CheckStatusDOWN, detail.Timeline[0].Status)
	assert.Equal(t, CheckStatusUP, detail.Timeline[1].Status)
}

func TestIncident_Fields_NilPointers(t *testing.T) {
	t.Parallel()

	incident := Incident{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		StartedAt: time.Now().UTC(),
		Status:    IncidentStatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	assert.Nil(t, incident.EndedAt)
	assert.Nil(t, incident.DurationSeconds)
}
