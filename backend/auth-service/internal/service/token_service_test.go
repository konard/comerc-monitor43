package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/internal/service/mocks"
	apperrors "github.com/raul/monitor/backend/auth-service/pkg/errors"
)

func TestRefreshToken(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		tokenHash := model.HashToken("old-refresh-token")
		storedToken := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		user := &model.User{ID: userID, Email: "user@example.com", Tier: "Free"}

		refreshRepo := mocks.NewRefreshTokenStore(t)
		refreshRepo.On("GetByTokenHash", mock.Anything, tokenHash).Return(storedToken, nil)
		refreshRepo.On("Revoke", mock.Anything, storedToken.ID).Return(nil)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(&model.RefreshToken{}, nil)
		userGetter := mocks.NewUserGetter(t)
		userGetter.On("GetByID", mock.Anything, userID).Return(user, nil)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), refreshRepo, mocks.NewSessionStore(t), userGetter, auditLogger, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.RefreshToken(context.Background(), &dto.RefreshTokenRequest{
			RefreshToken: "old-refresh-token",
		}, "127.0.0.1", "test-agent")

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "Bearer", resp.TokenType)
	})

	t.Run("token_not_found", func(t *testing.T) {
		t.Parallel()

		refreshRepo := mocks.NewRefreshTokenStore(t)
		refreshRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), refreshRepo, mocks.NewSessionStore(t), mocks.NewUserGetter(t), auditLogger, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.RefreshToken(context.Background(), &dto.RefreshTokenRequest{
			RefreshToken: "nonexistent",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInvalidRefreshToken, appErr.Code)
	})

	t.Run("token_revoked", func(t *testing.T) {
		t.Parallel()

		revokedTime := time.Now().Add(-1 * time.Hour)
		storedToken := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			TokenHash: "hash",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			RevokedAt: &revokedTime,
		}

		refreshRepo := mocks.NewRefreshTokenStore(t)
		refreshRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(storedToken, nil)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), refreshRepo, mocks.NewSessionStore(t), mocks.NewUserGetter(t), auditLogger, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.RefreshToken(context.Background(), &dto.RefreshTokenRequest{
			RefreshToken: "revoked-token",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInvalidRefreshToken, appErr.Code)
	})

	t.Run("user_not_found", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		storedToken := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    userID,
			TokenHash: "hash",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		refreshRepo := mocks.NewRefreshTokenStore(t)
		refreshRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(storedToken, nil)
		userGetter := mocks.NewUserGetter(t)
		userGetter.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), refreshRepo, mocks.NewSessionStore(t), userGetter, auditLogger, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.RefreshToken(context.Background(), &dto.RefreshTokenRequest{
			RefreshToken: "valid-token",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestValidateToken(t *testing.T) {
	t.Parallel()

	t.Run("valid_token", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{ID: userID, Email: "user@example.com", Tier: "Pro"}

		jwtSvc := newTestJWTService(t)
		accessToken, _, err := jwtSvc.GenerateTokenPair(context.Background(), userID, "user@example.com", "Pro")
		require.NoError(t, err)

		userGetter := mocks.NewUserGetter(t)
		userGetter.On("GetByID", mock.Anything, userID).Return(user, nil)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(jwtSvc, mocks.NewRefreshTokenStore(t), mocks.NewSessionStore(t), userGetter, auditLogger, otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.ValidateToken(context.Background(), accessToken)

		require.NoError(t, err)
		assert.True(t, resp.Valid)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, "user@example.com", resp.Email)
		assert.Equal(t, "Pro", resp.Tier)
	})

	t.Run("invalid_token", func(t *testing.T) {
		t.Parallel()

		svc := NewTokenService(newTestJWTService(t), mocks.NewRefreshTokenStore(t), mocks.NewSessionStore(t), mocks.NewUserGetter(t), mocks.NewAuditLogger(t), otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.ValidateToken(context.Background(), "invalid.jwt.token")

		require.NoError(t, err)
		assert.False(t, resp.Valid)
	})

	t.Run("valid_jwt_but_user_deleted", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		jwtSvc := newTestJWTService(t)
		accessToken, _, err := jwtSvc.GenerateTokenPair(context.Background(), userID, "deleted@example.com", "Free")
		require.NoError(t, err)

		userGetter := mocks.NewUserGetter(t)
		userGetter.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)

		svc := NewTokenService(jwtSvc, mocks.NewRefreshTokenStore(t), mocks.NewSessionStore(t), userGetter, mocks.NewAuditLogger(t), otel.GetTracerProvider().Tracer("test"))

		resp, err := svc.ValidateToken(context.Background(), accessToken)

		require.NoError(t, err)
		assert.False(t, resp.Valid)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("with_refresh_and_session", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		tokenHash := model.HashToken("my-refresh")
		storedToken := &model.RefreshToken{ID: uuid.New(), UserID: userID, TokenHash: tokenHash}

		refreshRepo := mocks.NewRefreshTokenStore(t)
		refreshRepo.On("GetByTokenHash", mock.Anything, tokenHash).Return(storedToken, nil)
		refreshRepo.On("Revoke", mock.Anything, storedToken.ID).Return(nil)
		sessionStore := mocks.NewSessionStore(t)
		sessionStore.On("Delete", mock.Anything, "session-123").Return(nil)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), refreshRepo, sessionStore, mocks.NewUserGetter(t), auditLogger, otel.GetTracerProvider().Tracer("test"))

		err := svc.Logout(context.Background(), userID, "my-refresh", "session-123", "127.0.0.1", "agent")

		require.NoError(t, err)
	})

	t.Run("with_empty_refresh_and_session", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), mocks.NewRefreshTokenStore(t), mocks.NewSessionStore(t), mocks.NewUserGetter(t), auditLogger, otel.GetTracerProvider().Tracer("test"))

		err := svc.Logout(context.Background(), userID, "", "", "127.0.0.1", "agent")

		require.NoError(t, err)
	})

	t.Run("refresh_token_not_found_still_succeeds", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()

		refreshRepo := mocks.NewRefreshTokenStore(t)
		refreshRepo.On("GetByTokenHash", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		sessionStore := mocks.NewSessionStore(t)
		sessionStore.On("Delete", mock.Anything, "session-456").Return(nil)
		auditLogger := mocks.NewAuditLogger(t)
		auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewTokenService(newTestJWTService(t), refreshRepo, sessionStore, mocks.NewUserGetter(t), auditLogger, otel.GetTracerProvider().Tracer("test"))

		err := svc.Logout(context.Background(), userID, "unknown-token", "session-456", "127.0.0.1", "")

		require.NoError(t, err)
	})
}

func TestRefreshToken_create_fails(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenHash := model.HashToken("valid-token")
	storedToken := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	user := &model.User{ID: userID, Email: "user@example.com", Tier: "Free"}

	refreshRepo := mocks.NewRefreshTokenStore(t)
	refreshRepo.On("GetByTokenHash", mock.Anything, tokenHash).Return(storedToken, nil)
	refreshRepo.On("Revoke", mock.Anything, storedToken.ID).Return(nil)
	refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil, assert.AnError)
	userGetter := mocks.NewUserGetter(t)
	userGetter.On("GetByID", mock.Anything, userID).Return(user, nil)
	auditLogger := mocks.NewAuditLogger(t)
	auditLogger.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil).Maybe()

	svc := NewTokenService(newTestJWTService(t), refreshRepo, mocks.NewSessionStore(t), userGetter, auditLogger, otel.GetTracerProvider().Tracer("test"))

	_, err := svc.RefreshToken(context.Background(), &dto.RefreshTokenRequest{
		RefreshToken: "valid-token",
	}, "127.0.0.1", "agent")

	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
}
