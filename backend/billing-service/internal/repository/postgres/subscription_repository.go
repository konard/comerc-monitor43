package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/repository/interfaces"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// NewSubscriptionRepository создаёт новый SubscriptionRepository.
func NewSubscriptionRepository(db *DB) interfaces.SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

type subscriptionRepository struct {
	db *DB
}

// Create создаёт новую подписку.
func (r *subscriptionRepository) Create(ctx context.Context, sub *model.Subscription) error {
	const query = `
		INSERT INTO subscriptions (id, user_id, plan_id, status, started_at, expires_at,
		                           canceled_at, auto_renew, provider_payment_id, metadata,
		                           created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	metadataJSON, err := json.Marshal(sub.Metadata)
	if err != nil {
		return errors.Wrap(err, "failed to marshal metadata")
	}

	_, err = r.db.ExecContext(ctx, query,
		sub.ID, sub.UserID, sub.PlanID, sub.Status, sub.StartedAt, sub.ExpiresAt,
		sub.CanceledAt, sub.AutoRenew, sub.ProviderPaymentID, metadataJSON,
		sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create subscription")
	}

	return nil
}

// GetByID возвращает подписку по ID.
func (r *subscriptionRepository) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	const query = `
		SELECT id, user_id, plan_id, status, started_at, expires_at,
		       canceled_at, auto_renew, provider_payment_id, metadata,
		       created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	return r.scanOneSubscription(ctx, query, id)
}

// GetActiveByUserID возвращает активную подписку пользователя.
func (r *subscriptionRepository) GetActiveByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	const query = `
		SELECT id, user_id, plan_id, status, started_at, expires_at,
		       canceled_at, auto_renew, provider_payment_id, metadata,
		       created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at DESC
		LIMIT 1
	`

	sub, err := r.scanOneSubscription(ctx, query, userID)
	if err != nil {
		return nil, billingerrors.NotFound("subscription", userID)
	}

	return sub, nil
}

// GetByUserID возвращает подписки пользователя с пагинацией.
func (r *subscriptionRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	const query = `
		SELECT id, user_id, plan_id, status, started_at, expires_at,
		       canceled_at, auto_renew, provider_payment_id, metadata,
		       created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subscriptions by user id")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			return
		}
	}()

	var subs []*model.Subscription
	for rows.Next() {
		sub, err := r.scanSubscription(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan subscription row")
		}
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate subscriptions")
	}

	return subs, nil
}

// Update обновляет подписку.
func (r *subscriptionRepository) Update(ctx context.Context, sub *model.Subscription) error {
	const query = `
		UPDATE subscriptions
		SET status = $2, expires_at = $3, canceled_at = $4, auto_renew = $5,
		    provider_payment_id = $6, metadata = $7, updated_at = $8
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(sub.Metadata)
	if err != nil {
		return errors.Wrap(err, "failed to marshal metadata")
	}

	result, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.Status, sub.ExpiresAt, sub.CanceledAt, sub.AutoRenew,
		sub.ProviderPaymentID, metadataJSON, sub.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update subscription")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return billingerrors.NotFound("subscription", sub.ID)
	}

	return nil
}

// Delete удаляет подписку (soft delete через статус).
func (r *subscriptionRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	const query = `UPDATE subscriptions SET status = 'CANCELED', canceled_at = $2, updated_at = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return errors.Wrap(err, "failed to delete subscription")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return billingerrors.NotFound("subscription", id)
	}

	return nil
}

// scanOneSubscription сканирует одну подписку из БД.
func (r *subscriptionRepository) scanOneSubscription(ctx context.Context, query string, args ...any) (*model.Subscription, error) {
	row := r.db.QueryRowxContext(ctx, query, args...)
	sub, err := r.scanSubscription(row)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// scanSubscription сканирует подписку из row.
func (r *subscriptionRepository) scanSubscription(row interface {
	Scan(dest ...any) error
}) (*model.Subscription, error) {
	var sub model.Subscription
	var metadataJSON []byte

	err := row.Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.StartedAt, &sub.ExpiresAt,
		&sub.CanceledAt, &sub.AutoRenew, &sub.ProviderPaymentID, &metadataJSON,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan subscription")
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal metadata")
		}
	}

	if sub.Metadata == nil {
		sub.Metadata = make(map[string]any)
	}

	return &sub, nil
}
