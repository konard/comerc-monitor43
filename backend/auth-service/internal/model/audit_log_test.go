package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLog(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	log := NewAuditLog(&userID, EventTypeLogin, "", true, "127.0.0.1", "Mozilla/5.0", "")

	require.NotNil(t, log)
	assert.NotEmpty(t, log.ID)
	assert.Equal(t, &userID, log.UserID)
	assert.Equal(t, EventTypeLogin, log.EventType)
	assert.True(t, log.Success)
	assert.Equal(t, "127.0.0.1", log.IPAddress)
	assert.Equal(t, "Mozilla/5.0", log.UserAgent)
	assert.Empty(t, log.ErrorMessage)
	assert.False(t, log.CreatedAt.IsZero())
}

func TestNewAuditLog_nil_user(t *testing.T) {
	t.Parallel()

	log := NewAuditLog(nil, EventTypeLogin, "", false, "10.0.0.1", "", "user not found")
	require.NotNil(t, log)
	assert.Nil(t, log.UserID)
	assert.False(t, log.Success)
	assert.Equal(t, "user not found", log.ErrorMessage)
}

func TestAuditLog_eventType_constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "login", EventTypeLogin)
	assert.Equal(t, "logout", EventTypeLogout)
	assert.Equal(t, "oauth_callback", EventTypeOAuthCallback)
	assert.Equal(t, "session_created", EventTypeSessionCreated)
	assert.Equal(t, "session_revoked", EventTypeSessionRevoked)
	assert.Equal(t, "account_locked", EventTypeAccountLocked)
	assert.Equal(t, "account_unlocked", EventTypeAccountUnlocked)
	assert.Equal(t, "token_refresh", EventTypeTokenRefresh)
	assert.Equal(t, "token_validate", EventTypeTokenValidate)
	assert.Equal(t, "register", EventTypeRegister)
}
