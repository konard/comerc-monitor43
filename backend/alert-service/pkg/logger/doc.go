// Package logger предоставляет структурированное логирование для alert-service.
//
// Основные компоненты:
//   - Logger: Структурированный logger с поддержкой уровней
//   - Config: Конфигурация для logger (log level, output format)
//   - NewLogger: Создание logger с конфигурацией
//
// Использование:
//
//	logger := NewLogger("info")
//	logger.Info("user logged in",
//	    "user_id", userID,
//	    "ip_address", ipAddress,
//	)
//	logger.Error("failed to process request",
//	    "error", err,
//	    "request_id", requestID,
//	)
//
// Ключевые особенности:
//   - Структурированное логирование с JSON форматом
//   - Поддержка разных уровней логирования (debug, info, warn, error)
//   - Контекстные атрибуты для distributed tracing
//   - Интеграция с OpenTelemetry для trace_id/span_id
//
// Контекст DOD:
//   - Соответствует требованию DOD 2.3 Structured Logging
//   - Все логи в JSON формате с structured fields
//   - Log levels: debug, info, warn, error, fatal
//   - Включение trace_id и span_id из OpenTelemetry context
package logger
