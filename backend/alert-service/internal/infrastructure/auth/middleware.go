package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthMiddleware валидирует аутентификацию для gRPC
type AuthMiddleware struct {
	authenticator *Authenticator
	jwtSecret     string
}

// NewAuthMiddleware создаёт новый middleware для аутентификации
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		authenticator: NewAuthenticator(jwtSecret),
		jwtSecret:     jwtSecret,
	}
}

// UnaryInterceptor создаёт gRPC unary interceptor для аутентификации
func (m *AuthMiddleware) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Пропускаем health check без авторизации — docker health probe
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		// Извлекаем JWT токен из метаданных gRPC запроса
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Ищем токен в заголовке authorization
		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Парсим токен (формат: "Bearer <token>")
		var tokenString string
		for _, h := range authHeader {
			if len(h) > 7 && h[:7] == "Bearer " {
				tokenString = h[7:]
				break
			}
		}

		if tokenString == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Валидируем токен и извлекаем claims
		claims, err := m.authenticator.ValidateToken(ctx, tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)

		return handler(ctx, req)
	}
}

// StreamInterceptor создаёт gRPC stream interceptor для аутентификации
func (m *AuthMiddleware) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Извлекаем JWT токен из метаданных gRPC запроса
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Ищем токен в заголовке authorization
		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Парсим токен (формат: "Bearer <token>")
		var tokenString string
		for _, h := range authHeader {
			if len(h) > 7 && h[:7] == "Bearer " {
				tokenString = h[7:]
				break
			}
		}

		if tokenString == "" {
			return status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Валидируем токен и извлекаем claims
		claims, err := m.authenticator.ValidateToken(ctx, tokenString)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		// Добавляем user ID в контекст
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          context.WithValue(context.WithValue(ctx, UserIDKey, claims.UserID), RoleKey, claims.Role),
		}

		return handler(srv, wrapped)
	}
}

// wrappedStream оборачивает grpc.ServerStream для обновления контекста
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
