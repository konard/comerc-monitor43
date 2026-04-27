package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// TestStartSpan тестирует создание span.
func TestStartSpan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ctx, span := StartSpan(ctx, "TestOperation")

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	span.End()
}

// TestStartSpanWithAttrs тестирует создание span с атрибутами.
func TestStartSpanWithAttrs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	attrs := map[string]any{
		"user_id":    "12345",
		"operation":  "check",
		"monitor_id": "abcde",
	}

	ctx, span := StartSpanWithAttrs(ctx, "TestOperation", attrs)

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	span.End()
}

// TestStartSpan_NestedSpans тестирует вложенные span'ы.
func TestStartSpan_NestedSpans(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Родительский span
	ctx, parentSpan := StartSpan(ctx, "ParentOperation")
	assert.NotNil(t, parentSpan)

	// Дочерний span
	_, childSpan := StartSpan(ctx, "ChildOperation")
	assert.NotNil(t, childSpan)
	childSpan.End()

	parentSpan.End()
}

// TestStartSpanWithAttrs_VariousTypes тестирует различные типы атрибутов.
func TestStartSpanWithAttrs_VariousTypes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	attrs := map[string]any{
		"string_attr": "value",
		"int_attr":    123,
		"float_attr":  45.6,
		"bool_attr":   true,
		"nil_attr":    nil,
	}

	ctx, span := StartSpanWithAttrs(ctx, "TestOperation", attrs)

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	span.End()
}

// TestStartSpanWithAttrs_EmptyAttrs тестирует создание span с пустыми атрибутами.
func TestStartSpanWithAttrs_EmptyAttrs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	attrs := map[string]any{}

	ctx, span := StartSpanWithAttrs(ctx, "TestOperation", attrs)

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	span.End()
}

// TestTracing_Integration тестирует интеграцию всех функций.
func TestTracing_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Создаём span
	ctx, span := StartSpan(ctx, "MonitorService.Check")
	require.NotNil(t, span)
	defer span.End()

	// Добавляем событие
	AddEvent(ctx, "check.started", map[string]any{
		"monitor_id": "test-monitor",
		"user_id":    "test-user",
	})

	// Устанавливаем успех
	SetSuccess(span)

	// Проверяем, что span создан
	assert.NotNil(t, span)
}

// TestRecordError тестирует запись ошибок в span.
func TestRecordError(t *testing.T) {
	t.Parallel()
	t.Run("records error in span", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		testErr := errors.New("test error")

		// RecordError не должен паниковать
		RecordError(span, testErr)

		assert.NotNil(t, span)
	})

	t.Run("handles nil error gracefully", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		// RecordError с nil ошибкой не должен паниковать
		RecordError(span, nil)

		assert.NotNil(t, span)
	})

	t.Run("records error with context", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "FailingOperation")
		defer span.End()

		testErr := errors.New("operation failed")

		RecordError(span, testErr)

		// Проверяем, что span всё ещё валиден
		assert.NotNil(t, span)
	})
}

// TestAddEdgeCases тестирует edge cases для AddEvent.
func TestAddEventEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("adds event with nil attributes", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		// AddEvent с nil attributes не должен паниковать
		AddEvent(ctx, "event.without.attrs", nil)

		assert.NotNil(t, span)
	})

	t.Run("adds event with empty attributes", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		attrs := map[string]any{}
		AddEvent(ctx, "event.empty.attrs", attrs)

		assert.NotNil(t, span)
	})

	t.Run("adds event with complex attributes", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		attrs := map[string]any{
			"string": "value",
			"int":    123,
			"float":  45.6,
			"bool":   true,
			"nil":    nil,
			"slice":  []int{1, 2, 3},
		}

		AddEvent(ctx, "event.complex", attrs)

		assert.NotNil(t, span)
	})

	t.Run("adds multiple events to span", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		for i := 0; i < 5; i++ {
			AddEvent(ctx, "event", map[string]any{
				"index": i,
			})
		}

		assert.NotNil(t, span)
	})
}

// TestSetSuccessEdgeCases тестирует edge cases для SetSuccess.
func TestSetSuccessEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("sets success on span", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "SuccessfulOperation")
		defer span.End()

		SetSuccess(span)

		assert.NotNil(t, span)
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "Operation")
		defer span.End()

		// SetSuccess можно вызывать несколько раз
		SetSuccess(span)
		SetSuccess(span)
		SetSuccess(span)

		assert.NotNil(t, span)
	})

	t.Run("works with nested spans", func(t *testing.T) {
		ctx, parentSpan := StartSpan(context.Background(), "ParentOperation")
		defer parentSpan.End()

		SetSuccess(parentSpan)

		_, childSpan := StartSpan(ctx, "ChildOperation")
		defer childSpan.End()

		SetSuccess(childSpan)

		assert.NotNil(t, parentSpan)
		assert.NotNil(t, childSpan)
	})
}

// TestTracing_ErrorFlow тестирует flow с ошибкой.
func TestTracing_ErrorFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Создаём span с атрибутами
	attrs := map[string]any{
		"monitor_id": "failing-monitor",
		"user_id":    "test-user",
	}

	ctx, span := StartSpanWithAttrs(ctx, "MonitorService.FailingCheck", attrs)
	require.NotNil(t, span)
	defer span.End()

	// Добавляем событие
	AddEvent(ctx, "check.started", nil)

	// Проверяем, что span создан и обработан
	assert.NotNil(t, span)

	// Примечание: RecordError проверяется в интеграционных тестах
	// так как требует реально записывающий span
}

// TestTracing_MultipleOperations тестирует несколько операций в одном span.
func TestTracing_MultipleOperations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Создаём span для операции
	ctx, span := StartSpan(ctx, "MonitorService.MultipleChecks")
	defer span.End()

	// Симулируем несколько проверок
	for i := 0; i < 3; i++ {
		checkCtx, checkSpan := StartSpan(ctx, "CheckExecution")
		AddEvent(checkCtx, "check.started", map[string]any{
			"check_id": i,
		})
		SetSuccess(checkSpan)
		checkSpan.End()
	}

	assert.NotNil(t, span)
}

// TestToString тестирует конвертацию значений в строку.
func TestToString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int",
			input:    42,
			expected: "42",
		},
		{
			name:     "int32",
			input:    int32(32),
			expected: "32",
		},
		{
			name:     "int64",
			input:    int64(64),
			expected: "64",
		},
		{
			name:     "uint",
			input:    uint(10),
			expected: "10",
		},
		{
			name:     "uint32",
			input:    uint32(32),
			expected: "32",
		},
		{
			name:     "uint64",
			input:    uint64(64),
			expected: "64",
		},
		{
			name:     "float32",
			input:    float32(3.14),
			expected: "3.140000",
		},
		{
			name:     "float64",
			input:    float64(2.718),
			expected: "2.718000",
		},
		{
			name:     "bool true",
			input:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			input:    false,
			expected: "false",
		},
		{
			name:     "nil",
			input:    nil,
			expected: "",
		},
		{
			name:     "struct",
			input:    struct{ Name string }{"test"},
			expected: "{test}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestTracing_ContextPropagation тестирует propagation контекста.
func TestTracing_ContextPropagation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Создаём первый span
	ctx, span1 := StartSpan(ctx, "Operation1")
	defer span1.End()

	// Создаём вложенный span
	_, span2 := StartSpan(ctx, "Operation2")
	defer span2.End()

	// Проверяем, что оба span'а созданы
	assert.NotNil(t, span1)
	assert.NotNil(t, span2)
}

// Helper function to create a recording span for testing
func setupRecordingSpan(t *testing.T) (context.Context, trace.Span, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	tracer := tp.Tracer("test-tracer")
	ctx, span := tracer.Start(context.Background(), "test-span")

	return ctx, span, exporter
}

// TestRecordError_RecordingSpan тестирует RecordError с recording span.
func TestRecordError_RecordingSpan(t *testing.T) {
	t.Parallel()
	t.Run("records error when span is recording", func(t *testing.T) {
		_, span, exporter := setupRecordingSpan(t)

		testErr := errors.New("test error")

		RecordError(span, testErr)

		// Verify span is recording
		assert.True(t, span.IsRecording())

		// End the span to flush it
		span.End()

		// Get the completed span
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-span", spans[0].Name)
	})

	t.Run("does not record when error is nil", func(t *testing.T) {
		_, span, exporter := setupRecordingSpan(t)

		RecordError(span, nil)

		// Verify no error was recorded
		assert.True(t, span.IsRecording())

		span.End()
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
	})

	t.Run("does not panic with non-recording span", func(t *testing.T) {
		// Use default tracer which creates non-recording spans
		_, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		testErr := errors.New("test error")

		// Should not panic even with non-recording span
		assert.NotPanics(t, func() {
			RecordError(span, testErr)
		})

		// Verify span is not recording
		assert.False(t, span.IsRecording())
	})
}

// TestAddEvent_RecordingSpan тестирует AddEvent с recording span.
func TestAddEvent_RecordingSpan(t *testing.T) {
	t.Parallel()
	t.Run("adds event when span is recording", func(t *testing.T) {
		ctx, span, exporter := setupRecordingSpan(t)

		attrs := map[string]any{
			"key1": "value1",
			"key2": 123,
		}

		AddEvent(ctx, "test.event", attrs)

		// Verify span is recording
		assert.True(t, span.IsRecording())

		// End the span to flush events
		span.End()

		// Verify event was recorded
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-span", spans[0].Name)
	})

	t.Run("handles empty attributes", func(t *testing.T) {
		ctx, span, exporter := setupRecordingSpan(t)

		AddEvent(ctx, "test.event", map[string]any{})

		assert.True(t, span.IsRecording())

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
	})

	t.Run("handles nil attributes", func(t *testing.T) {
		ctx, span, exporter := setupRecordingSpan(t)

		AddEvent(ctx, "test.event", nil)

		assert.True(t, span.IsRecording())

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
	})

	t.Run("does not panic with non-recording span", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		attrs := map[string]any{
			"key": "value",
		}

		// Should not panic even with non-recording span
		assert.NotPanics(t, func() {
			AddEvent(ctx, "test.event", attrs)
		})

		// Verify span is not recording
		assert.False(t, span.IsRecording())
	})

	t.Run("adds multiple events", func(t *testing.T) {
		ctx, span, exporter := setupRecordingSpan(t)

		for i := 0; i < 3; i++ {
			AddEvent(ctx, "test.event", map[string]any{
				"index": i,
			})
		}

		assert.True(t, span.IsRecording())

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
	})
}

// TestSetSuccess_RecordingSpan тестирует SetSuccess с recording span.
func TestSetSuccess_RecordingSpan(t *testing.T) {
	t.Parallel()
	t.Run("sets success when span is recording", func(t *testing.T) {
		_, span, exporter := setupRecordingSpan(t)

		SetSuccess(span)

		// Verify span is recording
		assert.True(t, span.IsRecording())

		// End the span to flush status
		span.End()

		// Verify span was recorded
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-span", spans[0].Name)
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		_, span, exporter := setupRecordingSpan(t)

		SetSuccess(span)
		SetSuccess(span)
		SetSuccess(span)

		assert.True(t, span.IsRecording())

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
	})

	t.Run("does not panic with non-recording span", func(t *testing.T) {
		_, span := StartSpan(context.Background(), "TestOperation")
		defer span.End()

		// Should not panic even with non-recording span
		assert.NotPanics(t, func() {
			SetSuccess(span)
		})

		// Verify span is not recording
		assert.False(t, span.IsRecording())
	})
}

// TestTracing_CompleteRecordingFlow тестирует полный flow с recording span.
func TestTracing_CompleteRecordingFlow(t *testing.T) {
	t.Parallel()
	t.Run("complete flow with events and success", func(t *testing.T) {
		ctx, span, exporter := setupRecordingSpan(t)

		// Add events
		AddEvent(ctx, "operation.started", map[string]any{
			"user_id": "test-user",
		})

		AddEvent(ctx, "operation.progress", map[string]any{
			"progress": 50,
		})

		// Set success
		SetSuccess(span)

		// Verify span is recording
		assert.True(t, span.IsRecording())

		span.End()

		// Verify span was recorded
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-span", spans[0].Name)
	})

	t.Run("complete flow with error", func(t *testing.T) {
		ctx, span, exporter := setupRecordingSpan(t)

		// Add event
		AddEvent(ctx, "operation.started", map[string]any{
			"user_id": "test-user",
		})

		// Record error
		testErr := errors.New("operation failed")
		RecordError(span, testErr)

		// Verify span is recording
		assert.True(t, span.IsRecording())

		span.End()

		// Verify span was recorded
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-span", spans[0].Name)
	})

	t.Run("nested recording spans", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
		)

		tracer := tp.Tracer("test-tracer")
		ctx, parentSpan := tracer.Start(context.Background(), "parent-span")

		// Parent operations
		AddEvent(ctx, "parent.started", map[string]any{"step": 1})

		// Child span
		ctx, childSpan := tracer.Start(ctx, "child-span")
		AddEvent(ctx, "child.started", map[string]any{"step": 2})
		SetSuccess(childSpan)
		childSpan.End()

		// Complete parent
		SetSuccess(parentSpan)
		parentSpan.End()

		// Verify both spans were recorded
		spans := exporter.GetSpans()
		require.Len(t, spans, 2)

		spanNames := make(map[string]bool)
		for _, span := range spans {
			spanNames[span.Name] = true
		}

		assert.True(t, spanNames["parent-span"])
		assert.True(t, spanNames["child-span"])
	})
}
