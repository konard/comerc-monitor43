//go:build bdd

package suite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"

	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// billingDBTables перечисляет таблицы billing-service, которые трункаются между сценариями.
// Порядок не важен благодаря CASCADE; subscription_plans не очищаем — это справочник,
// заполняемый миграциями.
var billingDBTables = []string{
	"billing_audit_log",
	"payments",
	"subscriptions",
}

// createBillingDB создаёт отдельную БД для billing-service в общем postgres контейнере.
// Возвращает DSN новой БД и cleanup, который удалит её по завершении.
func createBillingDB(ctx context.Context, pg *tcpostgres.PostgresContainer) (string, func(), error) {
	baseDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, fmt.Errorf("get base connection string: %w", err)
	}

	dbName := fmt.Sprintf("billing_bdd_%d", randSuffix())

	adminDB, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("open admin db: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем admin-соединение
		_ = adminDB.Close()
	}()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		return "", nil, fmt.Errorf("create database %q: %w", dbName, err)
	}

	u, err := url.Parse(baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("parse base dsn: %w", err)
	}
	u.Path = "/" + dbName
	billingDSN := u.String()

	cleanup := func() {
		cleanupCtx := context.Background()
		db, err := sql.Open("postgres", baseDSN)
		if err != nil {
			return
		}
		defer func() {
			//nolint:errcheck // закрываем admin-соединение
			_ = db.Close()
		}()
		//nolint:errcheck // best-effort drop
		_, _ = db.ExecContext(cleanupCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
	}

	return billingDSN, cleanup, nil
}

// billingMigrationsPath возвращает путь к миграциям billing-service.
func billingMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename: .../features/suite/billing_helpers.go → ../../../backend/billing-service/migrations
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "billing-service", "migrations")
}

// billingServiceDir возвращает путь к рабочей директории billing-service.
// Используется для exec.Cmd.Dir, чтобы встроенный goose.Up в main.go
// нашёл ./migrations (повторное применение — no-op).
func billingServiceDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "billing-service")
}
