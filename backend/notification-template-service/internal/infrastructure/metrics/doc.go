// Package metrics предоставляет Prometheus метрики для notification-template-service.
//
// Метрики соответствуют конвенциям Prometheus:
//
//   - Счётчики (_total) для монотонно возрастающих значений
//   - Гистограммы для распределения значений (duration)
//   - Gauge для текущих значений (active templates)
//
// Использование:
//
//	metrics.RecordTemplateCreate(channel, tmplType, userID)
//	metrics.RecordTemplateRender(channel, tmplType, true, duration.Seconds())
//
// HTTP endpoint для экспорта:
//
//	Добавить в main.go:
//	http.Handle("/metrics", promhttp.Handler())
package metrics
