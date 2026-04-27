// Package service предоставляет бизнес-логику alert-service.
//
// Основные компоненты:
//   - Субпакеты с конкретными сервисами:
//   - delivery: Сервисы для доставки уведомлений
//   - triggering: Сервисы для триггеринг алертов
//   - channels: Сервисы для работы с каналами уведомлений
//
// Использование:
//
//	alertDeliveryService := delivery.NewAlertDeliveryService(
//	    deliveryAttemptRepo,
//	    alertChannelRepo,
//	    alertRepo,
//	    telegramClient,
//	    emailClient,
//	    webhookClient,
//	    tracer,
//	    metrics,
//	)
//
// Ключевые особенности:
//   - Use case реализация как методы структур
//   - Dependency injection через конструкторы
//   - Группировка связанных операций в сервисах
//   - Контекст для отмены операций
//
// Контекст DOD:
//   - Соответствует требованию DOD 1.4 Service Layer (Use Cases)
//   - Сервисы как контейнеры для use cases
//   - Интерфейсы репозиториев для тестируемости
//   - Отсутствие внешних зависимостей в domain layer
package service
