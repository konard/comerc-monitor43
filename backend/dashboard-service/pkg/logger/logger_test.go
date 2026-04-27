package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_default_level(t *testing.T) {
	t.Parallel()

	l := New("info")

	require.NotNil(t, l)
}

func TestNew_all_levels(t *testing.T) {
	t.Parallel()

	for _, level := range []string{"debug", "info", "warn", "error", "unknown"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()

			l := New(level)
			assert.NotNil(t, l)
		})
	}
}

func TestLogger_methods_do_not_panic(t *testing.T) {
	t.Parallel()

	l := New("error")

	assert.NotPanics(t, func() { l.Debug("debug message", "key", "value") })
	assert.NotPanics(t, func() { l.Info("info message", "key", "value") })
	assert.NotPanics(t, func() { l.Warn("warn message", "key", "value") })
	assert.NotPanics(t, func() { l.Error("error message", "key", "value") })
}

func TestLogger_With_returns_new_logger(t *testing.T) {
	t.Parallel()

	l := New("info")

	child := l.With("component", "test")

	require.NotNil(t, child)
	assert.NotSame(t, l, child)
}
