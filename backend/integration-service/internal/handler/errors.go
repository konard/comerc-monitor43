package handler

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// errorToStatus конвертирует domain ошибки в gRPC status errors.
func errorToStatus(err error) error {
	if err == nil {
		return nil
	}

	// Проверяем конкретные ошибки
	switch {
	case errors.Is(err, model.ErrWebhookNotFound),
		errors.Is(err, model.ErrAPIKeyNotFound),
		errors.Is(err, model.ErrImportNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, model.ErrInvalidUUID):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, model.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, model.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, model.ErrWebhookDuplicateName),
		errors.Is(err, model.ErrAPIKeyDuplicateName):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, model.ErrWebhookInvalidURL),
		errors.Is(err, model.ErrWebhookInjection),
		errors.Is(err, model.ErrWebhookInvalidMethod),
		errors.Is(err, model.ErrEmptyWebhookName),
		errors.Is(err, model.ErrWebhookNameTooLong),
		errors.Is(err, model.ErrWebhookPayloadTooLarge),
		errors.Is(err, model.ErrEmptyAPIKeyName),
		errors.Is(err, model.ErrAPIKeyNameTooLong),
		errors.Is(err, model.ErrInvalidScope),
		errors.Is(err, model.ErrInsufficientScope),
		errors.Is(err, model.ErrIPNotAllowed),
		errors.Is(err, model.ErrInvalidImportFormat),
		errors.Is(err, model.ErrImportValidationFailed),
		errors.Is(err, model.ErrUnsupportedImportSource):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, model.ErrWebhookLimitReached),
		errors.Is(err, model.ErrAPIKeyLimitReached),
		errors.Is(err, model.ErrMonitorLimitExceeded),
		errors.Is(err, model.ErrRateLimitExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())

	case errors.Is(err, model.ErrDatabaseWriteFailed),
		errors.Is(err, model.ErrInternal):
		return status.Error(codes.Internal, err.Error())

	default:
		// Для обёрнутых ошибок пытаемся извлечь оригинальную
		unwrapped := errors.Unwrap(err)
		if unwrapped != nil {
			return errorToStatus(unwrapped)
		}

		// Логируем неожиданные ошибки
		return status.Error(codes.Internal, "internal server error")
	}
}

// invalidArgument создаёт InvalidArgument ошибку с деталями поля.
func invalidArgument(field, message string) error {
	return status.Error(codes.InvalidArgument, fmt.Sprintf("%s: %s", field, message))
}
