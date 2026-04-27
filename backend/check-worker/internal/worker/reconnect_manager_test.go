package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultReconnectConfig(t *testing.T) {
	config := DefaultReconnectConfig()

	assert.Equal(t, 1*time.Second, config.InitialDelay)
	assert.Equal(t, 60*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 0, config.MaxAttempts) // Бесконечные попытки
	assert.True(t, config.Jitter)
}

func TestNewReconnectManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		MaxAttempts:  5,
		Jitter:       false,
	}

	rm := NewReconnectManager(config, logger)

	assert.NotNil(t, rm)
	assert.False(t, rm.IsActive())
	assert.Equal(t, 0, rm.GetCurrentAttempt())
	assert.Equal(t, 0, rm.GetTotalAttempts())
}

func TestReconnectManager_SuccessfulReconnect(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  3,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	attempts := 0

	err := rm.StartReconnect(ctx, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("connection failed")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, 2, rm.GetTotalAttempts())
	assert.False(t, rm.IsActive())
}

func TestReconnectManager_MaxAttemptsReached(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  3,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	attempts := 0

	err := rm.StartReconnect(ctx, func(ctx context.Context) error {
		attempts++
		return errors.New("always fails")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max reconnect attempts")
	assert.Equal(t, 3, attempts)
	assert.Equal(t, 3, rm.GetTotalAttempts())
}

func TestReconnectManager_ExponentialBackoff(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  3,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	attempts := 0

	start := time.Now()
	err := rm.StartReconnect(ctx, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("fails")
		}
		return nil
	})
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)

	// Проверяем что было ожидание (accumulated delay: 10ms + 20ms = 30ms минимум)
	assert.Greater(t, duration, 25*time.Millisecond)
}

func TestReconnectManager_Stop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		MaxAttempts:  0,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		rm.Stop()
	}()

	attempts := 0
	err := rm.StartReconnect(ctx, func(ctx context.Context) error {
		attempts++
		return errors.New("always fails")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reconnect stopped")
	assert.LessOrEqual(t, attempts, 2) // Должен быть остановлен быстро
}

func TestReconnectManager_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		MaxAttempts:  0,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	attempts := 0
	err := rm.StartReconnect(ctx, func(ctx context.Context) error {
		attempts++
		return errors.New("always fails")
	})

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.LessOrEqual(t, attempts, 2)
}

func TestReconnectManager_GetAttempts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  2,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	assert.Error(t, rm.StartReconnect(ctx, func(ctx context.Context) error {
		return errors.New("fails")
	}))

	// Ждём завершения всех попыток
	time.Sleep(100 * time.Millisecond)

	attempts := rm.GetAttempts()
	assert.Len(t, attempts, 2)
	assert.Equal(t, 1, attempts[0].AttemptNumber)
	assert.Equal(t, 2, attempts[1].AttemptNumber)
	assert.NotNil(t, attempts[0].Error)
	assert.NotNil(t, attempts[1].Error)
}

func TestReconnectManager_GetStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  2,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	assert.Error(t, rm.StartReconnect(ctx, func(ctx context.Context) error {
		return errors.New("fails")
	}))

	// Ждём завершения всех попыток
	time.Sleep(100 * time.Millisecond)

	stats := rm.GetStats()
	assert.Equal(t, 2, stats["total_attempts"])
	assert.Equal(t, 2, stats["current_attempt"])
	assert.False(t, stats["is_active"].(bool))
	assert.NotNil(t, stats["last_attempt_error"])
	assert.NotNil(t, stats["last_attempt_timestamp"])
	assert.NotNil(t, stats["first_attempt_timestamp"])
	assert.NotNil(t, stats["total_duration"])
}

func TestReconnectManager_Reset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  2,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	assert.Error(t, rm.StartReconnect(ctx, func(ctx context.Context) error {
		return errors.New("fails")
	}))

	// Ждём завершения всех попыток
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, rm.GetTotalAttempts())
	assert.Equal(t, 2, rm.GetCurrentAttempt())

	rm.Reset()

	assert.Equal(t, 0, rm.GetTotalAttempts())
	assert.Equal(t, 0, rm.GetCurrentAttempt())
	assert.False(t, rm.IsActive())
}

func TestReconnectManager_GetLastAttemptError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
		MaxAttempts:  2,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	assert.Error(t, rm.StartReconnect(ctx, func(ctx context.Context) error {
		return expectedErr
	}))

	// Ждём завершения всех попыток
	time.Sleep(100 * time.Millisecond)

	lastErr := rm.GetLastAttemptError()
	assert.Error(t, lastErr)
	assert.Equal(t, expectedErr, lastErr)
}

func TestReconnectManager_AlreadyActive(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := DefaultReconnectConfig()
	rm := NewReconnectManager(config, logger)

	ctx, cancel := context.WithCancel(context.Background())
	reconnectErrCh := make(chan error, 1)

	// Запускаем первую попытку в фоне
	go func() {
		reconnectErrCh <- rm.StartReconnect(ctx, func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
	}()

	time.Sleep(10 * time.Millisecond) // Даём первой попытке начаться

	// Пытаемся запустить вторую
	err := rm.StartReconnect(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")

	cancel()

	select {
	case reconnectErr := <-reconnectErrCh:
		require.ErrorIs(t, reconnectErr, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("reconnect goroutine did not finish")
	}
}

func TestCalculateDelay(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := ReconnectConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1000 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}
	rm := NewReconnectManager(config, logger)

	tests := []struct {
		attempt          int
		expectedMinDelay time.Duration
		expectedMaxDelay time.Duration
	}{
		{1, 100 * time.Millisecond, 100 * time.Millisecond},
		{2, 200 * time.Millisecond, 200 * time.Millisecond},
		{3, 400 * time.Millisecond, 400 * time.Millisecond},
		{4, 800 * time.Millisecond, 800 * time.Millisecond},
		{5, 1000 * time.Millisecond, 1000 * time.Millisecond}, // Max delay
		{6, 1000 * time.Millisecond, 1000 * time.Millisecond}, // Max delay
	}

	for _, tt := range tests {
		t.Run(tt.expectedMinDelay.String(), func(t *testing.T) {
			delay := rm.calculateDelay(tt.attempt)
			assert.GreaterOrEqual(t, delay, tt.expectedMinDelay)
			assert.LessOrEqual(t, delay, tt.expectedMaxDelay)
		})
	}
}
