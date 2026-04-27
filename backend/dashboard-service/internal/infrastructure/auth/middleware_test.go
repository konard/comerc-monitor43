package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryInterceptor_valid_token(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("test-secret")
	interceptor := mw.UnaryInterceptor()

	userID := uuid.New()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenString := createTestToken("test-secret", claims)

	md := metadata.Pairs("authorization", "Bearer "+tokenString)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var handlerCalled bool
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return nil, nil
	}

	// Act
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)

	// Assert
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestUnaryInterceptor_missing_metadata(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("test-secret")
	interceptor := mw.UnaryInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Act
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUnaryInterceptor_missing_authorization_header(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("test-secret")
	interceptor := mw.UnaryInterceptor()

	md := metadata.Pairs("x-custom", "value")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Act
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUnaryInterceptor_invalid_format(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("test-secret")
	interceptor := mw.UnaryInterceptor()

	md := metadata.Pairs("authorization", "Bearer")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Act
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUnaryInterceptor_invalid_token(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("test-secret")
	interceptor := mw.UnaryInterceptor()

	md := metadata.Pairs("authorization", "Bearer bad-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Act
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUnaryInterceptor_extracts_user_id(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("test-secret")
	interceptor := mw.UnaryInterceptor()

	expectedID := uuid.New()
	claims := &Claims{
		UserID: expectedID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenString := createTestToken("test-secret", claims)

	md := metadata.Pairs("authorization", "Bearer "+tokenString)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var receivedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		receivedCtx = ctx
		return nil, nil
	}

	// Act
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedID, ExtractUserID(receivedCtx))
}
