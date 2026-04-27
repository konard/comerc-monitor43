package logger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger(LevelInfo, nil)

	if logger.level != LevelInfo {
		t.Errorf("expected level %v, got %v", LevelInfo, logger.level)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
		{"unknown", LevelInfo},
		{"", LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  Level
		testLevel Level
		shouldLog bool
	}{
		{"debug logs at debug level", LevelDebug, LevelDebug, true},
		{"debug logs at info level", LevelDebug, LevelInfo, true},
		{"info does not log at debug level", LevelInfo, LevelDebug, false},
		{"info logs at info level", LevelInfo, LevelInfo, true},
		{"warn does not log at error level", LevelError, LevelWarn, false},
		{"error logs at error level", LevelError, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.logLevel, &buf)

			msg := "test message"

			switch tt.testLevel {
			case LevelDebug:
				logger.Debug(msg)
			case LevelInfo:
				logger.Info(msg)
			case LevelWarn:
				logger.Warn(msg)
			case LevelError:
				logger.Error(msg)
			}

			output := buf.String()
			didLog := output != ""

			if didLog != tt.shouldLog {
				t.Errorf("expected didLog=%v, got %v", tt.shouldLog, didLog)
			}

			if didLog && !strings.Contains(output, msg) {
				t.Errorf("expected output to contain %q, got %q", msg, output)
			}
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	logger.Info("test message",
		StringField("user_id", "123"),
		IntField("count", 42),
		DurationField("latency", 100*time.Millisecond),
	)

	output := buf.String()

	if !strings.Contains(output, "user_id") {
		t.Error("expected output to contain user_id")
	}
	if !strings.Contains(output, "123") {
		t.Error("expected output to contain 123")
	}
	if !strings.Contains(output, "count") {
		t.Error("expected output to contain count")
	}
	if !strings.Contains(output, "42") {
		t.Error("expected output to contain 42")
	}
	if !strings.Contains(output, "latency") {
		t.Error("expected output to contain latency")
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	loggerWithField := logger.With(StringField("request_id", "abc-123"))
	loggerWithField.Info("test message")

	output := buf.String()

	if !strings.Contains(output, "request_id") {
		t.Error("expected output to contain request_id")
	}
	if !strings.Contains(output, "abc-123") {
		t.Error("expected output to contain abc-123")
	}
}

func TestLoggerStructuredOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	logger.Info("structured message",
		StringField("service", "api-gateway"),
		IntField("status", 200),
	)

	output := buf.String()

	// Verify JSON structure
	if !strings.Contains(output, `"level"`) {
		t.Error("expected output to contain level field")
	}
	if !strings.Contains(output, `"message"`) {
		t.Error("expected output to contain message field")
	}
	if !strings.Contains(output, `"timestamp"`) {
		t.Error("expected output to contain timestamp field")
	}
}

func TestFieldHelpers(t *testing.T) {
	t.Run("StringField", func(t *testing.T) {
		f := StringField("key", "value")
		if f.Key != "key" || f.Value != "value" {
			t.Errorf("StringField() = %+v, want Key=key, Value=value", f)
		}
	})

	t.Run("IntField", func(t *testing.T) {
		f := IntField("key", 42)
		if f.Key != "key" || f.Value != 42 {
			t.Errorf("IntField() = %+v, want Key=key, Value=42", f)
		}
	})

	t.Run("DurationField", func(t *testing.T) {
		f := DurationField("key", 100*time.Millisecond)
		if f.Key != "key" {
			t.Error("DurationField() key mismatch")
		}
		if _, ok := f.Value.(int64); !ok {
			t.Error("DurationField() should return int64 value")
		}
	})

	t.Run("ErrorField", func(t *testing.T) {
		err := fmt.Errorf("test error")
		f := ErrorField(err)
		if f.Key != "error" {
			t.Error("ErrorField() key should be 'error'")
		}
		if f.Value != "test error" {
			t.Errorf("ErrorField() value = %v, want 'test error'", f.Value)
		}
	})
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.expected {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger("info")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.level != LevelInfo {
		t.Errorf("expected level %v, got %v", LevelInfo, logger.level)
	}
}

func TestLoggerWithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LevelInfo, &buf)

	child := logger.WithField("trace_id", "xyz-999")
	child.Info("with field message")

	output := buf.String()
	if !strings.Contains(output, "trace_id") {
		t.Error("expected output to contain trace_id")
	}
	if !strings.Contains(output, "xyz-999") {
		t.Error("expected output to contain xyz-999")
	}
}

func TestLoggerWarnFiltered(t *testing.T) {
	// Logger at Error level must not log Warn messages.
	var buf bytes.Buffer
	logger := NewLogger(LevelError, &buf)

	logger.Warn("this should be filtered")

	if buf.Len() != 0 {
		t.Errorf("expected no output for Warn at Error level, got: %q", buf.String())
	}
}

func TestLoggerWarnLogged(t *testing.T) {
	// Logger at Warn level must log Warn messages.
	var buf bytes.Buffer
	logger := NewLogger(LevelWarn, &buf)

	logger.Warn("warn message logged")

	output := buf.String()
	if !strings.Contains(output, "warn message logged") {
		t.Errorf("expected output to contain warn message, got: %q", output)
	}
}

func TestLoggerWithInheritsFields(t *testing.T) {
	var buf bytes.Buffer
	base := NewLogger(LevelInfo, &buf)
	base = base.WithField("service", "gateway")

	child := base.With(StringField("handler", "login"))
	child.Info("inherited fields test")

	output := buf.String()
	if !strings.Contains(output, "service") {
		t.Error("expected output to contain inherited service field")
	}
	if !strings.Contains(output, "handler") {
		t.Error("expected output to contain handler field")
	}
}
