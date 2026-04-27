package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

type webhookDeliveryRepositoryImpl struct {
	db *DB
}

// NewWebhookDeliveryRepository создаёт новый WebhookDeliveryRepository.
func NewWebhookDeliveryRepository(db *DB) interfaces.WebhookDeliveryRepository {
	return &webhookDeliveryRepositoryImpl{db: db}
}

// Create создаёт запись о попытке доставки.
func (r *webhookDeliveryRepositoryImpl) Create(ctx context.Context, attempt *model.WebhookDeliveryAttempt) error {
	query := `
		INSERT INTO webhook_delivery_attempts (
			id, webhook_id, alert_id, status, retry_count, next_retry_at,
			http_status_code, response_time_ms, error_message, error_code,
			payload_size_bytes, payload_truncated, signature_algorithm,
			created_at, sent_at
		) VALUES (
			:id, :webhook_id, :alert_id, :status, :retry_count, :next_retry_at,
			:http_status_code, :response_time_ms, :error_message, :error_code,
			:payload_size_bytes, :payload_truncated, :signature_algorithm,
			:created_at, :sent_at
		)
	`

	args := map[string]any{
		"id":                  attempt.ID,
		"webhook_id":          attempt.WebhookID,
		"alert_id":            attempt.AlertID,
		"status":              attempt.Status,
		"retry_count":         attempt.RetryCount,
		"next_retry_at":       attempt.NextRetryAt,
		"http_status_code":    attempt.HTTPStatusCode,
		"response_time_ms":    attempt.ResponseTimeMs,
		"error_message":       attempt.ErrorMessage,
		"error_code":          attempt.ErrorCode,
		"payload_size_bytes":  attempt.PayloadSizeBytes,
		"payload_truncated":   attempt.PayloadTruncated,
		"signature_algorithm": attempt.SignatureAlgorithm,
		"created_at":          attempt.CreatedAt,
		"sent_at":             attempt.SentAt,
	}

	_, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to create webhook delivery attempt")
	}

	return nil
}

// GetByID возвращает попытку доставки по ID.
func (r *webhookDeliveryRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (_ *model.WebhookDeliveryAttempt, err error) {
	query := `
		SELECT
			id, webhook_id, alert_id, status, retry_count, next_retry_at,
			http_status_code, response_time_ms, error_message, error_code,
			payload_size_bytes, payload_truncated, signature_algorithm,
			created_at, sent_at
		FROM webhook_delivery_attempts
		WHERE id = :id
	`

	args := map[string]any{"id": id}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook delivery attempt")
	}
	defer closeResource(row, &err, "failed to close webhook delivery rows")

	if !row.Next() {
		return nil, errors.New("webhook delivery attempt not found")
	}

	var attempt model.WebhookDeliveryAttempt
	if err := row.StructScan(&attempt); err != nil {
		return nil, errors.Wrap(err, "failed to scan webhook delivery attempt")
	}

	return &attempt, nil
}

// ListByWebhookID возвращает попытки доставки для webhook.
func (r *webhookDeliveryRepositoryImpl) ListByWebhookID(ctx context.Context, webhookID uuid.UUID, limit, offset int) (_ []*model.WebhookDeliveryAttempt, err error) {
	query := `
		SELECT
			id, webhook_id, alert_id, status, retry_count, next_retry_at,
			http_status_code, response_time_ms, error_message, error_code,
			payload_size_bytes, payload_truncated, signature_algorithm,
			created_at, sent_at
		FROM webhook_delivery_attempts
		WHERE webhook_id = :webhook_id
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`

	args := map[string]any{
		"webhook_id": webhookID,
		"limit":      limit,
		"offset":     offset,
	}

	rows, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list webhook delivery attempts")
	}
	defer closeResource(rows, &err, "failed to close webhook delivery rows")

	var attempts []*model.WebhookDeliveryAttempt
	for rows.Next() {
		var attempt model.WebhookDeliveryAttempt
		if err := rows.StructScan(&attempt); err != nil {
			return nil, errors.Wrap(err, "failed to scan webhook delivery attempt")
		}
		attempts = append(attempts, &attempt)
	}

	return attempts, nil
}

// ListPendingForRetry возвращает попытки, которые нужно retry.
func (r *webhookDeliveryRepositoryImpl) ListPendingForRetry(ctx context.Context, limit int) (_ []*model.WebhookDeliveryAttempt, err error) {
	query := `
		SELECT
			id, webhook_id, alert_id, status, retry_count, next_retry_at,
			http_status_code, response_time_ms, error_message, error_code,
			payload_size_bytes, payload_truncated, signature_algorithm,
			created_at, sent_at
		FROM webhook_delivery_attempts
		WHERE status = 'retry_scheduled' AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC
		LIMIT :limit
	`

	args := map[string]any{"limit": limit}

	rows, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pending retries")
	}
	defer closeResource(rows, &err, "failed to close pending retry rows")

	var attempts []*model.WebhookDeliveryAttempt
	for rows.Next() {
		var attempt model.WebhookDeliveryAttempt
		if err := rows.StructScan(&attempt); err != nil {
			return nil, errors.Wrap(err, "failed to scan webhook delivery attempt")
		}
		attempts = append(attempts, &attempt)
	}

	return attempts, nil
}

// Update обновляет попытку доставки.
func (r *webhookDeliveryRepositoryImpl) Update(ctx context.Context, attempt *model.WebhookDeliveryAttempt) error {
	query := `
		UPDATE webhook_delivery_attempts SET
			status = :status,
			retry_count = :retry_count,
			next_retry_at = :next_retry_at,
			http_status_code = :http_status_code,
			response_time_ms = :response_time_ms,
			error_message = :error_message,
			error_code = :error_code,
			sent_at = :sent_at
		WHERE id = :id
	`

	args := map[string]any{
		"id":               attempt.ID,
		"status":           attempt.Status,
		"retry_count":      attempt.RetryCount,
		"next_retry_at":    attempt.NextRetryAt,
		"http_status_code": attempt.HTTPStatusCode,
		"response_time_ms": attempt.ResponseTimeMs,
		"error_message":    attempt.ErrorMessage,
		"error_code":       attempt.ErrorCode,
		"sent_at":          attempt.SentAt,
	}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to update webhook delivery attempt")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("webhook delivery attempt not found")
	}

	return nil
}

// DeleteOldAttempts удаляет старые попытки доставки.
func (r *webhookDeliveryRepositoryImpl) DeleteOldAttempts(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
		DELETE FROM webhook_delivery_attempts
		WHERE created_at < NOW() - INTERVAL '1 day' * :days
	`

	args := map[string]any{"days": olderThanDays}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete old webhook delivery attempts")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get rows affected")
	}

	return rowsAffected, nil
}
