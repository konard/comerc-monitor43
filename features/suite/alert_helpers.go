//go:build bdd

package suite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// alertDBTables перечисляет таблицы alert-service, которые тruncаются между сценариями.
var alertDBTables = []string{
	"alert_channel_priorities",
	"delivery_attempts",
	"audit_logs",
	"alert_escalations",
	"alert_mutes",
	"maintenance_windows",
	"monitor_status_changes",
	"alerts",
	"alert_rules",
	"alert_channels",
}

// createAlertDB создаёт отдельную БД для alert-service в общем postgres контейнере.
// Возвращает DSN новой БД и cleanup, который удалит её по завершении.
func createAlertDB(ctx context.Context, pg *tcpostgres.PostgresContainer) (string, func(), error) {
	baseDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, fmt.Errorf("get base connection string: %w", err)
	}

	dbName := fmt.Sprintf("alert_bdd_%d", randSuffix())

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
	alertDSN := u.String()

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

	return alertDSN, cleanup, nil
}

// applyAlertFKStubs создаёт минимальные таблицы users(id) и monitors(id) до применения
// миграций alert-service. Миграции alert-service содержат FOREIGN KEY на users(id) и
// monitors(id) — без этих stub-таблиц goose up падает на CREATE FK ошибкой.
func applyAlertFKStubs(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open alert db for fk stubs: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем временное соединение
		_ = db.Close()
	}()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY)`,
		`CREATE TABLE IF NOT EXISTS monitors (id UUID PRIMARY KEY)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create fk stub table: %w", err)
		}
	}
	return nil
}

// ensureAlertUserStub гарантирует наличие пользователя во внешней FK-таблице alert-БД.
func ensureAlertUserStub(ctx context.Context, stack *Stack, userID uuid.UUID) error {
	if stack == nil || stack.AlertDB == nil || userID == uuid.Nil {
		return nil
	}
	if _, err := stack.AlertDB.ExecContext(ctx,
		`INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		return fmt.Errorf("ensure alert user stub: %w", err)
	}
	return nil
}

// alertMigrationsPath возвращает путь к миграциям alert-service.
func alertMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename: .../features/suite/alert_helpers.go → ../../../backend/alert-service/migrations
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "alert-service", "migrations")
}

// alertServiceDir возвращает путь к рабочей директории alert-service.
// Используется для exec.Cmd.Dir, чтобы встроенный goose.Up в main.go
// нашёл ./migrations (повторное применение — no-op).
func alertServiceDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "alert-service")
}
