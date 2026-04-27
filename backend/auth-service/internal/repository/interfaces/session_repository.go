package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// SessionRepository defines interface for session operations
type SessionRepository interface {
	// Create creates a new session
	Create(ctx context.Context, session *model.Session) error

	// GetByID retrieves a session by ID
	GetByID(ctx context.Context, sessionID string) (*model.Session, error)

	// GetByUserID retrieves all sessions for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Session, int, error)

	// Update updates a session
	Update(ctx context.Context, session *model.Session) error

	// Delete deletes a session
	Delete(ctx context.Context, sessionID string) error

	// DeleteByUserID deletes all sessions for a user
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// UpdateLastActive updates last active timestamp for a session
	UpdateLastActive(ctx context.Context, sessionID string) error

	// DeleteExpired deletes all expired sessions
	DeleteExpired(ctx context.Context) error
}
