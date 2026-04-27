package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// Mock subscription repository
type mockSubscriptionRepository struct {
	subscriptions map[string]*model.Subscription
}

func newMockSubscriptionRepository() *mockSubscriptionRepository {
	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"

	return &mockSubscriptionRepository{
		subscriptions: map[string]*model.Subscription{
			userID.String(): sub,
		},
	}
}

func (m *mockSubscriptionRepository) Create(_ context.Context, sub *model.Subscription) error {
	m.subscriptions[sub.UserID.String()] = sub
	return nil
}

func (m *mockSubscriptionRepository) GetByID(_ context.Context, id string) (*model.Subscription, error) {
	for _, sub := range m.subscriptions {
		if sub.ID.String() == id {
			return sub, nil
		}
	}
	return nil, billingerrors.NotFound("subscription", id)
}

func (m *mockSubscriptionRepository) GetActiveByUserID(_ context.Context, userID string) (*model.Subscription, error) {
	sub, ok := m.subscriptions[userID]
	if !ok {
		return nil, billingerrors.NotFound("subscription", userID)
	}
	return sub, nil
}

func (m *mockSubscriptionRepository) GetByUserID(_ context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	sub, ok := m.subscriptions[userID]
	if !ok {
		return []*model.Subscription{}, nil
	}
	return []*model.Subscription{sub}, nil
}

func (m *mockSubscriptionRepository) Update(_ context.Context, sub *model.Subscription) error {
	m.subscriptions[sub.UserID.String()] = sub
	return nil
}

func (m *mockSubscriptionRepository) Delete(_ context.Context, id string) error {
	return nil
}

func TestSubscriptionService_GetActiveSubscription(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	service := NewSubscriptionService(subRepo, planRepo)

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
	sub.Status = "ACTIVE"
	subRepo.subscriptions[userID.String()] = sub

	response, err := service.GetActiveSubscription(ctx, userID.String())
	require.NoError(t, err)
	assert.Equal(t, userID.String(), response.UserID)
	assert.Equal(t, "ACTIVE", response.Status)
}

func TestSubscriptionService_CreateSubscription(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	service := NewSubscriptionService(subRepo, planRepo)

	t.Run("create new subscription", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.CreateSubscriptionRequest{
			UserID:   userID.String(),
			PlanID:   "TIER_STARTER",
			Duration: 30,
		}

		resp, err := service.CreateSubscription(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, "TIER_STARTER", resp.PlanID)
		assert.Equal(t, "PENDING", resp.Status)
	})

	t.Run("plan not found", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.CreateSubscriptionRequest{
			UserID:   userID.String(),
			PlanID:   "TIER_NONEXISTENT",
			Duration: 30,
		}

		_, err := service.CreateSubscription(ctx, req)
		assert.Error(t, err)
	})

	t.Run("plan not active", func(t *testing.T) {
		// Deactivate plan
		plan := planRepo.plans["TIER_STARTER"]
		plan.IsActive = false

		userID := uuid.New()
		req := &dto.CreateSubscriptionRequest{
			UserID:   userID.String(),
			PlanID:   "TIER_STARTER",
			Duration: 30,
		}

		_, err := service.CreateSubscription(ctx, req)
		assert.Error(t, err)

		// Restore
		plan.IsActive = true
	})
}

func TestSubscriptionService_CancelSubscription(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	service := NewSubscriptionService(subRepo, planRepo)

	t.Run("cancel active subscription", func(t *testing.T) {
		userID := uuid.New()
		sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
		sub.Status = "ACTIVE"
		subRepo.subscriptions[userID.String()] = sub

		req := &dto.CancelSubscriptionRequest{
			UserID: userID.String(),
			Reason: "test cancellation",
		}

		err := service.CancelSubscription(ctx, req)
		require.NoError(t, err)

		updated := subRepo.subscriptions[userID.String()]
		assert.Equal(t, "CANCELED", updated.Status)
		assert.NotNil(t, updated.CanceledAt)
	})

	t.Run("subscription not found", func(t *testing.T) {
		userID := uuid.New()
		req := &dto.CancelSubscriptionRequest{
			UserID: userID.String(),
			Reason: "test",
		}

		err := service.CancelSubscription(ctx, req)
		assert.Error(t, err)
	})
}

func TestSubscriptionService_RenewSubscription(t *testing.T) {
	ctx := context.Background()
	subRepo := newMockSubscriptionRepository()
	planRepo := newMockPlanRepository()
	service := NewSubscriptionService(subRepo, planRepo)

	t.Run("renew subscription", func(t *testing.T) {
		userID := uuid.New()
		sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
		sub.Status = "ACTIVE"
		subRepo.subscriptions[userID.String()] = sub

		oldExpiry := sub.ExpiresAt

		resp, err := service.RenewSubscription(ctx, sub.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", resp.Status)

		updated := subRepo.subscriptions[userID.String()]
		assert.True(t, updated.ExpiresAt.After(oldExpiry))
	})

	t.Run("subscription not found", func(t *testing.T) {
		_, err := service.RenewSubscription(ctx, uuid.New().String())
		assert.Error(t, err)
	})
}
