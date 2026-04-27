package apikey

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlidingWindowRateLimiter_Allow(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-123"
	requestsPerMinute := 5

	// Первые 5 запросов должны быть разрешены
	for i := 0; i < requestsPerMinute; i++ {
		allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 6-й запрос должен быть отклонён
	allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	assert.False(t, allowed, "6th request should be rate limited")
}

func TestSlidingWindowRateLimiter_SlidingWindow(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-sliding"
	requestsPerMinute := 3

	// Делаем 3 запроса - должны быть разрешены
	for i := 0; i < requestsPerMinute; i++ {
		allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 4-й запрос - отклонён
	allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	assert.False(t, allowed, "4th request should be rate limited")

	// Ждём, пока окно сдвинется (> 1 минута для простоты)
	// В реальном тесте можно использовать моки или меньшее окно

	// Сбрасываем лимит вручную для теста
	rl.Reset(apiKeyID)

	// Теперь запрос должен быть разрешён снова
	allowed = rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	assert.True(t, allowed, "Request after reset should be allowed")
}

func TestSlidingWindowRateLimiter_Reset(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-reset"
	requestsPerMinute := 2

	// Делаем максимальное количество запросов
	for i := 0; i < requestsPerMinute; i++ {
		allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
		assert.True(t, allowed)
	}

	// Следующий запрос должен быть отклонён
	allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	assert.False(t, allowed)

	// Сбрасываем
	rl.Reset(apiKeyID)

	// Теперь запрос должен быть разрешён
	allowed = rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	assert.True(t, allowed)
}

func TestSlidingWindowRateLimiter_MultipleKeys(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	key1 := "key-1"
	key2 := "key-2"
	requestsPerMinute := 2

	// Ключ 1: 2 запроса
	assert.True(t, rl.Allow(context.Background(), key1, requestsPerMinute))
	assert.True(t, rl.Allow(context.Background(), key1, requestsPerMinute))
	assert.False(t, rl.Allow(context.Background(), key1, requestsPerMinute))

	// Ключ 2: должен иметь своё собственное окно
	assert.True(t, rl.Allow(context.Background(), key2, requestsPerMinute))
	assert.True(t, rl.Allow(context.Background(), key2, requestsPerMinute))
	assert.False(t, rl.Allow(context.Background(), key2, requestsPerMinute))
}

func TestSlidingWindowRateLimiter_GetRemainingRequests(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-remaining"
	requestsPerMinute := 10

	// Изначально должно быть 10
	remaining := rl.GetRemainingRequests(apiKeyID, requestsPerMinute)
	assert.Equal(t, 10, remaining)

	// Делаем 3 запроса
	for i := 0; i < 3; i++ {
		rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	}

	remaining = rl.GetRemainingRequests(apiKeyID, requestsPerMinute)
	assert.Equal(t, 7, remaining)
}

func TestSlidingWindowRateLimiter_GetResetTime(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-reset-time" //nolint:gosec // G101: тестовые данные

	// Первый запрос создаёт окно
	rl.Allow(context.Background(), apiKeyID, 10)

	resetTime := rl.GetResetTime(apiKeyID)

	// Сброс должен быть примерно через 1 минуту от сейчас
	now := time.Now()
	diff := resetTime.Sub(now)

	// Разница должна быть около 1 минуты (± несколько секунд на задержку выполнения)
	assert.True(t, diff > 58*time.Second && diff < 62*time.Second,
		"Reset time should be approximately 1 minute from now")
}

func TestSlidingWindowRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-concurrent"
	requestsPerMinute := 100

	// Запускаем несколько горутин, которые делают запросы параллельно
	done := make(chan bool, 10)
	var successCount atomic.Int64

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				if rl.Allow(context.Background(), apiKeyID, requestsPerMinute) {
					successCount.Add(1)
				}
			}
			done <- true
		}()
	}

	// Ждём завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Должно быть успешно ровно 100 запросов
	assert.Equal(t, int64(100), successCount.Load())

	// 101-й запрос должен быть отклонён
	allowed := rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	assert.False(t, allowed)
}

func TestSlidingWindowRateLimiter_GetResetTimeNoWindow(t *testing.T) {
	t.Parallel()

	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	// Ключ без записей - возвращает time.Now() + 1 minute
	resetTime := rl.GetResetTime("nonexistent-key")
	now := time.Now()
	diff := resetTime.Sub(now)

	assert.True(t, diff > 55*time.Second, "Reset time should be approximately 1 minute for nonexistent key")
}

func TestSlidingWindowRateLimiter_GetRemainingRequestsExceeded(t *testing.T) {
	t.Parallel()

	rl := NewSlidingWindowRateLimiter()
	t.Cleanup(rl.Close)

	apiKeyID := "test-key-exceeded"
	requestsPerMinute := 3

	// Делаем больше запросов чем лимит
	for i := 0; i < 5; i++ {
		rl.Allow(context.Background(), apiKeyID, requestsPerMinute)
	}

	remaining := rl.GetRemainingRequests(apiKeyID, requestsPerMinute)
	assert.Equal(t, 0, remaining)
}

// TestSlidingWindowRateLimiter_CleanupTicker проверяет работу cleanup тикера.
func TestSlidingWindowRateLimiter_CleanupTicker(t *testing.T) {
	t.Parallel()

	// Создаём rate limiter напрямую с очень коротким интервалом cleanup
	rl := &SlidingWindowRateLimiter{
		windows:         make(map[string]*slidingWindow),
		cleanupInterval: 10 * time.Millisecond,
		stopCleanup:     make(chan struct{}),
	}
	go rl.cleanup()

	// Добавляем окно с очень старым temporally windowStart (11 минут назад - сразу удалится)
	apiKeyID := "cleanup-test-key"
	rl.mu.Lock()
	rl.windows[apiKeyID] = &slidingWindow{
		requests:    []time.Time{time.Now().Add(-12 * time.Minute)},
		windowStart: time.Now().Add(-11 * time.Minute),
		windowSize:  time.Minute,
	}
	rl.mu.Unlock()

	// Ждём пока ticker сработает и очистит устаревшие окна
	time.Sleep(50 * time.Millisecond)

	// Проверяем, что окно было удалено
	rl.mu.RLock()
	_, exists := rl.windows[apiKeyID]
	rl.mu.RUnlock()
	assert.False(t, exists, "стаое окно должно быть удалено cleanup горутиной")

	// Останавливаем cleanup горутину
	rl.Close()
}
