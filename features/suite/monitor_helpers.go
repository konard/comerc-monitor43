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

// monitorDBTables перечисляет таблицы monitor-service, которые тruncаются между сценариями.
var monitorDBTables = []string{
	"monitor_audit_log",
	"check_results",
	"incidents",
	"maintenance_window_monitors",
	"maintenance_windows",
	"monitors",
}

// createMonitorDB создаёт отдельную БД для monitor-service в общем postgres контейнере.
func createMonitorDB(ctx context.Context, pg *tcpostgres.PostgresContainer) (string, func(), error) {
	baseDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, fmt.Errorf("get base connection string: %w", err)
	}

	dbName := fmt.Sprintf("monitor_bdd_%d", randSuffix())

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
	monitorDSN := u.String()

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

	return monitorDSN, cleanup, nil
}

// monitorMigrationsPath возвращает путь к миграциям monitor-service.
func monitorMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename: .../features/suite/monitor_helpers.go → ../../backend/monitor-service/migrations
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "monitor-service", "migrations")
}
