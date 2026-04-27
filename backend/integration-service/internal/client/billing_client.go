package client

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	apiv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// BillingClient gRPC клиент для Billing Service.
type BillingClient struct {
	client apiv1.BillingServiceClient
}

// NewBillingClient создаёт новый BillingClient.
func NewBillingClient(conn *grpc.ClientConn) *BillingClient {
	return &BillingClient{
		client: apiv1.NewBillingServiceClient(conn),
	}
}

func withForwardedAuthorization(ctx context.Context) context.Context {
	if md, ok := metadata.FromOutgoingContext(ctx); ok && len(md.Get("authorization")) > 0 {
		return ctx
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", values[0])
}

// GetSubscription получает подписку пользователя.
func (c *BillingClient) GetSubscription(ctx context.Context, userID uuid.UUID) (*apiv1.Subscription, error) {
	req := &apiv1.GetSubscriptionRequest{
		UserId: userID.String(),
	}

	ctx = withForwardedAuthorization(ctx)
	resp, err := c.client.GetSubscription(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetSubscriptionPlans получает все доступные планы подписок.
func (c *BillingClient) GetSubscriptionPlans(ctx context.Context) (*apiv1.SubscriptionPlans, error) {
	ctx = withForwardedAuthorization(ctx)
	return c.client.GetSubscriptionPlans(ctx, &apiv1.Empty{})
}

// CheckMonitorLimit проверяет, может ли пользователь создать ещё монитор.
// Возвращает true, если лимит не превышен, и false если превышен.
func (c *BillingClient) CheckMonitorLimit(ctx context.Context, userID uuid.UUID, currentMonitorCount int) (bool, error) {
	// Получаем подписку пользователя
	subscription, err := c.GetSubscription(ctx, userID)
	if err != nil {
		return false, err
	}

	// Если подписка не активна, запрещаем создание
	if subscription.Status != apiv1.SubscriptionStatus_STATUS_ACTIVE {
		return false, nil
	}

	// Получаем планы, чтобы узнать лимиты
	plans, err := c.GetSubscriptionPlans(ctx)
	if err != nil {
		return false, err
	}

	// Ищем план пользователя
	var maxMonitors int32
	for _, plan := range plans.Plans {
		if plan.Id == subscription.PlanId {
			maxMonitors = plan.MaxMonitors
			break
		}
	}

	// Если лимит не найден, используем дефолтный (например, 10 для free tier)
	if maxMonitors == 0 {
		maxMonitors = 10
	}

	// Проверяем лимит
	return currentMonitorCount < int(maxMonitors), nil
}

// CheckWebhookLimit проверяет, может ли пользователь создать ещё webhook.
// Примечание: Лимиты основаны на subscription tier.
func (c *BillingClient) CheckWebhookLimit(ctx context.Context, userID uuid.UUID, currentWebhookCount int) (bool, error) {
	// Получаем подписку пользователя
	subscription, err := c.GetSubscription(ctx, userID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get user subscription")
	}

	// В зависимости от tier устанавливаем лимиты
	var maxWebhooks int
	switch subscription.Tier {
	case apiv1.SubscriptionTier_TIER_FREE:
		maxWebhooks = 5
	case apiv1.SubscriptionTier_TIER_STARTER:
		maxWebhooks = 20
	case apiv1.SubscriptionTier_TIER_PROFESSIONAL:
		maxWebhooks = 100
	case apiv1.SubscriptionTier_TIER_BUSINESS:
		maxWebhooks = 1000
	default:
		maxWebhooks = 5 // консервативный дефолт
	}

	return currentWebhookCount < maxWebhooks, nil
}

// CheckAPIKeyLimit проверяет, может ли пользователь создать ещё API ключ.
// Примечание: Лимиты основаны на subscription tier.
func (c *BillingClient) CheckAPIKeyLimit(ctx context.Context, userID uuid.UUID, currentAPIKeyCount int) (bool, error) {
	// Получаем подписку пользователя
	subscription, err := c.GetSubscription(ctx, userID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get user subscription")
	}

	// В зависимости от tier устанавливаем лимиты
	var maxAPIKeys int
	switch subscription.Tier {
	case apiv1.SubscriptionTier_TIER_FREE:
		maxAPIKeys = 2
	case apiv1.SubscriptionTier_TIER_STARTER:
		maxAPIKeys = 10
	case apiv1.SubscriptionTier_TIER_PROFESSIONAL:
		maxAPIKeys = 100
	case apiv1.SubscriptionTier_TIER_BUSINESS:
		maxAPIKeys = 1000
	default:
		maxAPIKeys = 2 // консервативный дефолт
	}

	return currentAPIKeyCount < maxAPIKeys, nil
}
