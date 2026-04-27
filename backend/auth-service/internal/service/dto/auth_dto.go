package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// PasswordLoginRequest represents a password login request
type PasswordLoginRequest struct {
	Email    string
	Password string
}

// OAuthCallbackRequest represents an OAuth callback request
type OAuthCallbackRequest struct {
	Code        string
	State       string
	RedirectURI string
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string
	Password string
	FullName string
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	RefreshToken string
	SessionID    string
}

// ValidateTokenRequest represents a token validation request
type ValidateTokenRequest struct {
	AccessToken string
}

// ValidateTokenResponse represents a token validation response
type ValidateTokenResponse struct {
	Valid     bool
	UserID    string
	Email     string
	Tier      string
	ExpiresAt *time.Time
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
	User         *UserDTO
}

// UserDTO represents a user data transfer object
type UserDTO struct {
	ID          uuid.UUID
	Email       string
	FullName    string
	Tier        string
	CreatedAt   time.Time
	LastLoginAt *time.Time
}

// ToUserDTO converts a User model to UserDTO
func ToUserDTO(user *model.User) *UserDTO {
	return &UserDTO{
		ID:          user.ID,
		Email:       user.Email,
		FullName:    user.FullName,
		Tier:        user.Tier,
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}
