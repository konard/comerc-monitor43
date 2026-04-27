package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents type of error
type ErrorCode string

const (
	// InternalError indicates an internal server error
	InternalError ErrorCode = "INTERNAL_ERROR"
	// InvalidRequest indicates a bad request
	InvalidRequest ErrorCode = "INVALID_REQUEST"
	// Unauthorized indicates unauthorized access
	Unauthorized ErrorCode = "UNAUTHORIZED"
	// NotFound indicates a resource was not found
	NotFound ErrorCode = "NOT_FOUND"
	// ServiceUnavailable indicates a downstream service is unavailable
	ServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	// Timeout indicates a timeout occurred
	Timeout ErrorCode = "TIMEOUT"
	// TooManyRequests indicates rate limit exceeded
	TooManyRequests ErrorCode = "TOO_MANY_REQUESTS"
	// Forbidden indicates forbidden access
	Forbidden ErrorCode = "FORBIDDEN"
	// InvalidToken indicates an invalid access token
	InvalidToken ErrorCode = "INVALID_TOKEN"
	// ExpiredToken indicates an expired access token
	ExpiredToken ErrorCode = "EXPIRED_TOKEN"
	// InvalidCredentials indicates invalid email or password
	//nolint:gosec // Это код ошибки, а не credential.
	InvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	// AccountLocked indicates account is locked
	AccountLocked ErrorCode = "ACCOUNT_LOCKED"
	// UserNotFound indicates user not found
	UserNotFound ErrorCode = "USER_NOT_FOUND"
)

// AppError represents an application error
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	StatusCode int       `json:"-"`
	Err        error     `json:"-"`
	RequestID  string    `json:"request_id,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		StatusCode: e.StatusCode,
		Err:        e.Err,
		RequestID:  requestID,
	}
}

// New creates a new AppError
func New(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an existing error
func Wrap(code ErrorCode, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Common error constructors
var (
	// ErrInternal indicates an internal server error
	ErrInternal = New(InternalError, "Internal server error", http.StatusInternalServerError)

	// ErrInvalidRequest indicates a bad request
	ErrInvalidRequest = New(InvalidRequest, "Invalid request", http.StatusBadRequest)

	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = New(Unauthorized, "Unauthorized", http.StatusUnauthorized)

	// ErrNotFound indicates a resource was not found
	ErrNotFound = New(NotFound, "Resource not found", http.StatusNotFound)

	// ErrServiceUnavailable indicates a downstream service is unavailable
	ErrServiceUnavailable = New(ServiceUnavailable, "Service unavailable", http.StatusServiceUnavailable)

	// ErrTimeout indicates a timeout occurred
	ErrTimeout = New(Timeout, "Request timeout", http.StatusGatewayTimeout)

	// ErrForbidden indicates forbidden access
	ErrForbidden = New(Forbidden, "Forbidden access", http.StatusForbidden)

	// ErrInvalidToken indicates an invalid access token
	ErrInvalidToken = New(InvalidToken, "Invalid access token", http.StatusUnauthorized)

	// ErrExpiredToken indicates an expired access token
	ErrExpiredToken = New(ExpiredToken, "Access token has expired", http.StatusUnauthorized)

	// ErrInvalidCredentials indicates invalid email or password
	ErrInvalidCredentials = New(InvalidCredentials, "Invalid email or password", http.StatusUnauthorized)

	// ErrAccountLocked indicates account is locked
	ErrAccountLocked = New(AccountLocked, "Account is locked", http.StatusForbidden)

	// ErrUserNotFound indicates user not found
	ErrUserNotFound = New(UserNotFound, "User not found", http.StatusNotFound)
)

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetStatusCode returns HTTP status code for an error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// GetErrorCode returns the error code for an error
func GetErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return string(appErr.Code)
	}
	return string(InternalError)
}
