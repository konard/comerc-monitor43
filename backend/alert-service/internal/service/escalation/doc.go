// Package escalation предоставляет бизнес-логику эскалации алертов.
//
// Эскалация автоматически повышает приоритет уведомлений для алертов,
// которые не были подтверждены в течение заданного таймаута.
//
// Основные компоненты:
//   - EscalationService: Сервис управления эскалациями алертов
//
// Использование:
//
//	escalationService := escalation.NewEscalationService(
//	    escalationRepo,
//	    alertRepo,
//	    channelRepo,
//	    auditService,
//	    tracer,
//	    metrics,
//	)
//	err := escalationService.CheckAutoEscalation(ctx)
//	level, err := escalationService.GetEscalationLevel(ctx, alertID)
//
// Ключевые особенности:
//   - Автоматическая эскалация: таймаут 30 минут без подтверждения
//   - Создание записей эскалации с указанием уровня и причины
//   - Уведомление следующего приоритетного канала при эскалации
//   - Запись действий в журнал аудита
//   - OpenTelemetry трейсинг для всех операций
package escalation
