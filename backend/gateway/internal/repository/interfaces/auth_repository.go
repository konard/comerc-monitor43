package interfaces

import (
	"context"

	"github.com/raul/monitor/gateway/internal/model"
)

// AuthRepository defines the interface for authentication operations
type AuthRepository interface {
	// ValidateToken validates an access token and returns user context
	ValidateToken(ctx context.Context, token string) (*model.TokenValidationResult, error)

	// CheckHealth checks if the auth service is healthy
	CheckHealth(ctx context.Context) (bool, error)
}
