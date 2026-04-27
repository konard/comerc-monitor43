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

// integrationDBTables перечисляет таблицы integration-service, которые трункаются между сценариями.
var integrationDBTables = []string{
	"webhook_delivery_attempts",
	"webhook_integrations",
	"api_key_usage_logs",
	"api_keys",
	"import_history",
}

// createIntegrationDB создаёт отдельную БД для integration-service в общем postgres контейнере.
// Возвращает DSN новой БД и cleanup, который удалит её по завершении.
func createIntegrationDB(ctx context.Context, pg *tcpostgres.PostgresContainer) (string, func(), error) {
	baseDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, fmt.Errorf("get base connection string: %w", err)
	}

	dbName := fmt.Sprintf("integration_bdd_%d", randSuffix())

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
	integrationDSN := u.String()

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

	return integrationDSN, cleanup, nil
}

// integrationMigrationsPath возвращает путь к миграциям integration-service.
func integrationMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "integration-service", "migrations")
}

// integrationServiceDir возвращает путь к рабочей директории integration-service.
// Используется для exec.Cmd.Dir, чтобы встроенный goose.Up в main.go нашёл ./migrations.
func integrationServiceDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "integration-service")
}
