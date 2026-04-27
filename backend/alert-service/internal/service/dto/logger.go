package dto

import (
	"log/slog"
)

// Logger предоставляет structured logging интерфейс
type Logger struct {
	*slog.Logger
}

// NewLogger создаёт новый structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: slog.Default(),
	}
}
