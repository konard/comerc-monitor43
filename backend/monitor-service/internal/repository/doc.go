// Package repository обеспечивает доступ к данным (Data Access Layer).
//
// Пакет содержит интерфейсы и реализации для работы с хранилищами данных.
// Следует паттерну Repository для инкапсуляции логики доступа к данным.
//
// Структура:
//   - interfaces/: Интерфейсы репозиториев (для тестирования)
//   - postgres/: PostgreSQL реализации репозиториев
//
// Интерфейсы (interfaces/):
//
//	Определяют контракты для доступа к данным. Используются в сервисах для
//	dependency injection и позволяют мокать в тестах.
//
//	Основные интерфейсы:
//	- MonitorRepository: CRUD операции для мониторов
//	- CheckResultRepository: Сохранение и чтение результатов проверок
//	- MaintenanceWindowRepository: Управление окнами обслуживания
//	- IncidentRepository: Управление инцидентами
//	- AuditRepository: Аудит логирование
//
// Реализации (postgres/):
//
//	Содержат SQL-запросы и маппинг результатов в domain entities.
//	Используют pgx для работы с PostgreSQL.
//
// Конвенции:
//   - Все методы принимают context.Context первым параметром
//   - Параметризованные запросы (без конкатенации строк)
//   - Используют pgxpool для connection pooling
//   - Ошибки оборачиваются с контекстом
//   - Поддерживают transactions для сложных операций
//
// Пример использования:
//
//	interfaces := repository.NewPostgresMonitorRepository(dbPool)
//	monitor, err := repo.Create(ctx, &domain.Monitor{...})
//	if err != nil {
//	    return errors.Wrap(err, "failed to create monitor")
//	}
package repository
