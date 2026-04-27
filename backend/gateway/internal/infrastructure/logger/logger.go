package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Level represents the log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value any
}

// Logger is a structured logger
type Logger struct {
	level  Level
	logger *log.Logger
	fields map[string]any
}

// NewLogger creates a new structured logger
func NewLogger(level Level, out io.Writer) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(out, "", 0),
		fields: make(map[string]any),
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger(level string) *Logger {
	logLevel := parseLevel(level)
	return NewLogger(logLevel, os.Stdout)
}

// parseLevel parses a log level string
func parseLevel(level string) Level {
	switch level {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// With creates a new logger with additional fields
func (l *Logger) With(fields ...Field) *Logger {
	newLogger := &Logger{
		level:  l.level,
		logger: l.logger,
		fields: make(map[string]any),
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Add new fields
	for _, f := range fields {
		newLogger.fields[f.Key] = f.Value
	}
	return newLogger
}

// WithField adds a single field
func (l *Logger) WithField(key string, value any) *Logger {
	return l.With(Field{Key: key, Value: value})
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	if l.level <= LevelDebug {
		l.log("debug", msg, fields)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	if l.level <= LevelInfo {
		l.log("info", msg, fields)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	if l.level <= LevelWarn {
		l.log("warn", msg, fields)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Field) {
	if l.level <= LevelError {
		l.log("error", msg, fields)
	}
}

// log is the internal logging method
func (l *Logger) log(level, msg string, fields []Field) {
	entry := map[string]any{
		"level":     level,
		"message":   msg,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Add persistent fields
	for k, v := range l.fields {
		entry[k] = v
	}

	// Add temporary fields
	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal log entry: %v\n", err)
		return
	}

	l.logger.Println(string(jsonData))
}

// StringField creates a string field
func StringField(key, value string) Field {
	return Field{Key: key, Value: value}
}

// IntField creates an int field
func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// DurationField creates a duration field
func DurationField(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.Milliseconds()}
}

// ErrorField creates an error field
func ErrorField(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}
