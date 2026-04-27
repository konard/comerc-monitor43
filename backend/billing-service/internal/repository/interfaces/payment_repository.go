package interfaces

import (
	"context"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

// PaymentRepository определяет интерфейс репозитория платежей.
type PaymentRepository interface {
	// Create создаёт новый платеж.
	Create(ctx context.Context, payment *model.Payment) error

	// GetByID возвращает платеж по ID.
	GetByID(ctx context.Context, id string) (*model.Payment, error)

	// GetByUserID возвращает платежи пользователя с пагинацией.
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Payment, error)

	// GetByProviderPaymentID возвращает платеж по ID провайдера.
	GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*model.Payment, error)

	// Update обновляет платёж.
	Update(ctx context.Context, payment *model.Payment) error

	// CountByUserID возвращает количество платежей пользователя.
	CountByUserID(ctx context.Context, userID string) (int64, error)
}
