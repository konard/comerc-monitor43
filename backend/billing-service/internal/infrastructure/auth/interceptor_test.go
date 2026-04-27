package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func makeTestJWT(t *testing.T, secret, userID string) string {
	t.Helper()
	claims := &jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func TestJWTInterceptor_ExtractUserID(t *testing.T) {
	interceptor := NewJWTInterceptor("test-secret")

	t.Run("valid token with Bearer prefix", func(t *testing.T) {
		token := makeTestJWT(t, "test-secret", "test-user-id-123")
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"authorization": []string{"Bearer " + token},
		})

		userID, err := interceptor.extractUserID(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-user-id-123", userID)
	})

	t.Run("no metadata", func(t *testing.T) {
		ctx := context.Background()

		_, err := interceptor.extractUserID(ctx)
		assert.Error(t, err)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"other-header": []string{"value"},
		})

		_, err := interceptor.extractUserID(ctx)
		assert.Error(t, err)
	})

	t.Run("invalid authorization format", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"authorization": []string{"InvalidFormat token"},
		})

		_, err := interceptor.extractUserID(ctx)
		assert.Error(t, err)
	})

	t.Run("empty token", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"authorization": []string{"Bearer "},
		})

		_, err := interceptor.extractUserID(ctx)
		assert.Error(t, err)
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("user ID exists in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-id")

		userID, ok := GetUserID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test-user-id", userID)
	})

	t.Run("user ID not in context", func(t *testing.T) {
		ctx := context.Background()

		userID, ok := GetUserID(ctx)
		assert.False(t, ok)
		assert.Empty(t, userID)
	})
}

func TestRequireAuth(t *testing.T) {
	t.Run("authenticated user", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey{}, "test-user-id")

		userID, err := RequireAuth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-user-id", userID)
	})

	t.Run("unauthenticated request", func(t *testing.T) {
		ctx := context.Background()

		_, err := RequireAuth(ctx)
		assert.Error(t, err)
	})
}

func TestMockAuthInterceptor(t *testing.T) {
	interceptorFunc := MockAuthInterceptor

	t.Run("with user-id metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"user-id": []string{"test-user-123"},
		})

		// Call interceptor
		resp, err := interceptorFunc(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
			// Check user ID in context
			userID, ok := GetUserID(ctx)
			assert.True(t, ok)
			assert.Equal(t, "test-user-123", userID)
			return "response", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("without user-id metadata", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"other-header": []string{"value"},
		})

		resp, err := interceptorFunc(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
			// User ID should not be in context
			_, ok := GetUserID(ctx)
			assert.False(t, ok)
			return "response", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
}
