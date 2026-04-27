package service

import (
	"context"

	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
)

// PlanService определяет интерфейс сервиса тарифных планов.
type PlanService interface {
	// GetAllPlans возвращает все тарифные планы.
	GetAllPlans(ctx context.Context) (*dto.PlansResponse, error)

	// GetPlan возвращает план по ID.
	GetPlan(ctx context.Context, planID string) (*dto.PlanResponse, error)

	// ValidatePlanLimits проверяет лимиты плана.
	ValidatePlanLimits(ctx context.Context, planID string, monitorsCount, checkInterval int) error
}

// SubscriptionService определяет интерфейс сервиса подписок.
type SubscriptionService interface {
	// GetActiveSubscription возвращает активную подписку пользователя.
	GetActiveSubscription(ctx context.Context, userID string) (*dto.SubscriptionResponse, error)

	// CreateSubscription создаёт новую подписку.
	CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error)

	// CancelSubscription отменяет подписку.
	CancelSubscription(ctx context.Context, req *dto.CancelSubscriptionRequest) error

	// RenewSubscription продлевает подписку.
	RenewSubscription(ctx context.Context, subscriptionID string) (*dto.SubscriptionResponse, error)

	// GetUserSubscriptionHistory возвращает историю подписок.
	GetUserSubscriptionHistory(ctx context.Context, userID string, page, pageSize int32) (*dto.SubscriptionHistoryResponse, error)

	// SetPublisher устанавливает publisher для событий.
	SetPublisher(publisher EventPublisher)
}

// PaymentService определяет интерфейс сервиса платежей.
type PaymentService interface {
	// CreateCheckout создаёт платёжную сессию.
	CreateCheckout(ctx context.Context, req *dto.CreatePaymentRequest) (*dto.CheckoutResponse, error)

	// GetPayment возвращает платёж по ID.
	GetPayment(ctx context.Context, paymentID string) (*dto.PaymentResponse, error)

	// GetPaymentHistory возвращает историю платежей.
	GetPaymentHistory(ctx context.Context, req *dto.GetPaymentHistoryRequest) (*dto.PaymentHistoryResponse, error)

	// ProcessWebhook обрабатывает вебхук от платежной системы.
	ProcessWebhook(ctx context.Context, req *dto.WebhookRequest) error

	// RefundPayment выполняет возврат платежа.
	RefundPayment(ctx context.Context, paymentID string) error

	// SetPublisher устанавливает publisher для событий.
	SetPublisher(publisher EventPublisher)
}

// EventPublisher определяет интерфейс для публикации событий.
type EventPublisher interface {
	// PublishSubscriptionCreated публикует событие создания подписки.
	PublishSubscriptionCreated(ctx context.Context, event any) error

	// PublishSubscriptionUpgraded публикует событие улучшения подписки.
	PublishSubscriptionUpgraded(ctx context.Context, event any) error

	// PublishSubscriptionCanceled публикует событие отмены подписки.
	PublishSubscriptionCanceled(ctx context.Context, event any) error

	// PublishPaymentFailed публикует событие неудачного платежа.
	PublishPaymentFailed(ctx context.Context, event any) error
}
