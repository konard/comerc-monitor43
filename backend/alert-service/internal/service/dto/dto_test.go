package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
}

func TestAlertRequest_Fields(t *testing.T) {
	req := AlertRequest{
		MonitorID: "monitor-1",
		RuleID:    "rule-1",
		Status:    "triggered",
		Metadata:  map[string]any{"key": "value"},
	}
	assert.Equal(t, "monitor-1", req.MonitorID)
	assert.Equal(t, "rule-1", req.RuleID)
	assert.Equal(t, "triggered", req.Status)
}

func TestAlertResponse_Fields(t *testing.T) {
	resp := AlertResponse{
		ID:        "alert-1",
		MonitorID: "monitor-1",
		RuleID:    "rule-1",
		Status:    "triggered",
		CreatedAt: "2026-03-31T00:00:00Z",
	}
	assert.Equal(t, "alert-1", resp.ID)
	assert.Equal(t, "triggered", resp.Status)
}
