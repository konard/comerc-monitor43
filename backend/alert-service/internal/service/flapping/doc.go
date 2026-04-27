// Package flapping предоставляет бизнес-логику обнаружения флаппинга мониторов.
//
// Флаппинг — частые переключения статуса монитора (up/down),
// указывающие на нестабильность. Обнаружение предотвращает ложные алерты.
//
// Основные компоненты:
//   - FlappingService: Сервис обнаружения и управления состоянием флаппинга
//
// Использование:
//
//	flappingService := flapping.NewFlappingService(
//	    statusChangeRepo,
//	    alertRepo,
//	    auditService,
//	    tracer,
//	    metrics,
//	)
//	err := flappingService.RecordStatusChange(ctx, userID, monitorID, "up", "down")
//	result, err := flappingService.CheckFlapping(ctx, monitorID)
//
// Ключевые особенности:
//   - Обнаружение: 5+ смен статуса за 10 минут (скользящее окно)
//   - Выход из флаппинга: отсутствие смен статуса в течение 15 минут
//   - Запись в журнал аудита при обнаружении
//   - OpenTelemetry трейсинг для всех операций
package flapping
