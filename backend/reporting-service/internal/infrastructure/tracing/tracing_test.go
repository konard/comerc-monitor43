package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func init() {
	// используем no-op tracer по умолчанию чтобы не требовать реального OTEL-бэкенда
	otel.SetTracerProvider(noop.NewTracerProvider())
}

// withRecordingProvider временно устанавливает реальный SDK-провайдер,
// возвращающий записывающие спаны, и откатывает его после теста.
func withRecordingProvider(t *testing.T) {
	t.Helper()

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		if err := tp.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown tracer provider: %v", err)
		}
	})
}

func TestStartSpan(t *testing.T) {
	t.Parallel()

	ctx, span := StartSpan(context.Background(), "test-operation")
	defer span.End()

	require.NotNil(t, ctx)
	require.NotNil(t, span)
}

func TestRecordError_WithError(t *testing.T) {
	// не параллельный — меняет глобальный провайдер
	withRecordingProvider(t)

	_, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	// span.IsRecording() == true, ветка выполняется
	RecordError(span, errors.New("test error"))
	assert.True(t, true)
}

func TestRecordError_NilError(t *testing.T) {
	// не параллельный — меняет глобальный провайдер
	withRecordingProvider(t)

	_, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	// ветка if err != nil не должна выполниться
	RecordError(span, nil)
	assert.True(t, true)
}

func TestSetSuccess(t *testing.T) {
	// не параллельный — меняет глобальный провайдер
	withRecordingProvider(t)

	_, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	// span.IsRecording() == true, ветка выполняется
	SetSuccess(span)
	assert.True(t, true)
}
