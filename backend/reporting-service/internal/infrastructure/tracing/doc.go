// Package tracing предоставляет утилиты для OpenTelemetry tracing.
//
// Пакет содержит helper функции для создания span'ов в handler и service методах
// согласно DOD требованиям к observability.
//
// Использование:
//
//	ctx, span := tracing.StartSpan(ctx, "Handler.Method")
//	defer span.End()
//
//	tracing.RecordError(span, err)
//	tracing.SetSuccess(span)
package tracing
