// Package handler предоставляет gRPC обработчики для сервиса алертов.
//
// Основные компоненты:
//   - AlertServiceServer: Реализация gRPC сервиса AlertService
//   - Конвертеры: Преобразования между protobuf и domain моделями
//   - Interceptors: gRPC middleware для аутентификации, трейсинга и метрик
//
// Использование:
//
//	server := handler.NewAlertServiceServer(alertService, channelService, ruleService, authMiddleware)
//	alertproto.RegisterAlertServiceServer(gRPCServer, server)
//
// Ключевые особенности:
//   - Делегирование бизнес-логики в сервисный слой
//   - Валидация входных данных с детализированными ошибками
//   - Извлечение user ID из контекста аутентификации
//   - Интеграция с OpenTelemetry для трейсинга
//
// Контекст DOD:
//   - Соответствует требованию DOD 2.4 gRPC Patterns
//   - Ссылки на use cases из feature files
//   - Структурированная обработка ошибок с gRPC status codes
package handler
