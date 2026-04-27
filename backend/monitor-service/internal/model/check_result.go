package domain

import (
	"time"

	"github.com/google/uuid"
)

// CheckResult представляет результат единичной проверки монитора.
type CheckResult struct {
	// ID уникальный идентификатор результата
	ID uuid.UUID
	// MonitorID идентификатор монитора
	MonitorID uuid.UUID
	// Status статус проверки (UP, DOWN, DEGRADED, PAUSED)
	Status MonitorStatus
	// ResponseTimeMs время ответа в миллисекундах
	ResponseTimeMs *int
	// StatusCode HTTP статус код (опционально)
	StatusCode *int
	// ErrorMessage сообщение об ошибке (если есть)
	ErrorMessage *string
	// ErrorCode код ошибки (если есть)
	ErrorCode *CheckErrorCode
	// CheckedAt время выполнения проверки
	CheckedAt time.Time
	// CreatedAt время записи результата в БД
	CreatedAt time.Time
}

// NewCheckResult создаёт новый результат проверки.
func NewCheckResult(monitorID uuid.UUID, status MonitorStatus) *CheckResult {
	now := time.Now()
	return &CheckResult{
		ID:        uuid.New(),
		MonitorID: monitorID,
		Status:    status,
		CheckedAt: now,
		CreatedAt: now,
	}
}

// WithResponseTime устанавливает время ответа.
func (r *CheckResult) WithResponseTime(ms int) *CheckResult {
	r.ResponseTimeMs = &ms
	return r
}

// WithStatusCode устанавливает HTTP статус код.
func (r *CheckResult) WithStatusCode(code int) *CheckResult {
	r.StatusCode = &code
	return r
}

// WithError устанавливает сообщение об ошибке.
func (r *CheckResult) WithError(err error) *CheckResult {
	if err != nil {
		msg := err.Error()
		r.ErrorMessage = &msg
	}
	return r
}

// WithErrorCode устанавливает код ошибки.
func (r *CheckResult) WithErrorCode(code CheckErrorCode) *CheckResult {
	r.ErrorCode = &code
	return r
}

// IsSuccess возвращает true, если проверка прошла успешно (UP).
func (r *CheckResult) IsSuccess() bool {
	return r.Status == StatusUp
}

// IsFailure возвращает true, если проверка не удалась (DOWN).
func (r *CheckResult) IsFailure() bool {
	return r.Status == StatusDown
}

// IsDegraded возвращает true, если проверка показала ухудшение.
func (r *CheckResult) IsDegraded() bool {
	return r.Status == StatusDegraded
}

// GetResponseTime возвращает время ответа или 0, если не задано.
func (r *CheckResult) GetResponseTime() int {
	if r.ResponseTimeMs == nil {
		return 0
	}
	return *r.ResponseTimeMs
}

// GetStatusCode возвращает статус код или 0, если не задан.
func (r *CheckResult) GetStatusCode() int {
	if r.StatusCode == nil {
		return 0
	}
	return *r.StatusCode
}

// GetErrorMessage возвращает сообщение об ошибке или пустую строку.
func (r *CheckResult) GetErrorMessage() string {
	if r.ErrorMessage == nil {
		return ""
	}
	return *r.ErrorMessage
}

// CheckErrorCode представляет код ошибки при выполнении проверки.
type CheckErrorCode string

const (
	ErrorCodeConnectionTimeout       CheckErrorCode = "CONNECTION_TIMEOUT"
	ErrorCodeDNSResolutionFailed     CheckErrorCode = "DNS_RESOLUTION_FAILED"
	ErrorCodeSSLCertificateExpired   CheckErrorCode = "SSL_CERTIFICATE_EXPIRED"
	ErrorCodeEmptyResponse           CheckErrorCode = "EMPTY_RESPONSE"
	ErrorCodeResponseTooLarge        CheckErrorCode = "RESPONSE_TOO_LARGE"
	ErrorCodeTooManyRedirects        CheckErrorCode = "TOO_MANY_REDIRECTS"
	ErrorCodeDegradedSlowResponse    CheckErrorCode = "DEGRADED_SLOW_RESPONSE"
	ErrorCodeDegradedHighFailure     CheckErrorCode = "DEGRADED_HIGH_FAILURE_RATE"
	ErrorCodeCheckSkippedMaintenance CheckErrorCode = "CHECK_SKIPPED_MAINTENANCE"
	ErrorCodeServerError             CheckErrorCode = "SERVER_ERROR"
	ErrorCodeClientError             CheckErrorCode = "CLIENT_ERROR"
	ErrorCodeNetworkError            CheckErrorCode = "NETWORK_ERROR"
)

// CheckResultSlice represents a slice of CheckResult for helper methods.
type CheckResultSlice []CheckResult

// CalculateFailureRate вычисляет процент failed проверок (DOWN).
// Формула: (DOWN / Total) * 100
func (s CheckResultSlice) CalculateFailureRate() float64 {
	if len(s) == 0 {
		return 0.0
	}

	failed := 0
	for _, r := range s {
		if r.Status == StatusDown {
			failed++
		}
	}

	return (float64(failed) / float64(len(s))) * 100
}

// CalculateAverageResponseTime вычисляет среднее время ответа.
// Игнорирует результаты без времени ответа.
func (s CheckResultSlice) CalculateAverageResponseTime() int {
	if len(s) == 0 {
		return 0
	}

	total := 0
	count := 0
	for _, r := range s {
		if r.ResponseTimeMs != nil {
			total += *r.ResponseTimeMs
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / count
}

// FilterByStatus фильтрует результаты по статусу.
func (s CheckResultSlice) FilterByStatus(status MonitorStatus) CheckResultSlice {
	filtered := make(CheckResultSlice, 0)
	for _, r := range s {
		if r.Status == status {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterActive возвращает только активные результаты (исключая PAUSED).
func (s CheckResultSlice) FilterActive() CheckResultSlice {
	filtered := make(CheckResultSlice, 0)
	for _, r := range s {
		if r.Status != StatusPaused {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// CountByStatus возвращает количество результатов для каждого статуса.
func (s CheckResultSlice) CountByStatus() map[MonitorStatus]int {
	counts := make(map[MonitorStatus]int)
	for _, r := range s {
		counts[r.Status]++
	}
	return counts
}
