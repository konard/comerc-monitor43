package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

// ReconnectConfig содержит конфигурацию для переподключения
type ReconnectConfig struct {
	InitialDelay time.Duration // Начальная задержка перед переподключением
	MaxDelay     time.Duration // Максимальная задержка
	Multiplier   float64       // Множитель для экспоненциального backoff
	MaxAttempts  int           // Максимальное количество попыток (0 = бесконечно)
	Jitter       bool          // Добавлять случайный jitter
}

// DefaultReconnectConfig возвращает конфигурацию по умолчанию
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		MaxAttempts:  0, // Бесконечные попытки
		Jitter:       true,
	}
}

// ReconnectAttemptInfo содержит информацию о попытке переподключения
type ReconnectAttemptInfo struct {
	AttemptNumber int
	Delay         time.Duration
	Error         error
	Timestamp     time.Time
}

// ReconnectManager управляет переподключением с exponential backoff
type ReconnectManager struct {
	mu             sync.RWMutex
	config         ReconnectConfig
	logger         *slog.Logger
	attempts       []ReconnectAttemptInfo
	currentAttempt int
	stopChan       chan struct{}
	active         bool
}

// NewReconnectManager создаёт новый менеджер переподключения
func NewReconnectManager(config ReconnectConfig, logger *slog.Logger) *ReconnectManager {
	if config.InitialDelay <= 0 {
		config.InitialDelay = 1 * time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 60 * time.Second
	}
	if config.Multiplier <= 1.0 {
		config.Multiplier = 2.0
	}

	return &ReconnectManager{
		config:   config,
		logger:   logger,
		attempts: make([]ReconnectAttemptInfo, 0),
		stopChan: make(chan struct{}),
	}
}

// StartReconnect запускает процесс переподключения
func (rm *ReconnectManager) StartReconnect(
	ctx context.Context,
	reconnectFunc func(ctx context.Context) error,
) error {
	rm.mu.Lock()
	if rm.active {
		rm.mu.Unlock()
		return fmt.Errorf("reconnect already in progress")
	}
	rm.active = true
	rm.currentAttempt = 0
	rm.attempts = make([]ReconnectAttemptInfo, 0)
	rm.stopChan = make(chan struct{})
	rm.mu.Unlock()

	defer func() {
		rm.mu.Lock()
		rm.active = false
		rm.mu.Unlock()
	}()

	return rm.reconnectLoop(ctx, reconnectFunc)
}

// reconnectLoop выполняет цикл переподключения с exponential backoff
func (rm *ReconnectManager) reconnectLoop(
	ctx context.Context,
	reconnectFunc func(ctx context.Context) error,
) error {
	for {
		rm.mu.RLock()
		stopChan := rm.stopChan
		rm.mu.RUnlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stopChan:
			return fmt.Errorf("reconnect stopped")
		default:
		}

		rm.mu.Lock()
		attemptNum := rm.currentAttempt + 1 // Следующая попытка

		// Проверяем max attempts перед попыткой
		if rm.config.MaxAttempts > 0 && attemptNum > rm.config.MaxAttempts {
			rm.mu.Unlock()
			return fmt.Errorf("max reconnect attempts (%d) reached", rm.config.MaxAttempts)
		}

		rm.currentAttempt = attemptNum

		// Вычисляем задержку с exponential backoff
		delay := rm.calculateDelay(attemptNum)

		attemptInfo := ReconnectAttemptInfo{
			AttemptNumber: attemptNum,
			Delay:         delay,
			Timestamp:     time.Now(),
		}
		rm.attempts = append(rm.attempts, attemptInfo)
		rm.mu.Unlock()

		// Логируем попытку
		rm.logger.Info("reconnect attempt",
			"attempt", attemptNum,
			"delay", delay,
		)

		// Ждём перед попыткой
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stopChan:
			return fmt.Errorf("reconnect stopped")
		case <-time.After(delay):
		}

		// Выполняем попытку переподключения
		err := reconnectFunc(ctx)

		rm.mu.Lock()
		if len(rm.attempts) > 0 {
			rm.attempts[len(rm.attempts)-1].Error = err
		}
		rm.mu.Unlock()

		if err == nil {
			// Успешное переподключение
			rm.logger.Info("reconnect successful",
				"attempt", attemptNum,
				"total_duration", time.Since(attemptInfo.Timestamp),
			)
			return nil
		}

		// Ошибка переподключения
		rm.logger.Warn("reconnect attempt failed",
			"attempt", attemptNum,
			"error", err,
		)
	}
}

// calculateDelay вычисляет задержку с exponential backoff и jitter
func (rm *ReconnectManager) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: initial * (multiplier ^ (attempt - 1))
	delay := float64(rm.config.InitialDelay) * math.Pow(rm.config.Multiplier, float64(attempt-1))

	// Ограничиваем максимальную задержку
	if delay > float64(rm.config.MaxDelay) {
		delay = float64(rm.config.MaxDelay)
	}

	// Добавляем jitter если включено
	if rm.config.Jitter {
		// Jitter: +/- 25% от задержки
		jitterRange := delay * 0.25
		delay = delay - jitterRange + (2 * jitterRange * float64(time.Now().UnixNano()%1000) / 1000)
	}

	return time.Duration(delay)
}

// Stop останавливает процесс переподключения
func (rm *ReconnectManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.active {
		close(rm.stopChan)
	}
}

// IsActive проверяет, активен ли процесс переподключения
func (rm *ReconnectManager) IsActive() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.active
}

// GetAttempts возвращает информацию о попытках переподключения
func (rm *ReconnectManager) GetAttempts() []ReconnectAttemptInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Возвращаем копию чтобы избежать race conditions
	attempts := make([]ReconnectAttemptInfo, len(rm.attempts))
	copy(attempts, rm.attempts)
	return attempts
}

// GetCurrentAttempt возвращает номер текущей попытки
func (rm *ReconnectManager) GetCurrentAttempt() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.currentAttempt
}

// Reset сбрасывает состояние менеджера
func (rm *ReconnectManager) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.currentAttempt = 0
	rm.attempts = make([]ReconnectAttemptInfo, 0)
	rm.active = false
}

// GetTotalAttempts возвращает общее количество попыток
func (rm *ReconnectManager) GetTotalAttempts() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.attempts)
}

// GetLastAttemptError возвращает ошибку последней попытки
func (rm *ReconnectManager) GetLastAttemptError() error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if len(rm.attempts) == 0 {
		return nil
	}

	return rm.attempts[len(rm.attempts)-1].Error
}

// GetStats возвращает статистику переподключения
func (rm *ReconnectManager) GetStats() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := map[string]any{
		"current_attempt": rm.currentAttempt,
		"total_attempts":  len(rm.attempts),
		"is_active":       rm.active,
	}

	if len(rm.attempts) > 0 {
		lastAttempt := rm.attempts[len(rm.attempts)-1]
		stats["last_attempt_error"] = lastAttempt.Error
		stats["last_attempt_timestamp"] = lastAttempt.Timestamp
		stats["first_attempt_timestamp"] = rm.attempts[0].Timestamp
		stats["total_duration"] = time.Since(rm.attempts[0].Timestamp)
	}

	return stats
}
