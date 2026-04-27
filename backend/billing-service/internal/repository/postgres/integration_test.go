package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

func TestPlanRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPlanRepository(db)

	t.Run("get all plans", func(t *testing.T) {
		plans, err := repo.GetAll(ctx, true)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(plans), 2, "should have at least 2 plans")

		// Check Free plan exists
		freePlan := findPlan(plans, "TIER_FREE")
		require.NotNil(t, freePlan)
		assert.Equal(t, "Free", freePlan.Name)
		assert.Equal(t, int64(0), freePlan.PriceKopeks)
		assert.True(t, freePlan.IsActive)
	})

	t.Run("get plan by ID", func(t *testing.T) {
		plan, err := repo.GetByID(ctx, "TIER_FREE")
		require.NoError(t, err)
		assert.Equal(t, "TIER_FREE", plan.ID)
		assert.Equal(t, "Free", plan.Name)
		assert.Len(t, plan.Features, 1)
	})

	t.Run("get plan by tier", func(t *testing.T) {
		plan, err := repo.GetByTier(ctx, "TIER_STARTER")
		require.NoError(t, err)
		assert.Equal(t, "TIER_STARTER", plan.ID)
		assert.Equal(t, "Starter", plan.Name)
		assert.Equal(t, int64(29900), plan.PriceKopeks)
	})

	t.Run("plan not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "TIER_NONEXISTENT")
		assert.Error(t, err)
	})

	t.Run("get all active only", func(t *testing.T) {
		plans, err := repo.GetAll(ctx, true)
		require.NoError(t, err)

		// All plans should be active
		for _, plan := range plans {
			assert.True(t, plan.IsActive, "plan %s should be active", plan.ID)
		}
	})
}

func TestSubscriptionRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSubscriptionRepository(db)

	t.Run("create subscription", func(t *testing.T) {
		userID := uuid.New()
		sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*24*time.Hour)

		err := repo.Create(ctx, sub)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, sub.ID)
		assert.Equal(t, userID, sub.UserID)
		assert.Equal(t, "PENDING", sub.Status)
	})

	t.Run("get subscription by ID", func(t *testing.T) {
		userID := uuid.New()
		sub := CreateTestSubscription(t, db, userID, "TIER_FREE")

		fetched, err := repo.GetByID(ctx, sub.ID.String())
		require.NoError(t, err)
		assert.Equal(t, sub.ID, fetched.ID)
		assert.Equal(t, userID, fetched.UserID)
		assert.Equal(t, "PENDING", fetched.Status)
	})

	t.Run("get active subscription by user ID", func(t *testing.T) {
		userID := uuid.New()

		// Create active subscription
		sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)
		sub.Status = "ACTIVE"
		err := repo.Create(ctx, sub)
		require.NoError(t, err)

		// Fetch active subscription
		fetched, err := repo.GetActiveByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", fetched.Status)
		assert.Equal(t, userID, fetched.UserID)
	})

	t.Run("update subscription", func(t *testing.T) {
		userID := uuid.New()
		sub := CreateTestSubscription(t, db, userID, "TIER_FREE")

		// Update status
		sub.Status = "ACTIVE"
		err := repo.Update(ctx, sub)
		require.NoError(t, err)

		// Verify update
		fetched, err := repo.GetByID(ctx, sub.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", fetched.Status)
	})

	t.Run("delete subscription (soft delete)", func(t *testing.T) {
		userID := uuid.New()
		sub := CreateTestSubscription(t, db, userID, "TIER_STARTER")

		err := repo.Delete(ctx, sub.ID.String())
		require.NoError(t, err)

		// Verify soft delete (status = CANCELED)
		fetched, err := repo.GetByID(ctx, sub.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "CANCELED", fetched.Status)
		assert.NotNil(t, fetched.CanceledAt)
	})

	t.Run("get subscriptions by user ID with pagination", func(t *testing.T) {
		userID := uuid.New()

		// Create multiple subscriptions
		for i := 0; i < 5; i++ {
			sub := model.NewSubscription(userID, "TIER_FREE", 30*24*time.Hour)
			sub.Status = "CANCELED"
			require.NoError(t, repo.Create(ctx, sub))
		}

		// Get first page
		subs, err := repo.GetByUserID(ctx, userID.String(), 3, 0)
		require.NoError(t, err)
		assert.Len(t, subs, 3)

		// Get second page
		subs2, err := repo.GetByUserID(ctx, userID.String(), 3, 3)
		require.NoError(t, err)
		assert.Len(t, subs2, 2)
	})
}

func TestPaymentRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewPaymentRepository(db)

	t.Run("create payment", func(t *testing.T) {
		userID := uuid.New()
		sub := CreateTestSubscription(t, db, userID, "TIER_STARTER")

		payment := model.NewPayment(userID, sub.ID, model.ProviderYookassa, "yp_test_create", 29900, "RUB")

		err := repo.Create(ctx, payment)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, payment.ID)
		assert.Equal(t, userID, payment.UserID)
		assert.Equal(t, "PENDING", payment.Status)
	})

	t.Run("get payment by ID", func(t *testing.T) {
		userID := uuid.New()
		subID := uuid.New()
		payment := CreateTestPayment(t, db, userID, subID, "yp_test_get")

		fetched, err := repo.GetByID(ctx, payment.ID.String())
		require.NoError(t, err)
		assert.Equal(t, payment.ID, fetched.ID)
		assert.Equal(t, "yp_test_get", fetched.ProviderPaymentID)
		assert.Equal(t, "PENDING", fetched.Status)
	})

	t.Run("get payment by provider payment ID", func(t *testing.T) {
		userID := uuid.New()
		subID := uuid.New()
		providerPaymentID := "yp_test_provider_id"
		_ = CreateTestPayment(t, db, userID, subID, providerPaymentID)

		fetched, err := repo.GetByProviderPaymentID(ctx, providerPaymentID)
		require.NoError(t, err)
		assert.Equal(t, providerPaymentID, fetched.ProviderPaymentID)
		assert.Equal(t, model.ProviderYookassa, fetched.Provider)
	})

	t.Run("update payment", func(t *testing.T) {
		userID := uuid.New()
		subID := uuid.New()
		payment := CreateTestPayment(t, db, userID, subID, "yp_test_update")

		// Mark as success
		err := payment.MarkAsSuccess()
		require.NoError(t, err)

		err = repo.Update(ctx, payment)
		require.NoError(t, err)

		// Verify update
		fetched, err := repo.GetByID(ctx, payment.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "SUCCESS", fetched.Status)
	})

	t.Run("count payments by user", func(t *testing.T) {
		userID := uuid.New()
		subID := uuid.New()

		// Create 3 payments
		for i := 0; i < 3; i++ {
			_ = CreateTestPayment(t, db, userID, subID, fmt.Sprintf("yp_test_count_%d", i))
		}

		count, err := repo.CountByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
	})

	t.Run("get payments by user with pagination", func(t *testing.T) {
		userID := uuid.New()
		subID := uuid.New()

		// Create 5 payments
		for i := 0; i < 5; i++ {
			_ = CreateTestPayment(t, db, userID, subID, fmt.Sprintf("yp_test_pag_%d", i))
		}

		// Get first page
		payments, err := repo.GetByUserID(ctx, userID.String(), 3, 0)
		require.NoError(t, err)
		assert.Len(t, payments, 3)

		// Get second page
		payments2, err := repo.GetByUserID(ctx, userID.String(), 3, 3)
		require.NoError(t, err)
		assert.Len(t, payments2, 2)
	})
}

// Helper function
func findPlan(plans []*model.Plan, id string) *model.Plan {
	for _, plan := range plans {
		if plan.ID == id {
			return plan
		}
	}
	return nil
}
