package model

import (
	"fmt"
)

// DomainError представляет ошибку доменной модели
type DomainError struct {
	code    string
	message string
}

// NewDomainError создаёт новую доменную ошибку
func NewDomainError(message string) *DomainError {
	return &DomainError{
		code:    "DOMAIN_ERROR",
		message: message,
	}
}

// Error реализует интерфейс error
func (e *DomainError) Error() string {
	return e.message
}

// Code возвращает код ошибки
func (e *DomainError) Code() string {
	return e.code
}

// WithCode устанавливает код ошибки
func (e *DomainError) WithCode(code string) *DomainError {
	e.code = code
	return e
}

// Wrap оборачивает ошибку с контекстом
func (e *DomainError) Wrap(message string) error {
	return &DomainError{
		code:    e.code,
		message: fmt.Sprintf("%s: %s", message, e.message),
	}
}

var (
	ErrInvalidConsecutiveFailures = NewDomainError("consecutive_failures must be between 1 and 5").WithCode("ALERT_INVALID_CONSECUTIVE_FAILURES")
	ErrInvalidChatID              = NewDomainError("invalid chat_id format").WithCode("INVALID_CHAT_ID")
	ErrInvalidEmail               = NewDomainError("invalid email format").WithCode("INVALID_EMAIL")
	ErrInvalidFormat              = NewDomainError("method must be one of: GET, POST, PUT, PATCH").WithCode("INVALID_FORMAT")
	ErrInvalidURL                 = NewDomainError("invalid URL format").WithCode("INVALID_URL")
	ErrFieldTooLong               = NewDomainError("name exceeds maximum length of 255 characters").WithCode("FIELD_TOO_LONG")
	ErrDuplicateChannel           = NewDomainError("channel already exists").WithCode("DUPLICATE_CHANNEL")
	ErrForbidden                  = NewDomainError("access denied").WithCode("FORBIDDEN")
	ErrChannelLimitReached        = NewDomainError("channel limit reached").WithCode("CHANNEL_LIMIT_REACHED")
	ErrAlertNotAcknowledgeable    = NewDomainError("alert cannot be acknowledged in its current status").WithCode("ALERT_NOT_ACKNOWLEDGEABLE")
	ErrAlertAlreadyAcknowledged   = NewDomainError("alert already acknowledged").WithCode("ALERT_ALREADY_ACKNOWLEDGED")
	ErrChannelVerificationFailed  = NewDomainError("channel verification failed").WithCode("CHANNEL_VERIFICATION_FAILED")
	ErrGuestNotAllowed            = NewDomainError("guest users cannot perform this action").WithCode("GUEST_NOT_ALLOWED")

	// Ошибки окон обслуживания
	ErrMaintenanceWindowNotFound         = NewDomainError("maintenance window not found").WithCode("MAINTENANCE_WINDOW_NOT_FOUND")
	ErrMaintenanceWindowDurationExceeded = NewDomainError("window duration exceeds maximum of 24 hours").WithCode("DURATION_EXCEEDS_MAXIMUM")
	ErrMaintenanceWindowDurationTooShort = NewDomainError("window duration must be at least 1 minute").WithCode("DURATION_TOO_SHORT")
	ErrMaintenanceWindowInPast           = NewDomainError("cannot create maintenance window in the past").WithCode("CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST")
	ErrEndTimeBeforeStartTime            = NewDomainError("end_time must be after start_time").WithCode("END_TIME_BEFORE_START_TIME")
	ErrOverlappingMaintenanceWindows     = NewDomainError("overlapping maintenance windows for the same monitor").WithCode("OVERLAPPING_MAINTENANCE_WINDOWS")
	ErrMaintenanceWindowNotScheduled     = NewDomainError("only scheduled maintenance windows can be modified").WithCode("WINDOW_NOT_SCHEDULED")
	ErrMaintenanceWindowConflict         = NewDomainError("maintenance window was modified by another request").WithCode("CONFLICT")
	ErrInsufficientPermissions           = NewDomainError("insufficient permissions for this operation").WithCode("INSUFFICIENT_PERMISSIONS")
)
