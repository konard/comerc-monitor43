// Package publisher предоставляет publisher для событий в RabbitMQ.
//
// Публикует события об изменениях подписок для других сервисов.
//
// События:
//
//	subscription.created - создана новая подписка
//	subscription.upgraded - подписка улучшена
//	subscription.canceled - подписка отменена
//	subscription.expired - подписка истекла
//	subscription.payment.failed - платёж неудачен
package publisher

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/pure-golang/adapters/queue"
)

// EventPublisher публикует события в RabbitMQ.
type EventPublisher struct {
	publisher queue.Publisher
	exchange  string
	log       *slog.Logger
}

// NewEventProducer создаёт новый publisher с использованием queue.Publisher.
func NewEventProducer(pub queue.Publisher, logger *slog.Logger) *EventPublisher {
	return &EventPublisher{
		publisher: pub,
		exchange:  "billing.events",
		log:       logger,
	}
}

// PublishSubscriptionCreated публикует событие создания подписки.
func (p *EventPublisher) PublishSubscriptionCreated(ctx context.Context, event any) error {
	return p.publish(ctx, "subscription.created", event)
}

// PublishSubscriptionUpgraded публикует событие улучшения подписки.
func (p *EventPublisher) PublishSubscriptionUpgraded(ctx context.Context, event any) error {
	return p.publish(ctx, "subscription.upgraded", event)
}

// PublishSubscriptionCanceled публикует событие отмены подписки.
func (p *EventPublisher) PublishSubscriptionCanceled(ctx context.Context, event any) error {
	return p.publish(ctx, "subscription.canceled", event)
}

// PublishSubscriptionExpired публикует событие истечения подписки.
func (p *EventPublisher) PublishSubscriptionExpired(ctx context.Context, event any) error {
	return p.publish(ctx, "subscription.expired", event)
}

// PublishPaymentFailed публикует событие неудачного платежа.
func (p *EventPublisher) PublishPaymentFailed(ctx context.Context, event any) error {
	return p.publish(ctx, "subscription.payment.failed", event)
}

// publish публикует событие в RabbitMQ.
func (p *EventPublisher) publish(ctx context.Context, routingKey string, event any) error {
	msg := queue.Message{
		Topic: routingKey,
		Body:  event,
		Headers: map[string]string{
			"content_type": "application/json",
			"source":       "billing-service",
		},
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return errors.Wrap(err, "failed to publish event")
	}

	p.log.InfoContext(ctx, "event published",
		"routing_key", routingKey,
		"event_type", fmt.Sprintf("%T", event),
	)

	return nil
}

// SubscriptionCreatedEvent событие создания подписки.
type SubscriptionCreatedEvent struct {
	SubscriptionID   string `json:"subscription_id"`
	UserID           string `json:"user_id"`
	PlanID           string `json:"plan_id"`
	MaxMonitors      int32  `json:"max_monitors"`
	MinCheckInterval int32  `json:"min_check_interval_seconds"`
	MaxAlerts        int32  `json:"max_alerts_per_day"`
}

// SubscriptionUpgradedEvent событие улучшения подписки.
type SubscriptionUpgradedEvent struct {
	SubscriptionID   string `json:"subscription_id"`
	UserID           string `json:"user_id"`
	OldPlanID        string `json:"old_plan_id"`
	NewPlanID        string `json:"new_plan_id"`
	MaxMonitors      int32  `json:"max_monitors"`
	MinCheckInterval int32  `json:"min_check_interval_seconds"`
	MaxAlerts        int32  `json:"max_alerts_per_day"`
}

// SubscriptionCanceledEvent событие отмены подписки.
type SubscriptionCanceledEvent struct {
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	PlanID         string `json:"plan_id"`
	Reason         string `json:"reason"`
}

// SubscriptionExpiredEvent событие истечения подписки.
type SubscriptionExpiredEvent struct {
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	PlanID         string `json:"plan_id"`
}

// PaymentFailedEvent событие неудачного платежа.
type PaymentFailedEvent struct {
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	AmountKopeks   int64  `json:"amount_kopeks"`
	Currency       string `json:"currency"`
	Reason         string `json:"reason"`
}
