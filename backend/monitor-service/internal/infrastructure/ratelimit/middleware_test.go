package ratelimit

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// TestNewRateLimiter тестирует создание RateLimiter.
func TestNewRateLimiter(t *testing.T) {
	t.Parallel()
	t.Run("creates rate limiter with valid config", func(t *testing.T) {
		limiter := NewRateLimiter(60)

		assert.NotNil(t, limiter)
		assert.NotNil(t, limiter.limiters)
		assert.Equal(t, 60, limiter.requests_per_minute)
		assert.Equal(t, 0, limiter.GetActiveLimitersCount())
	})

	t.Run("creates rate limiter with different rates", func(t *testing.T) {
		testCases := []struct {
			name     string
			rate     int
			expected int
		}{
			{"1 request per minute", 1, 1},
			{"60 requests per minute", 60, 60},
			{"600 requests per minute", 600, 600},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				limiter := NewRateLimiter(tc.rate)
				assert.Equal(t, tc.expected, limiter.requests_per_minute)
			})
		}
	})
}

// TestGetClientIP тестирует извлечение IP адреса клиента из контекста.
func TestGetClientIP(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(60)

	t.Run("extracts IP from context with peer info", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{
				IP:   net.ParseIP("192.168.1.1"),
				Port: 12345,
			},
		})

		clientIP := limiter.getClientIP(ctx)
		assert.Equal(t, "192.168.1.1", clientIP)
	})

	t.Run("returns unknown when no peer info", func(t *testing.T) {
		ctx := context.Background()
		clientIP := limiter.getClientIP(ctx)
		assert.Equal(t, "unknown", clientIP)
	})

	t.Run("returns string representation for non-TCP addresses", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.UDPAddr{
				IP:   net.ParseIP("10.0.0.1"),
				Port: 54321,
			},
		})

		clientIP := limiter.getClientIP(ctx)
		assert.Contains(t, clientIP, "10.0.0.1")
	})
}

// TestGetLimiter тестирует получение limiter для клиента.
func TestGetLimiter(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(60)

	t.Run("creates new limiter for unknown client", func(t *testing.T) {
		clientIP := "192.168.1.1"
		l := limiter.getLimiter(clientIP)

		assert.NotNil(t, l)
		assert.Equal(t, 1, limiter.GetActiveLimitersCount())
	})

	t.Run("returns existing limiter for known client", func(t *testing.T) {
		limiter := NewRateLimiter(60) // Create fresh limiter for this test
		clientIP := "192.168.1.2"
		l1 := limiter.getLimiter(clientIP)
		l2 := limiter.getLimiter(clientIP)

		assert.Same(t, l1, l2)
		assert.Equal(t, 1, limiter.GetActiveLimitersCount())
	})

	t.Run("creates different limiters for different clients", func(t *testing.T) {
		limiter := NewRateLimiter(60)

		l1 := limiter.getLimiter("192.168.1.1")
		l2 := limiter.getLimiter("192.168.1.2")

		assert.NotSame(t, l1, l2)
		assert.Equal(t, 2, limiter.GetActiveLimitersCount())
	})
}

// TestUnaryInterceptor тестирует gRPC interceptor.
func TestUnaryInterceptor(t *testing.T) {
	t.Parallel()
	t.Run("allows requests within rate limit", func(t *testing.T) {
		limiter := NewRateLimiter(60) // 60 запросов в минуту
		interceptor := limiter.UnaryInterceptor()

		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{
				IP:   net.ParseIP("192.168.1.1"),
				Port: 12345,
			},
		})

		handlerCalled := false
		handler := func(ctx context.Context, req any) (any, error) {
			handlerCalled = true
			return "response", nil
		}

		resp, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{}, handler)

		assert.Nil(t, err)
		assert.Equal(t, "response", resp)
		assert.True(t, handlerCalled)
	})

	t.Run("blocks requests exceeding rate limit", func(t *testing.T) {
		// Создаём limiter с низкой скоростью для тестирования
		limiter := NewRateLimiter(1) // 1 запрос в минуту
		interceptor := limiter.UnaryInterceptor()

		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{
				IP:   net.ParseIP("192.168.1.2"),
				Port: 12345,
			},
		})

		handler := func(ctx context.Context, req any) (any, error) {
			return "response", nil
		}

		// Первый запрос должен пройти
		_, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{}, handler)
		assert.Nil(t, err)

		// Второй запрос должен быть заблокирован
		_, err = interceptor(ctx, "request", &grpc.UnaryServerInfo{}, handler)
		assert.NotNil(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.ResourceExhausted, st.Code())
		assert.Contains(t, st.Message(), "rate limit exceeded")
	})

	t.Run("handles multiple clients independently", func(t *testing.T) {
		limiter := NewRateLimiter(30) // 30 запросов в минуту
		interceptor := limiter.UnaryInterceptor()

		handler := func(ctx context.Context, req any) (any, error) {
			return "response", nil
		}

		// Создаём контексты для разных клиентов
		ctx1 := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345},
		})
		ctx2 := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.2"), Port: 12345},
		})

		// Оба клиента должны получить свои лимиты
		_, err1 := interceptor(ctx1, "request", &grpc.UnaryServerInfo{}, handler)
		_, err2 := interceptor(ctx2, "request", &grpc.UnaryServerInfo{}, handler)

		assert.Nil(t, err1)
		assert.Nil(t, err2)
		assert.Equal(t, 2, limiter.GetActiveLimitersCount())
	})
}

// TestIsAllowed тестирует проверку разрешения запроса.
func TestIsAllowed(t *testing.T) {
	t.Parallel()
	t.Run("allows request when under limit", func(t *testing.T) {
		limiter := NewRateLimiter(60)
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345},
		})

		allowed := limiter.IsAllowed(ctx)
		assert.True(t, allowed)
	})

	t.Run("denies request when over limit", func(t *testing.T) {
		limiter := NewRateLimiter(1) // 1 запрос в минуту
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.2"), Port: 12345},
		})

		// Первый запрос
		allowed := limiter.IsAllowed(ctx)
		assert.True(t, allowed)

		// Второй запрос должен быть запрещён
		allowed = limiter.IsAllowed(ctx)
		assert.False(t, allowed)
	})
}

// TestWaitForRequest тестирует ожидание доступного слота.
func TestWaitForRequest(t *testing.T) {
	t.Parallel()
	t.Run("waits for available slot", func(t *testing.T) {
		limiter := NewRateLimiter(60)
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345},
		})

		err := limiter.WaitForRequest(ctx)
		assert.Nil(t, err)
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		limiter := NewRateLimiter(1) // 1 запрос в минуту
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.2"), Port: 12345},
		})

		ctx, cancel := context.WithCancel(ctx)

		// Первый запрос consumes token
		err := limiter.WaitForRequest(ctx)
		assert.Nil(t, err)

		// Отменяем контекст
		cancel()

		// Второй запрос должен вернуть ошибку
		err = limiter.WaitForRequest(ctx)
		assert.NotNil(t, err)
	})

	t.Run("returns error when context times out", func(t *testing.T) {
		limiter := NewRateLimiter(1) // 1 запрос в минуту
		baseCtx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.3"), Port: 12345},
		})

		ctx, cancel := context.WithTimeout(baseCtx, 100*time.Millisecond)
		defer cancel()

		// Первый запрос consumes token
		err := limiter.WaitForRequest(ctx)
		assert.Nil(t, err)

		// Второй запрос должен timeout
		err = limiter.WaitForRequest(ctx)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "would exceed context deadline")
	})
}

// TestCleanupLimiters тестирует очистку limiters.
func TestCleanupLimiters(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(60)

	// Создаём несколько limiters
	limiter.getLimiter("192.168.1.1")
	limiter.getLimiter("192.168.1.2")
	limiter.getLimiter("192.168.1.3")

	assert.Equal(t, 3, limiter.GetActiveLimitersCount())

	// Очищаем
	limiter.CleanupLimiters()

	assert.Equal(t, 0, limiter.GetActiveLimitersCount())
}

// TestGetActiveLimitersCount тестирует подсчёт активных limiters.
func TestGetActiveLimitersCount(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(60)

	assert.Equal(t, 0, limiter.GetActiveLimitersCount())

	limiter.getLimiter("192.168.1.1")
	assert.Equal(t, 1, limiter.GetActiveLimitersCount())

	limiter.getLimiter("192.168.1.2")
	assert.Equal(t, 2, limiter.GetActiveLimitersCount())

	// Повторный вызов для того же клиента
	limiter.getLimiter("192.168.1.1")
	assert.Equal(t, 2, limiter.GetActiveLimitersCount())
}

// TestConcurrentAccess тестирует конкурентный доступ к limiter.
func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(10000) // Очень высокий лимит для конкурентного теста
	interceptor := limiter.UnaryInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	// Запускаем 100 горутин
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	var successes int64 = 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Создаём уникальный контекст для каждой горутины
			ctx := peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP(fmt.Sprintf("192.168.1.%d", idx)), Port: 12345},
			})
			_, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{}, handler)
			if err != nil {
				errors <- err
			} else {
				atomic.AddInt64(&successes, 1)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Все запросы должны успешно выполниться (лимит высокий)
	errorCount := 0
	for range errors {
		errorCount++
	}

	assert.Equal(t, int64(100), atomic.LoadInt64(&successes))
	assert.Equal(t, 0, errorCount)
}

// TestRateLimitingAccuracy тестирует точность rate limiting.
func TestRateLimitingAccuracy(t *testing.T) {
	t.Parallel()
	// Этот тест проверяет, что rate limiting работает примерно с ожидаемой точностью
	limiter := NewRateLimiter(60) // 60 запросов в минуту
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345},
	})

	// Делаем несколько запросов - должны пройти
	successCount := 0
	for i := 0; i < 5; i++ {
		if limiter.IsAllowed(ctx) {
			successCount++
		}
	}

	assert.Equal(t, 1, successCount) // Только первый проходит сразу (burst=1)

	// Следующий запрос должен быть заблокирован
	allowed := limiter.IsAllowed(ctx)
	assert.False(t, allowed)
}

// TestMultipleClientsRateLimiting тестирует rate limiting для нескольких клиентов.
func TestMultipleClientsRateLimiting(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(60) // 60 запросов в минуту на клиента

	clients := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	successes := make(map[string]int)

	for _, client := range clients {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP(client), Port: 12345},
		})

		// Каждый клиент делает 2 запроса
		for i := 0; i < 2; i++ {
			if limiter.IsAllowed(ctx) {
				successes[client]++
			}
		}
	}

	// Каждый клиент должен иметь 1 успешный запрос (burst=1)
	for _, client := range clients {
		assert.Equal(t, 1, successes[client], "Client %s should have 1 successful request", client)
	}

	// Следующий запрос для любого клиента должен быть запрещён
	for _, client := range clients {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			Addr: &net.TCPAddr{IP: net.ParseIP(client), Port: 12345},
		})

		allowed := limiter.IsAllowed(ctx)
		assert.False(t, allowed, "Client %s should be rate limited", client)
	}
}
