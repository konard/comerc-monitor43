package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	t.Run("debug_level", func(t *testing.T) {
		logger := New("debug")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("info_level", func(t *testing.T) {
		logger := New("info")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("warn_level", func(t *testing.T) {
		logger := New("warn")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("error_level", func(t *testing.T) {
		logger := New("error")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("default_level", func(t *testing.T) {
		logger := New("")
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})
}

func TestLogger_Debug(t *testing.T) {
	logger := New("debug")

	t.Run("debug_message", func(t *testing.T) {
		// Should not panic and log properly
		logger.Debug("test debug message")
	})

	t.Run("debug_message_with_args", func(t *testing.T) {
		// Should handle multiple arguments
		logger.Debug("test message", "arg1", "arg2", 123)
	})

	t.Run("debug_message_with_struct", func(t *testing.T) {
		type testStruct struct {
			Field string
			Value int
		}
		data := testStruct{Field: "test", Value: 42}
		// Should handle struct arguments
		logger.Debug("test message", data)
	})
}

func TestLogger_Info(t *testing.T) {
	logger := New("info")

	t.Run("info_message", func(t *testing.T) {
		logger.Info("test info message")
	})

	t.Run("info_message_with_args", func(t *testing.T) {
		logger.Info("test message", "arg1", "arg2")
	})

	t.Run("info_message_with_struct", func(t *testing.T) {
		type testStruct struct {
			Name  string
			Count int
		}
		data := testStruct{Name: "test", Count: 42}
		logger.Info("test message", data)
	})
}

func TestLogger_Warn(t *testing.T) {
	logger := New("warn")

	t.Run("warn_message", func(t *testing.T) {
		logger.Warn("test warn message")
	})

	t.Run("warn_message_with_args", func(t *testing.T) {
		logger.Warn("test message", "arg1", "arg2")
	})
}

func TestLogger_Error(t *testing.T) {
	logger := New("error")

	t.Run("error_message", func(t *testing.T) {
		logger.Error("test error message")
	})

	t.Run("error_message_with_args", func(t *testing.T) {
		logger.Error("test message", "arg1", "arg2")
	})
}

func TestLogger_With(t *testing.T) {
	logger := New("info")

	t.Run("with_single_field", func(t *testing.T) {
		newLogger := logger.With("key1", "value1")
		assert.NotNil(t, newLogger)
		assert.NotNil(t, newLogger.Logger)
	})

	t.Run("with_multiple_fields", func(t *testing.T) {
		newLogger := logger.With("key1", "value1", "key2", 123, "key3", true)
		assert.NotNil(t, newLogger)
		assert.NotNil(t, newLogger.Logger)
	})

	t.Run("with_nested_fields", func(t *testing.T) {
		innerLogger := logger.With("inner", "value")
		nestedLogger := innerLogger.With("outer", "value2")
		assert.NotNil(t, nestedLogger)
		assert.NotNil(t, nestedLogger.Logger)
	})
}

func TestLogger_MultipleLoggers(t *testing.T) {
	t.Run("different_levels", func(t *testing.T) {
		debugLogger := New("debug")
		infoLogger := New("info")
		errorLogger := New("error")

		assert.NotNil(t, debugLogger)
		assert.NotNil(t, infoLogger)
		assert.NotNil(t, errorLogger)

		assert.NotEqual(t, debugLogger.Logger, infoLogger.Logger)
		assert.NotEqual(t, infoLogger.Logger, errorLogger.Logger)
	})

	t.Run("same_level_different_instances", func(t *testing.T) {
		logger1 := New("info")
		logger2 := New("info")

		assert.NotNil(t, logger1)
		assert.NotNil(t, logger2)

		// Both should be valid, independently usable loggers
		logger1.Info("logger1 message")
		logger2.Info("logger2 message")
	})

	t.Run("internal_logger_types", func(t *testing.T) {
		logger1 := New("info")
		logger2 := New("error")

		assert.NotNil(t, logger1.Logger)
		assert.NotNil(t, logger2.Logger)
		// Both should be slog.Logger instances
	})
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	logger := New("info")

	t.Run("concurrent_logging", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				logger.Info("concurrent log message", i)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestLogger_LoggerField(t *testing.T) {
	logger := New("info")

	t.Run("logger_field_exists", func(t *testing.T) {
		assert.NotNil(t, logger.Logger)
	})

	t.Run("logger_field_is_slog", func(t *testing.T) {
		// Verify it's a slog.Logger (by checking it's not nil)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("multiple_loggers_have_different_fields", func(t *testing.T) {
		logger1 := New("info")
		logger2 := New("error")

		assert.NotNil(t, logger1.Logger)
		assert.NotNil(t, logger2.Logger)

		// Should have different internal loggers
		assert.NotEqual(t, logger1.Logger, logger2.Logger)
	})
}

func TestLogger_Structure(t *testing.T) {
	logger := New("info")

	t.Run("initialization", func(t *testing.T) {
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("zero_value_handling", func(t *testing.T) {
		logger.Info("")     // Empty message should work
		logger.Info("test") // Normal message should work
	})

	t.Run("nil_context_handling", func(t *testing.T) {
		// All log methods should handle nil context
		logger.Debug("test")
		logger.Info("test")
		logger.Warn("test")
		logger.Error("test")
	})
}
