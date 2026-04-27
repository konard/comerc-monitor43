package postgres

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/repository/interfaces"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// NewPaymentRepository создаёт новый PaymentRepository.
func NewPaymentRepository(db *DB) interfaces.PaymentRepository {
	return &paymentRepository{db: db}
}

type paymentRepository struct {
	db *DB
}

// Create создаёт новый платеж.
func (r *paymentRepository) Create(ctx context.Context, payment *model.Payment) error {
	const query = `
		INSERT INTO payments (id, user_id, subscription_id, provider, provider_payment_id,
		                     status, amount_kopeks, currency, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := json.Marshal(payment.Metadata)
	if err != nil {
		return errors.Wrap(err, "failed to marshal metadata")
	}

	_, err = r.db.ExecContext(ctx, query,
		payment.ID, payment.UserID, payment.SubscriptionID, payment.Provider,
		payment.ProviderPaymentID, payment.Status, payment.AmountKopeks,
		payment.Currency, metadataJSON, payment.CreatedAt, payment.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create payment")
	}

	return nil
}

// GetByID возвращает платеж по ID.
func (r *paymentRepository) GetByID(ctx context.Context, id string) (*model.Payment, error) {
	const query = `
		SELECT id, user_id, subscription_id, provider, provider_payment_id,
		       status, amount_kopeks, currency, metadata, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	return r.scanOnePayment(ctx, query, id)
}

// GetByUserID возвращает платежи пользователя с пагинацией.
func (r *paymentRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Payment, error) {
	const query = `
		SELECT id, user_id, subscription_id, provider, provider_payment_id,
		       status, amount_kopeks, currency, metadata, created_at, updated_at
		FROM payments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get payments by user id")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			return
		}
	}()

	var payments []*model.Payment
	for rows.Next() {
		payment, err := r.scanPayment(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan payment row")
		}
		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate payments")
	}

	return payments, nil
}

// GetByProviderPaymentID возвращает платеж по ID провайдера.
func (r *paymentRepository) GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*model.Payment, error) {
	const query = `
		SELECT id, user_id, subscription_id, provider, provider_payment_id,
		       status, amount_kopeks, currency, metadata, created_at, updated_at
		FROM payments
		WHERE provider_payment_id = $1
	`

	return r.scanOnePayment(ctx, query, providerPaymentID)
}

// Update обновляет платёж.
func (r *paymentRepository) Update(ctx context.Context, payment *model.Payment) error {
	const query = `
		UPDATE payments
		SET status = $2, metadata = $3, updated_at = $4
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(payment.Metadata)
	if err != nil {
		return errors.Wrap(err, "failed to marshal metadata")
	}

	result, err := r.db.ExecContext(ctx, query,
		payment.ID, payment.Status, metadataJSON, payment.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update payment")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return billingerrors.NotFound("payment", payment.ID)
	}

	return nil
}

// CountByUserID возвращает количество платежей пользователя.
func (r *paymentRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	const query = `SELECT COUNT(*) FROM payments WHERE user_id = $1`

	var count int64
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count payments")
	}

	return count, nil
}

// scanOnePayment сканирует один платёж из БД.
func (r *paymentRepository) scanOnePayment(ctx context.Context, query string, args ...any) (*model.Payment, error) {
	row := r.db.QueryRowxContext(ctx, query, args...)
	payment, err := r.scanPayment(row)
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// scanPayment сканирует платёж из row.
func (r *paymentRepository) scanPayment(row interface {
	Scan(dest ...any) error
}) (*model.Payment, error) {
	var payment model.Payment
	var metadataJSON []byte

	err := row.Scan(
		&payment.ID, &payment.UserID, &payment.SubscriptionID, &payment.Provider,
		&payment.ProviderPaymentID, &payment.Status, &payment.AmountKopeks,
		&payment.Currency, &metadataJSON, &payment.CreatedAt, &payment.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan payment")
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &payment.Metadata); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal metadata")
		}
	}

	if payment.Metadata == nil {
		payment.Metadata = make(map[string]any)
	}

	return &payment, nil
}
