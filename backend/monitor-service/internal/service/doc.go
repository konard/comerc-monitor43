// Package service реализует бизнес-логику мониторинговой системы (use cases).
//
// Пакет содержит сервисы, которые реализуют use cases системы, координируют работу
// репозиториев и выполняют бизнес-операции. Каждый сервис группирует связанные операции.
//
// Основные сервисы:
//   - MonitorService: Управление мониторами и проверками
//   - CheckExecutorService: Выполнение проверок HTTP endpoints
//   - UptimeCalculatorService: Расчёт uptime статистики
//   - IncidentService: Управление инцидентами
//   - MaintenanceService: Управление окнами технического обслуживания
//
// DTO (Data Transfer Objects):
//
//	Подпакет dto содержит структуры для передачи данных между слоями:
//	- Request DTOs: Входные параметры для use cases
//	- Response DTOs: Выходные данные из use cases
//
// Конвенции:
//   - Все методы принимают context.Context первым параметром
//   - Все методы возвращают error последним значением
//   - Ошибки оборачиваются с контекстом: errors.Wrap(err, "failed to ...")
//   - Критические операции логируются (info级别 для бизнес-операций)
//   - Все операции trace'утся через OpenTelemetry
//
// Пример использования:
//
//	service := service.NewMonitorService(repo, executor, publisher, cfg)
//	monitor, err := service.CreateMonitor(ctx, &service.CreateMonitorRequest{
//	    UserID: userID,
//	    Name:   "My Monitor",
//	    URL:    "https://example.com",
//	})
//	if err != nil {
//	    return err
//	}
package service
