// Package retry предоставляет механизм повторных попыток с exponential backoff.
//
// Используется для сетевых операций (gRPC вызовы к scheduler-service и monitor-service)
// чтобы обеспечить resilience при временных сбоях сети или недоступности сервисов.
//
// Особенности:
//   - Exponential backoff с configurable multiplier
//   - Jitter для предотвращения thundering herd
//   - Max delay ceiling
//   - Context-aware (отмена при context cancellation)
package retry
