package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Checker provides health check functionality
type Checker struct {
	db          *sql.DB
	redisPinger RedisPinger
}

// NewChecker creates a new health checker
func NewChecker(db *sql.DB, redisPinger RedisPinger) *Checker {
	return &Checker{db: db, redisPinger: redisPinger}
}

// RedisPinger interface for Redis health checks
type RedisPinger interface {
	Ping(ctx context.Context) error
}

// CheckResult represents health check result
type CheckResult struct {
	Status   string          `json:"status"`
	Database *DatabaseStatus `json:"database,omitempty"`
	Redis    *RedisStatus    `json:"redis,omitempty"`
}

// DatabaseStatus represents database health
type DatabaseStatus struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Latency int64  `json:"latency_ms"`
}

// RedisStatus represents redis health
type RedisStatus struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Latency int64  `json:"latency_ms"`
}

const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
)

// Check performs all health checks
func (h *Checker) Check(ctx context.Context) *CheckResult {
	result := &CheckResult{Status: StatusHealthy}

	// Check database
	dbStatus := h.checkDatabase(ctx)
	if dbStatus.Status == StatusUnhealthy {
		result.Status = StatusUnhealthy
	}
	result.Database = dbStatus

	// Check redis
	redisStatus := h.checkRedis(ctx)
	if redisStatus.Status == StatusUnhealthy {
		result.Status = StatusUnhealthy
	}
	result.Redis = redisStatus

	return result
}

// checkDatabase verifies database connectivity
func (h *Checker) checkDatabase(ctx context.Context) *DatabaseStatus {
	start := time.Now()

	// Simple ping to database
	err := h.db.PingContext(ctx)
	if err != nil {
		return &DatabaseStatus{
			Status: StatusUnhealthy,
			Error:  err.Error(),
		}
	}

	// Verify we can query
	var one int
	err = h.db.QueryRowContext(ctx, "SELECT 1").Scan(&one)
	if err != nil {
		return &DatabaseStatus{
			Status: StatusUnhealthy,
			Error:  err.Error(),
		}
	}

	return &DatabaseStatus{
		Status:  StatusHealthy,
		Latency: time.Since(start).Milliseconds(),
	}
}

// checkRedis verifies redis connectivity
func (h *Checker) checkRedis(ctx context.Context) *RedisStatus {
	start := time.Now()

	// Try to ping redis
	err := h.redisPinger.Ping(ctx)
	if err != nil {
		return &RedisStatus{
			Status: StatusUnhealthy,
			Error:  err.Error(),
		}
	}

	return &RedisStatus{
		Status:  StatusHealthy,
		Latency: time.Since(start).Milliseconds(),
	}
}

// HealthHandler возвращает HTTP-обработчик для endpoint /health.
func (h *Checker) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := h.Check(ctx)
		w.Header().Set("Content-Type", "application/json")
		if result.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.WarnContext(ctx, "Failed to encode health response", slog.Any("err", err))
		}
	}
}

// ReadinessHandler возвращает HTTP-обработчик для endpoint /ready.
func (h *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := h.Check(ctx)
		w.Header().Set("Content-Type", "application/json")
		dbOK := result.Database != nil && result.Database.Status == StatusHealthy
		redisOK := result.Redis != nil && result.Redis.Status == StatusHealthy
		if result.Status == StatusHealthy && dbOK && redisOK {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.WarnContext(ctx, "Failed to encode readiness response", slog.Any("err", err))
		}
	}
}
