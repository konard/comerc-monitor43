// Package service реализует бизнес-логику scheduler-service.
//
// Сервисы пакета:
//   - WorkerService: управление жизненным циклом воркеров (регистрация, heartbeat, offline-detection)
//   - CheckScheduler: планирование проверок и назначение их воркерам
//
// Правила:
//   - Ошибки соединения/базы оборачиваются через errors.Wrap
//   - Бизнес-ошибки возвращаются как sentinel errors (WORKER_NOT_FOUND, NO_WORKERS_AVAILABLE)
//   - Логирование выполняется через slog с контекстом
package service
