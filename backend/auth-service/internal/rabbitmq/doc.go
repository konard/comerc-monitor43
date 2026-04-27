// Package rabbitmq декларирует AMQP-топологию сервиса аутентификации.
//
// Использование:
//
//	topo := rabbitmq.New(cfg, dialer)
//	if err := topo.Start(ctx); err != nil {
//	    return err
//	}
//
// Конфигурация:
//
//	RABBITMQ_EXCHANGE    — topic-exchange событий (default: auth.events)
//	RABBITMQ_QUEUE       — очередь событий      (default: auth.events)
//	RABBITMQ_ROUTING_KEY — ключ маршрутизации   (default: auth.events)
//
// Топология:
//
//	exchange auth.events (topic, durable)
//	  └─ queue auth.events  ← routing key auth.events
//
// Ограничения:
//
//   - Потокобезопасность: нет (каждый вызов Start открывает новый канал).
package rabbitmq
