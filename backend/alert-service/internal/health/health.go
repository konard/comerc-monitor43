package health

import (
	"context"
	"database/sql"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthChecker проверяет здоровье сервиса
type HealthChecker struct {
	db Pinger
}

// NewHealthChecker создаёт новый health checker из *sql.DB
func NewHealthChecker(db *sql.DB) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

// NewHealthCheckerFromInterface создаёт новый health checker из Pinger интерфейса
func NewHealthCheckerFromInterface(db Pinger) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

// NewHealthCheckerFromPostgres создаёт новый health checker из postgres.DB
func NewHealthCheckerFromPostgres(db Pinger) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

// Check проверяет здоровье сервиса
func (h *HealthChecker) Check(ctx context.Context) error {
	if h.db == nil {
		return status.Error(codes.Unavailable, "database not configured")
	}

	if err := h.db.PingContext(ctx); err != nil {
		return status.Error(codes.Unavailable, "database unhealthy")
	}

	return nil
}
