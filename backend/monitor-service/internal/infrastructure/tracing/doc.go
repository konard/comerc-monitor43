// Package tracing предоставляет утилиты для OpenTelemetry tracing.
//
// Пакет содержит helper функции для создания span'ов в service методах
// согласно DOD требованиям к observability.
//
// Использование:
//
//	ctx, span := tracing.StartSpan(ctx, "Service.Method")
//	defer span.End()
//
//	tracing.RecordError(span, err)
//	tracing.AddEvent(ctx, "event.name", map[string]interface{}{"key": "value"})
//	tracing.SetSuccess(span)
package tracing
