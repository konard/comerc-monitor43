package apikey

import (
	"context"
	"sync"
	"time"
)

// RateLimiter интерфейс для rate limiting.
type RateLimiter interface {
	// Allow проверяет, разрешён ли запрос.
	Allow(ctx context.Context, apiKeyID string, requestsPerMinute int) bool
	// Reset сбрасывает счётчик для API ключа.
	Reset(apiKeyID string)
}

// SlidingWindowRateLimiter реализует sliding window rate limiting в памяти.
// Для распределённых систем лучше использовать Redis-based implementation.
type SlidingWindowRateLimiter struct {
	mu              sync.RWMutex
	windows         map[string]*slidingWindow
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// slidingWindow окно для одного API ключа.
type slidingWindow struct {
	requests    []time.Time
	windowStart time.Time
	windowSize  time.Duration
}

// NewSlidingWindowRateLimiter создаёт новый SlidingWindowRateLimiter.
func NewSlidingWindowRateLimiter() *SlidingWindowRateLimiter {
	rl := &SlidingWindowRateLimiter{
		windows:         make(map[string]*slidingWindow),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	// Запускаем cleanup горутину
	go rl.cleanup()

	return rl
}

// Allow проверяет, разрешён ли запрос.
func (rl *SlidingWindowRateLimiter) Allow(ctx context.Context, apiKeyID string, requestsPerMinute int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowSize := time.Minute

	// Получаем или создаём окно для ключа
	window, exists := rl.windows[apiKeyID]
	if !exists {
		window = &slidingWindow{
			requests:    make([]time.Time, 0, requestsPerMinute),
			windowStart: now,
			windowSize:  windowSize,
		}
		rl.windows[apiKeyID] = window
	}

	// Если окно устарело (прошёл больше windowSize), сбрасываем
	if now.Sub(window.windowStart) > windowSize {
		window.requests = make([]time.Time, 0, requestsPerMinute)
		window.windowStart = now
	}

	// Удаляем старые запросы за пределами окна
	cutoff := now.Add(-windowSize)
	validRequests := make([]time.Time, 0, len(window.requests))
	for _, reqTime := range window.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	window.requests = validRequests

	// Проверяем лимит
	if len(window.requests) >= requestsPerMinute {
		return false
	}

	// Добавляем текущий запрос
	window.requests = append(window.requests, now)

	return true
}

// Reset сбрасывает счётчик для API ключа.
func (rl *SlidingWindowRateLimiter) Reset(apiKeyID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.windows, apiKeyID)
}

// cleanup удаляет устаревшие окна.
func (rl *SlidingWindowRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, window := range rl.windows {
				// Удаляем окна, которые не использовались более 10 минут
				if now.Sub(window.windowStart) > 10*time.Minute {
					delete(rl.windows, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Close закрывает rate limiter.
func (rl *SlidingWindowRateLimiter) Close() {
	close(rl.stopCleanup)
}

// GetRemainingRequests возвращает количество оставшихся запросов.
func (rl *SlidingWindowRateLimiter) GetRemainingRequests(apiKeyID string, requestsPerMinute int) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	window, exists := rl.windows[apiKeyID]
	if !exists {
		return requestsPerMinute
	}

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	validCount := 0
	for _, reqTime := range window.requests {
		if reqTime.After(cutoff) {
			validCount++
		}
	}

	remaining := requestsPerMinute - validCount
	if remaining < 0 {
		return 0
	}

	return remaining
}

// GetResetTime возвращает время сброса лимита.
func (rl *SlidingWindowRateLimiter) GetResetTime(apiKeyID string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	window, exists := rl.windows[apiKeyID]
	if !exists {
		return time.Now().Add(time.Minute)
	}

	// Время сброса = начало окна + 1 минута
	return window.windowStart.Add(time.Minute)
}
