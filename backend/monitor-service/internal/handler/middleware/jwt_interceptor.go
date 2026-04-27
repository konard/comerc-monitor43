package middleware

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// jwtClaims совместим с токенами, выдаваемыми auth-service.
type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTInterceptor валидирует JWT токены из gRPC metadata и добавляет claims в контекст.
type JWTInterceptor struct {
	secret []byte
}

// NewJWTInterceptor создаёт новый JWT interceptor.
func NewJWTInterceptor(secret string) *JWTInterceptor {
	return &JWTInterceptor{secret: []byte(secret)}
}

// Unary возвращает unary gRPC server interceptor.
func (i *JWTInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Пропускаем health check без авторизации — docker health probe
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		var tokenString string
		for _, h := range authHeader {
			if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
				tokenString = h[7:]
				break
			}
		}
		if tokenString == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		claims := &jwtClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, status.Error(codes.Unauthenticated, "unexpected signing method")
			}
			return i.secret, nil
		})
		if err != nil || !token.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		if claims.UserID == "" {
			return nil, status.Error(codes.Unauthenticated, "missing user_id in token")
		}

		role := claims.Role
		if role == "" {
			role = "USER"
		}
		tier := claims.Tier
		if tier == "" {
			tier = "Free"
		}

		ctx = ContextWithAuth(ctx, &AuthClaims{
			UserID: claims.UserID,
			Role:   role,
			Tier:   tier,
		})

		return handler(ctx, req)
	}
}
