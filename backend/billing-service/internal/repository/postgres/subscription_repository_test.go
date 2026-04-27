package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

func TestSubscriptionRepository_Create(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewSubscriptionRepository(&DB{DB: db})

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)

	err = repo.Create(ctx, sub)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, sub.ID)
}

func TestSubscriptionRepository_GetByID(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewSubscriptionRepository(&DB{DB: db})

	t.Run("existing subscription", func(t *testing.T) {
		subID := uuid.New().String()

		sub, err := repo.GetByID(ctx, subID)
		require.NoError(t, err)
		assert.Equal(t, subID, sub.ID.String())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New().String())
		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_GetActiveByUserID(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewSubscriptionRepository(&DB{DB: db})

	userID := uuid.New().String()

	sub, err := repo.GetActiveByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, sub.UserID.String())
	assert.Equal(t, "ACTIVE", sub.Status)
}

func TestSubscriptionRepository_Update(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewSubscriptionRepository(&DB{DB: db})

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)

	// Create subscription
	err = repo.Create(ctx, sub)
	require.NoError(t, err)

	// Update subscription status
	sub.Status = "ACTIVE"
	sub.UpdatedAt = time.Now()

	err = repo.Update(ctx, sub)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, sub.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "ACTIVE", updated.Status)
}

func TestSubscriptionRepository_Delete(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewSubscriptionRepository(&DB{DB: db})

	userID := uuid.New()
	sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)

	// Create subscription
	err = repo.Create(ctx, sub)
	require.NoError(t, err)

	// Delete subscription
	err = repo.Delete(ctx, sub.ID.String())
	require.NoError(t, err)

	// Verify deletion (status should be CANCELED)
	deleted, err := repo.GetByID(ctx, sub.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "CANCELED", deleted.Status)
	assert.NotNil(t, deleted.CanceledAt)
}
