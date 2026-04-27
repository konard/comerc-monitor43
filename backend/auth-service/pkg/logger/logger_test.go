package logger

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	require.NotNil(t, l)
}

func TestDebug(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	// не должно паниковать
	l.Debug(context.Background(), "debug message")
}

func TestInfo(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l.Info(context.Background(), "info message")
}

func TestWarn(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l.Warn(context.Background(), "warn message")
}

func TestError(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l.Error(context.Background(), "error message", fmt.Errorf("some error"))
}

func TestError_nil_error(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l.Error(context.Background(), "error message", nil)
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l2 := l.WithRequestID("req-123")

	require.NotNil(t, l2)
	// не должно паниковать
	l2.Info(context.Background(), "with request id")
}

func TestWithUserID(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l2 := l.WithUserID("user-456")

	require.NotNil(t, l2)
	l2.Info(context.Background(), "with user id")
}

func TestWithEmail(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l2 := l.WithEmail("user@example.com")

	require.NotNil(t, l2)
	l2.Info(context.Background(), "with email")
}

func TestWithTrace(t *testing.T) {
	t.Parallel()

	l := New("test-service")
	l2 := l.WithTrace("trace-abc", "span-xyz")

	require.NotNil(t, l2)
	l2.Info(context.Background(), "with trace")
}

func TestChainedWith(t *testing.T) {
	t.Parallel()

	l := New("test-service").
		WithRequestID("req-1").
		WithUserID("usr-1").
		WithEmail("test@example.com").
		WithTrace("t-1", "s-1")

	require.NotNil(t, l)
	l.Info(context.Background(), "chained")
}

func TestSlogLevel(t *testing.T) {
	t.Parallel()

	cases := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal, Level("unknown")}
	for _, level := range cases {
		t.Run(string(level), func(t *testing.T) {
			t.Parallel()
			// вызов не должен паниковать
			_ = slogLevel(level)
		})
	}
}

func TestExtractTraceID_no_span(t *testing.T) {
	t.Parallel()

	// без реального span → пустая строка
	traceID := ExtractTraceID(context.Background())
	assert.Equal(t, "", traceID)
}

func TestExtractSpanID_no_span(t *testing.T) {
	t.Parallel()

	spanID := ExtractSpanID(context.Background())
	assert.Equal(t, "", spanID)
}
