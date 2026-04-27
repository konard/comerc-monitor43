package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestContextWithUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.Background()
	userID := "test-user-456"

	// Act
	ctx = ContextWithUserID(ctx, userID)

	// Assert
	got, ok := GetUserID(ctx)
	assert.True(t, ok)
	assert.Equal(t, userID, got)
}

func TestJWTInterceptor_Unary(t *testing.T) {
	t.Parallel()

	interceptor := NewJWTInterceptor("secret")
	unary := interceptor.Unary()

	t.Run("valid_token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		token := makeTestJWT(t, "secret", "user-id-abc")
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"authorization": []string{"Bearer " + token},
		})

		var capturedCtx context.Context
		handler := func(ctx context.Context, req any) (any, error) {
			capturedCtx = ctx
			return "ok", nil
		}

		// Act
		resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{}, handler)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "ok", resp)

		userID, ok := GetUserID(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "user-id-abc", userID)
	})

	t.Run("missing_auth_header", func(t *testing.T) {
		t.Parallel()

		// Arrange
		ctx := context.Background() // нет metadata

		handler := func(ctx context.Context, req any) (any, error) {
			return "should_not_reach", nil
		}

		// Act
		resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{}, handler)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}
