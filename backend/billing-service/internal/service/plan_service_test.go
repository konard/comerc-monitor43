package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// Mock plan repository for testing
type mockPlanRepository struct {
	plans map[string]*model.Plan
}

func newMockPlanRepository() *mockPlanRepository {
	return &mockPlanRepository{
		plans: map[string]*model.Plan{
			"TIER_FREE": {
				ID:                      "TIER_FREE",
				Name:                    "Free",
				PriceKopeks:             0,
				BillingPeriodDays:       30,
				MaxMonitors:             5,
				MinCheckIntervalSeconds: 300,
				MaxAlertsPerDay:         10,
				IsActive:                true,
				Features:                []model.Feature{},
			},
			"TIER_STARTER": {
				ID:                      "TIER_STARTER",
				Name:                    "Starter",
				PriceKopeks:             29900,
				BillingPeriodDays:       30,
				MaxMonitors:             25,
				MinCheckIntervalSeconds: 60,
				MaxAlertsPerDay:         100,
				IsActive:                true,
				Features:                []model.Feature{},
			},
		},
	}
}

func (m *mockPlanRepository) GetByID(_ context.Context, id string) (*model.Plan, error) {
	plan, ok := m.plans[id]
	if !ok {
		return nil, billingerrors.NotFound("plan", id)
	}
	return plan, nil
}

func (m *mockPlanRepository) GetAll(_ context.Context, activeOnly bool) ([]*model.Plan, error) {
	var plans []*model.Plan
	for _, plan := range m.plans {
		if !activeOnly || plan.IsActive {
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func (m *mockPlanRepository) GetByTier(ctx context.Context, tier string) (*model.Plan, error) {
	return m.GetByID(ctx, tier)
}

func TestPlanService_GetAllPlans(t *testing.T) {
	ctx := context.Background()
	repo := newMockPlanRepository()
	service := NewPlanService(repo)

	plans, err := service.GetAllPlans(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, plans.Plans)
	assert.GreaterOrEqual(t, len(plans.Plans), 2)
}

func TestPlanService_GetPlan(t *testing.T) {
	ctx := context.Background()
	repo := newMockPlanRepository()
	service := NewPlanService(repo)

	t.Run("existing plan", func(t *testing.T) {
		plan, err := service.GetPlan(ctx, "TIER_FREE")
		require.NoError(t, err)
		assert.Equal(t, "TIER_FREE", plan.ID)
		assert.Equal(t, "Free", plan.Name)
		assert.Equal(t, int64(0), plan.PriceKopeks)
	})

	t.Run("plan not found", func(t *testing.T) {
		_, err := service.GetPlan(ctx, "TIER_NONEXISTENT")
		assert.Error(t, err)
	})
}

func TestPlanService_ValidatePlanLimits(t *testing.T) {
	ctx := context.Background()
	repo := newMockPlanRepository()
	service := NewPlanService(repo)

	t.Run("valid limits", func(t *testing.T) {
		err := service.ValidatePlanLimits(ctx, "TIER_STARTER", 10, 120)
		assert.NoError(t, err)
	})

	t.Run("exceeds monitors limit", func(t *testing.T) {
		err := service.ValidatePlanLimits(ctx, "TIER_FREE", 10, 300)
		assert.Error(t, err)
	})

	t.Run("check interval too short", func(t *testing.T) {
		err := service.ValidatePlanLimits(ctx, "TIER_STARTER", 5, 30)
		assert.Error(t, err)
	})

	t.Run("plan not found", func(t *testing.T) {
		err := service.ValidatePlanLimits(ctx, "TIER_NONEXISTENT", 5, 300)
		assert.Error(t, err)
	})
}
