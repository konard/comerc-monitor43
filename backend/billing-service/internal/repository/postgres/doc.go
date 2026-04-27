// Package postgres содержит PostgreSQL реализацию репозиториев.
//
// Использует sqlx для работы с базой данных.
//
// Структура:
//
//	DB - wrapper для *sqlx.DB с дополнительными методами
//	PlanRepository - репозиторий тарифных планов
//	SubscriptionRepository - репозиторий подписок
//	PaymentRepository - репозиторий платежей
//
// Все репозитории принимают context.Context для отмены операций.
// Все ошибки оборачиваются с контекстом через errors.Wrap().
package postgres
