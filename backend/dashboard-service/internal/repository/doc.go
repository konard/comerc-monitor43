// Package repository определяет интерфейсы хранилищ dashboard-service.
//
// Интерфейсы:
//   - MonitorStatusRepository: CRUD для статусов мониторов
//   - CheckHistoryRepository: создание и чтение истории проверок
//   - IncidentRepository: CRUD для инцидентов
//
// Реализации находятся в подпакете postgres.
package repository
