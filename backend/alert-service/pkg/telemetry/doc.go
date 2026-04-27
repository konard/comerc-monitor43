// Package telemetry предоставляет OpenTelemetry инструменты для observability.
//
// Основные компоненты:
//   - Tracer: Обёртка над OpenTelemetry tracer provider
//   - Metics: Обёртка над OpenTelemetry meter provider
//   - Config: Конфигурация для экспортеров и service name
//
// Использование:
//
//	tracer := telemetry.NewTracer("alert-service", "http://localhost:4318")
//	metrics := telemetry.NewMetrics("alert-service")
//	ctx, span := tracer.Start(ctx, "operationName")
//	defer span.End()
//
// Ключевые особенности:
//   - OpenTelemetry tracing с автоматическим export в Jaeger
//   - Metrics collection с Prometheus format
//   - Structured logging интеграция с trace context
//   - Proper span creation с атрибутами и error recording
//
// Контекст DOD:
//   - Соответствует требованию DOD 6.4 Tracing
//   - OpenTelemetry для всех операций (packageName.operation)
//   - Стандартные атрибуты для spans
//   - Метрики Prometheus с proper naming convention
package telemetry
