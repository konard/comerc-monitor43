package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Checker проверяет здоровье сервиса.
type Checker struct {
	db     *sqlx.DB
	logger *zerolog.Logger
}

// NewChecker создаёт новый health checker.
func NewChecker(db *sqlx.DB, logger *zerolog.Logger) *Checker {
	return &Checker{
		db:     db,
		logger: logger,
	}
}

// CheckResult результат проверки.
type CheckResult struct {
	Status    string           `json:"status"`
	Timestamp string           `json:"timestamp"`
	Checks    map[string]Check `json:"checks"`
}

// Check результат отдельной проверки.
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Check проверяет здоровье всех компонентов.
func (c *Checker) Check(ctx context.Context) CheckResult {
	checks := make(map[string]Check)

	// Проверка БД
	dbStatus := c.checkDatabase(ctx)
	checks["database"] = dbStatus

	// Определяем общий статус
	overallStatus := "pass"
	for _, check := range checks {
		if check.Status != "pass" {
			overallStatus = "fail"
			break
		}
	}

	return CheckResult{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}
}

// checkDatabase проверяет подключение к БД.
func (c *Checker) checkDatabase(ctx context.Context) Check {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.db.PingContext(checkCtx); err != nil {
		c.logger.Error().Err(err).Msg("database health check failed")
		return Check{
			Status:  "fail",
			Message: fmt.Sprintf("database connection failed: %v", err),
		}
	}

	return Check{
		Status: "pass",
	}
}

// Handler возвращает HTTP handler для health check.
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		result := c.Check(ctx)

		// Устанавливаем статус код
		statusCode := http.StatusOK
		if result.Status != "pass" {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		// Отправляем JSON
		// Для простоты используем fmt, можно использовать json.Marshal
		fmt.Fprintf(w, `{"status":"%s","timestamp":"%s",`, result.Status, result.Timestamp)
		for name, check := range result.Checks {
			fmt.Fprintf(w, `"%s":{%s},`, name, formatCheck(check))
		}
		fmt.Fprint(w, "}")
	}
}

// formatCheck форматирует проверку в JSON.
func formatCheck(c Check) string {
	if c.Message != "" {
		return fmt.Sprintf(`"status":"%s","message":"%s"`, c.Status, c.Message)
	}
	return fmt.Sprintf(`"status":"%s"`, c.Status)
}

// IsReady проверяет готовность сервиса.
func (c *Checker) IsReady(ctx context.Context) bool {
	result := c.Check(ctx)
	return result.Status == "pass"
}
