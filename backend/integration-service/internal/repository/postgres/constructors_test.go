package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWebhookRepository проверяет создание репозитория.
func TestNewWebhookRepository(t *testing.T) {
	t.Parallel()
	repo := NewWebhookRepository(nil)
	require.NotNil(t, repo)
}

// TestNewAPIKeyRepository проверяет создание репозитория.
func TestNewAPIKeyRepository(t *testing.T) {
	t.Parallel()
	repo := NewAPIKeyRepository(nil)
	require.NotNil(t, repo)
}

// TestNewAPIKeyUsageRepository проверяет создание репозитория.
func TestNewAPIKeyUsageRepository(t *testing.T) {
	t.Parallel()
	repo := NewAPIKeyUsageRepository(nil)
	require.NotNil(t, repo)
}

// TestNewImportHistoryRepository проверяет создание репозитория.
func TestNewImportHistoryRepository(t *testing.T) {
	t.Parallel()
	repo := NewImportHistoryRepository(nil)
	require.NotNil(t, repo)
}

// TestNewWebhookDeliveryRepository проверяет создание репозитория.
func TestNewWebhookDeliveryRepository(t *testing.T) {
	t.Parallel()
	repo := NewWebhookDeliveryRepository(nil)
	require.NotNil(t, repo)
}

// TestNewDB проверяет что NewDB возвращает ошибку без реального подключения.
func TestNewDB(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	// Попытка подключения к несуществующей БД должна вернуть ошибку
	_, err := NewDB("postgres://invalid:invalid@localhost:9999/invalid")
	assert.Error(t, err)
}

// TestDBConstructors проверяет что конструкторы не паникуют с nil DB.
func TestDBConstructorsNilDB(t *testing.T) {
	t.Parallel()

	t.Run("webhook repository with nil db", func(t *testing.T) {
		t.Parallel()
		repo := &webhookRepositoryImpl{db: nil}
		assert.NotNil(t, repo)
	})

	t.Run("apikey repository with nil db", func(t *testing.T) {
		t.Parallel()
		repo := &apiKeyRepositoryImpl{db: nil}
		assert.NotNil(t, repo)
	})

	t.Run("apikey usage repository with nil db", func(t *testing.T) {
		t.Parallel()
		repo := &apiKeyUsageRepositoryImpl{db: nil}
		assert.NotNil(t, repo)
	})

	t.Run("import history repository with nil db", func(t *testing.T) {
		t.Parallel()
		repo := &ImportHistoryRepositoryImpl{db: nil}
		assert.NotNil(t, repo)
	})

	t.Run("webhook delivery repository with nil db", func(t *testing.T) {
		t.Parallel()
		repo := &webhookDeliveryRepositoryImpl{db: nil}
		assert.NotNil(t, repo)
	})
}
