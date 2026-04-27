package service

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// NewSubscriptionService создаёт новый SubscriptionService.
func NewSubscriptionService(
	subRepo interfaces.SubscriptionRepository,
	planRepo interfaces.PlanRepository,
) SubscriptionService {
	return &subscriptionService{
		subRepo:  subRepo,
		planRepo: planRepo,
	}
}

type subscriptionService struct {
	subRepo   interfaces.SubscriptionRepository
	planRepo  interfaces.PlanRepository
	publisher EventPublisher // опционально
}

// SetPublisher устанавливает publisher для событий.
func (s *subscriptionService) SetPublisher(publisher EventPublisher) {
	s.publisher = publisher
}

// GetActiveSubscription возвращает активную подписку пользователя.
func (s *subscriptionService) GetActiveSubscription(ctx context.Context, userID string) (*dto.SubscriptionResponse, error) {
	sub, err := s.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active subscription")
	}

	return s.subscriptionToDTO(sub), nil
}

// CreateSubscription создаёт новую подписку.
func (s *subscriptionService) CreateSubscription(ctx context.Context, req *dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error) {
	// Проверяем существование плана
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, billingerrors.NotFound("plan", req.PlanID)
	}

	if !plan.IsActive {
		return nil, billingerrors.InvalidArgument("plan_id", "plan is not active")
	}

	// Проверяем, нет ли уже активной подписки
	activeSub, err := s.subRepo.GetActiveByUserID(ctx, req.UserID)
	if err == nil && activeSub != nil {
		return nil, billingerrors.AlreadyExists("subscription", req.UserID)
	}

	// Создаём подписку
	duration := time.Duration(req.Duration) * 24 * time.Hour
	sub := model.NewSubscription(uuid.MustParse(req.UserID), req.PlanID, duration)

	if err := s.subRepo.Create(ctx, sub); err != nil {
		return nil, errors.Wrap(err, "failed to create subscription")
	}

	// Публикуем событие создания
	if s.publisher != nil {
		if err := s.publishSubscriptionCreated(ctx, sub, plan); err != nil {
			slog.Default().WarnContext(ctx, "failed to publish subscription created event", "error", err)
		}
	}

	return s.subscriptionToDTO(sub), nil
}

// CancelSubscription отменяет подписку.
func (s *subscriptionService) CancelSubscription(ctx context.Context, req *dto.CancelSubscriptionRequest) error {
	sub, err := s.subRepo.GetActiveByUserID(ctx, req.UserID)
	if err != nil {
		return billingerrors.NotFound("subscription", req.UserID)
	}

	if err := sub.Cancel(req.Reason); err != nil {
		return errors.Wrap(err, "failed to cancel subscription")
	}

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return errors.Wrap(err, "failed to update subscription")
	}

	// Публикуем событие отмены
	if s.publisher != nil {
		if err := s.publishSubscriptionCanceled(ctx, sub, req.Reason); err != nil {
			slog.Default().WarnContext(ctx, "failed to publish subscription canceled event", "error", err)
		}
	}

	return nil
}

// RenewSubscription продлевает подписку.
func (s *subscriptionService) RenewSubscription(ctx context.Context, subscriptionID string) (*dto.SubscriptionResponse, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, billingerrors.NotFound("subscription", subscriptionID)
	}

	// Получаем план для определения периода
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, billingerrors.NotFound("plan", sub.PlanID)
	}

	duration := time.Duration(plan.BillingPeriodDays) * 24 * time.Hour

	if err := sub.Renew(duration); err != nil {
		return nil, errors.Wrap(err, "failed to renew subscription")
	}

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, errors.Wrap(err, "failed to update subscription")
	}

	return s.subscriptionToDTO(sub), nil
}

// GetUserSubscriptionHistory возвращает историю подписок.
func (s *subscriptionService) GetUserSubscriptionHistory(ctx context.Context, userID string, page, pageSize int32) (*dto.SubscriptionHistoryResponse, error) {
	limit := int(pageSize)
	offset := int(page * pageSize)

	subs, err := s.subRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subscription history")
	}

	response := &dto.SubscriptionHistoryResponse{
		Subscriptions: make([]*dto.SubscriptionResponse, 0, len(subs)),
		Page:          page,
		PageSize:      pageSize,
	}

	for _, sub := range subs {
		response.Subscriptions = append(response.Subscriptions, s.subscriptionToDTO(sub))
	}

	if len(subs) > math.MaxInt32 {
		response.Total = math.MaxInt32
	} else {
		response.Total = safeLenToInt32(len(subs))
	}

	return response, nil
}

// publishSubscriptionCreated публикует событие создания подписки.
func (s *subscriptionService) publishSubscriptionCreated(ctx context.Context, sub *model.Subscription, plan *model.Plan) error {
	event := map[string]any{
		"subscription_id":            sub.ID.String(),
		"user_id":                    sub.UserID.String(),
		"plan_id":                    sub.PlanID,
		"max_monitors":               plan.MaxMonitors,
		"min_check_interval_seconds": plan.MinCheckIntervalSeconds,
		"max_alerts_per_day":         plan.MaxAlertsPerDay,
	}

	if s.publisher != nil {
		return s.publisher.PublishSubscriptionCreated(ctx, event)
	}

	return nil
}

// publishSubscriptionCanceled публикует событие отмены подписки.
func (s *subscriptionService) publishSubscriptionCanceled(ctx context.Context, sub *model.Subscription, reason string) error {
	event := map[string]any{
		"subscription_id": sub.ID.String(),
		"user_id":         sub.UserID.String(),
		"plan_id":         sub.PlanID,
		"reason":          reason,
	}

	if s.publisher != nil {
		return s.publisher.PublishSubscriptionCanceled(ctx, event)
	}

	return nil
}

// subscriptionToDTO конвертирует модель в DTO.
func (s *subscriptionService) subscriptionToDTO(sub *model.Subscription) *dto.SubscriptionResponse {
	var canceledAt *string
	if sub.CanceledAt != nil {
		canceledAtStr := sub.CanceledAt.Format(time.RFC3339)
		canceledAt = &canceledAtStr
	}

	var providerPaymentID *string
	if sub.ProviderPaymentID != nil {
		providerPaymentID = sub.ProviderPaymentID
	}

	return &dto.SubscriptionResponse{
		ID:                sub.ID.String(),
		UserID:            sub.UserID.String(),
		PlanID:            sub.PlanID,
		Status:            sub.Status,
		StartedAt:         sub.StartedAt.Format(time.RFC3339),
		ExpiresAt:         sub.ExpiresAt.Format(time.RFC3339),
		CanceledAt:        canceledAt,
		AutoRenew:         sub.AutoRenew,
		ProviderPaymentID: providerPaymentID,
		CreatedAt:         sub.CreatedAt.Format(time.RFC3339),
	}
}

func safeLenToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}

	//nolint:gosec // Значение ограничено проверкой выше.
	return int32(v)
}
