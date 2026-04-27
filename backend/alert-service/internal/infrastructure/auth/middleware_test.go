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
	jwtSecret := "test-secret-key"

	middleware := NewAuthMiddleware(jwtSecret)

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.authenticator)
	assert.Equal(t, jwtSecret, middleware.jwtSecret)
}

func TestAuthMiddleware_UnaryInterceptor(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.UnaryInterceptor()

	assert.NotNil(t, interceptor)
}

func TestAuthMiddleware_UnaryInterceptor_ValidToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.UnaryInterceptor()

	ctx := context.Background()
	userID := uuid.New()

	// Create valid JWT token
	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	// Create context with Bearer token
	md := metadata.Pairs("authorization", "Bearer "+signedToken)
	ctxWithMeta := metadata.NewIncomingContext(ctx, md)

	// Mock handler that checks context
	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return nil, nil
	}

	// Call interceptor
	info := &grpc.UnaryServerInfo{FullMethod: "/alert.AlertService/TestMethod"}
	_, err = interceptor(ctxWithMeta, nil, info, handler)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestAuthMiddleware_UnaryInterceptor_NoToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.UnaryInterceptor()

	ctx := context.Background()

	// Create context without token
	md := metadata.Pairs()
	ctxWithoutToken := metadata.NewIncomingContext(ctx, md)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Call interceptor
	info := &grpc.UnaryServerInfo{FullMethod: "/alert.AlertService/TestMethod"}
	_, err := interceptor(ctxWithoutToken, nil, info, handler)

	require.Error(t, err)
}

func TestAuthMiddleware_UnaryInterceptor_InvalidToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.UnaryInterceptor()

	ctx := context.Background()

	// Create context with invalid token
	md := metadata.Pairs("authorization", "Bearer invalid.token")
	ctxWithInvalidToken := metadata.NewIncomingContext(ctx, md)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Call interceptor
	info := &grpc.UnaryServerInfo{FullMethod: "/alert.AlertService/TestMethod"}
	_, err := interceptor(ctxWithInvalidToken, nil, info, handler)

	require.Error(t, err)
}

func TestExtractUserID(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	extractedUserID := ExtractUserID(ctx)

	assert.Equal(t, userID, extractedUserID)
	assert.NotEqual(t, uuid.Nil, extractedUserID)
}

func TestExtractUserID_NotSet(t *testing.T) {
	ctx := context.Background()

	extractedUserID := ExtractUserID(ctx)

	assert.Equal(t, uuid.Nil, extractedUserID)
}

func TestGetUserID(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	extractedUserID, err := GetUserID(ctx)

	assert.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestGetUserID_NotSet(t *testing.T) {
	ctx := context.Background()

	_, err := GetUserID(ctx)

	require.Error(t, err)
}

func TestAuthMiddleware_StreamInterceptor_ValidToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.StreamInterceptor()

	ctx := context.Background()
	userID := uuid.New()

	// Create valid JWT token
	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	// Create mock server stream
	mockStream := &mockServerStream{
		ctx: ctx,
	}

	// Create context with Bearer token
	md := metadata.Pairs("authorization", "Bearer "+signedToken)
	mockStream.ctx = metadata.NewIncomingContext(ctx, md)

	// Mock handler
	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		// Verify that context has user ID
		newUserID := ExtractUserID(ss.Context())
		assert.Equal(t, userID, newUserID)
		return nil
	}

	// Call interceptor
	err = interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestAuthMiddleware_StreamInterceptor_NoToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.StreamInterceptor()

	ctx := context.Background()
	mockStream := &mockServerStream{
		ctx: ctx,
	}

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	// Call interceptor without token
	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthMiddleware_StreamInterceptor_InvalidToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	middleware := NewAuthMiddleware(jwtSecret)
	interceptor := middleware.StreamInterceptor()

	ctx := context.Background()

	// Create context with invalid token
	md := metadata.Pairs("authorization", "Bearer invalid.token")
	mockStream := &mockServerStream{
		ctx: metadata.NewIncomingContext(ctx, md),
	}

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	// Call interceptor
	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestWrappedStream_Context(t *testing.T) {
	userID := uuid.New()
	ctx := context.Background()
	ctxWithUserID := context.WithValue(ctx, UserIDKey, userID)

	wrapped := &wrappedStream{
		ctx: ctxWithUserID,
	}

	retrievedCtx := wrapped.Context()
	assert.Equal(t, ctxWithUserID, retrievedCtx)
	assert.Equal(t, userID, ExtractUserID(retrievedCtx))
}

func TestWrappedStream_Context_NilContext(t *testing.T) {
	wrapped := &wrappedStream{
		ctx: nil,
	}

	retrievedCtx := wrapped.Context()
	assert.Nil(t, retrievedCtx)
}

// mockServerStream implements grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func TestRole_IsAdmin(t *testing.T) {
	assert.True(t, RoleAdmin.IsAdmin())
	assert.False(t, RoleUser.IsAdmin())
	assert.False(t, RoleGuest.IsAdmin())
}

func TestRole_CanCreateChannels(t *testing.T) {
	assert.True(t, RoleUser.CanCreateChannels())
	assert.True(t, RoleAdmin.CanCreateChannels())
	assert.False(t, RoleGuest.CanCreateChannels())
}

func TestExtractRole_FromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), RoleKey, RoleAdmin)
	assert.Equal(t, RoleAdmin, ExtractRole(ctx))
}

func TestExtractRole_NotSet(t *testing.T) {
	assert.Equal(t, RoleGuest, ExtractRole(context.Background()))
}

func TestExtractRole_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), RoleKey, "not-a-role")
	assert.Equal(t, RoleGuest, ExtractRole(ctx))
}

func TestRequireAdmin_Admin(t *testing.T) {
	ctx := context.WithValue(context.Background(), RoleKey, RoleAdmin)
	assert.NoError(t, RequireAdmin(ctx))
}

func TestRequireAdmin_NotAdmin(t *testing.T) {
	ctx := context.WithValue(context.Background(), RoleKey, RoleUser)
	err := RequireAdmin(ctx)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestRequireChannelAccess_Owner(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)
	assert.NoError(t, RequireChannelAccess(ctx, userID))
}

func TestRequireChannelAccess_Admin(t *testing.T) {
	ctx := context.WithValue(context.WithValue(context.Background(), UserIDKey, uuid.New()), RoleKey, RoleAdmin)
	assert.NoError(t, RequireChannelAccess(ctx, uuid.New()))
}

func TestRequireChannelAccess_NotOwner(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)
	err := RequireChannelAccess(ctx, uuid.New())
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestRequireChannelAccess_Unauthenticated(t *testing.T) {
	err := RequireChannelAccess(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
