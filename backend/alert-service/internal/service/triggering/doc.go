// Package triggering предоставляет бизнес-логику триггеринг алертов.
//
// Основные компоненты:
//   - AlertTriggeringService: Сервис для триггеринг алертов на основе результатов проверок
//
// Использование:
//
//	triggeringService := NewAlertTriggeringService(
//	    alertRepo,
//	    ruleRepo,
//	    tracer,
//	    metrics,
//	)
//	alert, err := triggeringService.ProcessMonitorCheck(ctx, userID, monitorID, status, consecutiveFailures)
//
// Ключевые особенности:
//   - Создание алертов при достижении порога последовательных неудач
//   - Проверка правил алертов для монитора
//   - Защита от спама (cooldown period 30 минут)
//   - Интеграция с delivery service для отправки уведомлений
//   - OpenTelemetry трейсинг для всех операций
//
// Контекст DOD:
//   - Соответствует требованию DOD 6.4 Tracing
//   - Создание spans для значимых операций
//   - Структурированное логирование с контекстом
package triggering
