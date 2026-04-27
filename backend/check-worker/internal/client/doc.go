// Package client содержит gRPC-клиенты для взаимодействия
// с scheduler-service и monitor-service.
//
// Клиенты:
//   - SchedulerClient: подключение к scheduler-service для регистрации,
//     heartbeat, получения назначенных проверок и отключения
//   - MonitorClient: подключение к monitor-service для отправки
//     результатов проверок через TriggerCheck RPC
//
// Каждый клиент реализует соответствующий интерфейс (schedulerService, monitorService),
// объявленный в пакете worker, для возможности мокирования в тестах.
package client
