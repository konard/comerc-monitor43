// Package throttling предоставляет бизнес-логику ограничения алертов (throttling).
//
// Иерархия throttling (от высшего приоритета к низшему):
//  1. Alert Storm Detection: 5+ уникальных мониторов с алертами за 5 минут → группировка
//  2. Rate Limiting: на монитор + статус, не более 1 алерта за 5 минут
//  3. Cooldown: на монитор (любой статус), минимум 15 минут между алертами
//
// Использование:
//
//	svc := NewAlertThrottlingService(cfg, alertRepo, deliveryRepo, tracer, metrics)
//	result, err := svc.CheckThrottle(ctx, userID, monitorID, "DOWN")
//
// Контекст DOD:
//   - Защита от alert storm (массовых уведомлений)
//   - Rate limiting для предотвращения спама на канал
//   - Cooldown для устранения дублирующих алертов
//   - OpenTelemetry трейсинг для всех операций
package throttling
