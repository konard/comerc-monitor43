package errors

import (
	"errors"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// TestErrors_VerifyDeclaredErrors тестирует что ошибки объявлены корректно.
func TestErrors_VerifyDeclaredErrors(t *testing.T) {
	t.Run("ErrMonitorNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorNotFound)
		assert.Equal(t, "monitor not found", ErrMonitorNotFound.Error())
	})

	t.Run("ErrMonitorAlreadyExists is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorAlreadyExists)
		assert.Equal(t, "monitor already exists", ErrMonitorAlreadyExists.Error())
	})

	t.Run("ErrMonitorNameInvalid is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorNameInvalid)
		assert.Equal(t, "invalid monitor name", ErrMonitorNameInvalid.Error())
	})

	t.Run("ErrMonitorURLInvalid is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorURLInvalid)
		assert.Equal(t, "invalid monitor URL", ErrMonitorURLInvalid.Error())
	})

	t.Run("ErrMonitorIntervalInvalid is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorIntervalInvalid)
		assert.Equal(t, "invalid check interval", ErrMonitorIntervalInvalid.Error())
	})

	t.Run("ErrMonitorTimeoutInvalid is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorTimeoutInvalid)
		assert.Equal(t, "invalid timeout", ErrMonitorTimeoutInvalid.Error())
	})

	t.Run("ErrMonitorLimitExceeded is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorLimitExceeded)
		assert.Equal(t, "monitor limit exceeded", ErrMonitorLimitExceeded.Error())
	})

	t.Run("ErrMonitorIsPaused is defined", func(t *testing.T) {
		assert.NotNil(t, ErrMonitorIsPaused)
		assert.Equal(t, "monitor is paused", ErrMonitorIsPaused.Error())
	})

	t.Run("ErrCheckResultNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrCheckResultNotFound)
		assert.Equal(t, "check result not found", ErrCheckResultNotFound.Error())
	})

	t.Run("ErrIncidentNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrIncidentNotFound)
		assert.Equal(t, "incident not found", ErrIncidentNotFound.Error())
	})
}

// TestWrap тестирует функцию Wrap.
func TestWrap(t *testing.T) {
	t.Run("wrap error with message", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrappedErr := Wrap(originalErr, "failed to create monitor")

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), "failed to create monitor")
		assert.Contains(t, wrappedErr.Error(), "original error")
	})

	t.Run("wrap nil error returns nil", func(t *testing.T) {
		wrappedErr := Wrap(nil, "message")

		assert.Nil(t, wrappedErr)
	})

	t.Run("wrapped error contains original error", func(t *testing.T) {
		originalErr := errors.New("database connection failed")
		wrappedErr := Wrap(originalErr, "failed to connect")

		assert.Error(t, wrappedErr)
		assert.True(t, pkgerrors.Is(wrappedErr, originalErr))
	})
}

// TestWrapf тестирует функцию Wrapf.
func TestWrapf(t *testing.T) {
	t.Run("wrap error with formatted message", func(t *testing.T) {
		originalErr := errors.New("not found")
		wrappedErr := Wrapf(originalErr, "failed to get monitor: %s", "monitor-123")

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), "failed to get monitor: monitor-123")
		assert.Contains(t, wrappedErr.Error(), "not found")
	})

	t.Run("wrapf nil error returns nil", func(t *testing.T) {
		wrappedErr := Wrapf(nil, "message %s", "test")

		assert.Nil(t, wrappedErr)
	})

	t.Run("wrappedf error contains original error", func(t *testing.T) {
		originalErr := errors.New("timeout")
		wrappedErr := Wrapf(originalErr, "check failed: %v", originalErr)

		assert.Error(t, wrappedErr)
		assert.True(t, pkgerrors.Is(wrappedErr, originalErr))
	})

	t.Run("wrapf with multiple arguments", func(t *testing.T) {
		originalErr := errors.New("validation failed")
		wrappedErr := Wrapf(originalErr, "invalid field '%s' with value '%s'", "url", "invalid-url")

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), "invalid field 'url' with value 'invalid-url'")
		assert.Contains(t, wrappedErr.Error(), "validation failed")
	})
}

// TestNew тестирует функцию New.
func TestNew(t *testing.T) {
	t.Run("create new error", func(t *testing.T) {
		err := New("something went wrong")

		assert.Error(t, err)
		assert.Equal(t, "something went wrong", err.Error())
	})

	t.Run("new error is not nil", func(t *testing.T) {
		err := New("test error")

		assert.NotNil(t, err)
	})

	t.Run("new error contains message", func(t *testing.T) {
		message := "monitor validation failed"
		err := New(message)

		assert.Contains(t, err.Error(), message)
	})
}

// TestErrorUniqueness тестирует что все ошибки уникальны.
func TestErrorUniqueness(t *testing.T) {
	t.Run("all monitor errors are unique", func(t *testing.T) {
		errorsList := []error{
			ErrMonitorNotFound,
			ErrMonitorAlreadyExists,
			ErrMonitorNameInvalid,
			ErrMonitorURLInvalid,
			ErrMonitorIntervalInvalid,
			ErrMonitorTimeoutInvalid,
			ErrMonitorLimitExceeded,
			ErrMonitorIsPaused,
		}

		// Check that all errors are different
		uniqueErrors := make(map[string]struct{})
		for _, err := range errorsList {
			msg := err.Error()
			_, exists := uniqueErrors[msg]
			assert.False(t, exists, "Error message should be unique: "+msg)
			uniqueErrors[msg] = struct{}{}
		}

		assert.Equal(t, len(errorsList), len(uniqueErrors))
	})

	t.Run("check result and incident errors are unique", func(t *testing.T) {
		assert.NotEqual(t, ErrCheckResultNotFound.Error(), ErrIncidentNotFound.Error())
		assert.NotEqual(t, ErrCheckResultNotFound, ErrIncidentNotFound)
	})
}

// TestErrorTypes тестирует что ошибки имеют правильный тип.
func TestErrorTypes(t *testing.T) {
	t.Run("errors are from pkg/errors", func(t *testing.T) {
		// All errors should be fundamental errors from pkg/errors
		// which can be checked with errors.Is
		assert.True(t, pkgerrors.Is(ErrMonitorNotFound, ErrMonitorNotFound))
		assert.True(t, pkgerrors.Is(ErrMonitorAlreadyExists, ErrMonitorAlreadyExists))
	})

	t.Run("errors can be used with errors.Is", func(t *testing.T) {
		err := Wrap(ErrMonitorNotFound, "context")
		assert.True(t, pkgerrors.Is(err, ErrMonitorNotFound))
	})

	t.Run("wrapped errors preserve error type for Is", func(t *testing.T) {
		err := Wrap(ErrMonitorURLInvalid, "failed to validate")
		assert.True(t, pkgerrors.Is(err, ErrMonitorURLInvalid))
	})
}
