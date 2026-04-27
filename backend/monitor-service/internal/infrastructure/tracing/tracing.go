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

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan создаёт новый span с именем метода.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("monitor-service")
	return tracer.Start(ctx, name)
}

// StartSpanWithAttrs создаёт span с атрибутами.
func StartSpanWithAttrs(ctx context.Context, name string, attrs map[string]any) (context.Context, trace.Span) {
	tracer := otel.Tracer("monitor-service")

	// Convert map to proper KeyValue format
	kvPairs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kvPairs = append(kvPairs, attribute.String(k, toString(v)))
	}

	return tracer.Start(ctx, name, trace.WithAttributes(kvPairs...))
}

// RecordError записывает ошибку в текущий span.
func RecordError(span trace.Span, err error) {
	if err != nil && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddEvent добавляет событие в текущий span.
func AddEvent(ctx context.Context, name string, attrs map[string]any) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		kvPairs := make([]attribute.KeyValue, 0, len(attrs))
		for k, v := range attrs {
			kvPairs = append(kvPairs, attribute.String(k, toString(v)))
		}
		span.AddEvent(name, trace.WithAttributes(kvPairs...))
	}
}

// SetSuccess устанавливает статус успеха для текущего span.
func SetSuccess(span trace.Span) {
	if span.IsRecording() {
		span.SetStatus(codes.Ok, "success")
	}
}

// toString конвертирует любое значение в строку.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}
