// Package ratelimit предоставляет rate limiting middleware для gRPC сервера.
//
// Пакет содержит:
//   - RateLimiter: limiter на основе token bucket algorithm
//   - UnaryInterceptor: gRPC interceptor для применения rate limiting
//
// Использование:
//
//	limiter := ratelimit.NewRateLimiter(60) // 60 запросов в минуту
//	interceptor := limiter.UnaryInterceptor()
//	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor))
package ratelimit
