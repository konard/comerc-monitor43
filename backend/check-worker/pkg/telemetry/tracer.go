package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
}

func NewTracer(serviceName string, endpoint string) (*Tracer, error) {
	tp := otel.GetTracerProvider()

	tracer := tp.Tracer(
		"github.com/raul/monitor/backend/check-worker",
		trace.WithInstrumentationVersion("1.0.0"),
	)

	return &Tracer{
		provider: tp,
		tracer:   tracer,
	}, nil
}

func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	return nil
}

func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}
