// Package service содержит бизнес-логику notification-template-service.
//
// Основные сервисы:
//
//	TemplateService - CRUD операций с шаблонами
//	RendererService - рендеринг шаблонов
//	ValidatorService - валидация шаблонов
//
// Каждый сервис следует конвенциям:
//   - Конструктор: NewXXX(cfg Config) *XXX
//   - Методы возвращают domain модели, не DTO
//   - Ошибки оборачиваются с контекстом
package service
