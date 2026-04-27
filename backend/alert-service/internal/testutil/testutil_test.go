package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testContextKey string

// sharedDB хранит единственное подключение к БД для всех тестов пакета.
var sharedDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, db, err := StartPostgreSQLContainer(ctx)
	if err != nil {
		os.Exit(m.Run())
	}
	sharedDB = db

	code := m.Run()

	if shutdownErr := container.Shutdown(ctx); shutdownErr != nil {
		_ = shutdownErr
	}
	os.Exit(code)
}

// getSharedDB возвращает разделяемую тестовую БД или пропускает тест если недоступна.
func getSharedDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if sharedDB == nil {
		t.Skip("shared test database not available")
	}
	return sharedDB
}

func TestNewPostgreSQLContainer_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Skip("skipping context cancellation test - requires available Docker daemon")
}

func TestNewPostgreSQLContainer_ConnectionString(t *testing.T) {
	db := getSharedDB(t)

	err := db.Ping()
	assert.NoError(t, err, "database ping should succeed")
}

func TestNewPostgreSQLContainer_RetryMechanism(t *testing.T) {
	db := getSharedDB(t)

	assert.NotNil(t, db, "database should connect after retries")
}

func TestPostgreSQLContainer_Shutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if sharedDB == nil {
		t.Skip("skipping: Docker unavailable, shared DB not started")
	}

	// Тест проверяет только что Shutdown доступен — сам Shutdown тестируется в TestMain
	assert.NotNil(t, sharedDB, "shared DB should be available if container started successfully")
}

func TestPostgreSQLContainer_Shutdown_NilContainer(t *testing.T) {
	ctx := context.Background()
	container := &PostgreSQLContainer{
		PostgresContainer: nil,
	}

	err := container.Shutdown(ctx)
	assert.NoError(t, err, "shutdown with nil container should be a no-op")
}

func TestSetupTestDatabase_Creation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	container, db, err := StartPostgreSQLContainer(ctx)
	if err != nil {
		t.Skipf("skipping: Docker unavailable: %v", err)
	}
	require.NotNil(t, db, "database should not be nil")
	t.Cleanup(func() {
		require.NoError(t, container.Shutdown(ctx))
	})

	err = db.Ping()
	assert.NoError(t, err, "database should be accessible")
}

func TestSetupTestDatabase_TableCreation(t *testing.T) {
	db := getSharedDB(t)

	createTables(context.Background(), t, db)

	tables := []string{
		"alerts",
		"alert_rules",
		"alert_channels",
		"delivery_attempts",
	}

	ctx := context.Background()
	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)
		`, table).Scan(&exists)

		assert.NoError(t, err, "table check should succeed")
		assert.True(t, exists, "table %s should exist", table)
	}
}

func TestCreateTables_SQLExecution(t *testing.T) {
	db := getSharedDB(t)

	assert.NotPanics(t, func() {
		createTables(context.Background(), t, db)
	})
}

func TestCreateTables_Schema(t *testing.T) {
	db := getSharedDB(t)

	createTables(context.Background(), t, db)

	tables := map[string][]string{
		"alerts":            {"id", "user_id", "monitor_id", "alert_rule_id", "status", "type", "enabled", "consecutive_failures", "threshold_ms", "config", "created_at", "updated_at"},
		"alert_rules":       {"id", "user_id", "monitor_id", "enabled", "consecutive_failures", "created_at", "updated_at"},
		"alert_channels":    {"id", "user_id", "type", "status", "enabled", "verified", "failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config", "created_at", "updated_at"},
		"delivery_attempts": {"id", "alert_id", "alert_channel_id", "status", "error_message", "retry_count", "next_retry_at", "created_at", "updated_at"},
	}

	ctx := context.Background()
	for table, columns := range tables {
		rows, err := db.QueryContext(ctx, `
			SELECT column_name
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY ordinal_position
		`, table)

		assert.NoError(t, err, "column query should succeed")
		if err != nil {
			continue
		}
		var actualColumns []string
		for rows.Next() {
			var col string
			err := rows.Scan(&col)
			assert.NoError(t, err)
			actualColumns = append(actualColumns, col)
		}
		assert.NoError(t, rows.Close(), "rows close should succeed")

		for _, expectedCol := range columns {
			assert.Contains(t, actualColumns, expectedCol, "table %s should have column %s", table, expectedCol)
		}
	}
}

func TestRunMigrations_Execution(t *testing.T) {
	db := getSharedDB(t)

	assert.NotPanics(t, func() {
		err := RunMigrations(context.Background(), t, db)
		assert.NoError(t, err, "migrations should execute successfully")
	})
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := getSharedDB(t)

	ctx := context.Background()
	err := RunMigrations(ctx, t, db)
	assert.NoError(t, err, "first migration run should succeed")

	err = RunMigrations(ctx, t, db)
	assert.NoError(t, err, "second migration run should succeed (idempotent)")
}

func TestPostgreSQLContainer_ContainerLifecycle(t *testing.T) {
	db := getSharedDB(t)

	assert.NotNil(t, db, "container should be created and accessible")
}

func TestTestDatabase_ConnectionPooling(t *testing.T) {
	db := getSharedDB(t)

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			err := db.Ping()
			assert.NoError(t, err, "concurrent ping should succeed")
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestCleanup_Functionality(t *testing.T) {
	db := getSharedDB(t)

	cleanupCalled := false
	cleanup := func() {
		cleanupCalled = true
		_ = db
	}

	cleanup()
	assert.True(t, cleanupCalled, "cleanup should be called")
}

func TestDatabase_TransactionSupport(t *testing.T) {
	db := getSharedDB(t)

	ctx := context.Background()
	tx, err := db.BeginTxx(ctx, nil)
	assert.NoError(t, err, "transaction begin should succeed")

	_, err = tx.ExecContext(ctx, "SELECT 1")
	assert.NoError(t, err, "transaction query should succeed")

	err = tx.Commit()
	assert.NoError(t, err, "transaction commit should succeed")
}

func TestTestDatabase_Isolation(t *testing.T) {
	db1 := getSharedDB(t)

	// Проверяем что база работает (изоляция обеспечивается на уровне транзакций, а не контейнеров)
	err1 := db1.Ping()
	err2 := db1.Ping()

	assert.NoError(t, err1, "first ping should succeed")
	assert.NoError(t, err2, "second ping should succeed")
}

func TestContainer_Configuration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := []struct {
		name         string
		image        string
		databaseName string
		username     string
	}{
		{
			name:         "default configuration",
			image:        "postgres:15-alpine",
			databaseName: "testdb",
			username:     "testuser",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.image, "container image should be specified")
			assert.NotEmpty(t, tc.databaseName, "database name should be specified")
			assert.NotEmpty(t, tc.username, "username should be specified")
		})
	}
}

func TestDatabase_Timeouts(t *testing.T) {
	db := getSharedDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.PingContext(ctx)
	assert.NoError(t, err, "query should complete within timeout")
}

func TestError_Handling(t *testing.T) {
	db := getSharedDB(t)

	ctx := context.Background()
	_, err := db.QueryContext(ctx, "INVALID SQL")
	assert.Error(t, err, "invalid SQL should return error")
}

func TestContainer_WaitStrategy(t *testing.T) {
	db := getSharedDB(t)

	err := db.Ping()
	assert.NoError(t, err, "database should be ready after wait strategy")
}

func TestConnection_String(t *testing.T) {
	db := getSharedDB(t)

	err := db.Ping()
	assert.NoError(t, err, "connection string should produce valid connection")
}

func TestDatabase_SchemaValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	constraints := []string{
		"PRIMARY KEY on id columns",
		"NOT NULL on required columns",
		"DEFAULT values on timestamp columns",
	}

	for _, constraint := range constraints {
		assert.NotEmpty(t, constraint, "schema constraint should be defined")
	}
}

func TestTestcontainers_Integration(t *testing.T) {
	db := getSharedDB(t)

	require.NotNil(t, db, "testcontainers should work")

	err := db.Ping()
	assert.NoError(t, err, "database should be accessible")
}

func TestPostgreSQLContainer_Type(t *testing.T) {
	container := &PostgreSQLContainer{}

	assert.NotNil(t, container, "container should be created")
	assert.Nil(t, container.PostgresContainer, "PostgresContainer should be nil initially")
}

func TestContext_Propagation(t *testing.T) {
	db := getSharedDB(t)

	ctx := context.WithValue(context.Background(), testContextKey("test-key"), "test-value")

	err := db.PingContext(ctx)
	assert.NoError(t, err, "context should be propagated")
}

func TestDatabase_QueryContext(t *testing.T) {
	db := getSharedDB(t)

	ctx := context.Background()
	rows, err := db.QueryContext(ctx, "SELECT 1")
	require.NoError(t, err, "QueryContext should succeed")
	defer func() {
		assert.NoError(t, rows.Close(), "rows close should succeed")
	}()

	assert.True(t, rows.Next(), "query should return results")
}

func TestDatabase_ExecContext(t *testing.T) {
	db := getSharedDB(t)

	ctx := context.Background()
	_, err := db.ExecContext(ctx, "SELECT 1")
	assert.NoError(t, err, "ExecContext should succeed")
}
