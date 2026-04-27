// Package monitor предоставляет gRPC клиент к monitor-service.
//
// Клиент используется scheduler-service для:
//   - получения списка активных мониторов (ListMonitors)
//   - проверки статуса монитора (GetMonitor)
//   - проверки существования монитора перед планированием
//
// CircuitBreakerAdapter оборачивает Adapter в circuit breaker
// для защиты от каскадных сбоев при недоступности monitor-service.
//
// Конструктор не выполняет I/O. Соединение устанавливается через Connect().
package monitor
