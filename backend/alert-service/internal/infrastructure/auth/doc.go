// Package auth предоставляет JWT аутентификацию и gRPC middleware.
//
// Основные компоненты:
//   - Authenticator: Валидация JWT токенов и извлечение claims
//   - Claims: Структура данных из JWT токена с методами интерфейса
//   - AuthMiddleware: gRPC interceptors для аутентификации запросов
//   - ContextKey: Контекстный ключ для хранения user ID
//   - ExtractUserID: Функция извлечения user ID из контекста
//
// Использование:
//
//	authenticator := auth.NewAuthenticator(jwtSecret)
//	authMiddleware := auth.NewAuthMiddleware(jwtSecret)
//	grpcServer := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(authMiddleware.UnaryInterceptor()),
//	)
//
// Ключевые особенности:
//   - JWT токен валидация с проверкой подписи и срока действия
//   - Bearer token extraction из gRPC metadata
//   - User ID propagation в контекст для использования в обработчиках
//   - Поддержка как unary так и stream gRPC requests
//
// Контекст DOD:
//   - Соответствует требованию DOD 7.2 Authentication and Authorization
//   - JWT Authentication implementation с proper validation
//   - gRPC interceptors для request processing
//   - RBAC авторизация через user context
package auth
