package health

import (
	"context"
	"database/sql"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Pinger интерфейс для проверки соединения
type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthChecker проверяет здоровье сервиса
type HealthChecker struct {
	db Pinger
}

// NewHealthChecker создаёт новый health checker
func NewHealthChecker(db Pinger) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

// Check проверяет здоровье всех компонентов
func (h *HealthChecker) Check(ctx context.Context) error {
	// Проверяем базу данных
	if h.db != nil {
		if err := h.db.PingContext(ctx); err != nil {
			return status.Error(codes.Unavailable, "database unhealthy")
		}
	}

	// Здесь можно добавить проверки других компонентов:
	// - Внешние сервисы (billing, monitor)
	// - Очереди (RabbitMQ)
	// - Кэши

	return nil
}

// NewHealthCheckerFromSQLDB создаёт health checker из *sql.DB
func NewHealthCheckerFromSQLDB(db *sql.DB) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

// NewHealthCheckerFromSQLxDB создаёт health checker из *sqlx.DB
func NewHealthCheckerFromSQLxDB(db interface {
	PingContext(ctx context.Context) error
}) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}
