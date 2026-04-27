// Package logger предоставляет структурированное логирование для monitor-service.
//
// Пакет инициализирует slog logger с JSON форматированием
// и предоставляет утилиты для создания логгеров с контекстом.
package logger

import (
	"log/slog"
	"os"
)

var (
	// DefaultLogger дефолтный logger
	DefaultLogger *slog.Logger
)

func init() {
	// Инициализируем JSON логгер для production
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	DefaultLogger = slog.New(handler)
}

// InitLogger инициализирует глобальный логгер.
func InitLogger(level string) error {
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
	DefaultLogger = slog.New(handler)
	slog.SetDefault(DefaultLogger)

	return nil
}

// With возвращает logger с дополнительными полями.
func With(args ...any) *slog.Logger {
	return DefaultLogger.With(args...)
}

// FromContext возвращает logger из контекста или дефолтный.
func FromContext(_ any) *slog.Logger {
	return DefaultLogger
}
