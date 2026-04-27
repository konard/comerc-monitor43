// Package model определяет доменные сущности alert-service.
//
// Основные компоненты:
//   - Alert: Сущность алерта с бизнес-логикой
//   - AlertRule: Сущность правила алертов
//   - AlertChannel: Сущность канала уведомлений
//   - DeliveryAttempt: Сущность попытки доставки уведомления
//   - Enums: Типы алертов, статусы, типы каналов
//
// Использование:
//
//	alert := &domain.Alert{
//	    ID: uuid.New(),
//	    MonitorID: monitorID,
//	    Type: domain.AlertTypeStatusCode,
//	}
//
// Ключевые особенности:
//   - Domain-центричная архитектура без внешних зависимостей
//   - Бизнес-логика в методах сущностей
//   - Proper use of Go time.Time вместо string
//   - Enum типы для типобезопасности
//
// Контекст DOD:
//   - Соответствует требованию DOD 1.1 Clean Architecture
//   - Domain layer без импортов из infrastructure
//   - Бизнес-логика в доменных сущностях
package model
