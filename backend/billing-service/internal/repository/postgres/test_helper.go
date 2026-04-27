package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

// SetupTestDB создаёт тестовую базу данных с помощью testcontainers.
func SetupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Запускаем PostgreSQL контейнер
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "billing_test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err, "failed to start postgres container")

	// Получаем порт
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err, "failed to get mapped port")

	// Подключаемся к БД
	host := "localhost"
	db, err := NewDB(host, int(port.Num()), "test", "test", "billing_test", "disable")
	require.NoError(t, err, "failed to connect to test database")

	// Применяем миграции
	if err := runMigrationsForTest(t, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Cleanup функция
	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

// SetupTestDBWithDockertest альтернативный способ с dockertest.
func SetupTestDBWithDockertest(t *testing.T) (*DB, func()) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "failed to create docker pool")

	// Запускаем PostgreSQL
	resource, err := pool.Run("postgres", "15-alpine", []string{
		"POSTGRES_USER=test",
		"POSTGRES_PASSWORD=test",
		"POSTGRES_DB=billing_test",
	})
	require.NoError(t, err, "failed to start postgres container")

	// Очистка при ошибке
	if err != nil {
		if purgeErr := pool.Purge(resource); purgeErr != nil {
			t.Logf("failed to purge resource: %v", purgeErr)
		}
		t.Fatalf("failed to start resource: %v", err)
	}

	_ = resource.GetHostPort("5432/tcp")
	var db *DB

	// Retry пока БД не будет готова
	err = pool.Retry(func() error {
		var err error
		db, err = NewDB("localhost", 5432, "test", "test", "billing_test", "disable")
		if err != nil {
			return err
		}
		return db.Ping()
	})
	require.NoError(t, err, "failed to connect to database")

	// Применяем миграции
	if err := runMigrationsForTest(t, db); err != nil {
		if purgeErr := pool.Purge(resource); purgeErr != nil {
			t.Logf("failed to purge resource: %v", purgeErr)
		}
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
		if err := pool.Purge(resource); err != nil {
			t.Logf("failed to purge resource: %v", err)
		}
	}

	return db, cleanup
}

// runMigrationsForTest применяет миграции для тестов.
func runMigrationsForTest(t *testing.T, db *DB) error {
	t.Helper()

	// Создаём таблицы напрямую для тестов
	ctx := context.Background()

	// Create subscription_plans table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS subscription_plans (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			price_kopeks BIGINT NOT NULL DEFAULT 0,
			billing_period_days INTEGER NOT NULL DEFAULT 30,
			max_monitors INTEGER NOT NULL,
			min_check_interval_seconds INTEGER NOT NULL,
			max_alerts_per_day INTEGER NOT NULL,
			features JSONB NOT NULL DEFAULT '[]',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create subscription_plans table: %v", err)
	}

	// Insert test plans
	_, err = db.ExecContext(ctx, `
		INSERT INTO subscription_plans (id, name, description, price_kopeks, billing_period_days, max_monitors, min_check_interval_seconds, max_alerts_per_day, features) VALUES
		('TIER_FREE', 'Free', 'Бесплатный тариф', 0, 30, 5, 300, 10, '[{"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"}]'),
		('TIER_STARTER', 'Starter', 'Начальный тариф', 29900, 30, 25, 60, 100, '[{"id":"basic_monitoring","name":"Basic monitoring"}]')
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to insert test plans: %v", err)
	}

	// Create subscriptions table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			plan_id VARCHAR(50) NOT NULL REFERENCES subscription_plans(id),
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			started_at TIMESTAMP NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMP NOT NULL,
			canceled_at TIMESTAMP,
			auto_renew BOOLEAN NOT NULL DEFAULT TRUE,
			provider_payment_id VARCHAR(255),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create subscriptions table: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_user_active_pending_unique
		ON subscriptions(user_id, status)
		WHERE status IN ('ACTIVE', 'PENDING')
	`)
	if err != nil {
		return fmt.Errorf("failed to create subscriptions unique index: %v", err)
	}

	// Create payments table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			subscription_id UUID REFERENCES subscriptions(id),
			provider VARCHAR(20) NOT NULL,
			provider_payment_id VARCHAR(255) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			amount_kopeks BIGINT NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create payments table: %v", err)
	}

	return nil
}

// CreateTestSubscription создаёт тестовую подписку.
func CreateTestSubscription(t *testing.T, db *DB, userID uuid.UUID, planID string) *model.Subscription {
	t.Helper()

	ctx := context.Background()
	sub := model.NewSubscription(userID, planID, 30*24*time.Hour)

	_, err := db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, user_id, plan_id, status, started_at, expires_at, auto_renew, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, sub.ID, sub.UserID, sub.PlanID, sub.Status, sub.StartedAt, sub.ExpiresAt, sub.AutoRenew, "{}", sub.CreatedAt, sub.UpdatedAt)
	require.NoError(t, err, "failed to create test subscription")

	return sub
}

// CreateTestPayment создаёт тестовый платёж.
func CreateTestPayment(t *testing.T, db *DB, userID, subscriptionID uuid.UUID, providerPaymentID string) *model.Payment {
	t.Helper()

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, user_id, plan_id, status, started_at, expires_at, auto_renew, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '30 days', $5, $6, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, subscriptionID, userID, "TIER_FREE", model.StatusPending, true, "{}")
	require.NoError(t, err, "failed to ensure test subscription for payment")

	payment := model.NewPayment(userID, subscriptionID, model.ProviderYookassa, providerPaymentID, 29900, "RUB")

	_, err = db.ExecContext(ctx, `
		INSERT INTO payments (id, user_id, subscription_id, provider, provider_payment_id, status, amount_kopeks, currency, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, payment.ID, payment.UserID, payment.SubscriptionID, payment.Provider, payment.ProviderPaymentID, payment.Status, payment.AmountKopeks, payment.Currency, "{}", payment.CreatedAt, payment.UpdatedAt)
	require.NoError(t, err, "failed to create test payment")

	return payment
}
