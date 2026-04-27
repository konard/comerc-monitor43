// Package consumer обрабатывает события из RabbitMQ для обновления read model.
//
// Использование:
//
//	consumer := consumer.NewEventConsumer(amqpURL, statusRepo, checkRepo, incidentRepo, hub, logger)
//	consumer.Start(ctx)
//
// Подписки:
//   - monitor.created/updated/deleted/paused/resumed
//   - check.completed/failed
//   - incident.detected/resolved
//   - alert.triggered/resolved
//
// Ограничения:
//   - Требует вызова Close() для освобождения соединения
//   - Eventual consistency (CQRS)
package consumer
