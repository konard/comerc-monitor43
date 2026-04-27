package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/auth-service/internal/handler"
	"github.com/raul/monitor/backend/auth-service/internal/handler/mocks"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
)

func TestGoogleOAuthLogin(t *testing.T) {
	t.Parallel()

	authSvc := mocks.NewAuthServicer(t)
	tokenSvc := mocks.NewTokenServicer(t)
	h := handler.NewAuthHandler(authSvc, tokenSvc)

	authSvc.On("OAuthLogin", mock.Anything, "google", "http://example.com/cb").
		Return("https://oauth.google.com/auth?state=abc", "abc", nil)

	resp, err := h.GoogleOAuthLogin(context.Background(), &authv1.OAuthLoginRequest{
		RedirectUri: "http://example.com/cb",
	})
	require.NoError(t, err)
	assert.Equal(t, "abc", resp.State)
	assert.Equal(t, "https://oauth.google.com/auth?state=abc", resp.AuthUrl)
}

func TestPasswordLogin(t *testing.T) {
	t.Parallel()

	authSvc := mocks.NewAuthServicer(t)
	tokenSvc := mocks.NewTokenServicer(t)
	h := handler.NewAuthHandler(authSvc, tokenSvc)

	userID := uuid.New()
	now := time.Now()

	authSvc.On("PasswordLogin",
		mock.Anything,
		mock.MatchedBy(func(req *dto.PasswordLoginRequest) bool {
			return req.Email == "user@example.com" && req.Password == "secret"
		}),
		mock.Anything,
		mock.Anything,
	).Return(&dto.AuthResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User: &dto.UserDTO{
			ID:        userID,
			Email:     "user@example.com",
			FullName:  "Test User",
			Tier:      "free",
			CreatedAt: now,
		},
	}, nil)

	resp, err := h.PasswordLogin(context.Background(), &authv1.PasswordLoginRequest{
		Email:    "user@example.com",
		Password: "secret",
	})
	require.NoError(t, err)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	assert.Equal(t, userID.String(), resp.User.Id)
}

func TestRegister(t *testing.T) {
	t.Parallel()

	authSvc := mocks.NewAuthServicer(t)
	tokenSvc := mocks.NewTokenServicer(t)
	h := handler.NewAuthHandler(authSvc, tokenSvc)

	userID := uuid.New()
	now := time.Now()

	authSvc.On("Register",
		mock.Anything,
		mock.MatchedBy(func(req *dto.RegisterRequest) bool {
			return req.Email == "new@example.com" && req.FullName == "New User"
		}),
		mock.Anything,
		mock.Anything,
	).Return(&dto.AuthResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User: &dto.UserDTO{
			ID:        userID,
			Email:     "new@example.com",
			FullName:  "New User",
			Tier:      "free",
			CreatedAt: now,
		},
	}, nil)

	resp, err := h.Register(context.Background(), &authv1.RegisterRequest{
		Email:    "new@example.com",
		Password: "password123",
		FullName: "New User",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, userID.String(), resp.User.Id)
}

func TestLogout(t *testing.T) {
	t.Parallel()

	authSvc := mocks.NewAuthServicer(t)
	tokenSvc := mocks.NewTokenServicer(t)
	h := handler.NewAuthHandler(authSvc, tokenSvc)

	tokenSvc.On("Logout",
		mock.Anything,
		uuid.Nil,
		"refresh-token-xyz",
		"session-id-123",
		mock.Anything,
		mock.Anything,
	).Return(nil)

	resp, err := h.Logout(context.Background(), &authv1.LogoutRequest{
		RefreshToken: "refresh-token-xyz",
		SessionId:    "session-id-123",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestValidateToken(t *testing.T) {
	t.Parallel()

	authSvc := mocks.NewAuthServicer(t)
	tokenSvc := mocks.NewTokenServicer(t)
	h := handler.NewAuthHandler(authSvc, tokenSvc)

	userID := uuid.New()

	tokenSvc.On("ValidateToken", mock.Anything, "valid-token").
		Return(&dto.ValidateTokenResponse{
			Valid:  true,
			UserID: userID.String(),
			Email:  "user@example.com",
			Tier:   "pro",
		}, nil)

	resp, err := h.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{
		AccessToken: "valid-token",
	})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, userID.String(), resp.UserId)
	assert.Equal(t, "user@example.com", resp.Email)
}
