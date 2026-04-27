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

func makeValidToken(t *testing.T, secret string) string {
	t.Helper()

	userID := uuid.New()
	claims := &Claims{
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

func TestAuthMiddleware_UnaryInterceptor_NoMetadata(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("secret")
	interceptor := mw.UnaryInterceptor()

	// контекст без метаданных
	_, err := interceptor(context.Background(), nil, nil, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestAuthMiddleware_UnaryInterceptor_NonBearerHeader(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("secret")
	interceptor := mw.UnaryInterceptor()

	// заголовок есть, но не в формате Bearer
	md := metadata.Pairs("authorization", "Basic abc123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, nil, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid authorization header format")
}

func TestAuthMiddleware_StreamInterceptor_NoMetadata(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("secret")
	interceptor := mw.StreamInterceptor()

	// контекст без метаданных
	mockStream := &mockServerStream{ctx: context.Background()}
	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, func(srv any, ss grpc.ServerStream) error {
		return nil
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestAuthMiddleware_StreamInterceptor_NonBearerHeader(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("secret")
	interceptor := mw.StreamInterceptor()

	md := metadata.Pairs("authorization", "Basic abc123")
	mockStream := &mockServerStream{ctx: metadata.NewIncomingContext(context.Background(), md)}

	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, func(srv any, ss grpc.ServerStream) error {
		return nil
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid authorization header format")
}

func TestAuthMiddleware_StreamInterceptor_InvalidToken(t *testing.T) {
	t.Parallel()

	mw := NewAuthMiddleware("secret")
	interceptor := mw.StreamInterceptor()

	md := metadata.Pairs("authorization", "Bearer invalid.token.here")
	mockStream := &mockServerStream{ctx: metadata.NewIncomingContext(context.Background(), md)}

	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, func(srv any, ss grpc.ServerStream) error {
		return nil
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}
