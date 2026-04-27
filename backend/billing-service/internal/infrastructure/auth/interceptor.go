package auth

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// jwtClaims соответствует формату токена auth-service.
type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	jwt.RegisteredClaims
}

// JWTInterceptor перехватывает gRPC запросы и валидирует JWT токены.
type JWTInterceptor struct {
	secretKey string
}

// NewJWTInterceptor создаёт новый JWT interceptor.
func NewJWTInterceptor(secretKey string) *JWTInterceptor {
	return &JWTInterceptor{
		secretKey: secretKey,
	}
}

// Unary возвращает unary interceptor для gRPC.
func (i *JWTInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Пропускаем health check без авторизации — docker health probe
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		// Извлекаем user ID из контекста (после валидации JWT)
		userID, err := i.extractUserID(ctx)
		if err != nil {
			return nil, err
		}

		// Добавляем user ID в контекст
		ctx = context.WithValue(ctx, userIDKey{}, userID)

		// Вызываем следующий handler
		return handler(ctx, req)
	}
}

// extractUserID извлекает и валидирует user ID из JWT токена.
func (i *JWTInterceptor) extractUserID(ctx context.Context) (string, error) {
	// Получаем metadata из контекста
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata not found")
	}

	// Извлекаем authorization header
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization token not found")
	}

	token := values[0]
	if len(token) < 7 || token[:7] != "Bearer " {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	jwtToken := token[7:]

	userID, err := i.validateToken(jwtToken)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}

	return userID, nil
}

// validateToken валидирует JWT токен и возвращает user ID.
func (i *JWTInterceptor) validateToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("invalid token")
	}

	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(i.secretKey), nil
	})
	if err != nil || !token.Valid {
		return "", errors.Wrap(err, "invalid token")
	}
	if claims.UserID == "" {
		return "", errors.New("missing user_id in token")
	}

	return claims.UserID, nil
}

// GetUserID извлекает user ID из контекста.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey{}).(string)
	return userID, ok
}

// userIDKey тип ключа для хранения user ID в контексте.
type userIDKey struct{}

// RequireAuth это helper для handlers, который требует аутентификации.
func RequireAuth(ctx context.Context) (string, error) {
	userID, ok := GetUserID(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "user not authenticated")
	}
	return userID, nil
}

// ContextWithUserID добавляет user ID в контекст. Используется в тестах.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// MockAuthInterceptor это mock interceptor для тестирования.
// Он извлекает user ID из metadata без реальной валидации.
func MockAuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return handler(ctx, req)
	}

	values := md.Get("user-id")
	if len(values) == 0 {
		return handler(ctx, req)
	}

	userID := values[0]
	ctx = context.WithValue(ctx, userIDKey{}, userID)

	return handler(ctx, req)
}
