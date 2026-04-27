package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	testcontainers "github.com/raul/monitor/backend/monitor-service/internal/testcontainers"
)

// runMigrations выполняет миграции базы данных для тестов.
func runMigrations(db *DB) error {
	// Простой способ: выполняем SQL миграции напрямую
	migration := `CREATE TABLE IF NOT EXISTS monitors (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL,
		name VARCHAR(255) NOT NULL,
		url VARCHAR(2048) NOT NULL,
		check_type VARCHAR(50) NOT NULL DEFAULT 'HTTP',
		interval_seconds INTEGER NOT NULL CHECK (interval_seconds >= 30 AND interval_seconds <= 3600),
		timeout_seconds INTEGER NOT NULL DEFAULT 30 CHECK (timeout_seconds < interval_seconds),
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
		working_hours_start TIME,
		working_hours_end TIME,
		working_days VARCHAR(20)[],
		degraded_response_time_threshold INTEGER,
		degraded_failure_rate_threshold INTEGER,
		last_check_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(user_id, name)
	);
	CREATE INDEX IF NOT EXISTS idx_monitors_user_id ON monitors(user_id);
	CREATE INDEX IF NOT EXISTS idx_monitors_status ON monitors(status);
	CREATE INDEX IF NOT EXISTS idx_monitors_created_at ON monitors(created_at DESC);`

	if _, err := db.Exec(migration); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	return nil
}

// TestPostgres_Integration тестирует репозиторий мониторов с PostgreSQL в testcontainers.
func TestPostgres_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := testcontainers.SetupPostgreSQLContainer(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, testcontainers.TeardownPostgreSQLContainer(context.Background(), pgContainer))
	})

	db, err := NewDB(pgContainer.ConnectionString)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	// Run migrations
	err = runMigrations(db)
	require.NoError(t, err, "failed to run migrations")

	repo := NewMonitorRepository(db.DB)

	// Test: Create monitor
	t.Run("CreateMonitor", func(t *testing.T) {
		userID := uuid.New()
		monitor, err := domain.NewMonitor(userID, "Test Monitor", "https://example.com", 60)

		require.NoError(t, err)
		require.NotNil(t, monitor)

		err = repo.Create(ctx, monitor)
		assert.NoError(t, err)

		// Проверяем, что ID установлен
		assert.NotEqual(t, uuid.Nil, monitor.ID)
	})

	// Test: Get monitor by ID
	t.Run("GetMonitor", func(t *testing.T) {
		userID := uuid.New()
		monitor, err := domain.NewMonitor(userID, "GetTest", "https://example.com", 60)
		require.NoError(t, err)
		err = repo.Create(ctx, monitor)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, monitor.ID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, monitor.ID, found.ID)
		assert.Equal(t, "GetTest", found.Name)
	})

	// Test: List monitors by user
	t.Run("ListMonitors", func(t *testing.T) {
		userID := uuid.New()

		// Создаём несколько мониторов
		for i := 0; i < 3; i++ {
			monitor, err := domain.NewMonitor(userID, fmt.Sprintf("Monitor %d", i), "https://example.com", 60)
			require.NoError(t, err)
			err = repo.Create(ctx, monitor)
			require.NoError(t, err)
		}

		monitors, err := repo.ListByUserID(ctx, userID, 10, 0)

		assert.NoError(t, err)
		assert.Len(t, monitors, 3)
	})

	// Test: Update status
	t.Run("UpdateStatus", func(t *testing.T) {
		userID := uuid.New()
		monitor, err := domain.NewMonitor(userID, "UpdateTest", "https://example.com", 60)
		require.NoError(t, err)
		err = repo.Create(ctx, monitor)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, monitor.ID, domain.StatusUp)

		assert.NoError(t, err)

		updated, err := repo.GetByID(ctx, monitor.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusUp, updated.Status)
	})

	// Test: Delete monitor
	t.Run("DeleteMonitor", func(t *testing.T) {
		userID := uuid.New()
		monitor, err := domain.NewMonitor(userID, "DeleteTest", "https://example.com", 60)
		require.NoError(t, err)
		err = repo.Create(ctx, monitor)
		require.NoError(t, err)

		err = repo.Delete(ctx, monitor.ID)

		assert.NoError(t, err)

		// Проверяем, что монитор удалён
		_, err = repo.GetByID(ctx, monitor.ID)
		assert.Error(t, err)
	})

	// Test: Count monitors
	t.Run("CountMonitors", func(t *testing.T) {
		userID := uuid.New()

		for i := 0; i < 5; i++ {
			monitor, err := domain.NewMonitor(userID, fmt.Sprintf("Monitor Count %d", i), "https://example.com", 60)
			require.NoError(t, err)
			err = repo.Create(ctx, monitor)
			require.NoError(t, err)
		}

		count, err := repo.CountByUserID(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	// Test: Check existence by name
	t.Run("ExistsByName", func(t *testing.T) {
		userID := uuid.New()
		monitor, err := domain.NewMonitor(userID, "UniqueTest", "https://example.com", 60)
		require.NoError(t, err)
		err = repo.Create(ctx, monitor)
		require.NoError(t, err)

		exists, err := repo.ExistsByName(ctx, userID, "UniqueTest")

		assert.NoError(t, err)
		assert.True(t, exists)

		// Проверяем несуществующий монитор
		notExists, err := repo.ExistsByName(ctx, userID, "NotExisting")
		assert.NoError(t, err)
		assert.False(t, notExists)
	})

	// Test: Get by user ID and name
	t.Run("GetByUserIDAndName", func(t *testing.T) {
		userID := uuid.New()
		monitor, err := domain.NewMonitor(userID, "GetByNameTest", "https://example.com", 60)
		require.NoError(t, err)
		err = repo.Create(ctx, monitor)
		require.NoError(t, err)

		found, err := repo.GetByUserIDAndName(ctx, userID, "GetByNameTest")

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, monitor.ID, found.ID)
		assert.Equal(t, "GetByNameTest", found.Name)

		// Проверяем несуществующий монитор
		_, err = repo.GetByUserIDAndName(ctx, userID, "NotExisting")
		assert.Error(t, err)
	})
}
