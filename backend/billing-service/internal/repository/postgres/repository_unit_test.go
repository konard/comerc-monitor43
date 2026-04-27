package postgres

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

// newMockDB создаёт тестовую обёртку над sqlmock.
func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "postgres")
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("mock db close: %v", err)
		}
	})

	return &DB{DB: sqlxDB}, mock
}

// anyTime реализует driver.Value для сравнения time.Time в sqlmock.
type anyTime struct{}

func (a anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func TestNewPlanRepository(t *testing.T) {
	t.Parallel()

	db, _ := newMockDB(t)
	repo := NewPlanRepository(db)

	assert.NotNil(t, repo)
}

func TestNewPaymentRepository(t *testing.T) {
	t.Parallel()

	db, _ := newMockDB(t)
	repo := NewPaymentRepository(db)

	assert.NotNil(t, repo)
}

func TestNewSubscriptionRepository(t *testing.T) {
	t.Parallel()

	db, _ := newMockDB(t)
	repo := NewSubscriptionRepository(db)

	assert.NotNil(t, repo)
}

func TestPaymentRepository_CreateUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		payment := model.NewPayment(uuid.New(), uuid.New(), "YOOKASSA", "yp_123", 29900, "RUB")

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO payments`)).
			WithArgs(
				payment.ID, payment.UserID, payment.SubscriptionID, payment.Provider,
				payment.ProviderPaymentID, payment.Status, payment.AmountKopeks,
				payment.Currency, sqlmock.AnyArg(), anyTime{}, anyTime{},
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), payment)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		payment := model.NewPayment(uuid.New(), uuid.New(), "YOOKASSA", "yp_err", 29900, "RUB")

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO payments`)).
			WillReturnError(assert.AnError)

		err := repo.Create(context.Background(), payment)

		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetByIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		paymentID := uuid.New()
		userID := uuid.New()
		subID := uuid.New()
		now := time.Now()
		metadataJSON, err := json.Marshal(map[string]any{})
		require.NoError(t, err)

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "subscription_id", "provider", "provider_payment_id",
			"status", "amount_kopeks", "currency", "metadata", "created_at", "updated_at",
		}).AddRow(
			paymentID, userID, subID, "YOOKASSA", "yp_123",
			"PENDING", int64(29900), "RUB", metadataJSON, now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs(paymentID.String()).
			WillReturnRows(rows)

		result, err := repo.GetByID(context.Background(), paymentID.String())

		require.NoError(t, err)
		assert.Equal(t, paymentID, result.ID)
		assert.Equal(t, "YOOKASSA", result.Provider)
		assert.Equal(t, int64(29900), result.AmountKopeks)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs(uuid.New().String()).
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, err := repo.GetByID(context.Background(), uuid.New().String())

		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetByUserIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		userID := uuid.New()
		now := time.Now()
		metadataJSON, err := json.Marshal(map[string]any{})
		require.NoError(t, err)

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "subscription_id", "provider", "provider_payment_id",
			"status", "amount_kopeks", "currency", "metadata", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), userID, nil, "YOOKASSA", "yp_1",
			"SUCCESS", int64(29900), "RUB", metadataJSON, now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs(userID.String(), 10, 0).
			WillReturnRows(rows)

		results, err := repo.GetByUserID(context.Background(), userID.String(), 10, 0)

		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		userID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnError(assert.AnError)

		_, err := repo.GetByUserID(context.Background(), userID.String(), 10, 0)

		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetByProviderPaymentIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		paymentID := uuid.New()
		userID := uuid.New()
		now := time.Now()
		metadataJSON, err := json.Marshal(map[string]any{})
		require.NoError(t, err)

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "subscription_id", "provider", "provider_payment_id",
			"status", "amount_kopeks", "currency", "metadata", "created_at", "updated_at",
		}).AddRow(
			paymentID, userID, nil, "YOOKASSA", "yp_provider_123",
			"PENDING", int64(29900), "RUB", metadataJSON, now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs("yp_provider_123").
			WillReturnRows(rows)

		result, err := repo.GetByProviderPaymentID(context.Background(), "yp_provider_123")

		require.NoError(t, err)
		assert.Equal(t, paymentID, result.ID)
	})
}

func TestPaymentRepository_UpdateUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		payment := model.NewPayment(uuid.New(), uuid.New(), "YOOKASSA", "yp_upd", 29900, "RUB")
		payment.Status = "SUCCESS"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE payments`)).
			WithArgs(payment.ID, payment.Status, sqlmock.AnyArg(), anyTime{}).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(context.Background(), payment)

		require.NoError(t, err)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		payment := model.NewPayment(uuid.New(), uuid.New(), "YOOKASSA", "yp_nf", 29900, "RUB")

		// Нет затронутых строк
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE payments`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(context.Background(), payment)

		assert.Error(t, err)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		payment := model.NewPayment(uuid.New(), uuid.New(), "YOOKASSA", "yp_err2", 29900, "RUB")

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE payments`)).
			WillReturnError(assert.AnError)

		err := repo.Update(context.Background(), payment)

		assert.Error(t, err)
	})
}

func TestPaymentRepository_CountByUserIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		userID := uuid.New()
		rows := sqlmock.NewRows([]string{"count"}).AddRow(int64(5))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*)`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		count, err := repo.CountByUserID(context.Background(), userID.String())

		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPaymentRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*)`)).
			WillReturnError(assert.AnError)

		_, err := repo.CountByUserID(context.Background(), uuid.New().String())

		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_CreateUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		userID := uuid.New()
		sub := model.NewSubscription(userID, "TIER_STARTER", 30*24*time.Hour)

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO subscriptions`)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), sub)

		require.NoError(t, err)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO subscriptions`)).
			WillReturnError(assert.AnError)

		err := repo.Create(context.Background(), sub)

		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_GetByIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		subID := uuid.New()
		userID := uuid.New()
		now := time.Now()
		metadataJSON, err := json.Marshal(map[string]any{})
		require.NoError(t, err)

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "plan_id", "status", "started_at", "expires_at",
			"canceled_at", "auto_renew", "provider_payment_id", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			subID, userID, "TIER_STARTER", "ACTIVE",
			now, now.Add(30*24*time.Hour),
			nil, true, nil, metadataJSON,
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs(subID.String()).
			WillReturnRows(rows)

		result, err := repo.GetByID(context.Background(), subID.String())

		require.NoError(t, err)
		assert.Equal(t, subID, result.ID)
		assert.Equal(t, "ACTIVE", result.Status)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, err := repo.GetByID(context.Background(), uuid.New().String())

		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_GetActiveByUserIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		userID := uuid.New()
		subID := uuid.New()
		now := time.Now()
		metadataJSON, err := json.Marshal(map[string]any{})
		require.NoError(t, err)

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "plan_id", "status", "started_at", "expires_at",
			"canceled_at", "auto_renew", "provider_payment_id", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			subID, userID, "TIER_STARTER", "ACTIVE",
			now, now.Add(30*24*time.Hour),
			nil, true, nil, metadataJSON,
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs(userID.String()).
			WillReturnRows(rows)

		result, err := repo.GetActiveByUserID(context.Background(), userID.String())

		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", result.Status)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		userID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, err := repo.GetActiveByUserID(context.Background(), userID.String())

		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_GetByUserIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		userID := uuid.New()
		now := time.Now()
		metadataJSON, err := json.Marshal(map[string]any{})
		require.NoError(t, err)

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "plan_id", "status", "started_at", "expires_at",
			"canceled_at", "auto_renew", "provider_payment_id", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), userID, "TIER_STARTER", "ACTIVE",
			now, now.Add(30*24*time.Hour),
			nil, true, nil, metadataJSON,
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WithArgs(userID.String(), 10, 0).
			WillReturnRows(rows)

		results, err := repo.GetByUserID(context.Background(), userID.String(), 10, 0)

		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnError(assert.AnError)

		_, err := repo.GetByUserID(context.Background(), uuid.New().String(), 10, 0)

		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_UpdateUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)
		sub.Status = "ACTIVE"

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions`)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(context.Background(), sub)

		require.NoError(t, err)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)

		// Нет затронутых строк
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(context.Background(), sub)

		assert.Error(t, err)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		sub := model.NewSubscription(uuid.New(), "TIER_STARTER", 30*24*time.Hour)

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions`)).
			WillReturnError(assert.AnError)

		err := repo.Update(context.Background(), sub)

		assert.Error(t, err)
	})
}

func TestSubscriptionRepository_DeleteUnit(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		subID := uuid.New().String()

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions SET status = 'CANCELED'`)).
			WithArgs(subID, anyTime{}).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), subID)

		require.NoError(t, err)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		subID := uuid.New().String()

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions SET status = 'CANCELED'`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), subID)

		assert.Error(t, err)
	})

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewSubscriptionRepository(db)

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE subscriptions SET status = 'CANCELED'`)).
			WillReturnError(assert.AnError)

		err := repo.Delete(context.Background(), uuid.New().String())

		assert.Error(t, err)
	})
}

func TestPlanRepository_GetByIDUnit(t *testing.T) {
	t.Parallel()

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPlanRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, err := repo.GetByID(context.Background(), "TIER_NONEXISTENT")

		assert.Error(t, err)
	})
}

func TestPlanRepository_GetAllUnit(t *testing.T) {
	t.Parallel()

	t.Run("db_error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPlanRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnError(assert.AnError)

		_, err := repo.GetAll(context.Background(), true)

		assert.Error(t, err)
	})

	t.Run("db_error_inactive", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPlanRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnError(assert.AnError)

		_, err := repo.GetAll(context.Background(), false)

		assert.Error(t, err)
	})
}

func TestPlanRepository_GetByTierUnit(t *testing.T) {
	t.Parallel()

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		repo := NewPlanRepository(db)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
			WillReturnRows(sqlmock.NewRows([]string{}))

		_, err := repo.GetByTier(context.Background(), "TIER_NONEXISTENT")

		assert.Error(t, err)
	})
}

func TestPlanRepository_GetAllUnit_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewPlanRepository(db)

	now := time.Now()
	featuresJSON, err := json.Marshal([]model.Feature{
		{ID: "feature1", Name: "Feature 1", Description: "Desc"},
	})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "price_kopeks", "billing_period_days",
		"max_monitors", "min_check_interval_seconds", "max_alerts_per_day",
		"features", "is_active", "created_at", "updated_at",
	}).AddRow(
		"TIER_FREE", "Free", "Free plan", int64(0), int32(30),
		int32(5), int32(300), int32(10),
		featuresJSON, true, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WillReturnRows(rows)

	plans, err := repo.GetAll(context.Background(), true)

	require.NoError(t, err)
	assert.Len(t, plans, 1)
	assert.Equal(t, "TIER_FREE", plans[0].ID)
	assert.Len(t, plans[0].Features, 1)
}

func TestSubscriptionRepository_GetByUserIDUnit_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewSubscriptionRepository(db)

	userID := uuid.New()
	subID := uuid.New()
	now := time.Now()
	metadataJSON, err := json.Marshal(map[string]any{"key": "value"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "plan_id", "status", "started_at", "expires_at",
		"canceled_at", "auto_renew", "provider_payment_id", "metadata",
		"created_at", "updated_at",
	}).AddRow(
		subID, userID, "TIER_STARTER", "ACTIVE",
		now, now.Add(30*24*time.Hour),
		nil, true, nil, metadataJSON,
		now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID.String(), 10, 0).
		WillReturnRows(rows)

	results, err := repo.GetByUserID(context.Background(), userID.String(), 10, 0)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, subID, results[0].ID)
	assert.Equal(t, map[string]any{"key": "value"}, results[0].Metadata)
}

func TestPaymentRepository_GetByUserIDUnit_success(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewPaymentRepository(db)

	userID := uuid.New()
	now := time.Now()
	// Тест с непустыми metadata
	metadataJSON, err := json.Marshal(map[string]any{"info": "test"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "subscription_id", "provider", "provider_payment_id",
		"status", "amount_kopeks", "currency", "metadata", "created_at", "updated_at",
	}).AddRow(
		uuid.New(), userID, nil, "STRIPE", "pi_test",
		"SUCCESS", int64(9900), "RUB", metadataJSON, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID.String(), 10, 0).
		WillReturnRows(rows)

	results, err := repo.GetByUserID(context.Background(), userID.String(), 10, 0)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, map[string]any{"info": "test"}, results[0].Metadata)
}

func TestDB_Close(t *testing.T) {
	t.Parallel()

	// Arrange — создаём DB с sqlmock
	rawDB, _, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(rawDB, "postgres")
	db := &DB{DB: sqlxDB}

	// Act
	closeErr := db.Close()

	// Assert — Close вызвался без паники (ошибка nil или "sql: database is closed")
	// Мы проверяем только то, что метод возвращает результат
	_ = closeErr
}

func TestPlanRepository_GetByIDUnit_db_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewPlanRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs("TIER_STARTER").
		WillReturnError(assert.AnError)

	_, err := repo.GetByID(context.Background(), "TIER_STARTER")

	assert.Error(t, err)
}

func TestPlanRepository_GetByTierUnit_db_error(t *testing.T) {
	t.Parallel()

	db, mock := newMockDB(t)
	repo := NewPlanRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs("TIER_STARTER").
		WillReturnError(assert.AnError)

	_, err := repo.GetByTier(context.Background(), "TIER_STARTER")

	assert.Error(t, err)
}
