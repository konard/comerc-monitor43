package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

// runIntegrationMigrations выполняет миграции базы данных для тестов
func runIntegrationMigrations(db *DB) error {
	migration := `
-- Table: webhook_integrations
CREATE TABLE IF NOT EXISTS webhook_integrations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB DEFAULT '{}',
    secret_key VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT true,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',
    severity_filter JSONB DEFAULT '["critical","warning","degraded"]',
    max_payload_size_bytes INTEGER NOT NULL DEFAULT 1048576,
    payload_handling_strategy VARCHAR(20) NOT NULL DEFAULT 'truncate',
    total_sent INTEGER NOT NULL DEFAULT 0,
    successful_sent INTEGER NOT NULL DEFAULT 0,
    failed_sent INTEGER NOT NULL DEFAULT 0,
    avg_response_time_ms INTEGER,
    last_sent_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    failure_count INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT webhook_integrations_user_id_name_unique UNIQUE (user_id, name),
    CONSTRAINT webhook_integrations_status_check CHECK (status IN ('active', 'inactive', 'failed', 'disabled')),
    CONSTRAINT webhook_integrations_priority_check CHECK (priority IN ('high', 'normal', 'low')),
    CONSTRAINT webhook_integrations_payload_strategy_check CHECK (payload_handling_strategy IN ('truncate', 'reject'))
);

CREATE INDEX IF NOT EXISTS idx_webhook_integrations_user_id ON webhook_integrations(user_id);
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_status ON webhook_integrations(status);
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_enabled ON webhook_integrations(enabled);
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_priority ON webhook_integrations(priority);
`

	if _, err := db.DB.Exec(migration); err != nil {
		return err
	}
	return nil
}

// setupTestDB создаёт тестовое подключение к PostgreSQL и выполняет миграции
func setupTestDB(t *testing.T) *DB {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Подключение к локальной PostgreSQL из docker-compose
	connStr := "host=localhost port=5433 user=postgres dbname=integration_service password=postgres sslmode=disable"
	db, err := NewDB(connStr)
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	// Run migrations
	err = runIntegrationMigrations(db)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Очистка таблиц перед каждым тестом
	ctx := context.Background()
	_, err = db.DB.ExecContext(ctx, "TRUNCATE TABLE webhook_integrations CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	return db
}

// TestWebhookRepository_Create тестирует создание webhook
func TestWebhookRepository_Create(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:                      uuid.New(),
		UserID:                  userID,
		Name:                    "Test Webhook",
		URL:                     "https://example.com/hook",
		Method:                  "POST",
		Headers:                 map[string]string{"Content-Type": "application/json"},
		Enabled:                 true,
		Status:                  model.WebhookStatusActive,
		Priority:                model.WebhookPriorityNormal,
		SeverityFilter:          []string{"critical", "warning"},
		MaxPayloadSizeBytes:     1048576,
		PayloadHandlingStrategy: model.PayloadHandlingStrategyTruncate,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	err := repo.Create(ctx, webhook)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, webhook.ID)
}

// TestWebhookRepository_Create_DuplicateName тестирует создание webhook с дубликатом имени
func TestWebhookRepository_Create_DuplicateName(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook1 := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Duplicate Name",
		URL:       "https://example.com/hook1",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	webhook2 := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Duplicate Name", // такое же имя
		URL:       "https://example.com/hook2",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook1)
	assert.NoError(t, err)

	err = repo.Create(ctx, webhook2)
	assert.Error(t, err)
}

// TestWebhookRepository_GetByID тестирует получение webhook по ID
func TestWebhookRepository_GetByID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test Webhook",
		URL:       "https://example.com/hook",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, webhook.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, webhook.ID, found.ID)
	assert.Equal(t, webhook.Name, found.Name)
	assert.Equal(t, webhook.URL, found.URL)
}

// TestWebhookRepository_GetByID_NotFound тестирует поиск несуществующего webhook
func TestWebhookRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)

	found, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, found)
}

// TestWebhookRepository_GetByUserIDAndName тестирует получение webhook по имени
func TestWebhookRepository_GetByUserIDAndName(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Named Webhook",
		URL:       "https://example.com/hook",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)

	found, err := repo.GetByUserIDAndName(ctx, userID, "Named Webhook")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, webhook.Name, found.Name)
}

// TestWebhookRepository_ListByUserID тестирует список webhooks пользователя
func TestWebhookRepository_ListByUserID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	// Создаём несколько webhooks
	for i := 1; i <= 3; i++ {
		webhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      fmt.Sprintf("Webhook %d", i),
			URL:       fmt.Sprintf("https://example.com/hook%d", i),
			Method:    "POST",
			Enabled:   true,
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, webhook)
		require.NoError(t, err)
	}

	webhooks, err := repo.ListByUserID(ctx, userID, 100, 0)
	assert.NoError(t, err)
	assert.Len(t, webhooks, 3)
}

// TestWebhookRepository_Update тестирует обновление webhook
func TestWebhookRepository_Update(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Original Name",
		URL:       "https://example.com/hook",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)

	// Обновляем
	webhook.Name = "Updated Name"
	webhook.URL = "https://example.com/updated"
	webhook.UpdatedAt = time.Now()

	err = repo.Update(ctx, webhook)
	assert.NoError(t, err)

	// Проверяем
	found, err := repo.GetByID(ctx, webhook.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "https://example.com/updated", found.URL)
}

// TestWebhookRepository_Delete тестирует удаление webhook
func TestWebhookRepository_Delete(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "To Delete",
		URL:       "https://example.com/hook",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)

	err = repo.Delete(ctx, webhook.ID)
	assert.NoError(t, err)

	// Проверяем, что удалён
	_, err = repo.GetByID(ctx, webhook.ID)
	assert.Error(t, err)
}

// TestWebhookRepository_ExistsByName тестирует проверку существования по имени
func TestWebhookRepository_ExistsByName(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Unique Name",
		URL:       "https://example.com/hook",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)

	// Существует
	exists, err := repo.ExistsByName(ctx, userID, "Unique Name")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Не существует
	exists, err = repo.ExistsByName(ctx, userID, "Non Existent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestWebhookRepository_CountByUserID тестирует подсчёт webhooks пользователя
func TestWebhookRepository_CountByUserID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	// Создаём несколько webhooks
	for i := 1; i <= 5; i++ {
		webhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      fmt.Sprintf("Webhook %d", i),
			URL:       fmt.Sprintf("https://example.com/hook%d", i),
			Method:    "POST",
			Enabled:   true,
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, webhook)
		require.NoError(t, err)
	}

	count, err := repo.CountByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

// TestWebhookRepository_ListActiveByUserID тестирует список активных webhooks
func TestWebhookRepository_ListActiveByUserID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	// Активный webhook
	webhook1 := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Active Webhook",
		URL:       "https://example.com/hook1",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Неактивный webhook
	webhook2 := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Inactive Webhook",
		URL:       "https://example.com/hook2",
		Method:    "POST",
		Enabled:   false,
		Status:    model.WebhookStatusInactive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook1)
	require.NoError(t, err)

	err = repo.Create(ctx, webhook2)
	require.NoError(t, err)

	webhooks, err := repo.ListActiveByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, webhooks, 1)
	assert.Equal(t, "Active Webhook", webhooks[0].Name)
}

// TestWebhookRepository_UpdateStats тестирует обновление статистики
func TestWebhookRepository_UpdateStats(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Stats Webhook",
		URL:       "https://example.com/hook",
		Method:    "POST",
		Enabled:   true,
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)

	// Обновляем статистику
	stats := &interfaces.WebhookStats{
		TotalSent:         10,
		SuccessfulSent:    9,
		FailedSent:        1,
		AvgResponseTimeMs: 150,
	}

	err = repo.UpdateStats(ctx, webhook.ID, stats)
	assert.NoError(t, err)

	// Проверяем
	found, err := repo.GetByID(ctx, webhook.ID)
	assert.NoError(t, err)
	assert.Equal(t, 10, found.TotalSent)
	assert.Equal(t, 9, found.SuccessfulSent)
	assert.Equal(t, 1, found.FailedSent)
	assert.Equal(t, 150, found.AvgResponseTimeMs)
}

// TestWebhookRepository_ListByUserID_Pagination тестирует пагинацию
func TestWebhookRepository_ListByUserID_Pagination(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewWebhookRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM webhook_integrations WHERE user_id = $1", userID)

	// Создаём 10 webhooks
	for i := 1; i <= 10; i++ {
		webhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      fmt.Sprintf("Webhook %02d", i),
			URL:       fmt.Sprintf("https://example.com/hook%d", i),
			Method:    "POST",
			Enabled:   true,
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, webhook)
		require.NoError(t, err)
	}

	// Первая страница
	page1, err := repo.ListByUserID(ctx, userID, 5, 0)
	assert.NoError(t, err)
	assert.Len(t, page1, 5)

	// Вторая страница
	page2, err := repo.ListByUserID(ctx, userID, 5, 5)
	assert.NoError(t, err)
	assert.Len(t, page2, 5)

	// Проверяем, что это разные записи
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}
