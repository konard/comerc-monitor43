package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthMiddleware struct {
	authenticator *Authenticator
}

func NewAuthMiddleware(secretKey string) *AuthMiddleware {
	return &AuthMiddleware{
		authenticator: NewAuthenticator(secretKey),
	}
}

func (m *AuthMiddleware) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Пропускаем health check без авторизации — docker health probe
		if info != nil && strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

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

		claims, err := m.authenticator.ValidateToken(ctx, tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)

		return handler(ctx, req)
	}
}

func (m *AuthMiddleware) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

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

		claims, err := m.authenticator.ValidateToken(ctx, tokenString)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          context.WithValue(ctx, UserIDKey, claims.UserID),
		}

		return handler(srv, wrapped)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
