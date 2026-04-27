package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

type webhookRepositoryImpl struct {
	db *DB
}

type dbWebhookIntegration struct {
	ID                      uuid.UUID                     `db:"id"`
	UserID                  uuid.UUID                     `db:"user_id"`
	Name                    string                        `db:"name"`
	URL                     string                        `db:"url"`
	Method                  string                        `db:"method"`
	Headers                 string                        `db:"headers"`
	SecretKey               *string                       `db:"secret_key"`
	Enabled                 bool                          `db:"enabled"`
	Status                  model.WebhookStatus           `db:"status"`
	Priority                model.WebhookPriority         `db:"priority"`
	SeverityFilter          string                        `db:"severity_filter"`
	MaxPayloadSizeBytes     int                           `db:"max_payload_size_bytes"`
	PayloadHandlingStrategy model.PayloadHandlingStrategy `db:"payload_handling_strategy"`
	TotalSent               int                           `db:"total_sent"`
	SuccessfulSent          int                           `db:"successful_sent"`
	FailedSent              int                           `db:"failed_sent"`
	AvgResponseTimeMs       *int                          `db:"avg_response_time_ms"`
	LastSentAt              *time.Time                    `db:"last_sent_at"`
	LastSuccessAt           *time.Time                    `db:"last_success_at"`
	LastFailureAt           *time.Time                    `db:"last_failure_at"`
	FailureCount            int                           `db:"failure_count"`
	ConsecutiveFailures     int                           `db:"consecutive_failures"`
	CreatedAt               time.Time                     `db:"created_at"`
	UpdatedAt               time.Time                     `db:"updated_at"`
}

func (w *dbWebhookIntegration) toModel() (*model.WebhookIntegration, error) {
	headers := map[string]string{}
	if w.Headers != "" {
		if err := json.Unmarshal([]byte(w.Headers), &headers); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal webhook headers")
		}
	}

	severityFilter := []string{}
	if w.SeverityFilter != "" {
		if err := json.Unmarshal([]byte(w.SeverityFilter), &severityFilter); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal webhook severity filter")
		}
	}

	return &model.WebhookIntegration{
		ID:                      w.ID,
		UserID:                  w.UserID,
		Name:                    w.Name,
		URL:                     w.URL,
		Method:                  w.Method,
		Headers:                 headers,
		SecretKey:               w.SecretKey,
		Enabled:                 w.Enabled,
		Status:                  w.Status,
		Priority:                w.Priority,
		SeverityFilter:          severityFilter,
		MaxPayloadSizeBytes:     w.MaxPayloadSizeBytes,
		PayloadHandlingStrategy: w.PayloadHandlingStrategy,
		TotalSent:               w.TotalSent,
		SuccessfulSent:          w.SuccessfulSent,
		FailedSent:              w.FailedSent,
		AvgResponseTimeMs:       w.AvgResponseTimeMs,
		LastSentAt:              w.LastSentAt,
		LastSuccessAt:           w.LastSuccessAt,
		LastFailureAt:           w.LastFailureAt,
		FailureCount:            w.FailureCount,
		ConsecutiveFailures:     w.ConsecutiveFailures,
		CreatedAt:               w.CreatedAt,
		UpdatedAt:               w.UpdatedAt,
	}, nil
}

func marshalWebhookHeaders(headers map[string]string) (string, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	data, err := json.Marshal(headers)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal webhook headers")
	}
	return string(data), nil
}

func marshalWebhookSeverityFilter(severityFilter []string) (string, error) {
	if severityFilter == nil {
		severityFilter = []string{}
	}
	data, err := json.Marshal(severityFilter)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal webhook severity filter")
	}
	return string(data), nil
}

// NewWebhookRepository создаёт новый WebhookRepository.
func NewWebhookRepository(db *DB) interfaces.WebhookRepository {
	return &webhookRepositoryImpl{db: db}
}

// Create создаёт новую webhook интеграцию.
func (r *webhookRepositoryImpl) Create(ctx context.Context, webhook *model.WebhookIntegration) error {
	query := `
		INSERT INTO webhook_integrations (
			id, user_id, name, url, method, headers, secret_key,
			enabled, status, priority, severity_filter,
			max_payload_size_bytes, payload_handling_strategy,
			total_sent, successful_sent, failed_sent, avg_response_time_ms,
			last_sent_at, last_success_at, last_failure_at,
			failure_count, consecutive_failures,
			created_at, updated_at
		) VALUES (
			:id, :user_id, :name, :url, :method, :headers, :secret_key,
			:enabled, :status, :priority, :severity_filter,
			:max_payload_size_bytes, :payload_handling_strategy,
			:total_sent, :successful_sent, :failed_sent, :avg_response_time_ms,
			:last_sent_at, :last_success_at, :last_failure_at,
			:failure_count, :consecutive_failures,
			:created_at, :updated_at
		)
	`
	headers, err := marshalWebhookHeaders(webhook.Headers)
	if err != nil {
		return err
	}
	severityFilter, err := marshalWebhookSeverityFilter(webhook.SeverityFilter)
	if err != nil {
		return err
	}

	args := map[string]any{
		"id":                        webhook.ID,
		"user_id":                   webhook.UserID,
		"name":                      webhook.Name,
		"url":                       webhook.URL,
		"method":                    webhook.Method,
		"headers":                   headers,
		"secret_key":                webhook.SecretKey,
		"enabled":                   webhook.Enabled,
		"status":                    webhook.Status,
		"priority":                  webhook.Priority,
		"severity_filter":           severityFilter,
		"max_payload_size_bytes":    webhook.MaxPayloadSizeBytes,
		"payload_handling_strategy": webhook.PayloadHandlingStrategy,
		"total_sent":                webhook.TotalSent,
		"successful_sent":           webhook.SuccessfulSent,
		"failed_sent":               webhook.FailedSent,
		"avg_response_time_ms":      webhook.AvgResponseTimeMs,
		"last_sent_at":              webhook.LastSentAt,
		"last_success_at":           webhook.LastSuccessAt,
		"last_failure_at":           webhook.LastFailureAt,
		"failure_count":             webhook.FailureCount,
		"consecutive_failures":      webhook.ConsecutiveFailures,
		"created_at":                webhook.CreatedAt,
		"updated_at":                webhook.UpdatedAt,
	}

	_, err = r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to create webhook integration")
	}

	return nil
}

// GetByID возвращает webhook по ID.
func (r *webhookRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (_ *model.WebhookIntegration, err error) {
	query := `
		SELECT
			id, user_id, name, url, method, headers, secret_key,
			enabled, status, priority, severity_filter,
			max_payload_size_bytes, payload_handling_strategy,
			total_sent, successful_sent, failed_sent, avg_response_time_ms,
			last_sent_at, last_success_at, last_failure_at,
			failure_count, consecutive_failures,
			created_at, updated_at
		FROM webhook_integrations
		WHERE id = :id
	`

	args := map[string]any{"id": id}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook integration")
	}
	defer closeResource(row, &err, "failed to close webhook integration rows")

	if !row.Next() {
		return nil, interfaces.ErrWebhookNotFound
	}

	var dbWebhook dbWebhookIntegration
	if err := row.StructScan(&dbWebhook); err != nil {
		return nil, errors.Wrap(err, "failed to scan webhook integration")
	}

	return dbWebhook.toModel()
}

// GetByUserIDAndName возвращает webhook по userID и name.
func (r *webhookRepositoryImpl) GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (_ *model.WebhookIntegration, err error) {
	query := `
		SELECT
			id, user_id, name, url, method, headers, secret_key,
			enabled, status, priority, severity_filter,
			max_payload_size_bytes, payload_handling_strategy,
			total_sent, successful_sent, failed_sent, avg_response_time_ms,
			last_sent_at, last_success_at, last_failure_at,
			failure_count, consecutive_failures,
			created_at, updated_at
		FROM webhook_integrations
		WHERE user_id = :user_id AND name = :name
	`

	args := map[string]any{
		"user_id": userID,
		"name":    name,
	}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook integration")
	}
	defer closeResource(row, &err, "failed to close webhook integration rows")

	if !row.Next() {
		return nil, interfaces.ErrWebhookNotFound
	}

	var dbWebhook dbWebhookIntegration
	if err := row.StructScan(&dbWebhook); err != nil {
		return nil, errors.Wrap(err, "failed to scan webhook integration")
	}

	return dbWebhook.toModel()
}

// ListByUserID возвращает список webhooks пользователя.
func (r *webhookRepositoryImpl) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) (_ []*model.WebhookIntegration, err error) {
	query := `
		SELECT
			id, user_id, name, url, method, headers, secret_key,
			enabled, status, priority, severity_filter,
			max_payload_size_bytes, payload_handling_strategy,
			total_sent, successful_sent, failed_sent, avg_response_time_ms,
			last_sent_at, last_success_at, last_failure_at,
			failure_count, consecutive_failures,
			created_at, updated_at
		FROM webhook_integrations
		WHERE user_id = :user_id
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`

	args := map[string]any{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	}

	rows, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list webhook integrations")
	}
	defer closeResource(rows, &err, "failed to close webhook integration rows")

	var webhooks []*model.WebhookIntegration
	for rows.Next() {
		var dbWebhook dbWebhookIntegration
		if err := rows.StructScan(&dbWebhook); err != nil {
			return nil, errors.Wrap(err, "failed to scan webhook integration")
		}
		webhook, err := dbWebhook.toModel()
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// ListActiveByUserID возвращает активные webhooks пользователя.
func (r *webhookRepositoryImpl) ListActiveByUserID(ctx context.Context, userID uuid.UUID) (_ []*model.WebhookIntegration, err error) {
	query := `
		SELECT
			id, user_id, name, url, method, headers, secret_key,
			enabled, status, priority, severity_filter,
			max_payload_size_bytes, payload_handling_strategy,
			total_sent, successful_sent, failed_sent, avg_response_time_ms,
			last_sent_at, last_success_at, last_failure_at,
			failure_count, consecutive_failures,
			created_at, updated_at
		FROM webhook_integrations
		WHERE user_id = :user_id AND enabled = true AND status = 'active'
		ORDER BY priority DESC, created_at ASC
	`

	args := map[string]any{"user_id": userID}

	rows, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list active webhook integrations")
	}
	defer closeResource(rows, &err, "failed to close active webhook rows")

	var webhooks []*model.WebhookIntegration
	for rows.Next() {
		var dbWebhook dbWebhookIntegration
		if err := rows.StructScan(&dbWebhook); err != nil {
			return nil, errors.Wrap(err, "failed to scan webhook integration")
		}
		webhook, err := dbWebhook.toModel()
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// Update обновляет webhook.
func (r *webhookRepositoryImpl) Update(ctx context.Context, webhook *model.WebhookIntegration) error {
	query := `
		UPDATE webhook_integrations SET
			name = :name,
			url = :url,
			method = :method,
			headers = :headers,
			secret_key = :secret_key,
			enabled = :enabled,
			status = :status,
			priority = :priority,
			severity_filter = :severity_filter,
			max_payload_size_bytes = :max_payload_size_bytes,
			payload_handling_strategy = :payload_handling_strategy,
			total_sent = :total_sent,
			successful_sent = :successful_sent,
			failed_sent = :failed_sent,
			avg_response_time_ms = :avg_response_time_ms,
			last_sent_at = :last_sent_at,
			last_success_at = :last_success_at,
			last_failure_at = :last_failure_at,
			failure_count = :failure_count,
			consecutive_failures = :consecutive_failures,
			updated_at = :updated_at
		WHERE id = :id
	`
	headers, err := marshalWebhookHeaders(webhook.Headers)
	if err != nil {
		return err
	}
	severityFilter, err := marshalWebhookSeverityFilter(webhook.SeverityFilter)
	if err != nil {
		return err
	}

	args := map[string]any{
		"id":                        webhook.ID,
		"name":                      webhook.Name,
		"url":                       webhook.URL,
		"method":                    webhook.Method,
		"headers":                   headers,
		"secret_key":                webhook.SecretKey,
		"enabled":                   webhook.Enabled,
		"status":                    webhook.Status,
		"priority":                  webhook.Priority,
		"severity_filter":           severityFilter,
		"max_payload_size_bytes":    webhook.MaxPayloadSizeBytes,
		"payload_handling_strategy": webhook.PayloadHandlingStrategy,
		"total_sent":                webhook.TotalSent,
		"successful_sent":           webhook.SuccessfulSent,
		"failed_sent":               webhook.FailedSent,
		"avg_response_time_ms":      webhook.AvgResponseTimeMs,
		"last_sent_at":              webhook.LastSentAt,
		"last_success_at":           webhook.LastSuccessAt,
		"last_failure_at":           webhook.LastFailureAt,
		"failure_count":             webhook.FailureCount,
		"consecutive_failures":      webhook.ConsecutiveFailures,
		"updated_at":                webhook.UpdatedAt,
	}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to update webhook integration")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return interfaces.ErrWebhookNotFound
	}

	return nil
}

// UpdateStats обновляет статистику webhook.
func (r *webhookRepositoryImpl) UpdateStats(ctx context.Context, id uuid.UUID, stats *interfaces.WebhookStats) error {
	query := `
		UPDATE webhook_integrations SET
			total_sent = :total_sent,
			successful_sent = :successful_sent,
			failed_sent = :failed_sent,
			avg_response_time_ms = :avg_response_time_ms,
			updated_at = NOW()
		WHERE id = :id
	`

	args := map[string]any{
		"id":                   id,
		"total_sent":           stats.TotalSent,
		"successful_sent":      stats.SuccessfulSent,
		"failed_sent":          stats.FailedSent,
		"avg_response_time_ms": stats.AvgResponseTimeMs,
	}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to update webhook stats")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return interfaces.ErrWebhookNotFound
	}

	return nil
}

// Delete удаляет webhook.
func (r *webhookRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhook_integrations WHERE id = :id`

	args := map[string]any{"id": id}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to delete webhook integration")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return interfaces.ErrWebhookNotFound
	}

	return nil
}

// CountByUserID возвращает количество webhooks пользователя.
func (r *webhookRepositoryImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM webhook_integrations WHERE user_id = :user_id`

	args := map[string]any{"user_id": userID}

	var count int
	boundQuery, boundArgs, err := r.db.DB.BindNamed(query, args)
	if err != nil {
		return 0, errors.Wrap(err, "failed to bind count webhook integrations query")
	}
	err = r.db.DB.GetContext(ctx, &count, boundQuery, boundArgs...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count webhook integrations")
	}

	return count, nil
}

// ExistsByName проверяет существование webhook с именем.
func (r *webhookRepositoryImpl) ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM webhook_integrations
			WHERE user_id = :user_id AND name = :name
		)
	`

	args := map[string]any{
		"user_id": userID,
		"name":    name,
	}

	var exists bool
	boundQuery, boundArgs, err := r.db.DB.BindNamed(query, args)
	if err != nil {
		return false, errors.Wrap(err, "failed to bind webhook existence query")
	}
	err = r.db.DB.GetContext(ctx, &exists, boundQuery, boundArgs...)
	if err != nil {
		return false, errors.Wrap(err, "failed to check webhook existence")
	}

	return exists, nil
}
