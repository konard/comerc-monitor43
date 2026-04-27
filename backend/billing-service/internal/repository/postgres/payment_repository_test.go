package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

func TestPaymentRepository_Create(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPaymentRepository(&DB{DB: db})

	userID := uuid.New()
	subscriptionID := uuid.New()

	payment := model.NewPayment(
		userID,
		subscriptionID,
		model.ProviderYookassa,
		"yp_123456",
		29900,
		"RUB",
	)

	err = repo.Create(ctx, payment)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
}

func TestPaymentRepository_GetByID(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPaymentRepository(&DB{DB: db})

	paymentID := uuid.New().String()

	payment, err := repo.GetByID(ctx, paymentID)
	require.NoError(t, err)
	assert.Equal(t, paymentID, payment.ID.String())
}

func TestPaymentRepository_GetByUserID(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPaymentRepository(&DB{DB: db})

	userID := uuid.New().String()

	payments, err := repo.GetByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)
	assert.Greater(t, len(payments), 0)
}

func TestPaymentRepository_GetByProviderPaymentID(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPaymentRepository(&DB{DB: db})

	providerPaymentID := "yp_test_123456"

	payment, err := repo.GetByProviderPaymentID(ctx, providerPaymentID)
	require.NoError(t, err)
	assert.Equal(t, providerPaymentID, payment.ProviderPaymentID)
}

func TestPaymentRepository_Update(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPaymentRepository(&DB{DB: db})

	userID := uuid.New()
	subscriptionID := uuid.New()

	payment := model.NewPayment(
		userID,
		subscriptionID,
		model.ProviderYookassa,
		"yp_update_test",
		29900,
		"RUB",
	)

	// Create payment
	err = repo.Create(ctx, payment)
	require.NoError(t, err)

	// Update payment status
	err = payment.MarkAsSuccess()
	require.NoError(t, err)

	err = repo.Update(ctx, payment)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, payment.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", updated.Status)
}

func TestPaymentRepository_CountByUserID(t *testing.T) {
	t.Skip("integration test - requires test database")

	ctx := context.Background()
	db, err := sqlx.Connect("postgres", "test-dsn")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	repo := NewPaymentRepository(&DB{DB: db})

	userID := uuid.New().String()

	count, err := repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}
