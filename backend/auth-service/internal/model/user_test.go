package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUser_IsLocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		lockedUntil *time.Time
		want        bool
	}{
		{"nil_locked_until", nil, false},
		{"past_locked_until", pastTime(), false},
		{"future_locked_until", futureTime(1 * time.Hour), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := &User{LockedUntil: tc.lockedUntil}
			assert.Equal(t, tc.want, u.IsLocked())
		})
	}
}

func TestUser_ShouldLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		loginAttempt int
		maxAttempts  int
		want         bool
	}{
		{"below_max", 2, 5, false},
		{"at_max", 5, 5, true},
		{"above_max", 6, 5, true},
		{"zero_attempts", 0, 5, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := &User{LoginAttempts: tc.loginAttempt}
			assert.Equal(t, tc.want, u.ShouldLock(tc.maxAttempts))
		})
	}
}

func TestUser_ResetLoginAttempts(t *testing.T) {
	t.Parallel()

	u := &User{LoginAttempts: 5, LockedUntil: futureTime(1 * time.Hour)}
	u.ResetLoginAttempts()
	assert.Equal(t, 0, u.LoginAttempts)
	assert.Nil(t, u.LockedUntil)
}

func TestUser_IncrementLoginAttempts(t *testing.T) {
	t.Parallel()

	u := &User{LoginAttempts: 3}
	u.IncrementLoginAttempts()
	assert.Equal(t, 4, u.LoginAttempts)
}

func TestUser_UpdateLastLogin(t *testing.T) {
	t.Parallel()

	u := &User{}
	before := time.Now()
	u.UpdateLastLogin()
	assert.NotNil(t, u.LastLoginAt)
	assert.True(t, !u.LastLoginAt.Before(before))
	assert.True(t, !u.UpdatedAt.Before(before))
}

func TestUser_Lock(t *testing.T) {
	t.Parallel()

	u := &User{}
	until := time.Now().Add(30 * time.Minute)
	u.Lock(until)
	assert.NotNil(t, u.LockedUntil)
	assert.Equal(t, until, *u.LockedUntil)
}

func TestNewUser(t *testing.T) {
	t.Parallel()

	u := NewUser("test@example.com", "hash", "Test User", "Free")
	assert.NotEmpty(t, u.ID)
	assert.Equal(t, "test@example.com", u.Email)
	assert.Equal(t, "hash", u.PasswordHash)
	assert.Equal(t, "Test User", u.FullName)
	assert.Equal(t, "Free", u.Tier)
	assert.False(t, u.CreatedAt.IsZero())
	assert.False(t, u.UpdatedAt.IsZero())
}

func pastTime() *time.Time {
	t := time.Now().Add(-1 * time.Hour)
	return &t
}

func futureTime(d time.Duration) *time.Time {
	t := time.Now().Add(d)
	return &t
}
