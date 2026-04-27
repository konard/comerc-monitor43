package logger

import (
	"context"
	"log/slog"
	"os"
)

type Logger struct{ *slog.Logger }

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
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel})
	return &Logger{Logger: slog.New(handler)}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelDebug, msg, args...)
}
func (l *Logger) Info(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelInfo, msg, args...)
}
func (l *Logger) Warn(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelWarn, msg, args...)
}
func (l *Logger) Error(msg string, args ...any) {
	l.Log(context.Background(), slog.LevelError, msg, args...)
}
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}
