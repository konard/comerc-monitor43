// Package circuitbreaker реализует паттерн Circuit Breaker для защиты от каскадных сбоев.
//
// Состояния:
//   - Closed: запросы проходят нормально, ошибки считаются
//   - Open: запросы отклоняются сразу (fail-fast), ждём timeout
//   - HalfOpen: один пробный запрос, при успехе → Closed, при ошибке → Open
//
// Использование:
//
//	cb := circuitbreaker.New(circuitbreaker.Config{...})
//	result, err := cb.Execute(ctx, func(ctx context.Context) (*Result, error) { ... })
//
// Конфигурация:
//
//	CB_FAILURE_THRESHOLD  — ошибок до перехода в Open (default: 5)
//	CB_TIMEOUT            — время в Open до HalfOpen (default: 30s)
//	CB_SUCCESS_THRESHOLD  — успехов в HalfOpen до Closed (default: 1)
//
// Ограничения:
//   - Потокобезопасность: да
//   - Не требует Close()
package circuitbreaker
