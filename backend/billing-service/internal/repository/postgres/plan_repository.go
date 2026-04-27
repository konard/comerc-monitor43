package postgres

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/repository/interfaces"
)

// NewPlanRepository создаёт новый PlanRepository.
func NewPlanRepository(db *DB) interfaces.PlanRepository {
	return &planRepository{db: db}
}

type planRepository struct {
	db *DB
}

// GetByID возвращает план по ID.
func (r *planRepository) GetByID(ctx context.Context, id string) (*model.Plan, error) {
	const query = `
		SELECT id, name, description, price_kopeks, billing_period_days,
		       max_monitors, min_check_interval_seconds, max_alerts_per_day,
		       features, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1
	`

	var plan model.Plan
	var featuresJSON []byte

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&plan.ID, &plan.Name, &plan.Description, &plan.PriceKopeks,
		&plan.BillingPeriodDays, &plan.MaxMonitors, &plan.MinCheckIntervalSeconds,
		&plan.MaxAlertsPerDay, &featuresJSON, &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plan by id")
	}

	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal plan features")
	}

	return &plan, nil
}

// GetAll возвращает все планы.
func (r *planRepository) GetAll(ctx context.Context, activeOnly bool) ([]*model.Plan, error) {
	var query string
	var args []any

	if activeOnly {
		query = `
			SELECT id, name, description, price_kopeks, billing_period_days,
			       max_monitors, min_check_interval_seconds, max_alerts_per_day,
			       features, is_active, created_at, updated_at
			FROM subscription_plans
			WHERE is_active = true
			ORDER BY price_kopeks
		`
	} else {
		query = `
			SELECT id, name, description, price_kopeks, billing_period_days,
			       max_monitors, min_check_interval_seconds, max_alerts_per_day,
			       features, is_active, created_at, updated_at
			FROM subscription_plans
			ORDER BY price_kopeks
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plans")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			return
		}
	}()

	var plans []*model.Plan
	for rows.Next() {
		var plan model.Plan
		var featuresJSON []byte

		if err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Description, &plan.PriceKopeks,
			&plan.BillingPeriodDays, &plan.MaxMonitors, &plan.MinCheckIntervalSeconds,
			&plan.MaxAlertsPerDay, &featuresJSON, &plan.IsActive,
			&plan.CreatedAt, &plan.UpdatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan plan row")
		}

		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal plan features")
		}

		plans = append(plans, &plan)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate plans")
	}

	return plans, nil
}

// GetByTier возвращает план по tier.
func (r *planRepository) GetByTier(ctx context.Context, tier string) (*model.Plan, error) {
	const query = `
		SELECT id, name, description, price_kopeks, billing_period_days,
		       max_monitors, min_check_interval_seconds, max_alerts_per_day,
		       features, is_active, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1
	`

	var plan model.Plan
	var featuresJSON []byte

	row := r.db.QueryRowContext(ctx, query, tier)
	err := row.Scan(
		&plan.ID, &plan.Name, &plan.Description, &plan.PriceKopeks,
		&plan.BillingPeriodDays, &plan.MaxMonitors, &plan.MinCheckIntervalSeconds,
		&plan.MaxAlertsPerDay, &featuresJSON, &plan.IsActive,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plan by tier")
	}

	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal plan features")
	}

	return &plan, nil
}
