package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestCheckResult_WithResponseTime тестирует установку времени ответа.
func TestCheckResult_WithResponseTime(t *testing.T) {
	monitorID := uuid.New()
	result := NewCheckResult(monitorID, StatusUp)

	assert.Nil(t, result.ResponseTimeMs)
	assert.Equal(t, 0, result.GetResponseTime())

	result = result.WithResponseTime(500)
	assert.NotNil(t, result.ResponseTimeMs)
	assert.Equal(t, 500, result.GetResponseTime())
	assert.Equal(t, StatusUp, result.Status)
}

// TestCheckResult_WithStatusCode тестирует установку статус кода.
func TestCheckResult_WithStatusCode(t *testing.T) {
	monitorID := uuid.New()
	result := NewCheckResult(monitorID, StatusDown)

	assert.Nil(t, result.StatusCode)
	assert.Equal(t, 0, result.GetStatusCode())

	result = result.WithStatusCode(503)
	assert.NotNil(t, result.StatusCode)
	assert.Equal(t, 503, result.GetStatusCode())
	assert.Equal(t, StatusDown, result.Status)
}

// TestCheckResult_WithError тестирует установку ошибки.
func TestCheckResult_WithError(t *testing.T) {
	monitorID := uuid.New()
	result := NewCheckResult(monitorID, StatusDown)

	assert.Empty(t, result.GetErrorMessage())

	err := errors.New("connection timeout")
	result = result.WithError(err)

	assert.NotEmpty(t, result.GetErrorMessage())
	assert.Contains(t, result.GetErrorMessage(), "connection timeout")
}

// TestCheckResult_IsSuccess тестирует проверку на успешность.
func TestCheckResult_IsSuccess(t *testing.T) {
	monitorID := uuid.New()

	t.Run("UP status is success", func(t *testing.T) {
		result := NewCheckResult(monitorID, StatusUp)
		assert.True(t, result.IsSuccess())
		assert.False(t, result.IsFailure())
		assert.False(t, result.IsDegraded())
	})

	t.Run("DOWN status is failure", func(t *testing.T) {
		result := NewCheckResult(monitorID, StatusDown)
		assert.False(t, result.IsSuccess())
		assert.True(t, result.IsFailure())
		assert.False(t, result.IsDegraded())
	})

	t.Run("DEGRADED status is degraded", func(t *testing.T) {
		result := NewCheckResult(monitorID, StatusDegraded)
		assert.False(t, result.IsSuccess())
		assert.False(t, result.IsFailure())
		assert.True(t, result.IsDegraded())
	})

	t.Run("PAUSED status is not success/failure", func(t *testing.T) {
		result := NewCheckResult(monitorID, StatusPaused)
		assert.False(t, result.IsSuccess())
		assert.False(t, result.IsFailure())
		assert.False(t, result.IsDegraded())
	})
}

// TestCheckResult_GetMethods тестирует getter методы.
func TestCheckResult_GetMethods(t *testing.T) {
	monitorID := uuid.New()
	result := NewCheckResult(monitorID, StatusUp)

	t.Run("GetResponseTime with nil", func(t *testing.T) {
		assert.Equal(t, 0, result.GetResponseTime())
	})

	t.Run("GetResponseTime with value", func(t *testing.T) {
		result = result.WithResponseTime(250)
		assert.Equal(t, 250, result.GetResponseTime())
	})

	t.Run("GetStatusCode with nil", func(t *testing.T) {
		assert.Equal(t, 0, result.GetStatusCode())
	})

	t.Run("GetStatusCode with value", func(t *testing.T) {
		result = result.WithStatusCode(200)
		assert.Equal(t, 200, result.GetStatusCode())
	})

	t.Run("GetErrorMessage with nil", func(t *testing.T) {
		assert.Empty(t, result.GetErrorMessage())
	})

	t.Run("GetErrorMessage with value", func(t *testing.T) {
		result = result.WithError(errors.New("test error"))
		assert.NotEmpty(t, result.GetErrorMessage())
	})
}

// TestCheckResultSlice_CalculateAverageResponseTime тестирует вычисление среднего времени ответа.
func TestCheckResultSlice_CalculateAverageResponseTime(t *testing.T) {
	monitorID := uuid.New()

	t.Run("empty slice", func(t *testing.T) {
		var results CheckResultSlice
		assert.Equal(t, 0, results.CalculateAverageResponseTime())
	})

	t.Run("all results with response time", func(t *testing.T) {
		results := CheckResultSlice{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(200),
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(300),
		}
		assert.Equal(t, 200, results.CalculateAverageResponseTime())
	})

	t.Run("some results without response time", func(t *testing.T) {
		results := CheckResultSlice{
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(100),
			*NewCheckResult(monitorID, StatusUp), // No response time
			*NewCheckResult(monitorID, StatusUp).WithResponseTime(300),
		}
		// (100 + 300) / 2 = 200
		assert.Equal(t, 200, results.CalculateAverageResponseTime())
	})

	t.Run("all results without response time", func(t *testing.T) {
		results := CheckResultSlice{
			*NewCheckResult(monitorID, StatusUp),
			*NewCheckResult(monitorID, StatusUp),
			*NewCheckResult(monitorID, StatusUp),
		}
		assert.Equal(t, 0, results.CalculateAverageResponseTime())
	})
}

// TestCheckResultSlice_FilterByStatus тестирует фильтрацию по статусу.
func TestCheckResultSlice_FilterByStatus(t *testing.T) {
	monitorID := uuid.New()

	results := CheckResultSlice{
		*NewCheckResult(monitorID, StatusUp),
		*NewCheckResult(monitorID, StatusDown),
		*NewCheckResult(monitorID, StatusDegraded),
		*NewCheckResult(monitorID, StatusUp),
		*NewCheckResult(monitorID, StatusPaused),
	}

	t.Run("filter UP", func(t *testing.T) {
		filtered := results.FilterByStatus(StatusUp)
		assert.Len(t, filtered, 2)
		for _, r := range filtered {
			assert.Equal(t, StatusUp, r.Status)
		}
	})

	t.Run("filter DOWN", func(t *testing.T) {
		filtered := results.FilterByStatus(StatusDown)
		assert.Len(t, filtered, 1)
		assert.Equal(t, StatusDown, filtered[0].Status)
	})

	t.Run("filter PAUSED", func(t *testing.T) {
		filtered := results.FilterByStatus(StatusPaused)
		assert.Len(t, filtered, 1)
		assert.Equal(t, StatusPaused, filtered[0].Status)
	})

	t.Run("filter non-existent status", func(t *testing.T) {
		filtered := results.FilterByStatus(StatusPending)
		assert.Len(t, filtered, 0)
	})
}

// TestCheckResultSlice_FilterActive тестирует фильтрацию активных результатов.
func TestCheckResultSlice_FilterActive(t *testing.T) {
	monitorID := uuid.New()

	results := CheckResultSlice{
		*NewCheckResult(monitorID, StatusUp),
		*NewCheckResult(monitorID, StatusDown),
		*NewCheckResult(monitorID, StatusDegraded),
		*NewCheckResult(monitorID, StatusPaused),
		*NewCheckResult(monitorID, StatusUp),
	}

	filtered := results.FilterActive()
	assert.Len(t, filtered, 4) // All except PAUSED

	for _, r := range filtered {
		assert.NotEqual(t, StatusPaused, r.Status)
	}
}

// TestCheckResultSlice_CountByStatus тестирует подсчёт по статусам.
func TestCheckResultSlice_CountByStatus(t *testing.T) {
	monitorID := uuid.New()

	results := CheckResultSlice{
		*NewCheckResult(monitorID, StatusUp),
		*NewCheckResult(monitorID, StatusUp),
		*NewCheckResult(monitorID, StatusDown),
		*NewCheckResult(monitorID, StatusDown),
		*NewCheckResult(monitorID, StatusDegraded),
		*NewCheckResult(monitorID, StatusPaused),
	}

	counts := results.CountByStatus()

	assert.Equal(t, 2, counts[StatusUp])
	assert.Equal(t, 2, counts[StatusDown])
	assert.Equal(t, 1, counts[StatusDegraded])
	assert.Equal(t, 1, counts[StatusPaused])
	assert.Equal(t, 6, len(results))
}

func TestCheckResult_WithErrorCode(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()

	t.Run("sets error code", func(t *testing.T) {
		result := NewCheckResult(monitorID, StatusDown)
		assert.Nil(t, result.ErrorCode)

		result = result.WithErrorCode(ErrorCodeConnectionTimeout)
		assert.NotNil(t, result.ErrorCode)
		assert.Equal(t, ErrorCodeConnectionTimeout, *result.ErrorCode)
	})

	t.Run("preserves other fields", func(t *testing.T) {
		result := NewCheckResult(monitorID, StatusDown).
			WithResponseTime(150).
			WithError(errors.New("test"))

		result = result.WithErrorCode(ErrorCodeDNSResolutionFailed)

		assert.Equal(t, ErrorCodeDNSResolutionFailed, *result.ErrorCode)
		assert.Equal(t, 150, result.GetResponseTime())
		assert.Contains(t, result.GetErrorMessage(), "test")
	})
}
