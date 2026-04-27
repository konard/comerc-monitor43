package handler

import (
	"context"
	"errors"
	"math"
	"time"

	billingv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/billing-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/billing-service/internal/service"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// BillingHandler реализует billing.BillingServiceServer.
type BillingHandler struct {
	billingv1.UnimplementedBillingServiceServer
	planService         service.PlanService
	subscriptionService service.SubscriptionService
	paymentService      service.PaymentService
}

// NewBillingHandler создаёт новый BillingHandler.
func NewBillingHandler(
	planService service.PlanService,
	subscriptionService service.SubscriptionService,
	paymentService service.PaymentService,
) *BillingHandler {
	return &BillingHandler{
		planService:         planService,
		subscriptionService: subscriptionService,
		paymentService:      paymentService,
	}
}

// GetSubscriptionPlans возвращает список тарифных планов.
func (h *BillingHandler) GetSubscriptionPlans(ctx context.Context, req *billingv1.Empty) (*billingv1.SubscriptionPlans, error) {
	plans, err := h.planService.GetAllPlans(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plans: %v", err)
	}

	return &billingv1.SubscriptionPlans{
		Plans: h.plansToProto(plans.Plans),
	}, nil
}

// GetSubscription возвращает подписку пользователя.
func (h *BillingHandler) GetSubscription(ctx context.Context, req *billingv1.GetSubscriptionRequest) (*billingv1.Subscription, error) {
	// Валидация: user_id из запроса должен совпадать с user_id из токена
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Извлекаем userID из JWT токена
	authUserID, err := auth.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Проверяем, что пользователь запрашивает свою подписку
	if authUserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "can only access own subscription")
	}

	sub, err := h.subscriptionService.GetActiveSubscription(ctx, req.UserId)
	if err != nil {
		return nil, h.handleError(err)
	}

	return h.subscriptionToProto(sub), nil
}

// CreateCheckout создаёт платёжную сессию.
func (h *BillingHandler) CreateCheckout(ctx context.Context, req *billingv1.CreateCheckoutRequest) (*billingv1.CheckoutResponse, error) {
	// Извлекаем userID из JWT токена
	authUserID, err := auth.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Проверяем, что пользователь создаёт подписку для себя
	if req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "can only create subscription for yourself")
	}

	if req.PlanId == "" {
		return nil, status.Error(codes.InvalidArgument, "plan_id is required")
	}

	createReq := &dto.CreatePaymentRequest{
		UserID:    req.UserId,
		PlanID:    req.PlanId,
		ReturnURL: req.ReturnUrl,
	}

	checkout, err := h.paymentService.CreateCheckout(ctx, createReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	return &billingv1.CheckoutResponse{
		CheckoutUrl: checkout.CheckoutURL,
		PaymentId:   checkout.PaymentID,
	}, nil
}

// GetPaymentHistory возвращает историю платежей.
func (h *BillingHandler) GetPaymentHistory(ctx context.Context, req *billingv1.GetPaymentHistoryRequest) (*billingv1.PaymentHistoryResponse, error) {
	// Извлекаем userID из JWT токена
	authUserID, err := auth.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Проверяем, что пользователь запрашивает свою историю
	if req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "can only access own payment history")
	}

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	historyReq := &dto.GetPaymentHistoryRequest{
		UserID:   req.UserId,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	history, err := h.paymentService.GetPaymentHistory(ctx, historyReq)
	if err != nil {
		return nil, h.handleError(err)
	}

	payments := make([]*billingv1.Payment, 0, len(history.Payments))
	for _, p := range history.Payments {
		payments = append(payments, h.paymentToProto(p))
	}

	return &billingv1.PaymentHistoryResponse{
		Payments: payments,
		Total:    safeInt64ToInt32(history.Total),
		Page:     history.Page,
		PageSize: history.PageSize,
	}, nil
}

// CancelSubscription отменяет подписку.
func (h *BillingHandler) CancelSubscription(ctx context.Context, req *billingv1.CancelSubscriptionRequest) (*billingv1.Empty, error) {
	// Извлекаем userID из JWT токена
	authUserID, err := auth.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Проверяем, что пользователь отменяет свою подписку
	if req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "can only cancel own subscription")
	}

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	cancelReq := &dto.CancelSubscriptionRequest{
		UserID: req.UserId,
		Reason: req.CancelReason,
	}

	if err := h.subscriptionService.CancelSubscription(ctx, cancelReq); err != nil {
		return nil, h.handleError(err)
	}

	return &billingv1.Empty{}, nil
}

// HandleWebhook обрабатывает вебхук от платежной системы.
func (h *BillingHandler) HandleWebhook(ctx context.Context, req *billingv1.WebhookRequest) (*billingv1.Empty, error) {
	if req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	webhookReq := &dto.WebhookRequest{
		Provider:  req.Provider,
		Payload:   req.Payload,
		Signature: req.Signature,
	}

	if err := h.paymentService.ProcessWebhook(ctx, webhookReq); err != nil {
		return nil, h.handleError(err)
	}

	return &billingv1.Empty{}, nil
}

// plansToProto конвертирует DTO планы в Proto.
func (h *BillingHandler) plansToProto(plans []*dto.PlanResponse) []*billingv1.SubscriptionPlan {
	result := make([]*billingv1.SubscriptionPlan, 0, len(plans))
	for _, p := range plans {
		features := make([]*billingv1.Feature, 0, len(p.Features))
		for _, f := range p.Features {
			features = append(features, &billingv1.Feature{
				Id:          f.ID,
				Name:        f.Name,
				Description: f.Description,
			})
		}

		result = append(result, &billingv1.SubscriptionPlan{
			Id:                      p.ID,
			Tier:                    h.tierToProto(p.ID),
			Name:                    p.Name,
			Description:             p.Description,
			PriceKopeks:             p.PriceKopeks,
			BillingPeriodDays:       p.BillingPeriodDays,
			MaxMonitors:             p.MaxMonitors,
			MinCheckIntervalSeconds: p.MinCheckIntervalSeconds,
			Features:                features,
		})
	}
	return result
}

// subscriptionToProto конвертирует DTO подписку в Proto.
func (h *BillingHandler) subscriptionToProto(sub *dto.SubscriptionResponse) *billingv1.Subscription {
	startedAt, err := time.Parse(time.RFC3339, sub.StartedAt)
	if err != nil {
		startedAt = time.Unix(0, 0).UTC()
	}
	expiresAt, err := time.Parse(time.RFC3339, sub.ExpiresAt)
	if err != nil {
		expiresAt = time.Unix(0, 0).UTC()
	}

	protoSub := &billingv1.Subscription{
		Id:        sub.ID,
		UserId:    sub.UserID,
		PlanId:    sub.PlanID,
		Tier:      h.tierToProto(sub.PlanID),
		Status:    h.statusToProto(sub.Status),
		StartedAt: timestamppb.New(startedAt),
		ExpiresAt: timestamppb.New(expiresAt),
		AutoRenew: sub.AutoRenew,
	}

	if sub.CanceledAt != nil {
		canceledAt, err := time.Parse(time.RFC3339, *sub.CanceledAt)
		if err != nil {
			canceledAt = time.Unix(0, 0).UTC()
		}
		protoSub.CanceledAt = timestamppb.New(canceledAt)
	}

	return protoSub
}

// paymentToProto конвертирует DTO платёж в Proto.
func (h *BillingHandler) paymentToProto(p *dto.PaymentResponse) *billingv1.Payment {
	createdAt, err := time.Parse(time.RFC3339, p.CreatedAt)
	if err != nil {
		createdAt = time.Unix(0, 0).UTC()
	}
	updatedAt, err := time.Parse(time.RFC3339, p.UpdatedAt)
	if err != nil {
		updatedAt = time.Unix(0, 0).UTC()
	}

	protoPayment := &billingv1.Payment{
		Id:                p.ID,
		UserId:            p.UserID,
		Provider:          h.providerToProto(p.Provider),
		ProviderPaymentId: p.ProviderPaymentID,
		Status:            h.paymentStatusToProto(p.Status),
		AmountKopeks:      p.AmountKopeks,
		CreatedAt:         timestamppb.New(createdAt),
		UpdatedAt:         timestamppb.New(updatedAt),
	}

	if p.SubscriptionID != nil {
		protoPayment.SubscriptionId = *p.SubscriptionID
	}

	return protoPayment
}

// tierToProto маппит id плана (TIER_FREE/TIER_STARTER/…) в enum SubscriptionTier.
func (h *BillingHandler) tierToProto(planID string) billingv1.SubscriptionTier {
	switch planID {
	case "TIER_FREE":
		return billingv1.SubscriptionTier_TIER_FREE
	case "TIER_STARTER":
		return billingv1.SubscriptionTier_TIER_STARTER
	case "TIER_PROFESSIONAL":
		return billingv1.SubscriptionTier_TIER_PROFESSIONAL
	case "TIER_BUSINESS":
		return billingv1.SubscriptionTier_TIER_BUSINESS
	default:
		return billingv1.SubscriptionTier_TIER_UNSPECIFIED
	}
}

// statusToProto конвертирует статус подписки в Proto enum.
func (h *BillingHandler) statusToProto(subscriptionStatus string) billingv1.SubscriptionStatus {
	switch subscriptionStatus {
	case "ACTIVE":
		return billingv1.SubscriptionStatus_STATUS_ACTIVE
	case "PENDING":
		return billingv1.SubscriptionStatus_STATUS_PENDING
	case "CANCELED":
		return billingv1.SubscriptionStatus_STATUS_CANCELED
	case "EXPIRED":
		return billingv1.SubscriptionStatus_STATUS_EXPIRED
	default:
		return billingv1.SubscriptionStatus_STATUS_UNSPECIFIED
	}
}

// providerToProto конвертирует провайдера в Proto enum.
func (h *BillingHandler) providerToProto(provider string) billingv1.PaymentProvider {
	switch provider {
	case "YOOKASSA":
		return billingv1.PaymentProvider_PROVIDER_YOOKASSA
	case "STRIPE":
		return billingv1.PaymentProvider_PROVIDER_STRIPE
	default:
		return billingv1.PaymentProvider_PROVIDER_UNSPECIFIED
	}
}

// paymentStatusToProto конвертирует статус платежа в Proto enum.
func (h *BillingHandler) paymentStatusToProto(paymentStatus string) billingv1.PaymentStatus {
	switch paymentStatus {
	case "PENDING":
		return billingv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case "SUCCESS":
		return billingv1.PaymentStatus_PAYMENT_STATUS_SUCCESS
	case "FAILED":
		return billingv1.PaymentStatus_PAYMENT_STATUS_FAILED
	case "REFUNDED":
		return billingv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return billingv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

// handleError конвертирует ошибку в gRPC статус.
func (h *BillingHandler) handleError(err error) error {
	if err == nil {
		return nil
	}

	// Проверяем кастомные ошибки
	if errors.Is(err, billingerrors.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, billingerrors.ErrAlreadyExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, billingerrors.ErrInvalidArgument) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, billingerrors.ErrPermissionDenied) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, billingerrors.ErrUnauthenticated) {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	return status.Error(codes.Internal, err.Error())
}

func safeInt64ToInt32(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}

	return int32(v)
}
