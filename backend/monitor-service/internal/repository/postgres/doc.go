// Package postgres предоставляет реализации репозиториев на основе PostgreSQL.
//
// Пакет использует database/sql для работы с БД и поддерживает:
//   - Connection pooling
//   - Transactions
//   - Context cancellation
//   - Named queries
//
// Все repository методы принимают context.Context для отмены операций
// и propagate tracing информации.
package postgres
