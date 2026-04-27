package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

const (
	sessionKeyPrefix = "session:"
	sessionListKey   = "sessions:user:"
)

// SessionRepository implements session repository using Redis
type SessionRepository struct {
	client *Client
}

// NewSessionRepository creates a new SessionRepository
func NewSessionRepository(client *Client) *SessionRepository {
	return &SessionRepository{client: client}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Store session
	key := r.sessionKey(session.ID)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Hour // Default to 1 hour if already expired
	}

	if err := r.client.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	// Add to user's session list
	listKey := r.userSessionsListKey(session.UserID)
	if err := r.client.client.SAdd(ctx, listKey, session.ID).Err(); err != nil {
		return fmt.Errorf("failed to add session to user list: %v", err)
	}

	return nil
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*model.Session, error) {
	key := r.sessionKey(sessionID)

	data, err := r.client.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	var session model.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	return &session, nil
}

// GetByUserID retrieves all sessions for a user
func (r *SessionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Session, int, error) {
	listKey := r.userSessionsListKey(userID)

	// Get all session IDs for the user
	sessionIDs, err := r.client.client.SMembers(ctx, listKey).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user sessions: %v", err)
	}

	// Apply pagination
	total := len(sessionIDs)
	if offset >= total {
		return []*model.Session{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	pagedIDs := sessionIDs[offset:end]

	// Fetch each session
	sessions := make([]*model.Session, 0, len(pagedIDs))
	for _, sessionID := range pagedIDs {
		session, err := r.GetByID(ctx, sessionID)
		if err != nil {
			// Skip expired/deleted sessions
			continue
		}
		sessions = append(sessions, session)
	}

	// Total — количество реально существующих сессий; истёкшие из SET пропускаются
	return sessions, len(sessions), nil
}

// Update updates a session
func (r *SessionRepository) Update(ctx context.Context, session *model.Session) error {
	return r.Create(ctx, session)
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	key := r.sessionKey(sessionID)

	// Get session to get user ID for cleanup
	session, err := r.GetByID(ctx, sessionID)
	if err == nil {
		listKey := r.userSessionsListKey(session.UserID)
		if removeErr := r.client.client.SRem(ctx, listKey, sessionID).Err(); removeErr != nil {
			return fmt.Errorf("failed to remove session from user list: %v", removeErr)
		}
	}

	return r.client.Delete(ctx, key)
}

// DeleteByUserID deletes all sessions for a user
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	listKey := r.userSessionsListKey(userID)

	// Get all session IDs
	sessionIDs, err := r.client.client.SMembers(ctx, listKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %v", err)
	}

	// Delete each session and remove from list
	for _, sessionID := range sessionIDs {
		key := r.sessionKey(sessionID)
		if deleteErr := r.client.Delete(ctx, key); deleteErr != nil {
			return fmt.Errorf("failed to delete session %q: %v", sessionID, deleteErr)
		}
	}

	// Clear user's session list
	if err := r.client.client.Del(ctx, listKey).Err(); err != nil {
		return fmt.Errorf("failed to clear user session list: %v", err)
	}

	return nil
}

// UpdateLastActive updates the last active timestamp for a session
func (r *SessionRepository) UpdateLastActive(ctx context.Context, sessionID string) error {
	session, err := r.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	session.UpdateLastActive()
	return r.Create(ctx, session)
}

// DeleteExpired deletes all expired sessions
func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	// Scan for all session keys
	keys, err := r.client.Scan(ctx, sessionKeyPrefix+"*", 1000)
	if err != nil {
		return fmt.Errorf("failed to scan sessions: %v", err)
	}

	for _, key := range keys {
		// Check if session is expired
		data, err := r.client.Get(ctx, key)
		if err != nil {
			continue
		}

		var session model.Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			// Delete malformed sessions
			if deleteErr := r.client.Delete(ctx, key); deleteErr != nil {
				return fmt.Errorf("failed to delete malformed session %q: %v", key, deleteErr)
			}
			continue
		}

		if session.IsExpired() {
			if deleteErr := r.client.Delete(ctx, key); deleteErr != nil {
				return fmt.Errorf("failed to delete expired session %q: %v", key, deleteErr)
			}
		}
	}

	return nil
}

// sessionKey returns the Redis key for a session
func (r *SessionRepository) sessionKey(sessionID string) string {
	return sessionKeyPrefix + sessionID
}

// userSessionsListKey returns the Redis key for a user's session list
func (r *SessionRepository) userSessionsListKey(userID uuid.UUID) string {
	return sessionListKey + userID.String()
}
