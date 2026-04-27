package interfaces

import (
	"context"

	"github.com/raul/monitor/backend/billing-service/internal/model"
)

// PlanRepository определяет интерфейс репозитория тарифных планов.
type PlanRepository interface {
	// GetByID возвращает план по ID.
	GetByID(ctx context.Context, id string) (*model.Plan, error)

	// GetAll возвращает все планы.
	GetAll(ctx context.Context, activeOnly bool) ([]*model.Plan, error)

	// GetByTier возвращает план по tier (TIER_FREE, TIER_STARTER, etc).
	GetByTier(ctx context.Context, tier string) (*model.Plan, error)
}
