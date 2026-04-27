// Package errors предоставляет общие ошибки для monitor-service.
//
// Содержит базовые ошибки и утилиты для работы с ошибками.
package errors

import "github.com/pkg/errors"

// Ошибки бизнес-логики
var (
	// ErrMonitorNotFound монитор не найден
	ErrMonitorNotFound = errors.New("monitor not found")
	// ErrMonitorAlreadyExists монитор с таким именем уже существует
	ErrMonitorAlreadyExists = errors.New("monitor already exists")
	// ErrMonitorNameInvalid недопустимое имя монитора
	ErrMonitorNameInvalid = errors.New("invalid monitor name")
	// ErrMonitorURLInvalid недопустимый URL
	ErrMonitorURLInvalid = errors.New("invalid monitor URL")
	// ErrMonitorIntervalInvalid недопустимый интервал проверки
	ErrMonitorIntervalInvalid = errors.New("invalid check interval")
	// ErrMonitorTimeoutInvalid недопустимый таймаут
	ErrMonitorTimeoutInvalid = errors.New("invalid timeout")
	// ErrMonitorLimitExceeded превышен лимит мониторов для пользователя
	ErrMonitorLimitExceeded = errors.New("monitor limit exceeded")
	// ErrMonitorIsPaused монитор приостановлен
	ErrMonitorIsPaused = errors.New("monitor is paused")
	// ErrMonitorLimitReached достигнут лимит мониторов для subscription tier
	ErrMonitorLimitReached = errors.New("monitor limit reached for subscription tier")
	// ErrPermissionDenied доступ запрещён
	ErrPermissionDenied = errors.New("permission denied")
	// ErrUnknownTier неизвестный subscription tier
	ErrUnknownTier = errors.New("unknown subscription tier")

	// ErrCheckResultNotFound результат проверки не найден
	ErrCheckResultNotFound = errors.New("check result not found")

	// ErrIncidentNotFound инцидент не найден
	ErrIncidentNotFound = errors.New("incident not found")
)

// Wrap оборачивает ошибку с контекстом.
func Wrap(err error, message string) error {
	return errors.Wrap(err, message)
}

// Wrapf оборачивает ошибку с форматированным сообщением.
func Wrapf(err error, format string, args ...any) error {
	return errors.Wrapf(err, format, args...)
}

// New создаёт новую ошибку.
func New(message string) error {
	return errors.New(message)
}
