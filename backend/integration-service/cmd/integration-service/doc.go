// Package main является точкой входа для integration-service.
//
// Service запускает:
//   - gRPC сервер (WebhookIntegrationService, APIKeyService, ImportService)
//   - Webhook delivery service с worker pools
//   - RabbitMQ consumer для alert.triggered событий
//   - HTTP сервер с health check endpoints
//
// Интеграция с внешними системами:
//   - PostgreSQL для хранения данных
//   - RabbitMQ для приёма событий от Alert Service
//   - Jaeger для distributed tracing
//   - Prometheus для metrics
package main
