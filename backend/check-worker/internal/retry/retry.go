package retry

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"
)

// Config представляет конфигурацию retry механизма.
type Config struct {
	// MaxAttempts - максимальное количество попыток (включая первую)
	MaxAttempts int

	// BaseDelay - начальная задержка перед первой retry
	BaseDelay time.Duration

	// MaxDelay - максимальная задержка между попытками
	MaxDelay time.Duration

	// Multiplier - множитель для экспоненциального увеличения задержки
	Multiplier float64

	// Jitter - добавлять случайный jitter для предотвращения thundering herd
	Jitter bool
}

// DefaultConfig возвращает конфигурацию с разумными значениями по умолчанию.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 5,
		BaseDelay:   2 * time.Second,
		MaxDelay:    60 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
	}
}

// RetryableFunc представляет функцию, которая может быть повторена при ошибке.
type RetryableFunc func(ctx context.Context) error

// IsRetryable определяет, можно ли повторить операцию при данной ошибке.
type IsRetryable func(error) bool

// DefaultIsRetryable возвращает true для временных ошибок, которые стоит повторить.
func DefaultIsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// TODO: Добавить более sophisticated проверку
	// Например, проверку на gRPC status codes, temporary errors, etc.
	return true
}

// DoWithRetry выполняет функцию с retry логикой и exponential backoff.
func DoWithRetry(ctx context.Context, cfg Config, fn RetryableFunc) error {
	return DoWithRetryAndIsRetryable(ctx, cfg, fn, DefaultIsRetryable)
}

// DoWithRetryAndIsRetryable выполняет функцию с retry логикой и custom проверкой.
func DoWithRetryAndIsRetryable(
	ctx context.Context,
	cfg Config,
	fn RetryableFunc,
	isRetryable IsRetryable,
) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Вычисляем задержку
			delay := calculateDelay(cfg, attempt)

			// Добавляем jitter если включен
			if cfg.Jitter {
				delay = addJitter(delay)
			}

			// Ждем перед следующей попыткой
			select {
			case <-time.After(delay):
				// Продолжаем
			case <-ctx.Done():
				return fmt.Errorf("retry canceled after %d attempts: %w", attempt, ctx.Err())
			}
		}

		// Выполняем функцию
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Проверяем, можно ли повторить
		if !isRetryable(err) {
			return fmt.Errorf("non-retryable error on attempt %d: %w", attempt+1, err)
		}

		// Логируем неудачную попытку
		// TODO: Добавить структурированное логирование
	}

	return fmt.Errorf("max retry attempts (%d) reached, last error: %w", cfg.MaxAttempts, lastErr)
}

// calculateDelay вычисляет задержку для данной попытки с exponential backoff.
func calculateDelay(cfg Config, attempt int) time.Duration {
	// exponential backoff: baseDelay * multiplier^(attempt-1)
	exponential := float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))

	delay := time.Duration(exponential)
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}

	return delay
}

// addJitter добавляет случайный jitter до ±25% от задержки.
func addJitter(delay time.Duration) time.Duration {
	if delay == 0 {
		return delay
	}

	// Jitter ±25%
	jitterRange := float64(delay) * 0.25
	randomValue, err := rand.Int(rand.Reader, big.NewInt(1<<53))
	if err != nil {
		return delay
	}
	randomFloat := float64(randomValue.Int64()) / float64(1<<53)
	jitter := time.Duration(randomFloat*2*float64(jitterRange) - float64(jitterRange))

	return delay + jitter
}
