package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/repository/interfaces"
)

// RefreshTokenRepository implements refresh token repository using PostgreSQL
type RefreshTokenRepository struct {
	db *DB
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository
func NewRefreshTokenRepository(db *DB) interfaces.RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create creates a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) (*model.RefreshToken, error) {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, device_info, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, token_hash, device_info, ip_address, expires_at, revoked_at, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		token.ID, token.UserID, token.TokenHash, token.DeviceInfo, token.IPAddress,
		token.ExpiresAt, token.CreatedAt,
	).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.DeviceInfo, &token.IPAddress,
		&token.ExpiresAt, &token.RevokedAt, &token.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %v", err)
	}

	return token, nil
}

// GetByTokenHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, device_info, ip_address, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	token := &model.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.DeviceInfo, &token.IPAddress,
		&token.ExpiresAt, &token.RevokedAt, &token.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %v", err)
	}

	return token, nil
}

// GetByUserID retrieves all refresh tokens for a user
func (r *RefreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, device_info, ip_address, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh tokens: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			return
		}
	}()

	var tokens []*model.RefreshToken
	for rows.Next() {
		token := &model.RefreshToken{}
		err := rows.Scan(
			&token.ID, &token.UserID, &token.TokenHash, &token.DeviceInfo, &token.IPAddress,
			&token.ExpiresAt, &token.RevokedAt, &token.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %v", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// Delete deletes a refresh token
func (r *RefreshTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM refresh_tokens WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %v", err)
	}

	return nil
}

// DeleteByUserID deletes all refresh tokens for a user
func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete refresh tokens: %v", err)
	}

	return nil
}

// Revoke marks a refresh token as revoked
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %v", err)
	}

	return nil
}

// RevokeByUserID revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %v", err)
	}

	return nil
}

// DeleteExpired deletes all expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	const query = `DELETE FROM refresh_tokens WHERE expires_at < NOW()`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %v", err)
	}

	return nil
}
