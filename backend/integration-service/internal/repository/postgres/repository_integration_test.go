package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

// setupIntegrationDB создаёт подключение к тестовой БД и запускает миграции.
func setupIntegrationDB(t *testing.T) *DB {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	connStr := "host=localhost port=5433 user=postgres dbname=integration_service password=postgres sslmode=disable"
	db, err := NewDB(connStr)
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	// Запускаем все миграции
	if err := runAPIKeyMigrations(db); err != nil {
		t.Fatalf("failed to run api key migrations: %v", err)
	}
	if err := runIntegrationMigrations(db); err != nil {
		t.Fatalf("failed to run integration migrations: %v", err)
	}
	if err := runDeliveryMigrations(db); err != nil {
		t.Fatalf("failed to run delivery migrations: %v", err)
	}
	if err := runImportMigrations(db); err != nil {
		t.Fatalf("failed to run import migrations: %v", err)
	}
	if err := runUsageMigrations(db); err != nil {
		t.Fatalf("failed to run usage migrations: %v", err)
	}

	closeTestDB(t, db)
	return db
}

// runDeliveryMigrations создаёт таблицу delivery_attempts.
func runDeliveryMigrations(db *DB) error {
	migration := `
CREATE TABLE IF NOT EXISTS webhook_delivery_attempts (
    id UUID PRIMARY KEY,
    webhook_id UUID NOT NULL,
    alert_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    http_status_code INTEGER,
    response_time_ms INTEGER,
    error_message TEXT,
    error_code VARCHAR(100),
    payload_size_bytes INTEGER,
    payload_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    signature_algorithm VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_attempts_webhook_id ON webhook_delivery_attempts(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_attempts_status ON webhook_delivery_attempts(status);
`
	if _, err := db.DB.Exec(migration); err != nil {
		return err
	}
	return nil
}

// runImportMigrations создаёт таблицу import_history.
func runImportMigrations(db *DB) error {
	migration := `
CREATE TABLE IF NOT EXISTS import_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    source VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    total_monitors INTEGER NOT NULL DEFAULT 0,
    successful_imports INTEGER NOT NULL DEFAULT 0,
    failed_imports INTEGER NOT NULL DEFAULT 0,
    skipped_imports INTEGER NOT NULL DEFAULT 0,
    overwrite_existing BOOLEAN NOT NULL DEFAULT FALSE,
    imported_monitor_ids JSONB NOT NULL DEFAULT '[]',
    validation_errors JSONB NOT NULL DEFAULT '[]',
    conflict_resolution JSONB NOT NULL DEFAULT '{}',
    file_name VARCHAR(255),
    file_size_bytes INTEGER,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    error_message TEXT
);
CREATE INDEX IF NOT EXISTS idx_import_history_user_id ON import_history(user_id);
`
	if _, err := db.DB.Exec(migration); err != nil {
		return err
	}
	return nil
}

// runUsageMigrations создаёт таблицу api_key_usage_logs.
func runUsageMigrations(db *DB) error {
	migration := `
CREATE TABLE IF NOT EXISTS api_key_usage_logs (
    id UUID PRIMARY KEY,
    api_key_id UUID NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    http_status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    ip_address VARCHAR(50),
    user_agent TEXT,
    request_id UUID,
    rate_limited BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_api_key_id ON api_key_usage_logs(api_key_id);
`
	if _, err := db.DB.Exec(migration); err != nil {
		return err
	}
	return nil
}

// === DB методы ===

// TestDBPingAndClose проверяет Ping и Close.
func TestDBPingAndClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	connStr := "host=localhost port=5433 user=postgres dbname=integration_service password=postgres sslmode=disable"
	db, err := NewDB(connStr)
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	t.Run("ping succeeds", func(t *testing.T) {
		err := db.Ping()
		assert.NoError(t, err)
	})

	t.Run("ping context succeeds", func(t *testing.T) {
		err := db.PingContext(5 * time.Second)
		assert.NoError(t, err)
	})

	t.Run("close succeeds", func(t *testing.T) {
		err := db.Close()
		assert.NoError(t, err)
	})
}

// === API Key Repository ===

// TestAPIKeyRepositoryDeleteNotFound проверяет Delete несуществующего ключа.
func TestAPIKeyRepositoryDeleteNotFound(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, interfaces.ErrAPIKeyNotFound)
}

// TestAPIKeyRepositoryUpdateUsageNotFound проверяет UpdateUsage для несуществующего ключа.
func TestAPIKeyRepositoryUpdateUsageNotFound(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	stats := &interfaces.APIKeyUsageStats{
		TotalRequests: 1,
	}

	err := repo.UpdateUsage(ctx, uuid.New(), stats)
	assert.Error(t, err)
}

// === Webhook Repository ===

// TestWebhookRepositoryDeleteNotFound проверяет Delete несуществующего webhook.
func TestWebhookRepositoryDeleteNotFound(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assert.Error(t, err)
}

// TestWebhookDeliveryRepositoryNotFound проверяет GetByID для несуществующей попытки.
func TestWebhookDeliveryRepositoryNotFound(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookDeliveryRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

// TestWebhookDeliveryRepositoryListByWebhookIDEmpty проверяет ListByWebhookID без попыток.
func TestWebhookDeliveryRepositoryListByWebhookIDEmpty(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookDeliveryRepository(db)
	ctx := context.Background()

	attempts, err := repo.ListByWebhookID(ctx, uuid.New(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

// TestWebhookDeliveryRepositoryListPendingEmpty проверяет ListPendingForRetry без попыток.
func TestWebhookDeliveryRepositoryListPendingEmpty(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookDeliveryRepository(db)
	ctx := context.Background()

	attempts, err := repo.ListPendingForRetry(ctx, 100)
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

// TestWebhookDeliveryRepositoryDeleteOldAttempts проверяет DeleteOldAttempts.
func TestWebhookDeliveryRepositoryDeleteOldAttempts(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookDeliveryRepository(db)
	ctx := context.Background()

	deleted, err := repo.DeleteOldAttempts(ctx, 30)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(0))
}

// TestWebhookDeliveryRepositoryCreate проверяет Create delivery attempt.
func TestWebhookDeliveryRepositoryCreate(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookDeliveryRepository(db)
	ctx := context.Background()

	attempt := model.NewWebhookDeliveryAttempt(uuid.New(), uuid.New())

	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_delivery_attempts WHERE id = $1", attempt.ID)

	err := repo.Create(ctx, attempt)
	require.NoError(t, err)
}

// TestWebhookDeliveryRepositoryUpdate проверяет Create и Update delivery attempt.
func TestWebhookDeliveryRepositoryUpdate(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewWebhookDeliveryRepository(db)
	ctx := context.Background()

	attempt := model.NewWebhookDeliveryAttempt(uuid.New(), uuid.New())

	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_delivery_attempts WHERE id = $1", attempt.ID)

	err := repo.Create(ctx, attempt)
	require.NoError(t, err)

	statusCode := 200
	attempt.Status = "delivered"
	attempt.HTTPStatusCode = &statusCode

	err = repo.Update(ctx, attempt)
	require.NoError(t, err)
}

// === Import History Repository ===

// TestImportHistoryRepositoryGetByIDNotFound проверяет GetByID для несуществующей записи.
func TestImportHistoryRepositoryGetByIDNotFound(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

// TestImportHistoryRepositoryListByUserIDEmpty проверяет ListByUserID без записей.
func TestImportHistoryRepositoryListByUserIDEmpty(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	records, err := repo.ListByUserID(ctx, uuid.New(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, records)
}

// TestImportHistoryRepositoryCreateAndGet проверяет Create и GetByID.
func TestImportHistoryRepositoryCreateAndGet(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	userID := uuid.New()
	record := model.NewImportHistory(userID, model.ImportSourceCSV, "test.csv", 1024, false)

	cleanupExecContext(ctx, t, db, "DELETE FROM import_history WHERE user_id = $1", userID)

	err := repo.Create(ctx, record)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.ID, found.ID)
	assert.Equal(t, model.ImportSourceCSV, found.Source)
}

// TestImportHistoryRepositoryUpdate проверяет Update.
func TestImportHistoryRepositoryUpdate(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	userID := uuid.New()
	record := model.NewImportHistory(userID, model.ImportSourceCSV, "test.csv", 1024, false)

	cleanupExecContext(ctx, t, db, "DELETE FROM import_history WHERE user_id = $1", userID)

	err := repo.Create(ctx, record)
	require.NoError(t, err)

	record.MarkAsCompleted()

	err = repo.Update(ctx, record)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, record.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportStatusCompleted, found.Status)
}

// TestImportHistoryRepositoryUpdateStatus проверяет UpdateStatus.
func TestImportHistoryRepositoryUpdateStatus(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	userID := uuid.New()
	record := model.NewImportHistory(userID, model.ImportSourceCSV, "test.csv", 1024, false)

	cleanupExecContext(ctx, t, db, "DELETE FROM import_history WHERE user_id = $1", userID)

	err := repo.Create(ctx, record)
	require.NoError(t, err)

	err = repo.UpdateStatus(ctx, record.ID, model.ImportStatusFailed)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, record.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ImportStatusFailed, found.Status)
}

// TestImportHistoryRepositoryDelete проверяет Delete.
func TestImportHistoryRepositoryDelete(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	userID := uuid.New()
	record := model.NewImportHistory(userID, model.ImportSourceCSV, "test.csv", 1024, false)

	cleanupExecContext(ctx, t, db, "DELETE FROM import_history WHERE user_id = $1", userID)

	err := repo.Create(ctx, record)
	require.NoError(t, err)

	err = repo.Delete(ctx, record.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, record.ID)
	assert.Error(t, err)
}

// TestImportHistoryRepositoryListByUserIDWithData проверяет ListByUserID с данными.
func TestImportHistoryRepositoryListByUserIDWithData(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewImportHistoryRepository(db.DB)
	ctx := context.Background()

	userID := uuid.New()

	cleanupExecContext(ctx, t, db, "DELETE FROM import_history WHERE user_id = $1", userID)

	for i := 0; i < 3; i++ {
		record := model.NewImportHistory(userID, model.ImportSourceCSV, "test.csv", 1024, false)
		err := repo.Create(ctx, record)
		require.NoError(t, err)
	}

	records, err := repo.ListByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

// === API Key Usage Repository ===

// TestAPIKeyUsageRepositoryGetByIDNotFound проверяет GetByID для несуществующей записи.
func TestAPIKeyUsageRepositoryGetByIDNotFound(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewAPIKeyUsageRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

// TestAPIKeyUsageRepositoryListByAPIKeyIDEmpty проверяет ListByAPIKeyID без записей.
func TestAPIKeyUsageRepositoryListByAPIKeyIDEmpty(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewAPIKeyUsageRepository(db)
	ctx := context.Background()

	logs, err := repo.ListByAPIKeyID(ctx, uuid.New(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, logs)
}

// TestAPIKeyUsageRepositoryListByPeriodEmpty проверяет ListByAPIKeyIDAndPeriod без записей.
func TestAPIKeyUsageRepositoryListByPeriodEmpty(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewAPIKeyUsageRepository(db)
	ctx := context.Background()

	now := time.Now().Unix()
	past := now - 3600
	logs, err := repo.ListByAPIKeyIDAndPeriod(ctx, uuid.New(), past, now, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, logs)
}

// TestAPIKeyUsageRepositoryDeleteOldLogs проверяет DeleteOldLogs.
func TestAPIKeyUsageRepositoryDeleteOldLogs(t *testing.T) {
	t.Parallel()

	db := setupIntegrationDB(t)
	repo := NewAPIKeyUsageRepository(db)
	ctx := context.Background()

	deleted, err := repo.DeleteOldLogs(ctx, 90)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(0))
}
