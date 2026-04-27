package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.Equal(t, 2*time.Second, cfg.BaseDelay)
	assert.Equal(t, 60*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.True(t, cfg.Jitter)
}

func TestDoWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	attempts := 0

	fn := func(ctx context.Context) error {
		attempts++
		return nil
	}

	err := DoWithRetry(ctx, cfg, fn)

	require.NoError(t, err)
	assert.Equal(t, 1, attempts, "should succeed on first attempt")
}

func TestDoWithRetry_SuccessOnSecondAttempt(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false, // отключаем для стабильного теста
	}
	attempts := 0

	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	start := time.Now()
	err := DoWithRetry(ctx, cfg, fn)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "should succeed on second attempt")
	assert.GreaterOrEqual(t, elapsed, cfg.BaseDelay, "should wait at least BaseDelay")
}

func TestDoWithRetry_MaxAttemptsReached(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}
	attempts := 0

	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("persistent error")
	}

	err := DoWithRetry(ctx, cfg, fn)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retry attempts")
	assert.Equal(t, 3, attempts, "should attempt exactly MaxAttempts times")
}

func TestDoWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	attempts := 0

	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("non-retryable error")
	}

	isRetryable := func(err error) bool {
		return false // все ошибки не retryable
	}

	err := DoWithRetryAndIsRetryable(ctx, cfg, fn, isRetryable)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-retryable error")
	assert.Equal(t, 1, attempts, "should not retry non-retryable error")
}

func TestDoWithRetry_ContextCanceled(t *testing.T) {
	cfg := Config{
		MaxAttempts: 10,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	fn := func(ctx context.Context) error {
		attempts++
		if attempts == 2 {
			// Отменяем контекст после второй попытки
			cancel()
		}
		return errors.New("error")
	}

	err := DoWithRetry(ctx, cfg, fn)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry canceled")
	assert.LessOrEqual(t, attempts, 3, "should stop after context cancellation")
}

func TestCalculateDelay(t *testing.T) {
	cfg := Config{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
	}

	tests := []struct {
		name        string
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "first retry (attempt 1)",
			attempt:     1,
			expectedMin: 100 * time.Millisecond, // 100ms * 2^0
			expectedMax: 100 * time.Millisecond,
		},
		{
			name:        "second retry (attempt 2)",
			attempt:     2,
			expectedMin: 200 * time.Millisecond, // 100ms * 2^1
			expectedMax: 200 * time.Millisecond,
		},
		{
			name:        "third retry (attempt 3)",
			attempt:     3,
			expectedMin: 400 * time.Millisecond, // 100ms * 2^2
			expectedMax: 400 * time.Millisecond,
		},
		{
			name:        "large retry hits max delay",
			attempt:     20,
			expectedMin: 1 * time.Second, // capped at MaxDelay
			expectedMax: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateDelay(cfg, tt.attempt)
			assert.GreaterOrEqual(t, delay, tt.expectedMin)
			assert.LessOrEqual(t, delay, tt.expectedMax)
		})
	}
}

func TestAddJitter(t *testing.T) {
	baseDelay := 100 * time.Millisecond

	// Тестируем что jitter добавляет вариативность
	delays := make([]time.Duration, 100)
	for i := range delays {
		delays[i] = addJitter(baseDelay)
	}

	minDelay := delays[0]
	maxDelay := delays[0]
	for _, d := range delays[1:] {
		if d < minDelay {
			minDelay = d
		}
		if d > maxDelay {
			maxDelay = d
		}
	}

	// Jitter должен быть в пределах ±25%
	expectedMin := time.Duration(float64(baseDelay) * 0.75)
	expectedMax := time.Duration(float64(baseDelay) * 1.25)

	assert.LessOrEqual(t, minDelay, baseDelay, "jitter should sometimes reduce delay")
	assert.GreaterOrEqual(t, maxDelay, baseDelay, "jitter should sometimes increase delay")
	assert.GreaterOrEqual(t, minDelay, expectedMin-time.Millisecond, "min jitter should be within -25%")
	assert.LessOrEqual(t, maxDelay, expectedMax+time.Millisecond, "max jitter should be within +25%")
}

func TestDoWithRetry_ExponentialBackoffTiming(t *testing.T) {
	cfg := Config{
		MaxAttempts: 4,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	}

	start := time.Now()
	err := DoWithRetry(context.Background(), cfg, fn)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Equal(t, 4, attempts)

	// Expected delays: 0ms (first attempt) + 50ms + 100ms + 200ms = 350ms
	expectedMin := 350 * time.Millisecond
	expectedMax := 400 * time.Millisecond // small overhead

	assert.GreaterOrEqual(t, elapsed, expectedMin, "should wait for exponential backoff")
	assert.LessOrEqual(t, elapsed, expectedMax, "should not wait significantly longer than expected")
}
