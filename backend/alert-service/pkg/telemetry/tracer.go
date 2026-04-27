package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Tracer представляет OpenTelemetry tracer
type Tracer struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
}

// NewTracer создаёт новый tracer
func NewTracer(serviceName, endpoint string) *Tracer {
	tp := otel.GetTracerProvider()

	tracer := tp.Tracer(
		"github.com/raul/monitor/backend/alert-service",
		trace.WithInstrumentationVersion("1.0.0"),
	)

	return &Tracer{
		provider: tp,
		tracer:   tracer,
	}
}

// StartSpan начинает новый span
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

// shutdownProvider — минимальный интерфейс для завершения провайдера трейсинга
type shutdownProvider interface {
	ForceFlush(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// Shutdown сбрасывает буфер и завершает работу провайдера трейсинга
func (t *Tracer) Shutdown(ctx context.Context) error {
	if p, ok := t.provider.(shutdownProvider); ok {
		if err := p.ForceFlush(ctx); err != nil {
			return err
		}
		return p.Shutdown(ctx)
	}
	return nil
}

// GetTracer возвращает tracer для использования
func (t *Tracer) GetTracer() trace.Tracer {
	return t.tracer
}
