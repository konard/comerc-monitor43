package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// OAuthRepository defines interface for OAuth account operations
type OAuthRepository interface {
	// Create creates a new OAuth account
	Create(ctx context.Context, account *model.OAuthAccount) (*model.OAuthAccount, error)

	// GetByID retrieves an OAuth account by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.OAuthAccount, error)

	// GetByProviderUserID retrieves an OAuth account by provider and provider user ID
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*model.OAuthAccount, error)

	// GetByUserID retrieves all OAuth accounts for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.OAuthAccount, error)

	// Delete deletes an OAuth account
	Delete(ctx context.Context, id uuid.UUID) error
}
