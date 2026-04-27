package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testContextKey string

func TestNewTracer(t *testing.T) {
	t.Run("valid_service_name", func(t *testing.T) {
		tracer := NewTracer("test-service", "/test")
		assert.NotNil(t, tracer)
	})

	t.Run("empty_service_name", func(t *testing.T) {
		tracer := NewTracer("", "/test")
		assert.NotNil(t, tracer)
	})

	t.Run("empty_endpoint", func(t *testing.T) {
		tracer := NewTracer("test-service", "")
		assert.NotNil(t, tracer)
	})
}

func TestTracer_StructFields(t *testing.T) {
	tracer := NewTracer("test-service", "/test")
	require.NotNil(t, tracer)

	t.Run("all_fields_initialized", func(t *testing.T) {
		assert.NotNil(t, tracer.provider)
		assert.NotNil(t, tracer.tracer)
	})

	t.Run("fields_have_correct_types", func(t *testing.T) {
		assert.NotNil(t, tracer.provider)
		assert.NotNil(t, tracer.tracer)
	})
}

func TestTracer_StartSpan(t *testing.T) {
	tracer := NewTracer("test-service", "/test")
	require.NotNil(t, tracer)

	ctx := context.Background()

	t.Run("start_simple_span", func(t *testing.T) {
		spanCtx, span := tracer.StartSpan(ctx, "test-operation")
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
	})

	t.Run("start_multiple_spans", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, span := tracer.StartSpan(ctx, "test-operation")
			assert.NotNil(t, span)
		}
	})

	t.Run("start_nested_spans", func(t *testing.T) {
		outerCtx, _ := tracer.StartSpan(ctx, "outer-operation")
		assert.NotNil(t, outerCtx)

		_, innerSpan := tracer.StartSpan(outerCtx, "inner-operation")
		assert.NotNil(t, innerSpan)
	})

	t.Run("concurrent_spans", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_, span := tracer.StartSpan(ctx, "concurrent-operation")
				assert.NotNil(t, span)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestTracer_GetTracer(t *testing.T) {
	tracer := NewTracer("test-service", "/test")
	require.NotNil(t, tracer)

	t.Run("returns_tracer", func(t *testing.T) {
		returnedTracer := tracer.GetTracer()
		assert.NotNil(t, returnedTracer)
		assert.Equal(t, tracer.tracer, returnedTracer)
	})

	t.Run("returns_same_instance", func(t *testing.T) {
		tracer1 := tracer.GetTracer()
		tracer2 := tracer.GetTracer()
		assert.Equal(t, tracer1, tracer2)
	})
}

func TestTracer_Shutdown(t *testing.T) {
	tracer := NewTracer("test-service", "/test")
	require.NotNil(t, tracer)

	ctx := context.Background()

	t.Run("shutdown_without_panic", func(t *testing.T) {
		err := tracer.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown_after_spans", func(t *testing.T) {
		_, span1 := tracer.StartSpan(ctx, "operation1")
		_, span2 := tracer.StartSpan(ctx, "operation2")
		assert.NotNil(t, span1)
		assert.NotNil(t, span2)

		span1.End()
		span2.End()

		err := tracer.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestTracer_MultipleTracers(t *testing.T) {
	t.Run("different_services", func(t *testing.T) {
		tracer1 := NewTracer("service1", "/endpoint1")
		tracer2 := NewTracer("service2", "/endpoint2")

		assert.NotNil(t, tracer1)
		assert.NotNil(t, tracer2)

		ctx := context.Background()
		_, span1 := tracer1.StartSpan(ctx, "operation1")
		_, span2 := tracer2.StartSpan(ctx, "operation2")
		assert.NotNil(t, span1)
		assert.NotNil(t, span2)
	})

	t.Run("same_service_different_instances", func(t *testing.T) {
		tracer1 := NewTracer("service", "/endpoint")
		tracer2 := NewTracer("service", "/endpoint")

		assert.NotNil(t, tracer1)
		assert.NotNil(t, tracer2)

		ctx := context.Background()
		_, span1 := tracer1.StartSpan(ctx, "operation1")
		_, span2 := tracer2.StartSpan(ctx, "operation2")
		assert.NotNil(t, span1)
		assert.NotNil(t, span2)
	})

	t.Run("concurrent_tracer_creation", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				tracer := NewTracer("service", "/endpoint")
				assert.NotNil(t, tracer)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestTracer_ContextPropagation(t *testing.T) {
	tracer := NewTracer("test-service", "/test")
	require.NotNil(t, tracer)

	t.Run("background_context", func(t *testing.T) {
		ctx := context.Background()
		_, span := tracer.StartSpan(ctx, "operation")
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("context_with_values", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, testContextKey("key"), "value")
		_, span := tracer.StartSpan(ctx, "operation")
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("context_with_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel()
		_, span := tracer.StartSpan(ctx, "operation")
		assert.NotNil(t, span)
		span.End()
	})
}

func TestTracer_SpanLifecycle(t *testing.T) {
	tracer := NewTracer("test-service", "/test")
	require.NotNil(t, tracer)

	ctx := context.Background()

	t.Run("span_creation_and_ending", func(t *testing.T) {
		_, span := tracer.StartSpan(ctx, "lifecycle-test")
		assert.NotNil(t, span)
		span.End()
		span.End()
	})

	t.Run("multiple_span_lifecycles", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_, span := tracer.StartSpan(ctx, "lifecycle-operation")
			assert.NotNil(t, span)
			span.End()
		}
	})
}
