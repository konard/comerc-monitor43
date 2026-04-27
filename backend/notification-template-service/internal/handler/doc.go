// Package handler содержит gRPC handlers notification-template-service.
//
// Структура:
//
//	Handler - gRPC handler, реализующий NotificationTemplateServiceServer
//
// Конвенции:
//   - Преобразует gRPC запросы в domain модели
//   - Делегирует бизнес-логику сервису
//   - Оборачивает ошибки в статусы gRPC
//   - Логирует все запросы
package handler
