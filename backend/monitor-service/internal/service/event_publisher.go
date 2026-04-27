package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/pure-golang/adapters/queue"
)

// EventPublisherImpl реализация EventPublisher на основе RabbitMQ.
type EventPublisherImpl struct {
	publisher queue.Publisher
	logger    *slog.Logger
}

// NewEventPublisherImpl создаёт новый EventPublisherImpl.
func NewEventPublisherImpl(publisher queue.Publisher, logger *slog.Logger) *EventPublisherImpl {
	return &EventPublisherImpl{
		publisher: publisher,
		logger:    logger,
	}
}

// PublishStatusChange публикует событие изменения статуса.
func (p *EventPublisherImpl) PublishStatusChange(ctx context.Context, event *StatusChangeEvent) error {
	// Конвертируем в JSON compatible format
	eventData := map[string]any{
		"monitor_id": event.MonitorID.String(),
		"old_status": string(event.OldStatus),
		"new_status": string(event.NewStatus),
		"timestamp":  event.Timestamp.Unix(),
	}

	if event.ResponseTime != nil {
		eventData["response_time"] = *event.ResponseTime
	}
	if event.ErrorMessage != nil {
		eventData["error_message"] = *event.ErrorMessage
	}

	// Создаём сообщение для очереди
	message := queue.Message{
		Topic: fmt.Sprintf("monitor.status.%s", event.NewStatus),
		Body:  eventData,
		Headers: map[string]string{
			"monitor_id": event.MonitorID.String(),
			"old_status": string(event.OldStatus),
			"new_status": string(event.NewStatus),
			"source":     "monitor-service",
		},
	}

	// Публикуем через rabbitmq publisher
	if err := p.publisher.Publish(ctx, message); err != nil {
		return errors.Wrap(err, "failed to publish event")
	}

	p.logger.InfoContext(ctx, "Published status change event",
		"monitor_id", event.MonitorID,
		"old_status", event.OldStatus,
		"new_status", event.NewStatus,
		"topic", message.Topic,
	)

	return nil
}

// Check проверяет здоровье RabbitMQ соединения.
func (p *EventPublisherImpl) Check(ctx context.Context) error {
	// Проверяем publisher через Publish с test сообщением
	// Если publisher не может опубликовать сообщение, возвращаем ошибку
	if p.publisher == nil {
		return errors.New("rabbitmq publisher is nil")
	}

	// Пытаемся опубликовать test сообщение для проверки соединения
	// Используем специальный routing key для health check
	testMessage := queue.Message{
		Topic: "monitor.health.check",
		Body:  map[string]any{"timestamp": time.Now().Unix()},
	}

	// Публикуем с коротким timeout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := p.publisher.Publish(ctx, testMessage); err != nil {
		return errors.Wrap(err, "rabbitmq health check failed")
	}

	return nil
}
