package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

type apiKeyUsageRepositoryImpl struct {
	db *DB
}

// NewAPIKeyUsageRepository создаёт новый APIKeyUsageRepository.
func NewAPIKeyUsageRepository(db *DB) interfaces.APIKeyUsageRepository {
	return &apiKeyUsageRepositoryImpl{db: db}
}

// Create создаёт запись о логе использования.
func (r *apiKeyUsageRepositoryImpl) Create(ctx context.Context, log *model.APIKeyUsageLog) error {
	query := `
		INSERT INTO api_key_usage_logs (
			id, api_key_id, endpoint, method, http_status_code,
			response_time_ms, ip_address, user_agent, request_id,
			rate_limited, created_at
		) VALUES (
			:id, :api_key_id, :endpoint, :method, :http_status_code,
			:response_time_ms, :ip_address, :user_agent, :request_id,
			:rate_limited, :created_at
		)
	`

	args := map[string]any{
		"id":               log.ID,
		"api_key_id":       log.APIKeyID,
		"endpoint":         log.Endpoint,
		"method":           log.Method,
		"http_status_code": log.HTTPStatusCode,
		"response_time_ms": log.ResponseTimeMs,
		"ip_address":       log.IPAddress,
		"user_agent":       log.UserAgent,
		"request_id":       log.RequestID,
		"rate_limited":     log.RateLimited,
		"created_at":       log.CreatedAt,
	}

	_, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to create API key usage log")
	}

	return nil
}

// GetByID возвращает лог по ID.
func (r *apiKeyUsageRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (_ *model.APIKeyUsageLog, err error) {
	query := `
		SELECT
			id, api_key_id, endpoint, method, http_status_code,
			response_time_ms, ip_address, user_agent, request_id,
			rate_limited, created_at
		FROM api_key_usage_logs
		WHERE id = :id
	`

	args := map[string]any{"id": id}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get API key usage log")
	}
	defer closeResource(row, &err, "failed to close API key usage rows")

	if !row.Next() {
		return nil, errors.New("API key usage log not found")
	}

	var log model.APIKeyUsageLog
	if err := row.StructScan(&log); err != nil {
		return nil, errors.Wrap(err, "failed to scan API key usage log")
	}

	return &log, nil
}

// ListByAPIKeyID возвращает логи для API ключа.
func (r *apiKeyUsageRepositoryImpl) ListByAPIKeyID(ctx context.Context, apiKeyID uuid.UUID, limit, offset int) (_ []*model.APIKeyUsageLog, err error) {
	query := `
		SELECT
			id, api_key_id, endpoint, method, http_status_code,
			response_time_ms, ip_address, user_agent, request_id,
			rate_limited, created_at
		FROM api_key_usage_logs
		WHERE api_key_id = :api_key_id
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`

	args := map[string]any{
		"api_key_id": apiKeyID,
		"limit":      limit,
		"offset":     offset,
	}

	rows, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list API key usage logs")
	}
	defer closeResource(rows, &err, "failed to close API key usage rows")

	var logs []*model.APIKeyUsageLog
	for rows.Next() {
		var log model.APIKeyUsageLog
		if err := rows.StructScan(&log); err != nil {
			return nil, errors.Wrap(err, "failed to scan API key usage log")
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// ListByAPIKeyIDAndPeriod возвращает логи за период.
func (r *apiKeyUsageRepositoryImpl) ListByAPIKeyIDAndPeriod(ctx context.Context, apiKeyID uuid.UUID, from, to int64, limit, offset int) (_ []*model.APIKeyUsageLog, err error) {
	query := `
		SELECT
			id, api_key_id, endpoint, method, http_status_code,
			response_time_ms, ip_address, user_agent, request_id,
			rate_limited, created_at
		FROM api_key_usage_logs
		WHERE api_key_id = :api_key_id
			AND created_at >= to_timestamp(:from)
			AND created_at <= to_timestamp(:to)
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`

	args := map[string]any{
		"api_key_id": apiKeyID,
		"from":       from,
		"to":         to,
		"limit":      limit,
		"offset":     offset,
	}

	rows, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list API key usage logs by period")
	}
	defer closeResource(rows, &err, "failed to close API key usage rows")

	var logs []*model.APIKeyUsageLog
	for rows.Next() {
		var log model.APIKeyUsageLog
		if err := rows.StructScan(&log); err != nil {
			return nil, errors.Wrap(err, "failed to scan API key usage log")
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// DeleteOldLogs удаляет старые логи.
func (r *apiKeyUsageRepositoryImpl) DeleteOldLogs(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
		DELETE FROM api_key_usage_logs
		WHERE created_at < NOW() - INTERVAL '1 day' * :days
	`

	args := map[string]any{"days": olderThanDays}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete old API key usage logs")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get rows affected")
	}

	return rowsAffected, nil
}
