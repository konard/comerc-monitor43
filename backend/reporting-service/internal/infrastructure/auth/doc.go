// Package auth предоставляет JWT аутентификацию и gRPC middleware.
//
// Основные компоненты:
//   - Authenticator: Валидация JWT токенов и извлечение claims
//   - Claims: Структура данных из JWT токена с методами интерфейса
//   - AuthMiddleware: gRPC interceptors для аутентификации запросов
//   - UserIDKey: Контекстный ключ для хранения user ID
//   - ExtractUserID: Функция извлечения user ID из контекста
//   - GetUserID: Функция извлечения user ID с проверкой аутентификации
//
// Использование:
//
//	authMiddleware := auth.NewAuthMiddleware(jwtSecret)
//	grpcServer := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(authMiddleware.UnaryInterceptor()),
//	)
//
// Ключевые особенности:
//   - JWT токен валидация с проверкой подписи HS256 и срока действия
//   - Bearer token extraction из gRPC metadata
//   - User ID propagation в контекст для использования в обработчиках
//   - Поддержка как unary так и stream gRPC requests
//
// Контекст DOD:
//   - Соответствует требованию DOD 7.2 Authentication and Authorization
package auth
