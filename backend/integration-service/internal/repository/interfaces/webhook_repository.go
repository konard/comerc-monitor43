package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// WebhookRepository определяет интерфейс для работы с webhook интеграциями.
type WebhookRepository interface {
	// Create создаёт новую webhook интеграцию.
	Create(ctx context.Context, webhook *model.WebhookIntegration) error

	// GetByID возвращает webhook по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.WebhookIntegration, error)

	// GetByUserIDAndName возвращает webhook по userID и name.
	GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*model.WebhookIntegration, error)

	// ListByUserID возвращает список webhooks пользователя.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.WebhookIntegration, error)

	// ListActiveByUserID возвращает активные webhooks пользователя.
	ListActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*model.WebhookIntegration, error)

	// Update обновляет webhook.
	Update(ctx context.Context, webhook *model.WebhookIntegration) error

	// UpdateStats обновляет статистику webhook.
	UpdateStats(ctx context.Context, id uuid.UUID, stats *WebhookStats) error

	// Delete удаляет webhook.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByUserID возвращает количество webhooks пользователя.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// ExistsByName проверяет существование webhook с именем.
	ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error)
}

// WebhookDeliveryRepository определяет интерфейс для работы с попытками доставки webhook.
type WebhookDeliveryRepository interface {
	// Create создаёт запись о попытке доставки.
	Create(ctx context.Context, attempt *model.WebhookDeliveryAttempt) error

	// GetByID возвращает попытку доставки по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.WebhookDeliveryAttempt, error)

	// ListByWebhookID возвращает попытки доставки для webhook.
	ListByWebhookID(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*model.WebhookDeliveryAttempt, error)

	// ListPendingForRetry возвращает попытки, которые нужно retry.
	ListPendingForRetry(ctx context.Context, limit int) ([]*model.WebhookDeliveryAttempt, error)

	// Update обновляет попытку доставки.
	Update(ctx context.Context, attempt *model.WebhookDeliveryAttempt) error

	// DeleteOldAttempts удаляет старые попытки доставки.
	DeleteOldAttempts(ctx context.Context, olderThanDays int) (int64, error)
}

// WebhookStats статистика для обновления.
type WebhookStats struct {
	TotalSent         int
	SuccessfulSent    int
	FailedSent        int
	AvgResponseTimeMs int
}

var (
	// ErrWebhookNotFound webhook не найден.
	ErrWebhookNotFound = model.ErrWebhookNotFound
)
