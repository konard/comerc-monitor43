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

// runAPIKeyMigrations выполняет миграции для таблиц API keys
func runAPIKeyMigrations(db *DB) error {
	migration := `
-- Table: api_keys
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(50) NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]',
    rate_limit INTEGER,
    enabled BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    total_requests INTEGER NOT NULL DEFAULT 0,
    successful_requests INTEGER NOT NULL DEFAULT 0,
    failed_requests INTEGER NOT NULL DEFAULT 0,
    last_used_ip VARCHAR(50),
    most_used_endpoint VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT api_keys_user_id_name_unique UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);
`

	if _, err := db.DB.Exec(migration); err != nil {
		return err
	}
	return nil
}

// setupTestDBForAPIKeys создаёт тестовое подключение и выполняет миграции API keys
func setupTestDBForAPIKeys(t *testing.T) *DB {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	connStr := "host=localhost port=5433 user=postgres dbname=integration_service password=postgres sslmode=disable"
	db, err := NewDB(connStr)
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	// Run migrations
	err = runAPIKeyMigrations(db)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Очистка таблиц перед каждым тестом
	ctx := context.Background()
	_, err = db.DB.ExecContext(ctx, "TRUNCATE TABLE api_keys CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	return db
}

// TestAPIKeyRepository_Create тестирует создание API ключа
func TestAPIKeyRepository_Create(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	desc := "Test API key description"
	apiKey := &model.APIKey{
		ID:                 uuid.New(),
		UserID:             userID,
		Name:               "Test API Key",
		Description:        &desc,
		KeyHash:            "hash123456789",
		KeyPrefix:          "test_prefix",
		Scopes:             []string{"read", "write"},
		RateLimitPerMinute: 1000,
		Status:             model.APIKeyStatusActive,
		ExpiresAt:          nil,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, apiKey.ID)
}

// TestAPIKeyRepository_Create_DuplicateName тестирует создание с дубликатом имени
func TestAPIKeyRepository_Create_DuplicateName(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	key1 := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Duplicate Name",
		KeyHash:   "hash1",
		KeyPrefix: "prefix1",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	key2 := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Duplicate Name",
		KeyHash:   "hash2",
		KeyPrefix: "prefix2",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, key1)
	assert.NoError(t, err)

	err = repo.Create(ctx, key2)
	assert.Error(t, err)
}

// TestAPIKeyRepository_GetByID тестирует получение по ID
func TestAPIKeyRepository_GetByID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test Key",
		KeyHash:   "hash123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, apiKey.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, apiKey.ID, found.ID)
	assert.Equal(t, apiKey.Name, found.Name)
}

// TestAPIKeyRepository_GetByID_NotFound тестирует поиск несуществующего ключа
func TestAPIKeyRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)

	found, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, found)
}

// TestAPIKeyRepository_GetByKeyHash тестирует получение по хешу
func TestAPIKeyRepository_GetByKeyHash(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test Key",
		KeyHash:   "unique_hash_123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	found, err := repo.GetByKeyHash(ctx, "unique_hash_123")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, apiKey.Name, found.Name)
}

// TestAPIKeyRepository_GetByUserIDAndName тестирует получение по userID и name
func TestAPIKeyRepository_GetByUserIDAndName(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Named Key",
		KeyHash:   "hash123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	found, err := repo.GetByUserIDAndName(ctx, userID, "Named Key")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Named Key", found.Name)
}

// TestAPIKeyRepository_ListByUserID тестирует список ключей пользователя
func TestAPIKeyRepository_ListByUserID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	// Создаём несколько ключей
	for i := 1; i <= 3; i++ {
		key := &model.APIKey{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      fmt.Sprintf("Key %d", i),
			KeyHash:   fmt.Sprintf("hash%d", i),
			KeyPrefix: fmt.Sprintf("prefix%d", i),
			Status:    model.APIKeyStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	keys, err := repo.ListByUserID(ctx, userID, 100, 0)
	assert.NoError(t, err)
	assert.Len(t, keys, 3)
}

// TestAPIKeyRepository_Update тестирует обновление ключа
func TestAPIKeyRepository_Update(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Original Name",
		KeyHash:   "hash123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	// Обновляем
	apiKey.Name = "Updated Name"
	updatedDesc := "Updated description"
	apiKey.Description = &updatedDesc
	apiKey.UpdatedAt = time.Now()

	err = repo.Update(ctx, apiKey)
	assert.NoError(t, err)

	// Проверяем
	found, err := repo.GetByID(ctx, apiKey.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "Updated description", *found.Description)
}

// TestAPIKeyRepository_Delete тестирует удаление ключа
func TestAPIKeyRepository_Delete(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "To Delete",
		KeyHash:   "hash123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	err = repo.Delete(ctx, apiKey.ID)
	assert.NoError(t, err)

	// Проверяем, что удалён
	_, err = repo.GetByID(ctx, apiKey.ID)
	assert.Error(t, err)
}

// TestAPIKeyRepository_ExistsByName тестирует проверку существования по имени
func TestAPIKeyRepository_ExistsByName(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Unique Name",
		KeyHash:   "hash123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
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

// TestAPIKeyRepository_CountByUserID тестирует подсчёт ключей пользователя
func TestAPIKeyRepository_CountByUserID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	// Создаём несколько ключей
	for i := 1; i <= 7; i++ {
		key := &model.APIKey{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      fmt.Sprintf("Key %d", i),
			KeyHash:   fmt.Sprintf("hash%d", i),
			KeyPrefix: fmt.Sprintf("prefix%d", i),
			Status:    model.APIKeyStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	count, err := repo.CountByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 7, count)
}

// TestAPIKeyRepository_UpdateUsage тестирует обновление статистики использования
func TestAPIKeyRepository_UpdateUsage(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	apiKey := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Usage Key",
		KeyHash:   "hash123",
		KeyPrefix: "test_pre",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	// Обновляем статистику
	now := time.Now().Unix()
	ip := "192.168.1.1"
	endpoint := "/api/v1/test"

	stats := &interfaces.APIKeyUsageStats{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		LastUsedAt:         &now,
		LastUsedIP:         &ip,
		MostUsedEndpoint:   &endpoint,
	}

	err = repo.UpdateUsage(ctx, apiKey.ID, stats)
	assert.NoError(t, err)

	// Проверяем
	found, err := repo.GetByID(ctx, apiKey.ID)
	assert.NoError(t, err)
	assert.Equal(t, 100, found.TotalRequests)
	assert.Equal(t, 95, found.SuccessfulRequests)
	assert.Equal(t, 5, found.FailedRequests)
	assert.NotNil(t, found.LastUsedAt)
}

// TestAPIKeyRepository_ListByUserID_Pagination тестирует пагинацию
func TestAPIKeyRepository_ListByUserID_Pagination(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPIKeys(t)
	closeTestDB(t, db)

	ctx := context.Background()
	repo := NewAPIKeyRepository(db)
	userID := uuid.New()

	// Очистка после теста
	cleanupExecContext(ctx, t, db, "DELETE FROM api_keys WHERE user_id = $1", userID)

	// Создаём 15 ключей
	for i := 1; i <= 15; i++ {
		key := &model.APIKey{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      fmt.Sprintf("Key %02d", i),
			KeyHash:   fmt.Sprintf("hash%02d", i),
			KeyPrefix: fmt.Sprintf("pre%02d", i),
			Status:    model.APIKeyStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, key)
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

	// Третья страница
	page3, err := repo.ListByUserID(ctx, userID, 5, 10)
	assert.NoError(t, err)
	assert.Len(t, page3, 5)

	// Проверяем, что это разные записи
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
	assert.NotEqual(t, page2[0].ID, page3[0].ID)
}
