package webhook

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/payment"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// Handler обрабатывает вебхуки от платежных провайдеров.
type Handler struct {
	providers   map[string]payment.PaymentProvider
	subRepo     SubscriptionRepository
	paymentRepo PaymentRepository
	log         *slog.Logger
}

// SubscriptionRepository определяет интерфейс для работы с подписками.
type SubscriptionRepository interface {
	GetByID(ctx context.Context, id string) (*Subscription, error)
	Update(ctx context.Context, sub *Subscription) error
}

// PaymentRepository определяет интерфейс для работы с платежами.
type PaymentRepository interface {
	GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*Payment, error)
	Update(ctx context.Context, pmt *Payment) error
}

// Subscription представляет подписку.
type Subscription struct {
	ID        string
	UserID    string
	PlanID    string
	Status    string
	ExpiresAt string
}

// Payment представляет платёж.
type Payment struct {
	ID                string
	UserID            string
	SubscriptionID    *string
	Status            string
	Provider          string
	ProviderPaymentID string
}

// NewHandler создаёт новый Handler.
func NewHandler(
	yookassaProvider payment.PaymentProvider,
	stripeProvider payment.PaymentProvider,
	subRepo SubscriptionRepository,
	paymentRepo PaymentRepository,
	logger *slog.Logger,
) *Handler {
	providers := make(map[string]payment.PaymentProvider)
	if yookassaProvider != nil {
		providers["YOOKASSA"] = yookassaProvider
	}
	if stripeProvider != nil {
		providers["STRIPE"] = stripeProvider
	}

	return &Handler{
		providers:   providers,
		subRepo:     subRepo,
		paymentRepo: paymentRepo,
		log:         logger,
	}
}

// Handle обрабатывает вебхук.
func (h *Handler) Handle(ctx context.Context, provider string, payload []byte, signature string) error {
	// 1. Получаем провайдера
	p, ok := h.providers[provider]
	if !ok {
		return billingerrors.InvalidArgument("provider", "unknown provider")
	}

	// 2. Верифицируем сигнатуру
	if err := p.VerifyWebhookSignature(ctx, payload, signature); err != nil {
		h.log.ErrorContext(ctx, "webhook signature verification failed",
			"provider", provider,
			"error", err,
		)
		return billingerrors.ErrPermissionDenied
	}

	// 3. Парсим событие
	event, err := p.ParseWebhookEvent(ctx, payload)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to parse webhook event",
			"provider", provider,
			"error", err,
		)
		return errors.Wrap(err, "failed to parse webhook")
	}

	h.log.InfoContext(ctx, "processing webhook event",
		"provider", provider,
		"event_type", event.EventType,
		"payment_id", event.PaymentID,
	)

	// 4. Обрабатываем событие
	switch event.EventType {
	case "payment.succeeded", "payment_intent.succeeded":
		return h.handlePaymentSucceeded(ctx, provider, event)
	case "payment.canceled", "payment_intent.payment_failed":
		return h.handlePaymentFailed(ctx, provider, event)
	case "refund.succeeded", "charge.refunded":
		return h.handleRefundSucceeded(ctx, provider, event)
	default:
		h.log.WarnContext(ctx, "unknown webhook event type",
			"event_type", event.EventType,
		)
		return nil
	}
}

// handlePaymentSucceeded обрабатывает успешный платёж.
func (h *Handler) handlePaymentSucceeded(ctx context.Context, provider string, event *payment.WebhookEvent) error {
	// 1. Находим платёж по provider payment ID
	pmt, err := h.paymentRepo.GetByProviderPaymentID(ctx, event.PaymentID)
	if err != nil {
		h.log.ErrorContext(ctx, "payment not found",
			"provider_payment_id", event.PaymentID,
			"error", err,
		)
		return billingerrors.NotFound("payment", event.PaymentID)
	}

	// 2. Проверяем текущий статус
	if pmt.Status == "SUCCESS" {
		h.log.InfoContext(ctx, "payment already successful",
			"payment_id", pmt.ID,
		)
		return nil
	}

	// 3. Обновляем статус платежа
	pmt.Status = "SUCCESS"
	if err := h.paymentRepo.Update(ctx, pmt); err != nil {
		h.log.ErrorContext(ctx, "failed to update payment status",
			"payment_id", pmt.ID,
			"error", err,
		)
		return errors.Wrap(err, "failed to update payment")
	}

	// 4. Если есть связанная подписка, активируем её
	if pmt.SubscriptionID != nil {
		if err := h.activateSubscription(ctx, *pmt.SubscriptionID); err != nil {
			h.log.ErrorContext(ctx, "failed to activate subscription",
				"subscription_id", *pmt.SubscriptionID,
				"error", err,
			)
			return errors.Wrap(err, "failed to activate subscription")
		}
	}

	h.log.InfoContext(ctx, "payment processed successfully",
		"payment_id", pmt.ID,
		"user_id", pmt.UserID,
		"subscription_id", pmt.SubscriptionID,
	)

	return nil
}

// handlePaymentFailed обрабатывает неудачный платёж.
func (h *Handler) handlePaymentFailed(ctx context.Context, provider string, event *payment.WebhookEvent) error {
	// 1. Находим платёж
	pmt, err := h.paymentRepo.GetByProviderPaymentID(ctx, event.PaymentID)
	if err != nil {
		h.log.ErrorContext(ctx, "payment not found",
			"provider_payment_id", event.PaymentID,
			"error", err,
		)
		return billingerrors.NotFound("payment", event.PaymentID)
	}

	// 2. Обновляем статус платежа
	pmt.Status = "FAILED"
	if err := h.paymentRepo.Update(ctx, pmt); err != nil {
		h.log.ErrorContext(ctx, "failed to update payment status",
			"payment_id", pmt.ID,
			"error", err,
		)
		return errors.Wrap(err, "failed to update payment")
	}

	// 3. Если есть связанная подписка, отменяем её
	if pmt.SubscriptionID != nil {
		if err := h.cancelSubscription(ctx, *pmt.SubscriptionID, "payment failed"); err != nil {
			h.log.ErrorContext(ctx, "failed to cancel subscription",
				"subscription_id", *pmt.SubscriptionID,
				"error", err,
			)
			return errors.Wrap(err, "failed to cancel subscription")
		}
	}

	h.log.InfoContext(ctx, "payment marked as failed",
		"payment_id", pmt.ID,
		"user_id", pmt.UserID,
	)

	return nil
}

// handleRefundSucceeded обрабатывает успешный возврат.
func (h *Handler) handleRefundSucceeded(ctx context.Context, provider string, event *payment.WebhookEvent) error {
	// 1. Находим платёж
	pmt, err := h.paymentRepo.GetByProviderPaymentID(ctx, event.PaymentID)
	if err != nil {
		h.log.ErrorContext(ctx, "payment not found",
			"provider_payment_id", event.PaymentID,
			"error", err,
		)
		return billingerrors.NotFound("payment", event.PaymentID)
	}

	// 2. Обновляем статус платежа на REFUNDED
	pmt.Status = "REFUNDED"
	if err := h.paymentRepo.Update(ctx, pmt); err != nil {
		h.log.ErrorContext(ctx, "failed to update payment status",
			"payment_id", pmt.ID,
			"error", err,
		)
		return errors.Wrap(err, "failed to update payment")
	}

	// 3. Если есть связанная подписка, отменяем её
	if pmt.SubscriptionID != nil {
		if err := h.cancelSubscription(ctx, *pmt.SubscriptionID, "payment refunded"); err != nil {
			h.log.ErrorContext(ctx, "failed to cancel subscription",
				"subscription_id", *pmt.SubscriptionID,
				"error", err,
			)
			return errors.Wrap(err, "failed to cancel subscription")
		}
	}

	h.log.InfoContext(ctx, "payment refunded successfully",
		"payment_id", pmt.ID,
		"user_id", pmt.UserID,
	)

	return nil
}

// activateSubscription активирует подписку.
func (h *Handler) activateSubscription(ctx context.Context, subscriptionID string) error {
	sub, err := h.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	sub.Status = "ACTIVE"

	if err := h.subRepo.Update(ctx, sub); err != nil {
		return err
	}

	h.log.InfoContext(ctx, "subscription activated",
		"subscription_id", subscriptionID,
		"user_id", sub.UserID,
		"plan_id", sub.PlanID,
	)

	return nil
}

// cancelSubscription отменяет подписку.
func (h *Handler) cancelSubscription(ctx context.Context, subscriptionID, reason string) error {
	sub, err := h.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	sub.Status = "CANCELED"

	if err := h.subRepo.Update(ctx, sub); err != nil {
		return err
	}

	h.log.InfoContext(ctx, "subscription canceled",
		"subscription_id", subscriptionID,
		"user_id", sub.UserID,
		"reason", reason,
	)

	return nil
}
