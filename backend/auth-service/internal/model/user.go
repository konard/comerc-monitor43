package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user entity in the domain layer
type User struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"` // Never expose password hash
	FullName      string     `json:"full_name"`
	Tier          string     `json:"tier"` // Free, Basic, Pro, Enterprise
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	LoginAttempts int        `json:"login_attempts"`
	LockedUntil   *time.Time `json:"locked_until,omitempty"`
}

// IsLocked checks if the user account is currently locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return u.LockedUntil.After(time.Now())
}

// ShouldLock checks if the user should be locked based on login attempts
func (u *User) ShouldLock(maxAttempts int) bool {
	return u.LoginAttempts >= maxAttempts
}

// ResetLoginAttempts resets the failed login counter
func (u *User) ResetLoginAttempts() {
	u.LoginAttempts = 0
	u.LockedUntil = nil
}

// IncrementLoginAttempts increments the failed login counter
func (u *User) IncrementLoginAttempts() {
	u.LoginAttempts++
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// Lock locks the user account until the specified time
func (u *User) Lock(until time.Time) {
	u.LockedUntil = &until
	u.UpdatedAt = time.Now()
}

// NewUser creates a new user entity
func NewUser(email, passwordHash, fullName, tier string) *User {
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
		Tier:         tier,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}
