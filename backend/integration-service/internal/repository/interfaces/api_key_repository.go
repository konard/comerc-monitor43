package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// APIKeyRepository определяет интерфейс для работы с API ключами.
type APIKeyRepository interface {
	// Create создаёт новый API ключ.
	Create(ctx context.Context, key *model.APIKey) error

	// GetByID возвращает API ключ по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.APIKey, error)

	// GetByKeyHash возвращает API ключ по хешу ключа.
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)

	// GetByUserIDAndName возвращает API ключ по userID и name.
	GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*model.APIKey, error)

	// ListByUserID возвращает список API ключей пользователя.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.APIKey, error)

	// Update обновляет API ключ.
	Update(ctx context.Context, key *model.APIKey) error

	// UpdateUsage обновляет статистику использования API ключа.
	UpdateUsage(ctx context.Context, id uuid.UUID, stats *APIKeyUsageStats) error

	// Delete удаляет API ключ.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByUserID возвращает количество API ключей пользователя.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// ExistsByName проверяет существование API ключа с именем.
	ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error)
}

// APIKeyUsageRepository определяет интерфейс для работы с логами использования API ключей.
type APIKeyUsageRepository interface {
	// Create создаёт запись о логе использования.
	Create(ctx context.Context, log *model.APIKeyUsageLog) error

	// GetByID возвращает лог по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.APIKeyUsageLog, error)

	// ListByAPIKeyID возвращает логи для API ключа.
	ListByAPIKeyID(ctx context.Context, apiKeyID uuid.UUID, limit, offset int) ([]*model.APIKeyUsageLog, error)

	// ListByAPIKeyIDAndPeriod возвращает логи за период.
	ListByAPIKeyIDAndPeriod(ctx context.Context, apiKeyID uuid.UUID, from, to int64, limit, offset int) ([]*model.APIKeyUsageLog, error)

	// DeleteOldLogs удаляет старые логи.
	DeleteOldLogs(ctx context.Context, olderThanDays int) (int64, error)
}

// APIKeyUsageStats статистика для обновления.
type APIKeyUsageStats struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	LastUsedAt         *int64
	LastUsedIP         *string
	MostUsedEndpoint   *string
}

var (
	// ErrAPIKeyNotFound API ключ не найден.
	ErrAPIKeyNotFound = model.ErrAPIKeyNotFound
)
