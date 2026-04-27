package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// --- ToUserDTO ---

func TestToUserDTO(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastLogin := now.Add(-time.Hour)
	user := &model.User{
		ID:          uuid.New(),
		Email:       "user@example.com",
		FullName:    "Test User",
		Tier:        "Pro",
		CreatedAt:   now,
		LastLoginAt: &lastLogin,
	}

	result := ToUserDTO(user)

	require.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Email, result.Email)
	assert.Equal(t, user.FullName, result.FullName)
	assert.Equal(t, user.Tier, result.Tier)
	assert.Equal(t, user.CreatedAt, result.CreatedAt)
	assert.Equal(t, user.LastLoginAt, result.LastLoginAt)
}

func TestToUserDTO_nil_last_login(t *testing.T) {
	t.Parallel()

	user := &model.User{
		ID:        uuid.New(),
		Email:     "user@example.com",
		CreatedAt: time.Now(),
	}

	result := ToUserDTO(user)

	require.NotNil(t, result)
	assert.Nil(t, result.LastLoginAt)
}

// --- ToGetUserResponse ---

func TestToGetUserResponse(t *testing.T) {
	t.Parallel()

	now := time.Now()
	user := &model.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FullName:  "Test",
		Tier:      "Free",
		CreatedAt: now,
	}

	resp := ToGetUserResponse(user)

	require.NotNil(t, resp)
	assert.Equal(t, user.ID, resp.ID)
	assert.Equal(t, user.Email, resp.Email)
	assert.Equal(t, user.FullName, resp.FullName)
	assert.Equal(t, user.Tier, resp.Tier)
	assert.Equal(t, user.CreatedAt, resp.CreatedAt)
	assert.Nil(t, resp.LastLoginAt)
}

func TestToGetUserResponse_with_last_login(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastLogin := now.Add(-2 * time.Hour)
	user := &model.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		CreatedAt:   now,
		LastLoginAt: &lastLogin,
	}

	resp := ToGetUserResponse(user)

	require.NotNil(t, resp.LastLoginAt)
	assert.Equal(t, lastLogin, *resp.LastLoginAt)
}

// --- ToSessionDTO ---

func TestToSessionDTO(t *testing.T) {
	t.Parallel()

	sessionID := "sess-123"
	userID := uuid.New()
	now := time.Now()

	session := &model.Session{
		ID:           sessionID,
		UserID:       userID,
		DeviceInfo:   "Chrome/Linux",
		IPAddress:    "10.0.0.1",
		CreatedAt:    now,
		LastActiveAt: now,
	}

	result := ToSessionDTO(session, true)

	require.NotNil(t, result)
	assert.Equal(t, sessionID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Chrome/Linux", result.DeviceInfo)
	assert.Equal(t, "10.0.0.1", result.IPAddress)
	assert.Equal(t, now, result.CreatedAt)
	assert.Equal(t, now, result.LastActiveAt)
	assert.True(t, result.IsCurrent)
}

func TestToSessionDTO_not_current(t *testing.T) {
	t.Parallel()

	session := &model.Session{
		ID:     "sess-456",
		UserID: uuid.New(),
	}

	result := ToSessionDTO(session, false)

	assert.False(t, result.IsCurrent)
}

// --- ToSessionDTOs ---

func TestToSessionDTOs(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessions := []*model.Session{
		{ID: "sess-1", UserID: userID},
		{ID: "sess-2", UserID: userID},
		{ID: "sess-3", UserID: userID},
	}

	result := ToSessionDTOs(sessions, "sess-2")

	require.Len(t, result, 3)
	assert.False(t, result[0].IsCurrent)
	assert.True(t, result[1].IsCurrent)
	assert.False(t, result[2].IsCurrent)
}

func TestToSessionDTOs_empty(t *testing.T) {
	t.Parallel()

	result := ToSessionDTOs([]*model.Session{}, "")

	assert.Empty(t, result)
}
