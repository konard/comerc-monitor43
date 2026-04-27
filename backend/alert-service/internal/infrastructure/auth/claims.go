package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Role представляет роль пользователя в системе
type Role string

const (
	RoleGuest Role = "GUEST"
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// IsAdmin проверяет, является ли роль администраторской
func (r Role) IsAdmin() bool {
	return r == RoleAdmin
}

// CanCreateChannels проверяет, может ли роль создавать каналы
func (r Role) CanCreateChannels() bool {
	return r == RoleUser || r == RoleAdmin
}

// Claims представляют данные из JWT токена.
// Встраивает jwt.RegisteredClaims для корректной сериализации exp/iat в стандартном формате (Unix timestamp).
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
	Role   Role      `json:"role"`
}

type userIDContextKey struct{}
type roleContextKey struct{}

var (
	UserIDKey = &userIDContextKey{}
	RoleKey   = &roleContextKey{}
)

// ExtractUserID извлекает user ID из контекста
func ExtractUserID(ctx context.Context) uuid.UUID {
	userID := ctx.Value(UserIDKey)
	if userID == nil {
		return uuid.Nil
	}
	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// GetUserID возвращает user ID из контекста или ошибку аутентификации
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID := ctx.Value(UserIDKey)
	if userID == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return id, nil
}

// ExtractRole извлекает роль пользователя из контекста
func ExtractRole(ctx context.Context) Role {
	role := ctx.Value(RoleKey)
	if role == nil {
		return RoleGuest
	}
	r, ok := role.(Role)
	if !ok {
		return RoleGuest
	}
	return r
}

// RequireAdmin проверяет, что пользователь имеет роль ADMIN
func RequireAdmin(ctx context.Context) error {
	role := ExtractRole(ctx)
	if !role.IsAdmin() {
		return status.Error(codes.PermissionDenied, "admin access required")
	}
	return nil
}

// RequireChannelAccess проверяет, что пользователь имеет доступ к каналу
func RequireChannelAccess(ctx context.Context, channelOwnerID uuid.UUID) error {
	userID := ExtractUserID(ctx)
	if userID == uuid.Nil {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	role := ExtractRole(ctx)
	if role.IsAdmin() {
		return nil
	}
	if userID != channelOwnerID {
		return status.Error(codes.PermissionDenied, "access denied")
	}
	return nil
}
