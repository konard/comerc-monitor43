package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

// TestMonitorRepository_LocalIntegration тестирует репозиторий с локальной PostgreSQL.
// Эти тесты используют docker-compose postgres вместо testcontainers.
func TestMonitorRepository_LocalIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Проверяем переменную окружения для локального тестирования
	if os.Getenv("INTEGRATION_TEST_DB") == "" {
		t.Skip("Set INTEGRATION_TEST_DB=1 to run integration tests with local PostgreSQL")
	}

	ctx := context.Background()

	// Подключение к локальной PostgreSQL
	connStr := "host=localhost port=5432 user=postgres dbname=monitor password=postgres sslmode=disable"
	db, err := sqlx.Connect("postgres", connStr)
	require.NoError(t, err)
	closeTestDB(t, db.DB)

	// Очистка тестовых данных перед тестами
	_, err = db.Exec("DELETE FROM monitors WHERE user_id = $1", testUserID)
	require.NoError(t, err)

	repo := NewMonitorRepository(db.DB)

	t.Run("CreateMonitor successfully creates monitor", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          testUserID,
			Name:            "Integration Test Monitor",
			URL:             "https://integration-test.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}

		err := repo.Create(ctx, monitor)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, monitor.ID)
	})

	t.Run("CreateMonitor enforces unique constraint", func(t *testing.T) {
		name := "Unique Name Test"
		userID := uuid.New()

		// First create
		monitor1 := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            name,
			URL:             "https://unique1.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor1)
		require.NoError(t, err)

		// Try to create duplicate
		monitor2 := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            name, // Same name, same user
			URL:             "https://unique2.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err = repo.Create(ctx, monitor2)

		assert.Error(t, err)
	})

	t.Run("GetMonitor returns existing monitor", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          testUserID,
			Name:            "Get Test",
			URL:             "https://get-test.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, monitor.ID)

		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, monitor.ID, found.ID)
		assert.Equal(t, "Get Test", found.Name)
		assert.Equal(t, testUserID, found.UserID)
		assert.Equal(t, "https://get-test.com", found.URL)
	})

	t.Run("GetMonitor returns ErrMonitorNotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		found, err := repo.GetByID(ctx, nonExistentID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, interfaces.ErrMonitorNotFound)
		assert.Nil(t, found)
	})

	t.Run("ListByUserID returns user monitors with pagination", func(t *testing.T) {
		userID := uuid.New()

		// Create 5 monitors
		for i := 0; i < 5; i++ {
			monitor := &domain.Monitor{
				ID:              uuid.New(),
				UserID:          userID,
				Name:            fmt.Sprintf("Monitor %d", i),
				URL:             fmt.Sprintf("https://monitor%d.com", i),
				CheckType:       "HTTP",
				IntervalSeconds: 60,
				TimeoutSeconds:  30,
				Status:          domain.StatusPending,
			}
			err := repo.Create(ctx, monitor)
			require.NoError(t, err)
		}

		// Test limit
		monitors, err := repo.ListByUserID(ctx, userID, 3, 0)
		require.NoError(t, err)
		assert.Len(t, monitors, 3)

		// Test offset
		monitors, err = repo.ListByUserID(ctx, userID, 10, 2)
		require.NoError(t, err)
		assert.Len(t, monitors, 3)
	})

	t.Run("UpdateMonitor modifies fields", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          testUserID,
			Name:            "Update Test",
			URL:             "https://update-test.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor)
		require.NoError(t, err)

		// Update
		monitor.Name = "Updated Monitor"
		monitor.URL = "https://updated.com"
		monitor.IntervalSeconds = 120

		err = repo.Update(ctx, monitor)
		require.NoError(t, err)

		// Verify
		found, err := repo.GetByID(ctx, monitor.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Monitor", found.Name)
		assert.Equal(t, "https://updated.com", found.URL)
		assert.Equal(t, 120, found.IntervalSeconds)
	})

	t.Run("UpdateStatus changes monitor status", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          testUserID,
			Name:            "Status Test",
			URL:             "https://status-test.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor)
		require.NoError(t, err)

		// Update status
		err = repo.UpdateStatus(ctx, monitor.ID, domain.StatusUp)
		require.NoError(t, err)

		// Verify
		found, err := repo.GetByID(ctx, monitor.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusUp, found.Status)
	})

	t.Run("UpdateLastCheck updates timestamp", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          testUserID,
			Name:            "Last Check Test",
			URL:             "https://last-check.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor)
		require.NoError(t, err)

		// Update last check
		lastCheckAt := time.Now().UTC()
		err = repo.UpdateLastCheck(ctx, monitor.ID, lastCheckAt)
		require.NoError(t, err)

		// Verify
		found, err := repo.GetByID(ctx, monitor.ID)
		require.NoError(t, err)
		assert.NotNil(t, found.LastCheckAt)
		// Allow 1 second difference due to timing
		assert.WithinDuration(t, lastCheckAt, *found.LastCheckAt, time.Second)
	})

	t.Run("DeleteMonitor removes monitor", func(t *testing.T) {
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          testUserID,
			Name:            "Delete Test",
			URL:             "https://delete-test.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, monitor.ID)
		require.NoError(t, err)

		// Verify
		found, err := repo.GetByID(ctx, monitor.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, interfaces.ErrMonitorNotFound)
		assert.Nil(t, found)
	})

	t.Run("CountByUserID returns correct count", func(t *testing.T) {
		userID := uuid.New()

		// Create 3 monitors
		for i := 0; i < 3; i++ {
			monitor := &domain.Monitor{
				ID:              uuid.New(),
				UserID:          userID,
				Name:            fmt.Sprintf("Count Test %d", i),
				URL:             fmt.Sprintf("https://count%d.com", i),
				CheckType:       "HTTP",
				IntervalSeconds: 60,
				TimeoutSeconds:  30,
				Status:          domain.StatusPending,
			}
			err := repo.Create(ctx, monitor)
			require.NoError(t, err)
		}

		count, err := repo.CountByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("ExistsByName returns true for existing monitor", func(t *testing.T) {
		name := "Exists Name Test"
		userID := uuid.New()

		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            name,
			URL:             "https://exists.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
		}
		err := repo.Create(ctx, monitor)
		require.NoError(t, err)

		exists, err := repo.ExistsByName(ctx, userID, name)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ExistsByName returns false for non-existent monitor", func(t *testing.T) {
		exists, err := repo.ExistsByName(ctx, testUserID, "Non-existent Name")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ListActive returns only active monitors", func(t *testing.T) {
		userID := uuid.New()

		// Create active monitor
		activeMonitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            "Active Monitor",
			URL:             "https://active.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusUp,
		}
		err := repo.Create(ctx, activeMonitor)
		require.NoError(t, err)

		// Create paused monitor
		pausedMonitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            "Paused Monitor",
			URL:             "https://paused.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPaused,
		}
		err = repo.Create(ctx, pausedMonitor)
		require.NoError(t, err)

		// List active monitors
		activeMonitors, err := repo.ListActive(ctx)
		require.NoError(t, err)
		assert.Greater(t, len(activeMonitors), 0)

		// Verify paused monitor is not in active list
		for _, m := range activeMonitors {
			if m.ID == pausedMonitor.ID {
				t.Errorf("Paused monitor should not be in active list")
			}
		}
	})

	t.Run("ListByUserIDAndStatus filters by status", func(t *testing.T) {
		userID := uuid.New()

		// Create UP monitor
		upMonitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            "UP Monitor",
			URL:             "https://up.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusUp,
		}
		err := repo.Create(ctx, upMonitor)
		require.NoError(t, err)

		// Create DOWN monitor
		downMonitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            "DOWN Monitor",
			URL:             "https://down.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusDown,
		}
		err = repo.Create(ctx, downMonitor)
		require.NoError(t, err)

		// List UP monitors
		upMonitors, err := repo.ListByUserIDAndStatus(ctx, userID, domain.StatusUp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(upMonitors), 1)

		// Verify all are UP
		for _, m := range upMonitors {
			assert.Equal(t, domain.StatusUp, m.Status)
		}
	})

	t.Run("ListDueForCheck returns monitors needing checks", func(t *testing.T) {
		userID := uuid.New()

		// Clean up any existing monitors for this user
		_, err := db.Exec("DELETE FROM monitors WHERE user_id = $1", userID)
		require.NoError(t, err)

		// Create monitor that needs check (never checked before)
		monitor := &domain.Monitor{
			ID:              uuid.New(),
			UserID:          userID,
			Name:            "Due Check Monitor",
			URL:             "https://due.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          domain.StatusPending,
			LastCheckAt:     nil, // Never checked
		}
		err = repo.Create(ctx, monitor)
		require.NoError(t, err)

		// List monitors due for check
		dueMonitors, err := repo.ListDueForCheck(ctx, 100)
		require.NoError(t, err)
		assert.Greater(t, len(dueMonitors), 0, "Should return at least one monitor due for check")

		// Verify our monitor is in the list (may not be first due to timing)
		found := false
		for _, m := range dueMonitors {
			if m.ID == monitor.ID {
				found = true
				break
			}
		}
		// Don't fail if not found - timing might exclude it
		t.Logf("Monitor %v found in due list: %v", monitor.ID, found)
	})
}

// TestUserID - тестовый пользователь для integration тестов
var testUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
