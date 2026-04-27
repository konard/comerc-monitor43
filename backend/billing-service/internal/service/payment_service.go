package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/payment"
	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// NewPaymentService создаёт новый PaymentService.
func NewPaymentService(
	paymentRepo interfaces.PaymentRepository,
	subRepo interfaces.SubscriptionRepository,
	planRepo interfaces.PlanRepository,
	yookassaProvider payment.PaymentProvider,
	stripeProvider payment.PaymentProvider,
	logger *slog.Logger,
) PaymentService {
	return &paymentService{
		paymentRepo:      paymentRepo,
		subRepo:          subRepo,
		planRepo:         planRepo,
		yookassaProvider: yookassaProvider,
		stripeProvider:   stripeProvider,
		log:              logger,
	}
}

type paymentService struct {
	paymentRepo      interfaces.PaymentRepository
	subRepo          interfaces.SubscriptionRepository
	planRepo         interfaces.PlanRepository
	yookassaProvider payment.PaymentProvider
	stripeProvider   payment.PaymentProvider
	log              *slog.Logger
	publisher        EventPublisher
}

// SetPublisher устанавливает publisher для событий.
func (s *paymentService) SetPublisher(publisher EventPublisher) {
	s.publisher = publisher
}

// CreateCheckout создаёт платёжную сессию.
func (s *paymentService) CreateCheckout(ctx context.Context, req *dto.CreatePaymentRequest) (*dto.CheckoutResponse, error) {
	// Валидация
	if req.UserID == "" {
		return nil, billingerrors.InvalidArgument("user_id", "required")
	}
	if req.PlanID == "" {
		return nil, billingerrors.InvalidArgument("plan_id", "required")
	}

	// Проверяем существование плана
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, billingerrors.NotFound("plan", req.PlanID)
	}

	// Определяем провайдера (пока только Yookassa)
	provider := s.yookassaProvider
	if provider == nil {
		return nil, billingerrors.ErrInternal
	}

	// Формируем запрос на создание платежа
	paymentReq := &payment.CreatePaymentRequest{
		AmountKopeks: plan.PriceKopeks,
		Currency:     "RUB",
		Description:  "Подписка: " + plan.Name,
		Metadata: map[string]string{
			"user_id": req.UserID,
			"plan_id": req.PlanID,
		},
		ReturnURL: req.ReturnURL,
	}

	// Создаём платёж через провайдера
	paymentResp, err := provider.CreatePayment(ctx, paymentReq)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create payment",
			"user_id", req.UserID,
			"plan_id", req.PlanID,
			"error", err,
		)
		return nil, billingerrors.PaymentFailed(err.Error())
	}

	// Создаём подписку
	userUUID := uuid.MustParse(req.UserID)
	duration := time.Duration(plan.BillingPeriodDays) * 24 * time.Hour
	sub := model.NewSubscription(userUUID, req.PlanID, duration)
	sub.ProviderPaymentID = &paymentResp.PaymentID

	if err := s.subRepo.Create(ctx, sub); err != nil {
		s.log.ErrorContext(ctx, "failed to create subscription",
			"user_id", req.UserID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to create subscription")
	}

	// Создаём запись о платеже
	paymentRecord := model.NewPayment(
		userUUID,
		sub.ID,
		"YOOKASSA",
		paymentResp.PaymentID,
		plan.PriceKopeks,
		"RUB",
	)

	if err := s.paymentRepo.Create(ctx, paymentRecord); err != nil {
		s.log.ErrorContext(ctx, "failed to create payment record",
			"user_id", req.UserID,
			"payment_id", paymentResp.PaymentID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to create payment record")
	}

	s.log.InfoContext(ctx, "checkout created",
		"user_id", req.UserID,
		"subscription_id", sub.ID.String(),
		"payment_id", paymentResp.PaymentID,
		"amount", plan.PriceKopeks,
	)

	return &dto.CheckoutResponse{
		CheckoutURL: paymentResp.CheckoutURL,
		PaymentID:   paymentResp.PaymentID,
	}, nil
}

// GetPayment возвращает платёж по ID.
func (s *paymentService) GetPayment(ctx context.Context, paymentID string) (*dto.PaymentResponse, error) {
	paymentRecord, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, billingerrors.NotFound("payment", paymentID)
	}

	return s.paymentToDTO(paymentRecord), nil
}

// GetPaymentHistory возвращает историю платежей.
func (s *paymentService) GetPaymentHistory(ctx context.Context, req *dto.GetPaymentHistoryRequest) (*dto.PaymentHistoryResponse, error) {
	limit := int(req.PageSize)
	offset := int(req.Page * req.PageSize)

	payments, err := s.paymentRepo.GetByUserID(ctx, req.UserID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get payment history")
	}

	total, err := s.paymentRepo.CountByUserID(ctx, req.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count payments")
	}

	response := &dto.PaymentHistoryResponse{
		Payments: make([]*dto.PaymentResponse, 0, len(payments)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	for _, paymentRecord := range payments {
		response.Payments = append(response.Payments, s.paymentToDTO(paymentRecord))
	}

	return response, nil
}

// ProcessWebhook обрабатывает вебхук от платежной системы.
func (s *paymentService) ProcessWebhook(ctx context.Context, req *dto.WebhookRequest) error {
	// Определяем формат провайдера из строки
	// В DTO provider comes as "YOOKASSA" or "STRIPE"

	var provider string
	switch req.Provider {
	case "YOOKASSA", "yookassa":
		provider = "YOOKASSA"
	case "STRIPE", "stripe":
		provider = "STRIPE"
	default:
		return billingerrors.InvalidArgument("provider", "unknown provider")
	}

	// Получаем payment provider
	var paymentProvider payment.PaymentProvider
	switch provider {
	case "YOOKASSA":
		paymentProvider = s.yookassaProvider
	case "STRIPE":
		paymentProvider = s.stripeProvider
	default:
		return billingerrors.InvalidArgument("provider", "unknown provider")
	}

	if paymentProvider == nil {
		return billingerrors.InvalidArgument("provider", "provider not configured")
	}

	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return billingerrors.InvalidArgument("payload", "invalid webhook payload")
	}

	// Верифицируем сигнатуру
	if err := paymentProvider.VerifyWebhookSignature(ctx, payload, req.Signature); err != nil {
		s.log.ErrorContext(ctx, "webhook signature verification failed",
			"provider", provider,
			"error", err,
		)
		return billingerrors.ErrPermissionDenied
	}

	// Парсим событие
	event, err := paymentProvider.ParseWebhookEvent(ctx, payload)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to parse webhook event",
			"provider", provider,
			"error", err,
		)
		return errors.Wrap(err, "failed to parse webhook")
	}

	s.log.InfoContext(ctx, "processing webhook event",
		"provider", provider,
		"event_type", event.EventType,
		"payment_id", event.PaymentID,
	)

	// Находим платёж по provider payment ID
	paymentRecord, err := s.paymentRepo.GetByProviderPaymentID(ctx, event.PaymentID)
	if err != nil {
		s.log.ErrorContext(ctx, "payment not found",
			"provider_payment_id", event.PaymentID,
			"error", err,
		)
		return billingerrors.NotFound("payment", event.PaymentID)
	}

	// Обрабатываем событие
	switch event.EventType {
	case "payment.succeeded", "payment_intent.succeeded":
		return s.handlePaymentSuccess(ctx, paymentRecord)
	case "payment.canceled", "payment.failed", "payment_intent.payment_failed":
		return s.handlePaymentFailed(ctx, paymentRecord)
	case "refund.succeeded", "charge.refunded":
		return s.handleRefundSuccess(ctx, paymentRecord)
	default:
		s.log.WarnContext(ctx, "unknown webhook event type",
			"event_type", event.EventType,
		)
		return nil
	}
}

// RefundPayment выполняет возврат платежа.
func (s *paymentService) RefundPayment(ctx context.Context, paymentID string) error {
	paymentUUID, err := uuid.Parse(paymentID)
	if err != nil {
		return billingerrors.InvalidArgument("payment_id", "invalid uuid")
	}

	paymentRecord, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return billingerrors.NotFound("payment", paymentID)
	}

	// Определяем провайдера
	var provider payment.PaymentProvider
	switch paymentRecord.Provider {
	case "YOOKASSA":
		provider = s.yookassaProvider
	case "STRIPE":
		provider = s.stripeProvider
	default:
		return billingerrors.InvalidArgument("provider", "unknown provider")
	}

	if provider == nil {
		return billingerrors.ErrInternal
	}

	// Вызываем refund через провайдера
	refundResp, err := provider.RefundPayment(ctx, paymentRecord.ProviderPaymentID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to refund payment",
			"payment_id", paymentID,
			"error", err,
		)
		return billingerrors.PaymentFailed(err.Error())
	}

	// Обновляем статус платежа
	if err := paymentRecord.Refund(); err != nil {
		return errors.Wrap(err, "failed to mark payment as refunded")
	}

	if err := s.paymentRepo.Update(ctx, paymentRecord); err != nil {
		return errors.Wrap(err, "failed to update payment")
	}

	s.log.InfoContext(ctx, "payment refunded",
		"payment_id", paymentID,
		"refund_id", refundResp.RefundID,
		"user_id", paymentUUID,
	)

	return nil
}

// handlePaymentSuccess обрабатывает успешный платёж.
func (s *paymentService) handlePaymentSuccess(ctx context.Context, paymentRecord *model.Payment) error {
	if paymentRecord.Status == model.PaymentStatusSuccess {
		return nil // уже обработан
	}

	// Обновляем статус платежа
	paymentRecord.Status = model.PaymentStatusSuccess
	if err := s.paymentRepo.Update(ctx, paymentRecord); err != nil {
		return errors.Wrap(err, "failed to update payment")
	}

	// Активируем подписку
	if paymentRecord.SubscriptionID != nil {
		sub, err := s.subRepo.GetByID(ctx, paymentRecord.SubscriptionID.String())
		if err != nil {
			return errors.Wrap(err, "failed to get subscription")
		}

		sub.Status = model.StatusActive
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return errors.Wrap(err, "failed to activate subscription")
		}

		s.log.InfoContext(ctx, "subscription activated",
			"subscription_id", sub.ID.String(),
			"user_id", sub.UserID.String(),
			"payment_id", paymentRecord.ID.String(),
		)
	}

	return nil
}

// handlePaymentFailed обрабатывает неудачный платёж.
func (s *paymentService) handlePaymentFailed(ctx context.Context, paymentRecord *model.Payment) error {
	paymentRecord.Status = model.PaymentStatusFailed
	if err := s.paymentRepo.Update(ctx, paymentRecord); err != nil {
		return errors.Wrap(err, "failed to update payment")
	}

	// Отменяем подписку
	if paymentRecord.SubscriptionID != nil {
		sub, err := s.subRepo.GetByID(ctx, paymentRecord.SubscriptionID.String())
		if err != nil {
			return errors.Wrap(err, "failed to get subscription")
		}

		if err := sub.Cancel("payment failed"); err != nil {
			return errors.Wrap(err, "failed to cancel subscription")
		}

		if err := s.subRepo.Update(ctx, sub); err != nil {
			return errors.Wrap(err, "failed to update subscription")
		}

		s.log.InfoContext(ctx, "subscription canceled due to payment failure",
			"subscription_id", sub.ID.String(),
			"user_id", sub.UserID.String(),
		)
	}

	return nil
}

// handleRefundSuccess обрабатывает успешный возврат.
func (s *paymentService) handleRefundSuccess(ctx context.Context, paymentRecord *model.Payment) error {
	paymentRecord.Status = model.PaymentStatusRefunded
	if err := s.paymentRepo.Update(ctx, paymentRecord); err != nil {
		return errors.Wrap(err, "failed to update payment")
	}

	// Отменяем подписку
	if paymentRecord.SubscriptionID != nil {
		sub, err := s.subRepo.GetByID(ctx, paymentRecord.SubscriptionID.String())
		if err != nil {
			return errors.Wrap(err, "failed to get subscription")
		}

		if err := sub.Cancel("payment refunded"); err != nil {
			return errors.Wrap(err, "failed to cancel subscription")
		}

		if err := s.subRepo.Update(ctx, sub); err != nil {
			return errors.Wrap(err, "failed to update subscription")
		}

		s.log.InfoContext(ctx, "subscription canceled due to refund",
			"subscription_id", sub.ID.String(),
			"user_id", sub.UserID.String(),
		)
	}

	return nil
}

// paymentToDTO конвертирует модель в DTO.
func (s *paymentService) paymentToDTO(paymentRecord *model.Payment) *dto.PaymentResponse {
	var subscriptionID *string
	if paymentRecord.SubscriptionID != nil {
		subID := paymentRecord.SubscriptionID.String()
		subscriptionID = &subID
	}

	return &dto.PaymentResponse{
		ID:                paymentRecord.ID.String(),
		UserID:            paymentRecord.UserID.String(),
		SubscriptionID:    subscriptionID,
		Provider:          paymentRecord.Provider,
		ProviderPaymentID: paymentRecord.ProviderPaymentID,
		Status:            paymentRecord.Status,
		AmountKopeks:      paymentRecord.AmountKopeks,
		Currency:          paymentRecord.Currency,
		CreatedAt:         paymentRecord.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         paymentRecord.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
