package ratelimit

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter предоставляет rate limiting на основе token bucket algorithm.
type RateLimiter struct {
	mu                  sync.RWMutex
	limiters            map[string]*rate.Limiter
	requests_per_minute int
}

// NewRateLimiter создаёт новый RateLimiter.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		limiters:            make(map[string]*rate.Limiter),
		requests_per_minute: requestsPerMinute,
	}
}

// getLimiter возвращает limiter для конкретного клиента (IP адрес).
func (rl *RateLimiter) getLimiter(clientIP string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[clientIP]
	if !exists {
		// Создаём новый limiter: requestsPerMinute запросов в минуту
		// rate.Every создаёт duration между запросами
		interval := time.Duration(float64(time.Minute) / float64(rl.requests_per_minute))
		limiter = rate.NewLimiter(rate.Every(interval), 1)
		rl.limiters[clientIP] = limiter
	}

	return limiter
}

// getClientIP извлекает IP адрес клиента из контекста.
func (rl *RateLimiter) getClientIP(ctx context.Context) string {
	// Пытаемся получить IP из peer info
	if p, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			return tcpAddr.IP.String()
		}
		// Если не TCP, возвращаем строковое представление адреса
		return p.Addr.String()
	}

	// Если не удалось получить IP, используем default key
	return "unknown"
}

// UnaryInterceptor создаёт gRPC unary interceptor с rate limiting.
func (rl *RateLimiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Получаем IP клиента
		clientIP := rl.getClientIP(ctx)

		// Получаем limiter для этого клиента
		limiter := rl.getLimiter(clientIP)

		// Проверяем, разрешён ли запрос
		if !limiter.Allow() {
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded for client %s: maximum %d requests per minute allowed",
				clientIP, rl.requests_per_minute)
		}

		// Если лимит не превышен, вызываем handler
		return handler(ctx, req)
	}
}

// CleanupLimiters удаляет limiters для неактивных клиентов.
// Этот метод следует вызывать периодически для предотвращения утечки памяти.
func (rl *RateLimiter) CleanupLimiters() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// В текущей реализации мы просто очищаем все limiters
	// В продакшене можно добавить логику отслеживания активности
	rl.limiters = make(map[string]*rate.Limiter)
}

// GetActiveLimitersCount возвращает количество активных limiters.
func (rl *RateLimiter) GetActiveLimitersCount() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return len(rl.limiters)
}

// IsAllowed проверяет, разрешён ли запрос для клиента с указанным IP.
func (rl *RateLimiter) IsAllowed(ctx context.Context) bool {
	clientIP := rl.getClientIP(ctx)
	limiter := rl.getLimiter(clientIP)
	return limiter.Allow()
}

// WaitForRequest ожидает, пока будет доступен слот для запроса.
// Возвращает ошибку, если контекст отменён до того, как стал доступен слот.
func (rl *RateLimiter) WaitForRequest(ctx context.Context) error {
	clientIP := rl.getClientIP(ctx)
	limiter := rl.getLimiter(clientIP)

	err := limiter.Wait(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to wait for rate limit slot")
	}

	return nil
}
