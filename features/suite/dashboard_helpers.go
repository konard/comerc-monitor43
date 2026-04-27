//go:build bdd

package suite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// createDashboardDB создаёт отдельную БД для dashboard в общем postgres контейнере.
// Возвращает DSN новой БД и cleanup, который удалит её по завершении.
func createDashboardDB(ctx context.Context, pg *tcpostgres.PostgresContainer) (string, func(), error) {
	baseDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, fmt.Errorf("get base connection string: %w", err)
	}

	dbName := fmt.Sprintf("dashboard_bdd_%d", randSuffix())

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
	dashDSN := u.String()

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

	return dashDSN, cleanup, nil
}

// applyMigrationsTo применяет goose миграции из директории к указанному DSN.
func applyMigrationsTo(dsn, migrationsDir string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем временное соединение
		_ = db.Close()
	}()

	if err := waitForSQLReady(db, 30*time.Second); err != nil {
		return fmt.Errorf("wait for db ready: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// dashboardMigrationsPath возвращает путь к миграциям dashboard-service.
func dashboardMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename: .../features/suite/dashboard_helpers.go → ../../../backend/dashboard-service/migrations
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "dashboard-service", "migrations")
}

// randSuffix возвращает псевдослучайный суффикс для уникального имени БД.
func randSuffix() int64 {
	return time.Now().UnixNano()
}
