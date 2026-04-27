package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// GetUserRequest represents a get user request
type GetUserRequest struct {
	UserID uuid.UUID
}

// LockAccountRequest represents a lock account request
type LockAccountRequest struct {
	UserID        uuid.UUID
	DurationHours int64
}

// UnlockAccountRequest represents an unlock account request
type UnlockAccountRequest struct {
	UserID uuid.UUID
}

// UpdateUserRequest содержит данные для обновления профиля пользователя.
type UpdateUserRequest struct {
	UserID   uuid.UUID
	Email    string
	FullName string
}

// GetUserResponse represents a get user response
type GetUserResponse struct {
	ID          uuid.UUID
	Email       string
	FullName    string
	Tier        string
	CreatedAt   time.Time
	LastLoginAt *time.Time
}

// ToGetUserResponse converts a User model to GetUserResponse
func ToGetUserResponse(user *model.User) *GetUserResponse {
	return &GetUserResponse{
		ID:          user.ID,
		Email:       user.Email,
		FullName:    user.FullName,
		Tier:        user.Tier,
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}
