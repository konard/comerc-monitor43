// Package maintenance предоставляет бизнес-логику управления окнами обслуживания.
//
// Окно обслуживания (maintenance window) — плановый период, в течение которого
// алерты для монитора автоматически подавляются.
//
// Основные компоненты:
//   - MaintenanceService: Сервис управления окнами обслуживания
//
// Использование:
//
//	maintenanceService := maintenance.NewMaintenanceService(
//	    mwRepo,
//	    tracer,
//	    metrics,
//	)
//	err := maintenanceService.CreateWindow(ctx, userID, monitorID, startsAt, endsAt, &reason)
//	isUnder, err := maintenanceService.IsUnderMaintenance(ctx, monitorID)
//
// Ключевые особенности:
//   - Создание окон обслуживания с указанием начала, окончания и причины
//   - Проверка активного окна для конкретного монитора
//   - Подавление алертов во время окна обслуживания
//   - OpenTelemetry трейсинг для всех операций
package maintenance
