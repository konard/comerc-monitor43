package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token stored in database
type RefreshToken struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	TokenHash  string     `json:"-"` // Never expose token hash
	DeviceInfo string     `json:"device_info,omitempty"`
	IPAddress  string     `json:"ip_address,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// IsValid checks if refresh token is valid
func (rt *RefreshToken) IsValid() bool {
	if rt.RevokedAt != nil {
		return false
	}
	return time.Now().Before(rt.ExpiresAt)
}

// Revoke marks refresh token as revoked
func (rt *RefreshToken) Revoke() {
	now := time.Now()
	rt.RevokedAt = &now
}

// NewRefreshToken creates a new refresh token
func NewRefreshToken(userID uuid.UUID, tokenHash, deviceInfo, ipAddress string, expiryDuration time.Duration) *RefreshToken {
	return &RefreshToken{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
		ExpiresAt:  time.Now().Add(expiryDuration),
		CreatedAt:  time.Now(),
	}
}
