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
	"golang.org/x/crypto/bcrypt"

	"github.com/raul/monitor/backend/auth-service/internal/infrastructure/jwt"
	"github.com/raul/monitor/backend/auth-service/internal/infrastructure/oauth"
	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/internal/service/mocks"
	apperrors "github.com/raul/monitor/backend/auth-service/pkg/errors"
)

func newTestJWTService(t *testing.T) *jwt.TokenService {
	t.Helper()
	svc, err := jwt.NewTokenService("test-secret-key-min-32-chars-long", 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)
	return svc
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

func TestPasswordLogin(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		user := model.NewUser("user@example.com", hashPassword(t, "secret123"), "Test User", "Free")

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
		userRepo.On("ResetLoginAttempts", mock.Anything, user.ID).Return(nil)
		userRepo.On("UpdateLastLogin", mock.Anything, user.ID).Return(nil)
		refreshRepo := mocks.NewRefreshTokenRepo(t)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(&model.RefreshToken{}, nil)
		sessionCreator := mocks.NewSessionCreator(t)
		sessionCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.Session")).Return(nil)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)
		pub := mocks.NewEventPublisher(t)
		pub.On("PublishUserLoggedIn", mock.Anything, user.ID.String(), user.Email, "127.0.0.1", "test-agent").Return(nil)
		pub.On("PublishSessionCreated", mock.Anything, user.ID.String(), user.Email, mock.Anything, mock.Anything).Return(nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), refreshRepo, sessionCreator, auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t),
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.PasswordLogin(context.Background(), &dto.PasswordLoginRequest{
			Email: "user@example.com", Password: "secret123",
		}, "127.0.0.1", "test-agent")

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.NotNil(t, resp.User)
		assert.Equal(t, user.Email, resp.User.Email)
	})

	t.Run("user_not_found", func(t *testing.T) {
		t.Parallel()

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "missing@example.com").Return(nil, assert.AnError)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		resp, err := svc.PasswordLogin(context.Background(), &dto.PasswordLoginRequest{
			Email: "missing@example.com", Password: "secret123",
		}, "127.0.0.1", "test-agent")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInvalidCredentials, appErr.Code)
	})

	t.Run("account_locked", func(t *testing.T) {
		t.Parallel()

		lockedUntil := time.Now().Add(30 * time.Minute)
		user := &model.User{ID: uuid.New(), Email: "locked@example.com", PasswordHash: hashPassword(t, "pass"), LockedUntil: &lockedUntil}

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "locked@example.com").Return(user, nil)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		resp, err := svc.PasswordLogin(context.Background(), &dto.PasswordLoginRequest{
			Email: "locked@example.com", Password: "pass",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeAccountLocked, appErr.Code)
	})

	t.Run("wrong_password_should_lock", func(t *testing.T) {
		t.Parallel()

		user := &model.User{ID: uuid.New(), Email: "user@example.com", PasswordHash: hashPassword(t, "correct"), LoginAttempts: 5}

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
		userRepo.On("IncrementLoginAttempts", mock.Anything, user.ID).Return(nil)
		userRepo.On("LockAccount", mock.Anything, user.ID, mock.Anything).Return(nil)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)
		pub := mocks.NewEventPublisher(t)
		pub.On("PublishUserLocked", mock.Anything, user.ID.String(), user.Email, 5).Return(nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.PasswordLogin(context.Background(), &dto.PasswordLoginRequest{
			Email: "user@example.com", Password: "wrong",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeAccountLocked, appErr.Code)
	})

	t.Run("wrong_password_no_lock", func(t *testing.T) {
		t.Parallel()

		user := &model.User{ID: uuid.New(), Email: "user@example.com", PasswordHash: hashPassword(t, "correct"), LoginAttempts: 2}

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
		userRepo.On("IncrementLoginAttempts", mock.Anything, user.ID).Return(nil)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)
		pub := mocks.NewEventPublisher(t)
		pub.On("PublishUserFailedLogin", mock.Anything, user.ID.String(), user.Email, "127.0.0.1", "invalid password").Return(nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.PasswordLogin(context.Background(), &dto.PasswordLoginRequest{
			Email: "user@example.com", Password: "wrong",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInvalidCredentials, appErr.Code)
	})

	t.Run("reset_login_attempts_fails", func(t *testing.T) {
		t.Parallel()

		user := model.NewUser("user@example.com", hashPassword(t, "secret123"), "Test User", "Free")

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(user, nil)
		userRepo.On("ResetLoginAttempts", mock.Anything, user.ID).Return(assert.AnError)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		resp, err := svc.PasswordLogin(context.Background(), &dto.PasswordLoginRequest{
			Email: "user@example.com", Password: "secret123",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})
}

func TestOAuthLogin(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		provider := mocks.NewOAuthProvider(t)
		provider.On("GetAuthURL", mock.Anything, "https://callback.example.com").Return("https://accounts.google.com/oauth?state=abc")
		stateStore := mocks.NewStateStore(t)
		stateStore.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
			return len(key) > 11
		}), mock.Anything).Return(nil)

		svc := NewAuthService(
			mocks.NewUserRepo(t), mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), mocks.NewAuditCreator(t),
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		url, state, err := svc.OAuthLogin(context.Background(), "google", "https://callback.example.com")

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.NotEmpty(t, state)
	})

	t.Run("unsupported_provider", func(t *testing.T) {
		t.Parallel()

		svc := NewAuthService(
			mocks.NewUserRepo(t), mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), mocks.NewAuditCreator(t),
			newTestJWTService(t), map[string]OAuthProvider{}, mocks.NewStateStore(t),
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, _, err := svc.OAuthLogin(context.Background(), "facebook", "https://callback.example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})

	t.Run("state_store_failure", func(t *testing.T) {
		t.Parallel()

		provider := mocks.NewOAuthProvider(t)
		stateStore := mocks.NewStateStore(t)
		stateStore.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

		svc := NewAuthService(
			mocks.NewUserRepo(t), mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), mocks.NewAuditCreator(t),
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, _, err := svc.OAuthLogin(context.Background(), "google", "https://callback.example.com")

		assert.Error(t, err)
	})
}

func TestOAuthCallback(t *testing.T) {
	t.Parallel()

	t.Run("state_mismatch", func(t *testing.T) {
		t.Parallel()

		stateStore := mocks.NewStateStore(t)
		stateStore.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, assert.AnError)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			mocks.NewUserRepo(t), mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "bad-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeOAuthStateMismatch, appErr.Code)
	})

	t.Run("unknown_provider", func(t *testing.T) {
		t.Parallel()

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "unknownprovider", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			mocks.NewUserRepo(t), mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeOAuthFailed, appErr.Code)
	})

	t.Run("exchange_fails", func(t *testing.T) {
		t.Parallel()

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "bad-code", "https://example.com/cb").
			Return(nil, assert.AnError)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			mocks.NewUserRepo(t), mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "bad-code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeOAuthFailed, appErr.Code)
	})

	t.Run("existing_oauth_account", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{
			ID:    userID,
			Email: "google-user@gmail.com",
			Tier:  "Free",
		}
		oauthAccount := &model.OAuthAccount{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "g-123",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "valid-code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID:             "g-123",
				Email:          "google-user@gmail.com",
				Name:           "Google User",
				Provider:       "google",
				ProviderUserID: "g-123",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-123").Return(oauthAccount, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, userID).Return(nil)

		refreshRepo := mocks.NewRefreshTokenRepo(t)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(&model.RefreshToken{}, nil)

		sessionCreator := mocks.NewSessionCreator(t)
		sessionCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.Session")).Return(nil)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		pub := mocks.NewEventPublisher(t)
		pub.On("PublishSessionCreated", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		svc := NewAuthService(
			userRepo, oauthRepo, refreshRepo, sessionCreator, auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "valid-code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "google-user@gmail.com", resp.User.Email)
	})

	t.Run("new_user_created", func(t *testing.T) {
		t.Parallel()

		newUserID := uuid.New()
		newUser := &model.User{
			ID:    newUserID,
			Email: "newuser@gmail.com",
			Tier:  "Free",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID:             "g-new",
				Email:          "newuser@gmail.com",
				Name:           "New User",
				ProviderUserID: "g-new",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-new").Return(nil, assert.AnError)
		oauthRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.OAuthAccount")).Return(&model.OAuthAccount{}, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "newuser@gmail.com").Return(nil, assert.AnError)
		userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(newUser, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, newUserID).Return(nil)

		refreshRepo := mocks.NewRefreshTokenRepo(t)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(&model.RefreshToken{}, nil)

		sessionCreator := mocks.NewSessionCreator(t)
		sessionCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.Session")).Return(nil)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		pub := mocks.NewEventPublisher(t)
		pub.On("PublishUserCreated", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		pub.On("PublishSessionCreated", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		svc := NewAuthService(
			userRepo, oauthRepo, refreshRepo, sessionCreator, auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
	})

	t.Run("link_existing_user", func(t *testing.T) {
		t.Parallel()

		existingUserID := uuid.New()
		existingUser := &model.User{
			ID:    existingUserID,
			Email: "existing@example.com",
			Tier:  "Pro",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID:             "g-link",
				Email:          "existing@example.com",
				Name:           "Existing User",
				ProviderUserID: "g-link",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-link").Return(nil, assert.AnError)
		oauthRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.OAuthAccount")).Return(&model.OAuthAccount{}, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, existingUserID).Return(nil)

		refreshRepo := mocks.NewRefreshTokenRepo(t)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(&model.RefreshToken{}, nil)

		sessionCreator := mocks.NewSessionCreator(t)
		sessionCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.Session")).Return(nil)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		pub := mocks.NewEventPublisher(t)
		pub.On("PublishOAuthAccountLinked", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		pub.On("PublishSessionCreated", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		svc := NewAuthService(
			userRepo, oauthRepo, refreshRepo, sessionCreator, auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "existing@example.com", resp.User.Email)
	})

	t.Run("locked_account", func(t *testing.T) {
		t.Parallel()

		lockedUntil := time.Now().Add(time.Hour)
		userID := uuid.New()
		lockedUser := &model.User{
			ID:          userID,
			Email:       "locked@gmail.com",
			LockedUntil: &lockedUntil,
		}
		oauthAccount := &model.OAuthAccount{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "g-locked",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID: "g-locked", Email: "locked@gmail.com", ProviderUserID: "g-locked",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-locked").Return(oauthAccount, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByID", mock.Anything, userID).Return(lockedUser, nil)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, oauthRepo, mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeAccountLocked, appErr.Code)
	})

	t.Run("get_user_by_id_fails", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		oauthAccount := &model.OAuthAccount{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "g-xyz",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID: "g-xyz", Email: "user@gmail.com", ProviderUserID: "g-xyz",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-xyz").Return(oauthAccount, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, oauthRepo, mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})

	t.Run("update_last_login_fails", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{ID: userID, Email: "user@gmail.com", Tier: "Free"}
		oauthAccount := &model.OAuthAccount{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "g-upd",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID: "g-upd", Email: "user@gmail.com", ProviderUserID: "g-upd",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-upd").Return(oauthAccount, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, userID).Return(assert.AnError)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, oauthRepo, mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})

	t.Run("link_oauth_create_fails", func(t *testing.T) {
		t.Parallel()

		existingUser := &model.User{ID: uuid.New(), Email: "link@example.com", Tier: "Free"}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID: "g-link2", Email: "link@example.com", ProviderUserID: "g-link2",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-link2").Return(nil, assert.AnError)
		oauthRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.OAuthAccount")).Return(nil, assert.AnError)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByEmail", mock.Anything, "link@example.com").Return(existingUser, nil)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, oauthRepo, mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})

	t.Run("refresh_token_create_fails", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{ID: userID, Email: "user@gmail.com", Tier: "Free"}
		oauthAccount := &model.OAuthAccount{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "g-rt",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID: "g-rt", Email: "user@gmail.com", ProviderUserID: "g-rt",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-rt").Return(oauthAccount, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, userID).Return(nil)

		refreshRepo := mocks.NewRefreshTokenRepo(t)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil, assert.AnError)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, oauthRepo, refreshRepo, mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})

	t.Run("session_create_fails", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New()
		user := &model.User{ID: userID, Email: "user@gmail.com", Tier: "Free"}
		oauthAccount := &model.OAuthAccount{
			ID:             uuid.New(),
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "g-sess",
		}

		stateStore := mocks.NewStateStore(t)
		stateData := map[string]string{"provider": "google", "redirect_uri": "https://example.com/cb"}
		stateStore.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
		stateStore.On("Delete", mock.Anything, mock.Anything).Return(nil)

		provider := mocks.NewOAuthProvider(t)
		provider.On("Exchange", mock.Anything, "code", "https://example.com/cb").
			Return(&oauth.UserInfo{
				ID: "g-sess", Email: "user@gmail.com", ProviderUserID: "g-sess",
			}, nil)

		oauthRepo := mocks.NewOauthRepo(t)
		oauthRepo.On("GetByProviderUserID", mock.Anything, "google", "g-sess").Return(oauthAccount, nil)

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		userRepo.On("UpdateLastLogin", mock.Anything, userID).Return(nil)

		refreshRepo := mocks.NewRefreshTokenRepo(t)
		refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(&model.RefreshToken{}, nil)

		sessionCreator := mocks.NewSessionCreator(t)
		sessionCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.Session")).Return(assert.AnError)

		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)

		svc := NewAuthService(
			userRepo, oauthRepo, refreshRepo, sessionCreator, auditCreator,
			newTestJWTService(t), map[string]OAuthProvider{"google": provider}, stateStore,
			5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		_, err := svc.OAuthCallback(context.Background(), &dto.OAuthCallbackRequest{
			Code: "code", State: "valid-state",
		}, "127.0.0.1", "Chrome")

		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		createdUser := model.NewUser("new@example.com", "hash", "New User", "Free")

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("ExistsByEmail", mock.Anything, "new@example.com").Return(false, nil)
		userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(createdUser, nil)
		auditCreator := mocks.NewAuditCreator(t)
		auditCreator.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(&model.AuditLog{}, nil)
		pub := mocks.NewEventPublisher(t)
		pub.On("PublishUserCreated", mock.Anything, createdUser.ID.String(), createdUser.Email).Return(nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), auditCreator,
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"), WithPublisher(pub),
		)

		resp, err := svc.Register(context.Background(), &dto.RegisterRequest{
			Email: "new@example.com", Password: "secret123", FullName: "New User",
		}, "127.0.0.1", "test-agent")

		require.NoError(t, err)
		assert.NotNil(t, resp.User)
		assert.Equal(t, "new@example.com", resp.User.Email)
	})

	t.Run("user_already_exists", func(t *testing.T) {
		t.Parallel()

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("ExistsByEmail", mock.Anything, "existing@example.com").Return(true, nil)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), mocks.NewAuditCreator(t),
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		resp, err := svc.Register(context.Background(), &dto.RegisterRequest{
			Email: "existing@example.com", Password: "secret123", FullName: "Existing",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeUserExists, appErr.Code)
	})

	t.Run("exists_by_email_repo_error", func(t *testing.T) {
		t.Parallel()

		userRepo := mocks.NewUserRepo(t)
		userRepo.On("ExistsByEmail", mock.Anything, "error@example.com").Return(false, assert.AnError)

		svc := NewAuthService(
			userRepo, mocks.NewOauthRepo(t), mocks.NewRefreshTokenRepo(t), mocks.NewSessionCreator(t), mocks.NewAuditCreator(t),
			newTestJWTService(t), nil, mocks.NewStateStore(t), 5, 30*time.Minute, otel.GetTracerProvider().Tracer("test"),
		)

		resp, err := svc.Register(context.Background(), &dto.RegisterRequest{
			Email: "error@example.com", Password: "secret123", FullName: "Error",
		}, "127.0.0.1", "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		var appErr *apperrors.AppError
		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrorCodeInternalError, appErr.Code)
	})
}
