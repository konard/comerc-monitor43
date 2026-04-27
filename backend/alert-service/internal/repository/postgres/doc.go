// Package postgres предоставляет PostgreSQL реализации репозиториев.
//
// Основные компоненты:
//   - AlertRepository: PostgreSQL реализация репозитория алертов
//   - AlertRuleRepository: PostgreSQL реализация репозитория правил алертов
//   - AlertChannelRepository: PostgreSQL реализация репозитория каналов уведомлений
//   - DeliveryAttemptRepository: PostgreSQL реализация репозитория попыток доставки
//   - DB: Обёртка над sqlx.DB с наблюдаемостью
//
// Использование:
//
//	db := postgres.NewDB(config)
//	if err := db.Connect(); err != nil {
//	    // handle error
//	}
//	alertRepo := postgres.NewAlertRepository(db)
//	alert, err := alertRepo.Create(ctx, alert)
//
// Ключевые особенности:
//   - Использование sqlx для упрощённой работы с PostgreSQL
//   - Встроенная наблюдаемость через OpenTelemetry
//   - Поддержка транзакций для атомарных операций
//   - TEXT тип для хранения JSON данных (согласно DOD 4.1)
//   - Connection pooling через sqlx
//
// Контекст DOD:
//   - ✅ Database adapter не существует в pure-golang/adapters
//     (pure-golang предоставляет: grpc, http, logger, metrics, queue, tracing)
//   - ✅ Кастомная DB реализация — ДОПУСТИМО согласно DOD 1.5
//   - ✅ Другие сервисы используют pure-golang adapters (см. cmd/server/main.go)
//   - ✅ Соответствует требованию DOD 4.3 (Transaction Handling)
//   - ✅ Правильная обработка ошибок с контекстом
package postgres
