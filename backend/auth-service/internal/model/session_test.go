package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"expired_session", time.Now().Add(-1 * time.Hour), true},
		{"active_session", time.Now().Add(1 * time.Hour), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Session{ExpiresAt: tc.expiresAt}
			assert.Equal(t, tc.want, s.IsExpired())
		})
	}
}

func TestSession_UpdateLastActive(t *testing.T) {
	t.Parallel()

	s := &Session{LastActiveAt: time.Now().Add(-1 * time.Hour)}
	before := time.Now()
	s.UpdateLastActive()
	assert.True(t, !s.LastActiveAt.Before(before))
}

func TestNewSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	s := NewSession(userID, "browser", "192.168.1.1", 24*time.Hour)

	require.NotNil(t, s)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, userID, s.UserID)
	assert.Equal(t, "browser", s.DeviceInfo)
	assert.Equal(t, "192.168.1.1", s.IPAddress)
	assert.False(t, s.CreatedAt.IsZero())
	assert.False(t, s.LastActiveAt.IsZero())
	assert.True(t, s.ExpiresAt.After(s.CreatedAt))
}
