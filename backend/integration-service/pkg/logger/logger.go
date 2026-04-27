// Package logger предоставляет инициализацию логгера для пакета.
package logger

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

var (
	globalLogger *slog.Logger
	loggerMu     sync.RWMutex
)

// InitLogger инициализирует глобальный логгер с указанным уровнем.
func InitLogger(level string) error {
	var levelVar slog.Level
	switch level {
	case "debug":
		levelVar = slog.LevelDebug
	case "info":
		levelVar = slog.LevelInfo
	case "warn":
		levelVar = slog.LevelWarn
	case "error":
		levelVar = slog.LevelError
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}

	opts := &slog.HandlerOptions{
		Level: levelVar,
	}

	loggerMu.Lock()
	defer loggerMu.Unlock()

	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(globalLogger)

	return nil
}

// Default возвращает глобальный логгер.
func Default() *slog.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()

	if globalLogger == nil {
		return slog.Default()
	}
	return globalLogger
}
