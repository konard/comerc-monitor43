// Package worker реализует жизненный цикл check-worker.
//
// Воркер управляет:
//   - регистрацией в scheduler-service при запуске
//   - периодической отправкой heartbeat
//   - опросом scheduler-service на наличие назначенных проверок
//   - выполнением проверок через executor
//   - отправкой результатов в monitor-service
//   - graceful shutdown с доработкой текущих проверок
package worker
