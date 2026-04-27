package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("reporting-service")
	return tracer.Start(ctx, name)
}

func RecordError(span trace.Span, err error) {
	if err != nil && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

func SetSuccess(span trace.Span) {
	if span.IsRecording() {
		span.SetStatus(codes.Ok, "success")
	}
}
