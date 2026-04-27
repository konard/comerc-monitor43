package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// UserRepository defines interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *model.User) (*model.User, error)

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// Update updates a user
	Update(ctx context.Context, user *model.User) error

	// UpdateLastLogin updates last login timestamp
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error

	// IncrementLoginAttempts increments failed login counter
	IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error

	// ResetLoginAttempts resets failed login counter
	ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error

	// LockAccount locks a user account until specified time
	LockAccount(ctx context.Context, userID uuid.UUID, lockedUntil any) error

	// UnlockAccount unlocks a user account
	UnlockAccount(ctx context.Context, userID uuid.UUID) error

	// ExistsByEmail checks if a user with given email exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
