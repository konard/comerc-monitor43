// Package telemetry предоставляет обёртки для OpenTelemetry трейсинга и метрик.
//
// Использование:
//
//	tracer, err := telemetry.NewTracer("dashboard-service", endpoint)
//	if err != nil { ... }
//	metrics, err := telemetry.NewMetrics("dashboard-service")
//	if err != nil { ... }
//	db.SetObservability(tracer.GetTracer(), metrics)
package telemetry
