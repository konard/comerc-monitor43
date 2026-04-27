package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenService_short_secret(t *testing.T) {
	t.Parallel()

	_, err := NewTokenService("short", time.Minute, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32")
}

func TestNewTokenService_exact_32_chars(t *testing.T) {
	t.Parallel()

	secret := "12345678901234567890123456789012" // ровно 32
	svc, err := NewTokenService(secret, time.Minute, time.Hour)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestTokenService_GenerateTokenPair(t *testing.T) {
	t.Run("generates valid token pair", func(t *testing.T) {
		secret := "test-secret-key-min-32-chars-long"
		service, err := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		accessToken, refreshToken, err := service.GenerateTokenPair(context.Background(), userID, "test@example.com", "Pro")

		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
		assert.NotEqual(t, accessToken, refreshToken)
	})

	t.Run("different tokens are generated", func(t *testing.T) {
		secret := "test-secret-key-min-32-chars-long"
		service, err := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		accessToken1, refreshToken1, err := service.GenerateTokenPair(context.Background(), uuid.New(), "user@example.com", "Free")
		require.NoError(t, err)
		accessToken2, refreshToken2, err := service.GenerateTokenPair(context.Background(), uuid.New(), "user@example.com", "Free")
		require.NoError(t, err)

		assert.NotEqual(t, accessToken1, accessToken2)
		assert.NotEqual(t, refreshToken1, refreshToken2)
	})
}

func TestTokenService_ValidateAccessToken(t *testing.T) {
	t.Run("valid token is accepted", func(t *testing.T) {
		secret := "test-secret-key-min-32-chars-long"
		service, err := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		accessToken, _, err := service.GenerateTokenPair(context.Background(), userID, "test@example.com", "Pro")
		require.NoError(t, err)

		claims, err := service.ValidateAccessToken(context.Background(), accessToken)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "Pro", claims.Tier)
		assert.NotNil(t, claims.ExpiresAt)
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		secret := "different-secret-key-min-32-chars-long"
		service, err := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		_, err = service.ValidateAccessToken(context.Background(), "invalid.token.string")

		assert.Error(t, err)
	})

	t.Run("expired token returns error", func(t *testing.T) {
		secret := "test-secret-key-min-32-chars-long"
		// Create service with very short expiry
		service, err := NewTokenService(secret, 1*time.Millisecond, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		accessToken, _, err := service.GenerateTokenPair(context.Background(), userID, "test@example.com", "Pro")
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		_, validateErr := service.ValidateAccessToken(context.Background(), accessToken)

		assert.Error(t, validateErr)
	})
}
