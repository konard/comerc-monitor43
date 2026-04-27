// Package publisher предоставляет публикацию событий о мониторинге.
//
// Пакет публикует события в RabbitMQ для других сервисов:
//   - alerting: для отправки уведомлений
//   - dashboard: для обновления UI в реальном времени
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

// EventPublisher публикует события мониторинга.
type EventPublisher struct {
	channel  Channel
	exchange string
	logger   *slog.Logger
}

// Channel описывает интерфейс для AMQP channel.
type Channel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

// NewEventPublisher создаёт новый EventPublisher.
func NewEventPublisher(channel Channel, exchange string, logger *slog.Logger) *EventPublisher {
	return &EventPublisher{
		channel:  channel,
		exchange: exchange,
		logger:   logger,
	}
}

// SetupTopology настраивает topology RabbitMQ.
func (p *EventPublisher) SetupTopology(ctx context.Context) error {
	// Объявляем exchange
	if err := p.channel.ExchangeDeclare(
		p.exchange,
		"topic", // topic exchange для маршрутизации по ключам
		true,    // durable
		false,   // auto-delete
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	); err != nil {
		return errors.Wrap(err, "failed to declare exchange")
	}

	return nil
}

// MonitorStatusChangedEvent событие изменения статуса монитора.
type MonitorStatusChangedEvent struct {
	EventType    string    `json:"event_type"`
	MonitorID    uuid.UUID `json:"monitor_id"`
	UserID       uuid.UUID `json:"user_id"`
	OldStatus    string    `json:"old_status"`
	NewStatus    string    `json:"new_status"`
	Timestamp    int64     `json:"timestamp"`
	ResponseTime *int      `json:"response_time,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// PublishStatusChange публикует событие изменения статуса.
func (p *EventPublisher) PublishStatusChange(ctx context.Context, event *MonitorStatusChangedEvent) error {
	// Маршрутизируем по ключу: monitor.status.<new_status>
	routingKey := fmt.Sprintf("monitor.status.%s", event.NewStatus)
	// event_type дублирует routing key в теле — consumer может роутить по обоим
	event.EventType = routingKey

	body, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal event")
	}

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Сохранять сообщения при перезапуске
			Timestamp:    time.Now(),
		},
	); err != nil {
		return errors.Wrap(err, "failed to publish event")
	}

	p.logger.InfoContext(ctx, "Published status change event",
		"monitor_id", event.MonitorID,
		"old_status", event.OldStatus,
		"new_status", event.NewStatus,
		"routing_key", routingKey,
	)

	return nil
}

// MonitorCreatedEvent событие создания монитора.
type MonitorCreatedEvent struct {
	EventType string    `json:"event_type"`
	MonitorID uuid.UUID `json:"monitor_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Timestamp int64     `json:"timestamp"`
}

// PublishMonitorCreated публикует событие создания монитора.
func (p *EventPublisher) PublishMonitorCreated(ctx context.Context, event *MonitorCreatedEvent) error {
	routingKey := "monitor.created"
	event.EventType = routingKey

	body, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal event")
	}

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	); err != nil {
		return errors.Wrap(err, "failed to publish event")
	}

	p.logger.InfoContext(ctx, "Published monitor created event",
		"monitor_id", event.MonitorID,
		"user_id", event.UserID,
		"name", event.Name,
	)

	return nil
}

// MonitorDeletedEvent событие удаления монитора.
type MonitorDeletedEvent struct {
	EventType string    `json:"event_type"`
	MonitorID uuid.UUID `json:"monitor_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Timestamp int64     `json:"timestamp"`
}

// PublishMonitorDeleted публикует событие удаления монитора.
func (p *EventPublisher) PublishMonitorDeleted(ctx context.Context, event *MonitorDeletedEvent) error {
	routingKey := "monitor.deleted"
	event.EventType = routingKey

	body, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal event")
	}

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	); err != nil {
		return errors.Wrap(err, "failed to publish event")
	}

	p.logger.InfoContext(ctx, "Published monitor deleted event",
		"monitor_id", event.MonitorID,
		"user_id", event.UserID,
	)

	return nil
}

// Close закрывает publisher.
func (p *EventPublisher) Close() error {
	// Channel закрывается извне
	return nil
}

// NoopPublisher является no-op реализацией EventPublisher для тестов и отключенной интеграции.
type NoopPublisher struct{}

// NewNoopPublisher создаёт новый NoopPublisher.
func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

// PublishStatusChange не делает ничего.
func (p *NoopPublisher) PublishStatusChange(ctx context.Context, event any) error {
	return nil
}
