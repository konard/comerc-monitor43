package model

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a user session stored in Redis
type Session struct {
	ID           string    `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	DeviceInfo   string    `json:"device_info"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired checks if session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// UpdateLastActive updates the last active timestamp
func (s *Session) UpdateLastActive() {
	s.LastActiveAt = time.Now()
}

// NewSession creates a new session
func NewSession(userID uuid.UUID, deviceInfo, ipAddress string, expiryDuration time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		CreatedAt:    now,
		LastActiveAt: now,
		ExpiresAt:    now.Add(expiryDuration),
	}
}
