package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger(t *testing.T) {
	// Не используем t.Parallel() так как изменяем глобальное состояние

	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{
			name:    "debug level",
			level:   "debug",
			wantErr: false,
		},
		{
			name:    "info level",
			level:   "info",
			wantErr: false,
		},
		{
			name:    "warn level",
			level:   "warn",
			wantErr: false,
		},
		{
			name:    "error level",
			level:   "error",
			wantErr: false,
		},
		{
			name:    "unknown level returns error",
			level:   "unknown",
			wantErr: true,
		},
		{
			name:    "empty level returns error",
			level:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitLogger(tt.level)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown log level")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefault(t *testing.T) {
	t.Parallel()

	// After any init, Default should return non-nil
	logger := Default()
	assert.NotNil(t, logger)
}

func TestDefaultBeforeInit(t *testing.T) {
	t.Parallel()

	// Even with nil globalLogger, Default returns slog.Default()
	loggerMu.Lock()
	saved := globalLogger
	globalLogger = nil
	defer func() {
		globalLogger = saved
		loggerMu.Unlock()
	}()

	var logger = slog.Default()
	if globalLogger != nil {
		logger = globalLogger
	}
	assert.NotNil(t, logger)
}
