// Package postgres предоставляет PostgreSQL реализации репозиториев.
//
// Компоненты:
//   - MonitorStatusRepository: upsert/delete/list статусов мониторов
//   - CheckHistoryRepository: создание записей проверок, агрегация метрик
//   - IncidentRepository: upsert/list/update инцидентов
//   - DB: обёртка над sqlx.DB с observability
//
// Использование:
//
//	db := postgres.NewDB(dsn)
//	db.Connect(ctx)
//	repo := postgres.NewMonitorStatusRepository(db)
//
// Особенности:
//   - TEXT тип для JSON хранения (согласно DOD 4.1)
//   - OpenTelemetry трейсинг на каждом запросе
//   - Динамическая сборка WHERE для фильтрации
package postgres
