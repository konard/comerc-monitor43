package interfaces

import (
	"context"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

// SubscriptionRepository определяет интерфейс репозитория подписок.
type SubscriptionRepository interface {
	// Create создаёт новую подписку.
	Create(ctx context.Context, sub *model.Subscription) error

	// GetByID возвращает подписку по ID.
	GetByID(ctx context.Context, id string) (*model.Subscription, error)

	// GetActiveByUserID возвращает активную подписку пользователя.
	GetActiveByUserID(ctx context.Context, userID string) (*model.Subscription, error)

	// GetByUserID возвращает подписки пользователя с пагинацией.
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error)

	// Update обновляет подписку.
	Update(ctx context.Context, sub *model.Subscription) error

	// Delete удаляет подписку (soft delete через статус).
	Delete(ctx context.Context, id string) error
}
