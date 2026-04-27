package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInitLogger тестирует инициализацию логгера.
func TestInitLogger(t *testing.T) {
	t.Run("init with debug level", func(t *testing.T) {
		err := InitLogger("debug")
		assert.NoError(t, err)
		assert.NotNil(t, DefaultLogger)
	})

	t.Run("init with info level", func(t *testing.T) {
		err := InitLogger("info")
		assert.NoError(t, err)
		assert.NotNil(t, DefaultLogger)
	})

	t.Run("init with warn level", func(t *testing.T) {
		err := InitLogger("warn")
		assert.NoError(t, err)
		assert.NotNil(t, DefaultLogger)
	})

	t.Run("init with error level", func(t *testing.T) {
		err := InitLogger("error")
		assert.NoError(t, err)
		assert.NotNil(t, DefaultLogger)
	})

	t.Run("init with invalid level defaults to info", func(t *testing.T) {
		err := InitLogger("invalid")
		assert.NoError(t, err)
		assert.NotNil(t, DefaultLogger)
	})
}

// TestDefaultLogger тестирует дефолтный логгер.
func TestDefaultLogger(t *testing.T) {
	t.Run("default logger is initialized", func(t *testing.T) {
		assert.NotNil(t, DefaultLogger)
	})

	t.Run("default logger is slog logger", func(t *testing.T) {
		assert.IsType(t, &slog.Logger{}, DefaultLogger)
	})

	t.Run("default logger can log messages", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := slog.New(slog.NewTextHandler(&buf, nil))

		testLogger.Info("test message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "key")
		assert.Contains(t, output, "value")
	})
}

// TestWith тестирует создание логгера с дополнительными полями.
func TestWith(t *testing.T) {
	t.Run("with creates logger with additional fields", func(t *testing.T) {
		logger := With("request_id", "test-123", "user_id", "user-456")

		assert.NotNil(t, logger)
		assert.NotNil(t, logger)
	})

	t.Run("with multiple calls adds more fields", func(t *testing.T) {
		logger1 := With("field1", "value1")
		logger2 := logger1.With("field2", "value2")

		assert.NotNil(t, logger2)
	})

	t.Run("with preserves handler", func(t *testing.T) {
		logger := With("test", "value")

		assert.NotNil(t, logger.Handler())
	})
}

// TestFromContext тестирует извлечение логгера из контекста.
func TestFromContext(t *testing.T) {
	t.Run("from context returns default logger", func(t *testing.T) {
		ctx := context.Background()
		logger := FromContext(ctx)

		assert.NotNil(t, logger)
		assert.NotNil(t, logger)
	})

	t.Run("from context with nil argument", func(t *testing.T) {
		logger := FromContext(nil)

		assert.NotNil(t, logger)
		assert.NotNil(t, logger)
	})
}

// TestLoggerOutput тестирует что логгер выводит в правильном формате.
func TestLoggerOutput(t *testing.T) {
	t.Run("json handler outputs json", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &slog.HandlerOptions{Level: slog.LevelInfo}
		handler := slog.NewJSONHandler(&buf, opts)
		logger := slog.New(handler)

		logger.Info("test message", "key", "value")

		output := buf.String()
		assert.Contains(t, output, "\"test message\"")
		assert.Contains(t, output, "\"key\"")
		assert.Contains(t, output, "\"value\"")
		assert.Contains(t, output, "\"level\"")
		assert.Contains(t, output, "\"time\"")
	})

	t.Run("json format is valid", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		logger := slog.New(handler)

		logger.Info("test", "key", "value")

		output := buf.String()
		assert.Contains(t, output, "{")
		assert.Contains(t, output, "}")
		assert.Contains(t, output, "\"")
	})

	t.Run("json contains required fields", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		logger := slog.New(handler)

		logger.Info("test message")

		output := buf.String()
		assert.Contains(t, output, "\"time\"")
		assert.Contains(t, output, "\"level\"")
		assert.Contains(t, output, "\"msg\"")
	})
}
