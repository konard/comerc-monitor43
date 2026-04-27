package errors

import (
	"errors"
	"fmt"
)

// Обертки над стандартными ошибками для использования в проекте.

var (
	// ErrNotFound возвращается когда сущность не найдена.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists возвращается когда сущность уже существует.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidArgument возвращается при невалидных аргументах.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrPermissionDenied возвращается при недостатке прав.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrUnauthenticated возвращается когда пользователь не аутентифицирован.
	ErrUnauthenticated = errors.New("unauthenticated")

	// ErrInternal возвращается для внутренних ошибок.
	ErrInternal = errors.New("internal error")

	// ErrPaymentFailed возвращается при ошибке платежа.
	ErrPaymentFailed = errors.New("payment failed")

	// ErrSubscriptionExpired возвращается когда подписка истекла.
	ErrSubscriptionExpired = errors.New("subscription expired")

	// ErrSubscriptionCanceled возвращается когда подписка отменена.
	ErrSubscriptionCanceled = errors.New("subscription canceled")

	// ErrPlanLimitExceeded возвращается когда превышен лимит плана.
	ErrPlanLimitExceeded = errors.New("plan limit exceeded")
)

// NotFound оборачивает ошибку с контекстом "not found".
func NotFound(entity string, id any) error {
	return fmt.Errorf("%s %v: %w", entity, id, ErrNotFound)
}

// AlreadyExists оборачивает ошибку с контекстом "already exists".
func AlreadyExists(entity string, id any) error {
	return fmt.Errorf("%s %v: %w", entity, id, ErrAlreadyExists)
}

// InvalidArgument оборачивает ошибку с контекстом "invalid argument".
func InvalidArgument(field, reason string) error {
	return fmt.Errorf("invalid argument %s: %s: %w", field, reason, ErrInvalidArgument)
}

// PaymentFailed оборачивает ошибку платежа с деталями.
func PaymentFailed(reason string) error {
	return fmt.Errorf("payment failed: %s: %w", reason, ErrPaymentFailed)
}
