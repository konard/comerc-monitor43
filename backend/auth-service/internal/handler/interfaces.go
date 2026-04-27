package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/infrastructure/jwt"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
)

// authServicer определяет методы аутентификации, используемые хэндлером.
type authServicer interface {
	OAuthLogin(ctx context.Context, provider, redirectURI string) (string, string, error)
	OAuthCallback(ctx context.Context, req *dto.OAuthCallbackRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)
	PasswordLogin(ctx context.Context, req *dto.PasswordLoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)
	Register(ctx context.Context, req *dto.RegisterRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)
}

// tokenServicer определяет методы работы с токенами, используемые хэндлером.
type tokenServicer interface {
	RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)
	ValidateToken(ctx context.Context, accessToken string) (*dto.ValidateTokenResponse, error)
	Logout(ctx context.Context, userID uuid.UUID, refreshToken, sessionID, ipAddress, userAgent string) error
}

// sessionServicer определяет методы управления сессиями, используемые хэндлером.
type sessionServicer interface {
	ListSessions(ctx context.Context, req *dto.ListSessionsRequest) (*dto.ListSessionsResponse, error)
	RevokeSession(ctx context.Context, req *dto.RevokeSessionRequest) error
}

// userServicer определяет методы управления пользователями, используемые хэндлером.
type userServicer interface {
	GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.GetUserResponse, error)
	UpdateUser(ctx context.Context, req *dto.UpdateUserRequest) (*dto.GetUserResponse, error)
	LockAccount(ctx context.Context, req *dto.LockAccountRequest) error
	UnlockAccount(ctx context.Context, req *dto.UnlockAccountRequest) error
}

// jwtValidator определяет минимальный контракт JWT для session и user хэндлеров.
type jwtValidator interface {
	ValidateAccessToken(ctx context.Context, tokenStr string) (*jwt.Claims, error)
}
