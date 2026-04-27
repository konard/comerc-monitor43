package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger представляет структурированный логгер
type Logger struct {
	*slog.Logger
}

// New создаёт новый логгер с указанным уровнем логирования
func New(level string) *Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{Logger: logger}
}

// Debug логирует отладочное сообщение
func (l *Logger) Debug(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelDebug, msg, args...)
}

// Info логирует информационное сообщение
func (l *Logger) Info(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelInfo, msg, args...)
}

// Warn логирует предупреждение
func (l *Logger) Warn(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelWarn, msg, args...)
}

// Error логирует ошибку
func (l *Logger) Error(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelError, msg, args...)
}

// With возвращает новый логгер с дополнительными полями
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// Errorf логирует ошибку с форматированием
func (l *Logger) Errorf(msg string, args ...any) {
	l.Error(msg, args...)
}
