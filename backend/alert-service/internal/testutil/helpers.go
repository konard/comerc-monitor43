package testutil

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// SetupTestDatabase создаёт подключение к тестовой БД
func SetupTestDatabase(t *testing.T) (*sqlx.DB, func()) {
	ctx := context.Background()

	container, db, err := StartPostgreSQLContainer(ctx)
	require.NoError(t, err, "failed to create postgres container")
	require.NotNil(t, db, "database connection should not be nil")

	// Создаём таблицы для тестов
	createTables(ctx, t, db)

	// Возвращаем cleanup функцию
	cleanup := func() {
		if err := container.Shutdown(ctx); err != nil {
			t.Logf("failed to shutdown container: %v", err)
		}
	}

	return db, cleanup
}

// CreateTablesForTest создаёт таблицы для тестов (публичная версия для использования из других пакетов)
func CreateTablesForTest(t *testing.T, db *sqlx.DB) {
	createTables(context.Background(), t, db)
}

// createTables создаёт таблицы для тестов
func createTables(ctx context.Context, t *testing.T, db *sqlx.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS alerts (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			monitor_id UUID NOT NULL,
			alert_rule_id UUID,
			status VARCHAR(50) NOT NULL,
			type VARCHAR(50) NOT NULL,
			enabled BOOLEAN NOT NULL,
			consecutive_failures INTEGER NOT NULL,
			threshold_ms INTEGER,
			config TEXT,
			acknowledged_by UUID,
			acknowledged_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_rules (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			monitor_id UUID NOT NULL,
			enabled BOOLEAN NOT NULL,
			consecutive_failures INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_channels (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			enabled BOOLEAN NOT NULL,
			verified BOOLEAN NOT NULL,
			failure_count INTEGER NOT NULL,
			last_failure_at TIMESTAMP,
			telegram_config TEXT,
			email_config TEXT,
			webhook_config TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS delivery_attempts (
			id UUID PRIMARY KEY,
			alert_id UUID NOT NULL,
			alert_channel_id UUID NOT NULL,
			status VARCHAR(50) NOT NULL,
			error_message TEXT,
			retry_count INTEGER NOT NULL,
			next_retry_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY,
			user_id UUID,
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100) NOT NULL,
			resource_id TEXT,
			fields TEXT,
			ip_address TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_mutes (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			monitor_id UUID,
			scope VARCHAR(50) NOT NULL,
			muted_until TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by UUID NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_escalations (
			id UUID PRIMARY KEY,
			alert_id UUID NOT NULL,
			level INTEGER NOT NULL,
			escalated_to_channel_id UUID,
			reason TEXT NOT NULL,
			timeout_minutes INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS maintenance_windows (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			monitor_id UUID,
			name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'scheduled',
			recurrence TEXT NOT NULL DEFAULT 'once',
			is_global BOOLEAN NOT NULL DEFAULT false,
			pause_monitoring BOOLEAN NOT NULL DEFAULT false,
			suppress_alerts BOOLEAN NOT NULL DEFAULT false,
			safe_mode BOOLEAN NOT NULL DEFAULT false,
			monitor_ids TEXT[] NOT NULL DEFAULT '{}',
			starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
			ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
			activated_at TIMESTAMP WITH TIME ZONE,
			completed_at TIMESTAMP WITH TIME ZONE,
			version INTEGER NOT NULL DEFAULT 0,
			reason TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS monitor_status_changes (
			id UUID PRIMARY KEY,
			monitor_id UUID NOT NULL,
			user_id UUID NOT NULL,
			old_status VARCHAR(50) NOT NULL,
			new_status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_channel_priorities (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			alert_rule_id UUID NOT NULL,
			alert_channel_id UUID NOT NULL,
			priority INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, tableSQL := range tables {
		if _, err := db.ExecContext(ctx, tableSQL); err != nil {
			t.Fatalf("failed to create table: %v\nSQL: %s", err, tableSQL)
		}
	}
}

// RunMigrations запускает миграции
func RunMigrations(ctx context.Context, t *testing.T, db *sqlx.DB) error {
	createTables(ctx, t, db)
	return nil
}
