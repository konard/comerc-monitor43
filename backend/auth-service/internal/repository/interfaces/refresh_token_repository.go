package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// RefreshTokenRepository defines interface for refresh token operations
type RefreshTokenRepository interface {
	// Create creates a new refresh token
	Create(ctx context.Context, token *model.RefreshToken) (*model.RefreshToken, error)

	// GetByTokenHash retrieves a refresh token by its hash
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	// GetByUserID retrieves all refresh tokens for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error)

	// Delete deletes a refresh token
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID deletes all refresh tokens for a user
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// Revoke marks a refresh token as revoked
	Revoke(ctx context.Context, id uuid.UUID) error

	// RevokeByUserID revokes all refresh tokens for a user
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteExpired deletes all expired refresh tokens
	DeleteExpired(ctx context.Context) error
}
