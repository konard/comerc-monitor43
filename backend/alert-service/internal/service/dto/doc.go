// Package dto содержит Data Transfer Objects для слоя сервисов
//
// DTO объекты используются для передачи данных между слоями архитектуры:
// - От handler слоя к service слою (Request DTOs)
// - От service слоя к handler слою (Response DTOs)
// - Для внутренней коммуникации между сервисами
//
// Использование DTO позволяет:
// - Разделить внутренние модели (model) и API контракты
// - Скрыть чувствительные поля domain entities
// - Валидировать данные на границах слоев
// - Управлять форматом данных для external systems
//
// Структура пакета:
// - logger.go: Structured logging wrapper на базе slog
// - *_dto.go: Domain-specific DTO structures
package dto
