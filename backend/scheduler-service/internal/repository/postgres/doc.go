// Package postgres содержит PostgreSQL-реализации репозиториев scheduler-service.
//
// Реализации:
//   - WorkerRepository: хранение и поиск воркеров
//   - ScheduledCheckRepository: хранение и поиск запланированных проверок
//   - AuditRepository: аудит-логирование операций
//   - DistributedLocker: распределённая блокировка через advisory locks
//
// Все запросы используют parameterized SQL ($1, $2, ...) для предотвращения SQL-инъекций.
// Metadata воркеров сериализуется как JSONB TEXT.
package postgres
