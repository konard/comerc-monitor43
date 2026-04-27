package domain

import "fmt"

// Ошибки домена монитора
var (
	ErrEmptyMonitorName       = fmt.Errorf("monitor name cannot be empty")
	ErrMonitorNameTooLong     = fmt.Errorf("monitor name too long (max 255 characters)")
	ErrEmptyURL               = fmt.Errorf("URL cannot be empty")
	ErrURLTooLong             = fmt.Errorf("URL too long (max 2048 characters)")
	ErrInvalidURLScheme       = fmt.Errorf("URL must start with http:// or https://")
	ErrIntervalTooSmall       = fmt.Errorf("interval must be at least 30 seconds")
	ErrIntervalTooLarge       = fmt.Errorf("interval must be at most 3600 seconds (1 hour)")
	ErrTimeoutExceedsInterval = fmt.Errorf("timeout must be less than check interval")
)

// InvalidStatusTransitionError ошибка недопустимого перехода статуса.
type InvalidStatusTransitionError struct {
	From MonitorStatus
	To   MonitorStatus
}

func (e *InvalidStatusTransitionError) Error() string {
	return fmt.Sprintf("invalid status transition: %s -> %s", e.From, e.To)
}

// Is возвращает true для ошибок типа InvalidStatusTransitionError.
func (e *InvalidStatusTransitionError) Is(target error) bool {
	_, ok := target.(*InvalidStatusTransitionError)
	return ok
}
