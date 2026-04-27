// Package dto содержит Data Transfer Objects для scheduler-service.
//
// DTO используются для передачи данных между слоями:
// - Handler → Service: запросы
// - Service → Handler: ответы
//
// DTO изолируют бизнес-логику от протокол-специфичных форматов (protobuf).
package dto
