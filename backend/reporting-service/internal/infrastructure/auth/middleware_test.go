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

func TestNewAuthMiddleware(t *testing.T) {
	t.Parallel()

	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.authenticator)
}

func TestAuthMiddleware_UnaryInterceptor_ValidToken(t *testing.T) {
	t.Parallel()

	jwtSecret := "test-secret-key"
	mw := NewAuthMiddleware(jwtSecret)
	interceptor := mw.UnaryInterceptor()

	ctx := context.Background()
	userID := uuid.New()

	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+signedToken)
	ctxWithMeta := metadata.NewIncomingContext(ctx, md)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		extractedID := ExtractUserID(ctx)
		assert.Equal(t, userID, extractedID)
		return nil, nil
	}

	_, err = interceptor(ctxWithMeta, nil, nil, handler)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestAuthMiddleware_UnaryInterceptor_NoToken(t *testing.T) {
	t.Parallel()

	jwtSecret := "test-secret-key"
	mw := NewAuthMiddleware(jwtSecret)
	interceptor := mw.UnaryInterceptor()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs())

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	_, err := interceptor(ctx, nil, nil, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthMiddleware_UnaryInterceptor_InvalidToken(t *testing.T) {
	t.Parallel()

	jwtSecret := "test-secret-key"
	mw := NewAuthMiddleware(jwtSecret)
	interceptor := mw.UnaryInterceptor()

	md := metadata.Pairs("authorization", "Bearer invalid.token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	_, err := interceptor(ctx, nil, nil, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthMiddleware_StreamInterceptor_ValidToken(t *testing.T) {
	t.Parallel()

	jwtSecret := "test-secret-key"
	mw := NewAuthMiddleware(jwtSecret)
	interceptor := mw.StreamInterceptor()

	ctx := context.Background()
	userID := uuid.New()

	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	mockStream := &mockServerStream{
		ctx: metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer "+signedToken)),
	}

	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		assert.Equal(t, userID, ExtractUserID(ss.Context()))
		return nil
	}

	err = interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestAuthMiddleware_StreamInterceptor_NoToken(t *testing.T) {
	t.Parallel()

	jwtSecret := "test-secret-key"
	mw := NewAuthMiddleware(jwtSecret)
	interceptor := mw.StreamInterceptor()

	mockStream := &mockServerStream{
		ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs()),
	}

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestExtractUserID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	extractedUserID := ExtractUserID(ctx)
	assert.Equal(t, userID, extractedUserID)
	assert.NotEqual(t, uuid.Nil, extractedUserID)
}

func TestExtractUserID_NotSet(t *testing.T) {
	t.Parallel()

	extractedUserID := ExtractUserID(context.Background())
	assert.Equal(t, uuid.Nil, extractedUserID)
}

func TestGetUserID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	extractedUserID, err := GetUserID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestGetUserID_NotSet(t *testing.T) {
	t.Parallel()

	_, err := GetUserID(context.Background())
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestWrappedStream_Context(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := context.Background()
	ctxWithUserID := context.WithValue(ctx, UserIDKey, userID)

	wrapped := &wrappedStream{ctx: ctxWithUserID}
	retrievedCtx := wrapped.Context()
	assert.Equal(t, ctxWithUserID, retrievedCtx)
	assert.Equal(t, userID, ExtractUserID(retrievedCtx))
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}
