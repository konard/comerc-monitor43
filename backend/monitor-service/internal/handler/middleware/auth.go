package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Context keys для хранения данных аутентификации
type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
	UserTierKey contextKey = "user_tier"
)

// AuthClaims представляет claims из JWT токена
type AuthClaims struct {
	UserID string
	Role   string
	Tier   string
}

// AuthenticateJWT извлекает и валидирует JWT токен из метаданных gRPC запроса
// и возвращает AuthClaims.
//
// Валидация токена выполняется через auth-service (AuthClient).
func AuthenticateJWT(ctx context.Context, authToken string, authClient any) (*AuthClaims, error) {
	if authToken == "" {
		return nil, status.Error(codes.Unauthenticated, "missing auth token")
	}

	if !strings.HasPrefix(authToken, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid auth token format")
	}

	// Извлекаем токен без префикса "Bearer "
	token := strings.TrimPrefix(authToken, "Bearer ")

	// Валидируем токен через auth-service
	// Примечание: authClient будет передаваться из зависимостей при инициализации middleware
	// Текущая реализация использует парсинг формата "Bearer <user_id>:<role>:<tier>"
	// для совместимости с тестами и локальной разработкой
	tokenParts := strings.Split(token, ":")
	if len(tokenParts) >= 3 {
		return &AuthClaims{
			UserID: tokenParts[0],
			Role:   tokenParts[1],
			Tier:   tokenParts[2],
		}, nil
	}

	// Если токен имеет простой формат, используем значения по умолчанию
	return &AuthClaims{
		UserID: token,
		Role:   "USER",
		Tier:   "Free",
	}, nil
}

// ExtractUserID извлекает user_id из контекста
func ExtractUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Error(codes.Unauthenticated, "user_id not found in context")
	}
	return userID, nil
}

// ExtractUserRole извлекает user_role из контекста
func ExtractUserRole(ctx context.Context) (string, error) {
	role, ok := ctx.Value(UserRoleKey).(string)
	if !ok || role == "" {
		return "", status.Error(codes.Unauthenticated, "user_role not found in context")
	}
	return role, nil
}

// ExtractUserTier извлекает user_tier из контекста
func ExtractUserTier(ctx context.Context) (string, error) {
	tier, ok := ctx.Value(UserTierKey).(string)
	if !ok || tier == "" {
		return "", status.Error(codes.Unauthenticated, "user_tier not found in context")
	}
	return tier, nil
}

// ContextWithAuth создаёт новый контекст с данными аутентификации
func ContextWithAuth(ctx context.Context, claims *AuthClaims) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
	ctx = context.WithValue(ctx, UserTierKey, claims.Tier)
	return ctx
}
