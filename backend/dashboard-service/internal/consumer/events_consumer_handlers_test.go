package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

// --- заглушки репозиториев ---

type stubStatusRepo struct {
	upsertCalled bool
	deleteCalled bool
	deleteID     string
	err          error
}

func (s *stubStatusRepo) Upsert(_ context.Context, _ *model.MonitorStatusView) error {
	s.upsertCalled = true
	return s.err
}
func (s *stubStatusRepo) Delete(_ context.Context, id string) error {
	s.deleteCalled = true
	s.deleteID = id
	return s.err
}
func (s *stubStatusRepo) ListByUserID(_ context.Context, _ string, _ model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
	return nil, 0, nil
}
func (s *stubStatusRepo) GetByID(_ context.Context, _ string) (*model.MonitorStatusView, error) {
	return nil, nil
}
func (s *stubStatusRepo) GetOverallUptime(_ context.Context, _ string, _ []model.MonitorStatus) (float64, error) {
	return 0, nil
}

type stubCheckRepo struct {
	createCalled bool
	err          error
}

func (s *stubCheckRepo) Create(_ context.Context, _ *model.CheckHistoryEntry) error {
	s.createCalled = true
	return s.err
}
func (s *stubCheckRepo) ListByMonitorID(_ context.Context, _ string, _ model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
	return nil, 0, nil
}
func (s *stubCheckRepo) GetPeriodMetrics(_ context.Context, _ string, _, _ time.Time) (*model.PeriodMetrics, error) {
	return nil, nil
}

type stubIncidentRepo struct {
	upsertCalled bool
	updateCalled bool
	incident     *model.Incident
	err          error
	getErr       error
}

func (s *stubIncidentRepo) Upsert(_ context.Context, _ *model.Incident) error {
	s.upsertCalled = true
	return s.err
}
func (s *stubIncidentRepo) ListByMonitorID(_ context.Context, _ string, _ model.IncidentFilter) ([]*model.Incident, int, error) {
	return nil, 0, nil
}
func (s *stubIncidentRepo) GetByID(_ context.Context, _ string) (*model.Incident, error) {
	return s.incident, s.getErr
}
func (s *stubIncidentRepo) Update(_ context.Context, _ *model.Incident) error {
	s.updateCalled = true
	return s.err
}

// --- вспомогательная функция ---

func newConsumer(statusRepo *stubStatusRepo, checkRepo *stubCheckRepo, incidentRepo *stubIncidentRepo) *EventConsumer {
	hub := realtime.NewHub()
	go hub.Run()
	return NewEventConsumer(
		"amqp://localhost:5672",
		statusRepo,
		checkRepo,
		incidentRepo,
		hub,
		applogger.New("error"),
	)
}

func makeDelivery(t *testing.T, body map[string]any) amqp.Delivery {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	return amqp.Delivery{Body: data, RoutingKey: "test"}
}

// --- тесты handleMessage ---

func TestHandleMessage_monitor_created(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.created",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"name":       "my-monitor",
		"url":        "https://example.com",
		"status":     "UP",
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, statusRepo.upsertCalled)
}

func TestHandleMessage_monitor_updated(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.updated",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"name":       "updated",
		"url":        "https://updated.com",
		"status":     "DOWN",
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, statusRepo.upsertCalled)
}

func TestHandleMessage_monitor_deleted(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	monitorID := uuid.New().String()
	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.deleted",
		"monitor_id": monitorID,
		"user_id":    uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, statusRepo.deleteCalled)
	assert.Equal(t, monitorID, statusRepo.deleteID)
}

func TestHandleMessage_monitor_paused(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.paused",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"status":     "PAUSED",
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, statusRepo.upsertCalled)
}

func TestHandleMessage_monitor_resumed(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.resumed",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
		"status":     "UP",
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, statusRepo.upsertCalled)
}

func TestHandleMessage_check_completed(t *testing.T) {
	t.Parallel()

	checkRepo := &stubCheckRepo{}
	c := newConsumer(&stubStatusRepo{}, checkRepo, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type":       "check.completed",
		"monitor_id":       uuid.New().String(),
		"user_id":          uuid.New().String(),
		"status":           "UP",
		"response_time_ms": float64(42),
		"status_code":      float64(200),
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, checkRepo.createCalled)
}

func TestHandleMessage_check_failed_with_error_message(t *testing.T) {
	t.Parallel()

	checkRepo := &stubCheckRepo{}
	c := newConsumer(&stubStatusRepo{}, checkRepo, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type":    "check.failed",
		"monitor_id":    uuid.New().String(),
		"user_id":       uuid.New().String(),
		"status":        "DOWN",
		"error_message": "connection refused",
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, checkRepo.createCalled)
}

func TestHandleMessage_incident_detected(t *testing.T) {
	t.Parallel()

	incidentRepo := &stubIncidentRepo{}
	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, incidentRepo)

	msg := makeDelivery(t, map[string]any{
		"event_type":  "incident.detected",
		"incident_id": uuid.New().String(),
		"monitor_id":  uuid.New().String(),
		"user_id":     uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, incidentRepo.upsertCalled)
}

func TestHandleMessage_incident_resolved(t *testing.T) {
	t.Parallel()

	incidentID := uuid.New()
	startedAt := time.Now().Add(-time.Hour)
	incidentRepo := &stubIncidentRepo{
		incident: &model.Incident{
			ID:        incidentID,
			MonitorID: uuid.New(),
			StartedAt: startedAt,
			Status:    model.IncidentStatusActive,
			CreatedAt: startedAt,
			UpdatedAt: startedAt,
		},
	}
	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, incidentRepo)

	msg := makeDelivery(t, map[string]any{
		"event_type":  "incident.resolved",
		"incident_id": incidentID.String(),
		"user_id":     uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, incidentRepo.updateCalled)
}

func TestHandleMessage_unknown_event_type(t *testing.T) {
	t.Parallel()

	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "unknown.event",
		"user_id":    uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestHandleMessage_invalid_json(t *testing.T) {
	t.Parallel()

	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, &stubIncidentRepo{})

	delivery := amqp.Delivery{Body: []byte("not json"), RoutingKey: "test"}
	err := c.handleMessage(context.Background(), delivery)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal event")
}

func TestHandleMessage_monitor_upsert_repo_error(t *testing.T) {
	t.Parallel()

	statusRepo := &stubStatusRepo{err: assert.AnError}
	c := newConsumer(statusRepo, &stubCheckRepo{}, &stubIncidentRepo{})

	msg := makeDelivery(t, map[string]any{
		"event_type": "monitor.created",
		"monitor_id": uuid.New().String(),
		"user_id":    uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
}

func TestHandleMessage_incident_resolved_get_error(t *testing.T) {
	t.Parallel()

	incidentRepo := &stubIncidentRepo{getErr: assert.AnError}
	c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, incidentRepo)

	msg := makeDelivery(t, map[string]any{
		"event_type":  "incident.resolved",
		"incident_id": uuid.New().String(),
		"user_id":     uuid.New().String(),
	})

	err := c.handleMessage(context.Background(), msg)

	assert.Error(t, err)
}

func TestHandleMessage_alert_events_are_unknown(t *testing.T) {
	t.Parallel()

	for _, eventType := range []string{"alert.triggered", "alert.resolved"} {
		t.Run(eventType, func(t *testing.T) {
			t.Parallel()

			c := newConsumer(&stubStatusRepo{}, &stubCheckRepo{}, &stubIncidentRepo{})

			msg := makeDelivery(t, map[string]any{
				"event_type": eventType,
				"user_id":    uuid.New().String(),
			})

			// alert.* не обрабатываются — ожидаем ошибку unknown event type
			err := c.handleMessage(context.Background(), msg)
			assert.Error(t, err)
		})
	}
}
