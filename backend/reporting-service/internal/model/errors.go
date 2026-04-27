package model

import "fmt"

// ReportError представляет ошибку с кодом и сообщением для отчётности.
type ReportError struct {
	Code    string
	Message string
}

// Error возвращает строковое представление ошибки.
func (e *ReportError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewReportError создаёт новую ReportError с указанными кодом и сообщением.
func NewReportError(code, message string) *ReportError {
	return &ReportError{Code: code, Message: message}
}

var (
	ErrInvalidDateRange       = NewReportError("INVALID_DATE_RANGE", "invalid date range")
	ErrPeriodExceedsRetention = NewReportError("PERIOD_EXCEEDS_RETENTION", "period exceeds retention policy")
	ErrReportNotFound         = NewReportError("REPORT_NOT_FOUND", "report not found")
	ErrUnauthorizedAccess     = NewReportError("UNAUTHORIZED_ACCESS", "no access to this report")
	ErrMonitorNotFound        = NewReportError("MONITOR_NOT_FOUND", "monitor not found")
	ErrFuturePeriod           = NewReportError("INVALID_DATE_RANGE", "cannot generate report for future period")
)
