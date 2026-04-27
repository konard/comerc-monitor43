// Package model определяет доменные сущности scheduler-service.
//
// Контракт пакета:
//   - Worker: зарегистрированный check-worker с зоной и метриками
//   - ScheduledCheck: запланированная проверка с приоритетом и назначением
//   - WorkerStatus: IDLE, BUSY, OFFLINE
//   - CheckPriority: LOW, NORMAL, HIGH, CRITICAL
//   - CheckStatus: PENDING, IN_PROGRESS, COMPLETED, FAILED
//
// Бизнес-правила:
//   - Воркер регистрируется с именем и зоной
//   - Heartbeat timeout = 30s → OFFLINE
//   - Offline cleanup = 10m → удаление из реестра
//   - Проверки назначаются воркерам в той же зоне
//   - Проверки не назначаются для PAUSED мониторов
//   - Дублирование проверок предотвращается
package model
