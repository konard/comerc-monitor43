package postgres

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRepository_GetByID(t *testing.T) {
	// Skip if no test database
	t.Skip("integration test - requires test database")

	ctx := context.Background()

	// Setup test database
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPlanRepository(&DB{DB: db})

	// Test getting existing plan
	plan, err := repo.GetByID(ctx, "TIER_FREE")
	require.NoError(t, err)
	assert.Equal(t, "TIER_FREE", plan.ID)
	assert.Equal(t, "Free", plan.Name)
	assert.True(t, plan.IsActive)
}

func TestPlanRepository_GetAll(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPlanRepository(&DB{DB: db})

	// Test getting all active plans
	plans, err := repo.GetAll(ctx, true)
	require.NoError(t, err)
	assert.Greater(t, len(plans), 0)

	// All plans should be active
	for _, plan := range plans {
		assert.True(t, plan.IsActive, "plan %s should be active", plan.ID)
	}
}

func TestPlanRepository_GetByTier(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPlanRepository(&DB{DB: db})

	t.Run("get starter plan", func(t *testing.T) {
		plan, err := repo.GetByTier(ctx, "TIER_STARTER")
		require.NoError(t, err)
		assert.Equal(t, "TIER_STARTER", plan.ID)
		assert.Equal(t, "Starter", plan.Name)
		assert.Equal(t, int64(29900), plan.PriceKopeks)
	})

	t.Run("plan not found", func(t *testing.T) {
		_, err := repo.GetByTier(ctx, "TIER_NONEXISTENT")
		assert.Error(t, err)
	})
}
