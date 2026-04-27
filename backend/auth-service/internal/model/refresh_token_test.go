package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		revokedAt *time.Time
		expiresAt time.Time
		want      bool
	}{
		{"valid_token", nil, time.Now().Add(7 * 24 * time.Hour), true},
		{"revoked_token", pastTime(), time.Now().Add(7 * 24 * time.Hour), false},
		{"expired_token", nil, time.Now().Add(-1 * time.Hour), false},
		{"revoked_and_expired", pastTime(), time.Now().Add(-1 * time.Hour), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rt := &RefreshToken{RevokedAt: tc.revokedAt, ExpiresAt: tc.expiresAt}
			assert.Equal(t, tc.want, rt.IsValid())
		})
	}
}

func TestRefreshToken_Revoke(t *testing.T) {
	t.Parallel()

	rt := &RefreshToken{}
	assert.Nil(t, rt.RevokedAt)
	rt.Revoke()
	assert.NotNil(t, rt.RevokedAt)
}

func TestNewRefreshToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	rt := NewRefreshToken(userID, "hash123", "mobile", "10.0.0.1", 7*24*time.Hour)

	require.NotNil(t, rt)
	assert.NotEmpty(t, rt.ID)
	assert.Equal(t, userID, rt.UserID)
	assert.Equal(t, "hash123", rt.TokenHash)
	assert.Equal(t, "mobile", rt.DeviceInfo)
	assert.Equal(t, "10.0.0.1", rt.IPAddress)
	assert.True(t, rt.ExpiresAt.After(time.Now()))
	assert.Nil(t, rt.RevokedAt)
}
