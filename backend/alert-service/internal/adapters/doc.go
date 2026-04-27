// Package adapters предоставляет адаптеры для работы с бизнес-логикой.
//
// Основные компоненты:
//   - AlertServiceAdapter: Адаптер для управления алертами
//   - AlertRuleServiceAdapter: Адаптер для работы с правилами алертов
//   - ChannelServiceAdapter: Адаптер для работы с каналами уведомлений
//
// Использование:
//
//	alertServiceAdapter := adapters.NewAlertServiceAdapter(alertRepo, alertRuleRepo, alertChannelRepo)
//	server := grpc.NewAlertServiceServer(alertServiceAdapter, channelServiceAdapter, ruleServiceAdapter)
//
// Ключевые особенности:
//   - Конвертация между protobuf и domain моделями
//   - Обработка ошибок репозиториев с контекстом
//   - Простой интерфейс для gRPC обработчиков
//   - Поддержка пагинации и фильтрации
//
// Контекст DOD:
//   - Соответствует требованию DOD 1.2 Go-Way Framework Patterns
//   - Адаптеры для бизнес-логики
//   - Конструкторы с dependency injection
package adapters
