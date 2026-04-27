package model

import "github.com/pkg/errors"

// Webhook errors
var (
	ErrWebhookNotFound        = errors.New("webhook not found")
	ErrWebhookInvalidURL      = errors.New("invalid webhook URL")
	ErrWebhookInjection       = errors.New("webhook URL contains injection patterns")
	ErrWebhookTimeout         = errors.New("webhook request timeout")
	ErrWebhookAuthFailed      = errors.New("webhook authentication failed")
	ErrWebhookDeliveryFailed  = errors.New("webhook delivery failed")
	ErrWebhook5xxError        = errors.New("webhook returned 5xx error")
	ErrWebhookLimitReached    = errors.New("webhook limit reached for subscription tier")
	ErrWebhookDuplicateName   = errors.New("webhook with this name already exists")
	ErrEmptyWebhookName       = errors.New("webhook name cannot be empty")
	ErrWebhookNameTooLong     = errors.New("webhook name too long (max 255 chars)")
	ErrWebhookPayloadTooLarge = errors.New("webhook payload exceeds size limit")
	ErrWebhookInvalidMethod   = errors.New("invalid HTTP method (must be GET, POST, PUT, PATCH)")
	ErrDeliveryPermanentError = errors.New("permanent delivery error (4xx)")
)

// API Key errors
var (
	ErrAPIKeyNotFound         = errors.New("API key not found")
	ErrAPIKeyInvalid          = errors.New("invalid API key")
	ErrAPIKeyExpired          = errors.New("API key expired")
	ErrAPIKeyDeactivated      = errors.New("API key deactivated")
	ErrAPIKeyInactive         = errors.New("API key inactive")
	ErrAPIKeyUnderMaintenance = errors.New("API key under maintenance")
	ErrInsufficientScope      = errors.New("insufficient scope for this operation")
	ErrIPNotAllowed           = errors.New("IP address not allowed")
	ErrAPIKeyLimitReached     = errors.New("API key limit reached for subscription tier")
	ErrEmptyAPIKeyName        = errors.New("API key name cannot be empty")
	ErrAPIKeyNameTooLong      = errors.New("API key name too long (max 255 chars)")
	ErrAPIKeyDuplicateName    = errors.New("API key with this name already exists")
	ErrInvalidScope           = errors.New("invalid API key scope")
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
)

// Import errors
var (
	ErrImportNotFound          = errors.New("import not found")
	ErrInvalidImportFormat     = errors.New("invalid import format")
	ErrImportValidationFailed  = errors.New("import validation failed")
	ErrImportFileTooLarge      = errors.New("import file too large")
	ErrMonitorLimitExceeded    = errors.New("monitor limit exceeded for subscription tier")
	ErrUnsupportedImportSource = errors.New("unsupported import source")
)

// Common errors
var (
	ErrInvalidUUID         = errors.New("invalid UUID format")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrInternal            = errors.New("internal server error")
	ErrDatabaseWriteFailed = errors.New("database write failed")
)
