package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
)

// dbAPIKey — вспомогательная структура для сканирования строк из БД.
// Поля scopes и ip_whitelist объявлены как pq.StringArray, чтобы корректно
// декодировать PostgreSQL-массивы в []string.
type dbAPIKey struct {
	ID                 uuid.UUID          `db:"id"`
	UserID             uuid.UUID          `db:"user_id"`
	Name               string             `db:"name"`
	Description        *string            `db:"description"`
	KeyHash            string             `db:"key_hash"`
	KeyPrefix          string             `db:"key_prefix"`
	Scopes             pq.StringArray     `db:"scopes"`
	Status             model.APIKeyStatus `db:"status"`
	SecretKey          *string            `db:"secret_key"`
	ExpiresAt          *time.Time         `db:"expires_at"`
	IPWhitelist        pq.StringArray     `db:"ip_whitelist"`
	RateLimitPerMinute int                `db:"rate_limit_per_minute"`
	TotalRequests      int                `db:"total_requests"`
	SuccessfulRequests int                `db:"successful_requests"`
	FailedRequests     int                `db:"failed_requests"`
	LastUsedAt         *time.Time         `db:"last_used_at"`
	LastUsedIP         *string            `db:"last_used_ip"`
	MostUsedEndpoint   *string            `db:"most_used_endpoint"`
	CreatedAt          time.Time          `db:"created_at"`
	UpdatedAt          time.Time          `db:"updated_at"`
	LastRotatedAt      *time.Time         `db:"last_rotated_at"`
}

// toModel конвертирует dbAPIKey в model.APIKey.
func (r *dbAPIKey) toModel() *model.APIKey {
	key := &model.APIKey{
		ID:                 r.ID,
		UserID:             r.UserID,
		Name:               r.Name,
		Description:        r.Description,
		KeyHash:            r.KeyHash,
		KeyPrefix:          r.KeyPrefix,
		Status:             r.Status,
		SecretKey:          r.SecretKey,
		ExpiresAt:          r.ExpiresAt,
		RateLimitPerMinute: r.RateLimitPerMinute,
		TotalRequests:      r.TotalRequests,
		SuccessfulRequests: r.SuccessfulRequests,
		FailedRequests:     r.FailedRequests,
		LastUsedAt:         r.LastUsedAt,
		LastUsedIP:         r.LastUsedIP,
		MostUsedEndpoint:   r.MostUsedEndpoint,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
		LastRotatedAt:      r.LastRotatedAt,
	}
	if r.Scopes != nil {
		key.Scopes = []string(r.Scopes)
	}
	if r.IPWhitelist != nil {
		key.IPWhitelist = []string(r.IPWhitelist)
	}
	return key
}

type apiKeyRepositoryImpl struct {
	db *DB
}

// NewAPIKeyRepository создаёт новый APIKeyRepository.
func NewAPIKeyRepository(db *DB) interfaces.APIKeyRepository {
	return &apiKeyRepositoryImpl{db: db}
}

// Create создаёт новый API ключ.
func (r *apiKeyRepositoryImpl) Create(ctx context.Context, key *model.APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, user_id, name, description, key_hash, key_prefix,
			scopes, status, secret_key, expires_at,
			ip_whitelist, rate_limit_per_minute,
			total_requests, successful_requests, failed_requests,
			last_used_at, last_used_ip, most_used_endpoint,
			created_at, updated_at, last_rotated_at
		) VALUES (
			:id, :user_id, :name, :description, :key_hash, :key_prefix,
			:scopes, :status, :secret_key, :expires_at,
			:ip_whitelist, :rate_limit_per_minute,
			:total_requests, :successful_requests, :failed_requests,
			:last_used_at, :last_used_ip, :most_used_endpoint,
			:created_at, :updated_at, :last_rotated_at
		)
	`

	args := map[string]any{
		"id":                    key.ID,
		"user_id":               key.UserID,
		"name":                  key.Name,
		"description":           key.Description,
		"key_hash":              key.KeyHash,
		"key_prefix":            key.KeyPrefix,
		"scopes":                pq.Array(key.Scopes),
		"status":                key.Status,
		"secret_key":            key.SecretKey,
		"expires_at":            key.ExpiresAt,
		"ip_whitelist":          pq.Array(key.IPWhitelist),
		"rate_limit_per_minute": key.RateLimitPerMinute,
		"total_requests":        key.TotalRequests,
		"successful_requests":   key.SuccessfulRequests,
		"failed_requests":       key.FailedRequests,
		"last_used_at":          key.LastUsedAt,
		"last_used_ip":          key.LastUsedIP,
		"most_used_endpoint":    key.MostUsedEndpoint,
		"created_at":            key.CreatedAt,
		"updated_at":            key.UpdatedAt,
		"last_rotated_at":       key.LastRotatedAt,
	}

	_, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to create API key")
	}

	return nil
}

// GetByID возвращает API ключ по ID.
func (r *apiKeyRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (_ *model.APIKey, err error) {
	query := `
		SELECT
			id, user_id, name, description, key_hash, key_prefix,
			scopes, status, secret_key, expires_at,
			ip_whitelist, rate_limit_per_minute,
			total_requests, successful_requests, failed_requests,
			last_used_at, last_used_ip, most_used_endpoint,
			created_at, updated_at, last_rotated_at
		FROM api_keys
		WHERE id = :id
	`

	args := map[string]any{"id": id}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get API key")
	}
	defer closeResource(row, &err, "failed to close API key rows")

	if !row.Next() {
		return nil, interfaces.ErrAPIKeyNotFound
	}

	var dbKey dbAPIKey
	if err := row.StructScan(&dbKey); err != nil {
		return nil, errors.Wrap(err, "failed to scan API key")
	}

	return dbKey.toModel(), nil
}

// GetByKeyHash возвращает API ключ по хешу ключа.
func (r *apiKeyRepositoryImpl) GetByKeyHash(ctx context.Context, keyHash string) (_ *model.APIKey, err error) {
	query := `
		SELECT
			id, user_id, name, description, key_hash, key_prefix,
			scopes, status, secret_key, expires_at,
			ip_whitelist, rate_limit_per_minute,
			total_requests, successful_requests, failed_requests,
			last_used_at, last_used_ip, most_used_endpoint,
			created_at, updated_at, last_rotated_at
		FROM api_keys
		WHERE key_hash = :key_hash
	`

	args := map[string]any{"key_hash": keyHash}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get API key by hash")
	}
	defer closeResource(row, &err, "failed to close API key rows")

	if !row.Next() {
		return nil, interfaces.ErrAPIKeyNotFound
	}

	var dbKey dbAPIKey
	if err := row.StructScan(&dbKey); err != nil {
		return nil, errors.Wrap(err, "failed to scan API key")
	}

	return dbKey.toModel(), nil
}

// GetByUserIDAndName возвращает API ключ по userID и name.
func (r *apiKeyRepositoryImpl) GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (_ *model.APIKey, err error) {
	query := `
		SELECT
			id, user_id, name, description, key_hash, key_prefix,
			scopes, status, secret_key, expires_at,
			ip_whitelist, rate_limit_per_minute,
			total_requests, successful_requests, failed_requests,
			last_used_at, last_used_ip, most_used_endpoint,
			created_at, updated_at, last_rotated_at
		FROM api_keys
		WHERE user_id = :user_id AND name = :name
	`

	args := map[string]any{
		"user_id": userID,
		"name":    name,
	}

	row, err := r.db.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get API key")
	}
	defer closeResource(row, &err, "failed to close API key rows")

	if !row.Next() {
		return nil, interfaces.ErrAPIKeyNotFound
	}

	var dbKey dbAPIKey
	if err := row.StructScan(&dbKey); err != nil {
		return nil, errors.Wrap(err, "failed to scan API key")
	}

	return dbKey.toModel(), nil
}

// ListByUserID возвращает список API ключей пользователя.
func (r *apiKeyRepositoryImpl) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) (_ []*model.APIKey, err error) {
	query := `
		SELECT
			id, user_id, name, description, key_hash, key_prefix,
			scopes, status, secret_key, expires_at,
			ip_whitelist, rate_limit_per_minute,
			total_requests, successful_requests, failed_requests,
			last_used_at, last_used_ip, most_used_endpoint,
			created_at, updated_at, last_rotated_at
		FROM api_keys
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
		return nil, errors.Wrap(err, "failed to list API keys")
	}
	defer closeResource(rows, &err, "failed to close API key rows")

	var keys []*model.APIKey
	for rows.Next() {
		var dbKey dbAPIKey
		if err := rows.StructScan(&dbKey); err != nil {
			return nil, errors.Wrap(err, "failed to scan API key")
		}
		keys = append(keys, dbKey.toModel())
	}

	return keys, nil
}

// Update обновляет API ключ.
func (r *apiKeyRepositoryImpl) Update(ctx context.Context, key *model.APIKey) error {
	query := `
		UPDATE api_keys SET
			name = :name,
			description = :description,
			scopes = :scopes,
			status = :status,
			secret_key = :secret_key,
			expires_at = :expires_at,
			ip_whitelist = :ip_whitelist,
			rate_limit_per_minute = :rate_limit_per_minute,
			total_requests = :total_requests,
			successful_requests = :successful_requests,
			failed_requests = :failed_requests,
			last_used_at = :last_used_at,
			last_used_ip = :last_used_ip,
			most_used_endpoint = :most_used_endpoint,
			updated_at = :updated_at,
			last_rotated_at = :last_rotated_at
		WHERE id = :id
	`

	args := map[string]any{
		"id":                    key.ID,
		"name":                  key.Name,
		"description":           key.Description,
		"scopes":                pq.Array(key.Scopes),
		"status":                key.Status,
		"secret_key":            key.SecretKey,
		"expires_at":            key.ExpiresAt,
		"ip_whitelist":          pq.Array(key.IPWhitelist),
		"rate_limit_per_minute": key.RateLimitPerMinute,
		"total_requests":        key.TotalRequests,
		"successful_requests":   key.SuccessfulRequests,
		"failed_requests":       key.FailedRequests,
		"last_used_at":          key.LastUsedAt,
		"last_used_ip":          key.LastUsedIP,
		"most_used_endpoint":    key.MostUsedEndpoint,
		"updated_at":            key.UpdatedAt,
		"last_rotated_at":       key.LastRotatedAt,
	}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to update API key")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return interfaces.ErrAPIKeyNotFound
	}

	return nil
}

// UpdateUsage обновляет статистику использования API ключа.
func (r *apiKeyRepositoryImpl) UpdateUsage(ctx context.Context, id uuid.UUID, stats *interfaces.APIKeyUsageStats) error {
	query := `
		UPDATE api_keys SET
			total_requests = :total_requests,
			successful_requests = :successful_requests,
			failed_requests = :failed_requests,
			last_used_at = :last_used_at,
			last_used_ip = :last_used_ip,
			most_used_endpoint = :most_used_endpoint,
			updated_at = NOW()
		WHERE id = :id
	`

	args := map[string]any{
		"id":                  id,
		"total_requests":      stats.TotalRequests,
		"successful_requests": stats.SuccessfulRequests,
		"failed_requests":     stats.FailedRequests,
		"last_used_at":        stats.LastUsedAt,
		"last_used_ip":        stats.LastUsedIP,
		"most_used_endpoint":  stats.MostUsedEndpoint,
	}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to update API key usage")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return interfaces.ErrAPIKeyNotFound
	}

	return nil
}

// Delete удаляет API ключ.
func (r *apiKeyRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = :id`

	args := map[string]any{"id": id}

	result, err := r.db.DB.NamedExecContext(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "failed to delete API key")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return interfaces.ErrAPIKeyNotFound
	}

	return nil
}

// CountByUserID возвращает количество API ключей пользователя.
func (r *apiKeyRepositoryImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM api_keys WHERE user_id = :user_id`

	args := map[string]any{"user_id": userID}

	var count int
	boundQuery, boundArgs, err := r.db.DB.BindNamed(query, args)
	if err != nil {
		return 0, errors.Wrap(err, "failed to bind count API keys query")
	}
	err = r.db.DB.GetContext(ctx, &count, boundQuery, boundArgs...)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count API keys")
	}

	return count, nil
}

// ExistsByName проверяет существование API ключа с именем.
func (r *apiKeyRepositoryImpl) ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM api_keys
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
		return false, errors.Wrap(err, "failed to bind API key existence query")
	}
	err = r.db.DB.GetContext(ctx, &exists, boundQuery, boundArgs...)
	if err != nil {
		return false, errors.Wrap(err, "failed to check API key existence")
	}

	return exists, nil
}
