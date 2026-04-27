// Package auth предоставляет JWT аутентификацию для gRPC и WebSocket.
//
// Использование:
//
//	authenticator := auth.NewAuthenticator(secretKey)
//	claims, err := authenticator.ValidateToken(ctx, tokenString)
//
//	middleware := auth.NewAuthMiddleware(jwtSecret)
//	grpc.ChainUnaryInterceptor(middleware.UnaryInterceptor())
//
// Переменные окружения:
//   - JWT_SECRET: секретный ключ для валидации JWT токенов
package auth
