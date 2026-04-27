package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

// nopLogger возвращает logger, отбрасывающий все записи.
func nopLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

// TestNewChecker проверяет создание checker.
func TestNewChecker(t *testing.T) {
	t.Parallel()

	// Arrange
	db := &sqlx.DB{}
	logger := nopLogger()

	// Act
	c := NewChecker(db, logger)

	// Assert
	if c == nil {
		t.Fatal("NewChecker() returned nil")
	}
}

// TestCheck_DatabaseFail проверяет что при недоступной БД статус fail.
func TestCheck_DatabaseFail(t *testing.T) {
	t.Parallel()

	// Arrange — открываем БД с невалидным адресом
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable connect_timeout=1")
	if err != nil {
		t.Skipf("cannot open fake DB: %v", err)
	}
	closeTestDB(t, db)

	c := NewChecker(db, nopLogger())

	// Act
	result := c.Check(context.Background())

	// Assert
	if result.Status == "" {
		t.Error("Check() Status should not be empty")
	}
	if result.Timestamp == "" {
		t.Error("Check() Timestamp should not be empty")
	}
	if result.Status != "fail" {
		t.Errorf("Check() Status = %q, want %q when DB unavailable", result.Status, "fail")
	}
	dbCheck, ok := result.Checks["database"]
	if !ok {
		t.Fatal("Check() missing 'database' key in checks")
	}
	if dbCheck.Status != "fail" {
		t.Errorf("Check() database status = %q, want %q", dbCheck.Status, "fail")
	}
}

// TestIsReady проверяет IsReady при недоступной БД.
func TestIsReady(t *testing.T) {
	t.Parallel()

	// Arrange
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable connect_timeout=1")
	if err != nil {
		t.Skipf("cannot open fake DB: %v", err)
	}
	closeTestDB(t, db)

	c := NewChecker(db, nopLogger())

	// Act
	ready := c.IsReady(context.Background())

	// Assert — БД недоступна, поэтому ready должен быть false
	if ready {
		t.Error("IsReady() should return false when DB is unavailable")
	}
}

// TestHandler_Returns503WhenUnhealthy проверяет HTTP handler при нездоровом сервисе.
func TestHandler_Returns503WhenUnhealthy(t *testing.T) {
	t.Parallel()

	// Arrange
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable connect_timeout=1")
	if err != nil {
		t.Skipf("cannot open fake DB: %v", err)
	}
	closeTestDB(t, db)

	c := NewChecker(db, nopLogger())
	handler := c.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	// Act
	handler(rr, req)

	// Assert
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Handler() status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Handler() Content-Type = %q, want application/json", rr.Header().Get("Content-Type"))
	}
}

// TestFormatCheck проверяет форматирование результата проверки.
func TestFormatCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		check    Check
		contains string
	}{
		{
			name:     "without message",
			check:    Check{Status: "pass"},
			contains: `"status":"pass"`,
		},
		{
			name:     "with message",
			check:    Check{Status: "fail", Message: "DB down"},
			contains: "DB down",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := formatCheck(tc.check)

			// Assert
			if !strings.Contains(result, tc.contains) {
				t.Errorf("formatCheck() = %q, want contains %q", result, tc.contains)
			}
		})
	}
}
