// Package handler содержит gRPC handlers billing-service.
//
// Реализует интерфейс billing.BillingServiceServer из сгенерированного proto кода.
//
// Handlers валидируют запросы, конвертируют Proto <-> DTO,
// вызывают сервисы и конвертируют ошибки в gRPC статусы.
package handler
