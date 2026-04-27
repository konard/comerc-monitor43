// Package publisher предоставляет интерфейс для публикации событий в RabbitMQ.
//
// Пакет использует pure-golang RMQ адаптер для:
//   - Публикации сообщений с retry и error handling
//   - Автоматического переподключения
//   - Интеграции с OpenTelemetry (трассировка)
//   - Структурированного логирования
//
// Основные компоненты:
//   - Publisher: Интерфейс для публикации событий
//   - EventPublisher: Реализация Publisher для RabbitMQ
//   - Event: Структура события с метаданными
//
// Типы событий:
//
//   - monitor.created     - Создан новый монитор
//   - monitor.updated     - Обновлен монитор
//   - monitor.deleted     - Удален монитор
//   - monitor.paused      - Проверки приостановлены
//   - monitor.resumed     - Проверки возобновлены
//   - check.started       - Начата проверка
//   - check.completed     - Проверка завершена
//   - check.failed        - Проверка не удалась
//   - incident.detected   - Обнаружен инцидент
//   - incident.resolved   - Инцидент разрешен
//
// Использование:
//
//	pub := publisher.NewEventPublisher(rmqAdapter, logger)
//
//	err := pub.Publish(ctx, publisher.Event{
//	    Type:      "monitor.created",
//	    MonitorID: monitorID,
//	    UserID:    userID,
//	    Data:      monitorData,
//	})
//
// Формат сообщений:
//
//	{
//	  "type": "monitor.created",
//	  "timestamp": "2026-03-26T10:00:00Z",
//	  "monitor_id": "uuid",
//	  "user_id": "uuid",
//	  "data": { ... }
//	}
//
// Интеграции:
//
//   - Alert Service: Получает события для отправки уведомлений
//   - Billing Service: Получает события для тарификации
//   - Dashboard Service: Получает события для обновления UI
package publisher
