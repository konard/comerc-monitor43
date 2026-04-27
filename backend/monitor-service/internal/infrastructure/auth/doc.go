// Package auth предоставляет интеграцию с auth-service для валидации JWT токенов.
//
// Клиент:
//   - AuthClient: gRPC клиент для валидации токенов
//
// Использование:
//   - Создаётся при старте сервиса через NewAuthClient()
//   - Передаётся в grpc handlers для извлечения user_id из JWT
//   - Закрывается при shutdown сервиса
//
// Контракт:
//   - ValidateToken() возвращает user_id или ошибку Unauthenticated
//   - Все ошибки оборачиваются с контекстом
package auth
