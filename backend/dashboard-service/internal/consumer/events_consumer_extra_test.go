package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
)

// makeIncident создаёт тестовый Incident.
func makeIncident(id uuid.UUID) *model.Incident {
	return &model.Incident{
		ID:        id,
		MonitorID: uuid.New(),
		StartedAt: time.Now().Add(-time.Hour),
		Status:    model.IncidentStatusActive,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
}

// TestClose_sets_connected_false проверяет, что Close сбрасывает флаг connected.
func TestClose_sets_connected_false(t *testing.T) {
	t.Parallel()

	c := &EventConsumer{}
	c.connected.Store(true)

	err := c.Close()

	require.NoError(t, err)
	assert.False(t, c.IsConnected())
}

// TestHandleMessage_monitor_delete_repo_error проверяет ошибку удаления.
func TestHandleMessage_monitor_delete_repo_error(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{err: assert.AnError}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.deleted",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle monitor delete")
}

// TestHandleMessage_monitor_paused_repo_error проверяет ошибку смены статуса.
func TestHandleMessage_monitor_paused_repo_error(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{err: assert.AnError}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.paused",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"status":     "PAUSED",
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle monitor status change")
}

// TestHandleMessage_monitor_resumed_repo_error проверяет ошибку возобновления.
func TestHandleMessage_monitor_resumed_repo_error(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{err: assert.AnError}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.resumed",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"status":     "UP",
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
}

// TestHandleMessage_check_completed_repo_error проверяет ошибку сохранения check result.
func TestHandleMessage_check_completed_repo_error(t *testing.T) {
	t.Parallel()

	checkRepo := &stubCheckRepo{err: assert.AnError}
	c := newConsumer(&stubStatusRepo{}, checkRepo, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "check.completed",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"status":     "UP",
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle check result")
}

// TestHandleMessage_incident_detected_repo_error проверяет ошибку upsert инцидента.
func TestHandleMessage_incident_detected_repo_error(t *testing.T) {
	t.Parallel()

	incidentRepo := &stubIncidentRepo{err: assert.AnError}
	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, incidentRepo)

	msg := makeDelivery(t, map[string]any{
		"event_type":  "incident.detected",
		"incident_id": uuid.New().String(),
		"monitor_id":  uuid.New().String(),
		"user_id":     uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle incident detected")
}

// TestHandleMessage_incident_resolved_update_error проверяет ошибку обновления инцидента при resolve.
func TestHandleMessage_incident_resolved_update_error(t *testing.T) {
	t.Parallel()

	incidentID := uuid.New()
	incidentRepo := &stubIncidentRepo{
		incident: makeIncident(incidentID),
		err:      assert.AnError, // ошибка при Update
	}
	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, incidentRepo)

	msg := makeDelivery(t, map[string]any{
		"event_type":  "incident.resolved",
		"incident_id": incidentID.String(),
		"user_id":     uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle incident resolved")
}

// TestHandleMessage_check_failed_no_optional_fields проверяет обработку без опциональных полей.
func TestHandleMessage_check_failed_no_optional_fields(t *testing.T) {
	t.Parallel()

	checkRepo := &stubCheckRepo{}
	c := newConsumer(&stubStatusRepo{}, checkRepo, &stubIncidentRepo{})

	// нет response_time_ms, status_code, error_message
	msg := makeDelivery(t, map[string]any{
		"event_type": "check.failed",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"status":     "DOWN",
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, checkRepo.createCalled)
}

// TestHandleMessage_monitor_updated_repo_error проверяет ошибку при monitor.updated.
func TestHandleMessage_monitor_updated_repo_error(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{err: assert.AnError}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.updated",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"name":       "test",
		"url":        "http://example.com",
		"status":     "UP",
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle monitor upsert")
}
