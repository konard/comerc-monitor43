package consumer

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
	"github.com/raul/monitor/backend/dashboard-service/internal/repository"
	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

type EventConsumer struct {
	amqpURL      string
	conn         *amqp.Connection
	connected    atomic.Bool
	statusRepo   repository.MonitorStatusRepository
	checkRepo    repository.CheckHistoryRepository
	incidentRepo repository.IncidentRepository
	hub          *realtime.Hub
	logger       *applogger.Logger
}

func NewEventConsumer(
	amqpURL string,
	statusRepo repository.MonitorStatusRepository,
	checkRepo repository.CheckHistoryRepository,
	incidentRepo repository.IncidentRepository,
	hub *realtime.Hub,
	logger *applogger.Logger,
) *EventConsumer {
	return &EventConsumer{
		amqpURL:      amqpURL,
		statusRepo:   statusRepo,
		checkRepo:    checkRepo,
		incidentRepo: incidentRepo,
		hub:          hub,
		logger:       logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) error {
	conn, err := amqp.Dial(c.amqpURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to RabbitMQ")
	}
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open channel")
	}

	err = ch.ExchangeDeclare("monitor-events", "topic", true, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "failed to declare exchange")
	}

	err = ch.ExchangeDeclare("monitor-events-dlq", "fanout", true, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "failed to declare DLQ exchange")
	}

	q, err := ch.QueueDeclare(
		"dashboard-service-events", true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    "monitor-events-dlq",
			"x-dead-letter-routing-key": "dashboard-service-events-dlq",
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare queue")
	}

	_, err = ch.QueueDeclare("dashboard-service-events-dlq", true, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "failed to declare DLQ queue")
	}
	err = ch.QueueBind("dashboard-service-events-dlq", "", "monitor-events-dlq", false, nil)
	if err != nil {
		return errors.Wrap(err, "failed to bind DLQ queue")
	}

	topics := []string{
		"monitor.created", "monitor.updated", "monitor.deleted", "monitor.paused", "monitor.resumed",
		"check.completed", "check.failed",
		"incident.detected", "incident.resolved",
		"alert.triggered", "alert.resolved",
	}
	for _, topic := range topics {
		if err := ch.QueueBind(q.Name, topic, "monitor-events", false, nil); err != nil {
			return errors.Wrap(err, "failed to bind queue")
		}
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "failed to consume")
	}

	c.connected.Store(true)

	go func() {
		defer c.connected.Store(false)
		for {
			select {
			case <-ctx.Done():
				if err := ch.Close(); err != nil {
					c.logger.Error("failed to close channel", "error", err)
				}
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := c.handleMessage(ctx, msg); err != nil {
					c.logger.Error("failed to process event", "routing_key", msg.RoutingKey, "error", err)
					if err := msg.Nack(false, false); err != nil {
						c.logger.Error("failed to nack message", "error", err)
					}
				} else {
					if err := msg.Ack(false); err != nil {
						c.logger.Error("failed to ack message", "error", err)
					}
				}
			}
		}
	}()

	c.logger.Info("Event consumer started")
	return nil
}

func (c *EventConsumer) Close() error {
	c.connected.Store(false)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *EventConsumer) IsConnected() bool {
	return c.connected.Load()
}

func (c *EventConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var event map[string]any
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return errors.Wrap(err, "failed to unmarshal event")
	}

	eventType, _ := event["event_type"].(string)
	// fallback на routing key если publisher не включил event_type в тело
	if eventType == "" {
		eventType = msg.RoutingKey
	}
	userID, _ := event["user_id"].(string)

	switch eventType {
	case "monitor.created", "monitor.updated":
		if err := c.handleMonitorUpsert(ctx, event); err != nil {
			return errors.Wrap(err, "failed to handle monitor upsert")
		}
	case "monitor.deleted":
		if err := c.handleMonitorDelete(ctx, event); err != nil {
			return errors.Wrap(err, "failed to handle monitor delete")
		}
	case "monitor.paused", "monitor.resumed":
		if err := c.handleMonitorStatusChange(ctx, event); err != nil {
			return errors.Wrap(err, "failed to handle monitor status change")
		}
	case "check.completed", "check.failed":
		if err := c.handleCheckResult(ctx, event); err != nil {
			return errors.Wrap(err, "failed to handle check result")
		}
	case "incident.detected":
		if err := c.handleIncidentDetected(ctx, event); err != nil {
			return errors.Wrap(err, "failed to handle incident detected")
		}
	case "incident.resolved":
		if err := c.handleIncidentResolved(ctx, event); err != nil {
			return errors.Wrap(err, "failed to handle incident resolved")
		}
	default:
		return errors.New("unknown event type: " + eventType)
	}

	c.hub.BroadcastToUser(userID, &realtime.BroadcastMessage{
		Type:    eventType,
		Payload: event,
	})

	return nil
}

func (c *EventConsumer) handleMonitorUpsert(ctx context.Context, event map[string]any) error {
	id, _ := event["monitor_id"].(string)
	userID, _ := event["user_id"].(string)
	name, _ := event["name"].(string)
	url, _ := event["url"].(string)
	status, _ := event["status"].(string)

	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, "invalid monitor_id")
	}
	ud, err := uuid.Parse(userID)
	if err != nil {
		return errors.Wrap(err, "invalid user_id")
	}

	view := &model.MonitorStatusView{
		ID:        uid,
		UserID:    ud,
		Name:      name,
		URL:       url,
		Status:    model.MonitorStatus(status),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := c.statusRepo.Upsert(ctx, view); err != nil {
		return errors.Wrap(err, "failed to upsert monitor status")
	}
	return nil
}

func (c *EventConsumer) handleMonitorDelete(ctx context.Context, event map[string]any) error {
	id, _ := event["monitor_id"].(string)
	if err := c.statusRepo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "failed to delete monitor status")
	}
	return nil
}

func (c *EventConsumer) handleMonitorStatusChange(ctx context.Context, event map[string]any) error {
	id, _ := event["monitor_id"].(string)
	status, _ := event["status"].(string)

	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, "invalid monitor_id")
	}

	view := &model.MonitorStatusView{
		ID:     uid,
		UserID: uuid.Nil,
		Status: model.MonitorStatus(status),
	}
	if err := c.statusRepo.Upsert(ctx, view); err != nil {
		return errors.Wrap(err, "failed to update monitor status")
	}
	return nil
}

func (c *EventConsumer) handleCheckResult(ctx context.Context, event map[string]any) error {
	id := uuid.New().String()
	monitorID, _ := event["monitor_id"].(string)
	status, _ := event["status"].(string)

	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, "invalid check_id")
	}
	mid, err := uuid.Parse(monitorID)
	if err != nil {
		return errors.Wrap(err, "invalid monitor_id")
	}

	entry := &model.CheckHistoryEntry{
		ID:        uid,
		MonitorID: mid,
		Status:    model.CheckStatus(status),
		CheckedAt: time.Now(),
	}
	if rt, ok := event["response_time_ms"].(float64); ok {
		entry.ResponseTimeMs = &rt
	}
	if sc, ok := event["status_code"].(float64); ok {
		code := int(sc)
		entry.StatusCode = &code
	}
	if em, ok := event["error_message"].(string); ok && em != "" {
		entry.ErrorMessage = &em
	}

	if err := c.checkRepo.Create(ctx, entry); err != nil {
		return errors.Wrap(err, "failed to create check history")
	}
	return nil
}

func (c *EventConsumer) handleIncidentDetected(ctx context.Context, event map[string]any) error {
	id, _ := event["incident_id"].(string)
	monitorID, _ := event["monitor_id"].(string)

	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, "invalid incident_id")
	}
	mid, err := uuid.Parse(monitorID)
	if err != nil {
		return errors.Wrap(err, "invalid monitor_id")
	}

	incident := &model.Incident{
		ID:        uid,
		MonitorID: mid,
		StartedAt: time.Now(),
		Status:    model.IncidentStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := c.incidentRepo.Upsert(ctx, incident); err != nil {
		return errors.Wrap(err, "failed to upsert incident")
	}
	return nil
}

func (c *EventConsumer) handleIncidentResolved(ctx context.Context, event map[string]any) error {
	id, _ := event["incident_id"].(string)
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.Wrap(err, "invalid incident_id")
	}

	incident, err := c.incidentRepo.GetByID(ctx, uid.String())
	if err != nil {
		return errors.Wrap(err, "failed to get incident for resolve")
	}

	now := time.Now()
	duration := int64(now.Sub(incident.StartedAt).Seconds())
	incident.EndedAt = &now
	incident.DurationSeconds = &duration
	incident.Status = model.IncidentStatusResolved
	incident.UpdatedAt = now

	if err := c.incidentRepo.Update(ctx, incident); err != nil {
		return errors.Wrap(err, "failed to resolve incident")
	}
	return nil
}
