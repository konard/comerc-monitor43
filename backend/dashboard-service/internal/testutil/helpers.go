package testutil

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func SetupTestDatabase(t *testing.T) (*sqlx.DB, func()) {
	ctx := context.Background()

	container, db, err := NewPostgreSQLContainer(ctx)
	require.NoError(t, err, "failed to create postgres container")
	require.NotNil(t, db, "database connection should not be nil")

	createTables(ctx, t, db)

	cleanup := func() {
		if err := container.Shutdown(ctx); err != nil {
			t.Logf("failed to shutdown container: %v", err)
		}
	}

	return db, cleanup
}

func createTables(ctx context.Context, t *testing.T, db *sqlx.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS monitor_statuses (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			url TEXT NOT NULL,
			status VARCHAR(20) NOT NULL,
			uptime_percentage DECIMAL(5,2) DEFAULT 0,
			last_checked_at TIMESTAMPTZ,
			last_response_time_ms DECIMAL(10,2),
			tags TEXT DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS check_history (
			id UUID PRIMARY KEY,
			monitor_id UUID NOT NULL,
			status VARCHAR(20) NOT NULL,
			status_code INTEGER,
			response_time_ms DECIMAL(10,2),
			error_message TEXT,
			checked_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS incidents (
			id UUID PRIMARY KEY,
			monitor_id UUID NOT NULL,
			started_at TIMESTAMPTZ NOT NULL,
			ended_at TIMESTAMPTZ,
			duration_seconds INTEGER,
			check_count INTEGER DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, tableSQL := range tables {
		if _, err := db.ExecContext(ctx, tableSQL); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	extensions := []string{
		`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
	}
	for _, extSQL := range extensions {
		if _, err := db.ExecContext(ctx, extSQL); err != nil {
			t.Fatalf("failed to create extension: %v", err)
		}
	}
}
