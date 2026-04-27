package errors

import "fmt"

// Error codes for auth service responses
const (
	// Auth errors
	ErrorCodeInvalidCredentials  = "INVALID_CREDENTIALS" //nolint:gosec // error code constant, not credential
	ErrorCodeInvalidToken        = "INVALID_TOKEN"
	ErrorCodeExpiredToken        = "EXPIRED_TOKEN"
	ErrorCodeInvalidRefreshToken = "INVALID_REFRESH_TOKEN"
	ErrorCodeAccountLocked       = "ACCOUNT_LOCKED"
	ErrorCodeAccountNotFound     = "ACCOUNT_NOT_FOUND"
	ErrorCodeUserNotFound        = "USER_NOT_FOUND"
	ErrorCodeUserExists          = "USER_EXISTS"
	ErrorCodeInvalidRequest      = "INVALID_REQUEST"
	ErrorCodeOAuthFailed         = "OAUTH_FAILED"
	ErrorCodeOAuthStateMismatch  = "OAUTH_STATE_MISMATCH"
	ErrorCodeSessionNotFound     = "SESSION_NOT_FOUND"
	ErrorCodeUnauthorized        = "UNAUTHORIZED"
	ErrorCodeForbidden           = "FORBIDDEN"

	// Validation errors
	ErrorCodeInvalidEmail       = "INVALID_EMAIL"
	ErrorCodeInvalidPassword    = "INVALID_PASSWORD"
	ErrorCodeInvalidOAuthCode   = "INVALID_OAUTH_CODE"
	ErrorCodeInvalidRedirectURI = "INVALID_REDIRECT_URI"

	// Rate limit errors
	ErrorCodeTooManyRequests = "TOO_MANY_REQUESTS"

	// Internal errors
	ErrorCodeInternalError = "INTERNAL_ERROR"
	ErrorCodeDatabaseError = "DATABASE_ERROR"
	ErrorCodeRedisError    = "REDIS_ERROR"
)

// AppError represents an application error with code and details
type AppError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New creates a new AppError
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to an AppError
func (e *AppError) WithDetails(details map[string]any) *AppError {
	e.Details = details
	return e
}

// Common error constructors
func InvalidCredentials() *AppError {
	return New(ErrorCodeInvalidCredentials, "invalid email or password")
}

func InvalidToken() *AppError {
	return New(ErrorCodeInvalidToken, "invalid access token")
}

func ExpiredToken() *AppError {
	return New(ErrorCodeExpiredToken, "access token has expired")
}

func InvalidRefreshToken() *AppError {
	return New(ErrorCodeInvalidRefreshToken, "invalid or expired refresh token")
}

func AccountLocked() *AppError {
	return New(ErrorCodeAccountLocked, "account is locked")
}

func AccountNotFound() *AppError {
	return New(ErrorCodeAccountNotFound, "account not found")
}

func UserExists() *AppError {
	return New(ErrorCodeUserExists, "user with this email already exists")
}

func InvalidRequest(message string) *AppError {
	return New(ErrorCodeInvalidRequest, message)
}

func OAuthFailed(err error) *AppError {
	return New(ErrorCodeOAuthFailed, fmt.Sprintf("OAuth failed: %v", err))
}

func OAuthStateMismatch() *AppError {
	return New(ErrorCodeOAuthStateMismatch, "OAuth state mismatch")
}

func SessionNotFound() *AppError {
	return New(ErrorCodeSessionNotFound, "session not found")
}

func Unauthorized() *AppError {
	return New(ErrorCodeUnauthorized, "unauthorized access")
}

func InvalidEmail() *AppError {
	return New(ErrorCodeInvalidEmail, "invalid email address")
}

func InvalidPassword() *AppError {
	return New(ErrorCodeInvalidPassword, "password must be at least 8 characters")
}

func InvalidOAuthCode() *AppError {
	return New(ErrorCodeInvalidOAuthCode, "invalid OAuth code")
}

func TooManyRequests() *AppError {
	return New(ErrorCodeTooManyRequests, "too many requests, please try again later")
}

func Forbidden() *AppError {
	return New(ErrorCodeForbidden, "forbidden access")
}

func UserNotFound() *AppError {
	return New(ErrorCodeUserNotFound, "user not found")
}

func InternalError(err error) *AppError {
	return New(ErrorCodeInternalError, fmt.Sprintf("internal error: %v", err))
}
