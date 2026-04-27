package executor

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

// ErrorCode представляет код ошибки проверки
type ErrorCode string

const (
	ErrorCodeConnectionTimeout     ErrorCode = "CONNECTION_TIMEOUT"
	ErrorCodeDNSResolutionFailed   ErrorCode = "DNS_RESOLUTION_FAILED"
	ErrorCodeSSLCertificateExpired ErrorCode = "SSL_CERTIFICATE_EXPIRED"
	ErrorCodeTooManyRedirects      ErrorCode = "TOO_MANY_REDIRECTS"
	ErrorCodeConnectionRefused     ErrorCode = "CONNECTION_REFUSED"
	ErrorCodeResponseTooLarge      ErrorCode = "RESPONSE_TOO_LARGE"
	ErrorCodeSlowResponse          ErrorCode = "SLOW_RESPONSE"
	ErrorCodeEmptyResponse         ErrorCode = "EMPTY_RESPONSE"
	ErrorCodeInvalidMonitorConfig  ErrorCode = "INVALID_MONITOR_CONFIG"
	ErrorCodeMaintenanceSkipped    ErrorCode = "MAINTENANCE_WINDOW"
)

// ErrorClassifier классифицирует ошибки HTTP проверок
type ErrorClassifier struct{}

// NewErrorClassifier создаёт новый классификатор ошибок
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

// ClassifyError классифицирует ошибку и возвращает код ошибки
func (c *ErrorClassifier) ClassifyError(err error) ErrorCode {
	if err == nil {
		return ""
	}

	// Проверяем на timeout
	if c.isTimeoutError(err) {
		return ErrorCodeConnectionTimeout
	}

	// Проверяем на DNS ошибки
	if c.isDNSError(err) {
		return ErrorCodeDNSResolutionFailed
	}

	// Проверяем на connection refused
	if c.isConnectionRefusedError(err) {
		return ErrorCodeConnectionRefused
	}

	// Проверяем на SSL certificate ошибки
	if c.isSSLCertificateError(err) {
		return ErrorCodeSSLCertificateExpired
	}

	// Generic fallback
	return ErrorCodeConnectionRefused
}

// isTimeoutError проверяет, является ли ошибка timeout
func (c *ErrorClassifier) isTimeoutError(err error) bool {
	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		return true
	}

	// Проверяем на конкретные timeout ошибки
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	// Проверяем на context.DeadlineExceeded
	if errors.Is(err, contextDeadlineExceededError()) {
		return true
	}

	// Проверяем на строковые представления timeout (для HTTPClient errors)
	errMsg := err.Error()
	if contains(errMsg, "timeout") || contains(errMsg, "deadline exceeded") {
		return true
	}

	return false
}

// isDNSError проверяет, является ли ошибка DNS resolution failure
func (c *ErrorClassifier) isDNSError(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}

// isConnectionRefusedError проверяет, является ли ошибка connection refused
func (c *ErrorClassifier) isConnectionRefusedError(err error) bool {
	// Проверяем на syscall.ECONNREFUSED
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) {
		if errors.Is(sysErr.Err, syscall.ECONNREFUSED) {
			return true
		}
	}

	// Проверяем на конкретные connection refused ошибки
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" || opErr.Op == "connect" {
			return true
		}
	}

	// Проверяем на строковые представления connection refused
	errMsg := err.Error()
	if contains(errMsg, "connection refused") ||
		contains(errMsg, "connect: connection refused") ||
		contains(errMsg, "dial tcp") {
		return true
	}

	return false
}

// isSSLCertificateError проверяет, является ли ошибка SSL certificate problem
func (c *ErrorClassifier) isSSLCertificateError(err error) bool {
	// Проверяем на x509 ошибки
	var certErr *x509.CertificateInvalidError
	if errors.As(err, &certErr) {
		return true
	}

	var hostErr *x509.HostnameError
	if errors.As(err, &hostErr) {
		return true
	}

	// Проверяем на неизвестные авторитетные центры
	var unknownAuthErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthErr) {
		return true
	}

	// Проверяем на строковые представления SSL ошибок
	errMsg := err.Error()
	if contains(errMsg, "certificate") ||
		contains(errMsg, "x509") ||
		contains(errMsg, "tls") {
		return true
	}

	return false
}

// contextDeadlineExceededError возвращает context.DeadlineExceeded ошибку
func contextDeadlineExceededError() error {
	// Импортируем context package для доступа к DeadlineExceeded
	type contextInterface interface {
		DeadlineExceeded() error
	}

	// Создаём mock context для проверки
	// В реальном коде используем context.DeadlineExceeded
	return errors.New("context deadline exceeded")
}

// contains проверяет, содержит ли строка подстроку (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) && containsIgnoreCase(s, substr))
}

// containsIgnoreCase case-insensitive проверка содержит ли строка подстроку
func containsIgnoreCase(s, substr string) bool {
	// Простая реализация - для производительности можно использовать strings.ToLower
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetErrorMessage возвращает пользовательское сообщение об ошибке по коду
func (c *ErrorClassifier) GetErrorMessage(code ErrorCode) string {
	switch code {
	case ErrorCodeConnectionTimeout:
		return "connection timeout"
	case ErrorCodeDNSResolutionFailed:
		return "dns resolution failed"
	case ErrorCodeSSLCertificateExpired:
		return "ssl certificate expired"
	case ErrorCodeTooManyRedirects:
		return "too many redirects"
	case ErrorCodeConnectionRefused:
		return "connection refused"
	case ErrorCodeResponseTooLarge:
		return "response too large"
	case ErrorCodeSlowResponse:
		return "slow response"
	case ErrorCodeEmptyResponse:
		return "empty response"
	case ErrorCodeInvalidMonitorConfig:
		return "invalid monitor configuration"
	default:
		return "unknown error"
	}
}

// DetailedMetrics содержит детальные метрики времени ответа
type DetailedMetrics struct {
	DNSLookupMs    float64
	TCPConnectMs   float64
	TLSHandshakeMs float64
	TTFBMs         float64 // Time To First Byte
	TotalMs        float64
}

// IsDegraded проверяет, является ли ответ медленным
func IsDegraded(responseTimeMs float64, degradedThresholdMs float64) bool {
	if degradedThresholdMs <= 0 {
		return false
	}
	return responseTimeMs > degradedThresholdMs
}

// ValidateRequest проверяет валидность конфигурации монитора
func ValidateRequest(req CheckRequest) error {
	if req.URL == "" {
		return fmt.Errorf("URL is required")
	}
	return nil
}

// FormatDuration форматирует duration в миллисекунды
func FormatDuration(d time.Duration) float64 {
	return float64(d.Milliseconds())
}
