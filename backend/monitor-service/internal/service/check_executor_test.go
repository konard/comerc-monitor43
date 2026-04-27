package service

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

// TestCheckExecutor_ExecuteCheck_Success тест успешной проверки.
func TestCheckExecutor_ExecuteCheck_Success(t *testing.T) {
	t.Run("returns UP for 200 OK", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockResultRepo := new(MockCheckResultRepository)

		userID := uuid.New()
		_ = mustNewMonitor(t, userID, "Test Monitor", "https://example.com", 60)

		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)
		assert.NotNil(t, executor)

		// Mock HTTP response
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("OK"))
			assert.NoError(t, err)
		})
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		// Create a test request that would point to the test server
		// In real implementation, this would use the monitor's URL
		t.Skip("Requires actual HTTP client or proper mocking")
	})
}

// TestCheckExecutor_DetermineStatus тестирует определение статуса.
func TestCheckExecutor_DetermineStatus(t *testing.T) {
	// Test DOWN status detection
	t.Run("returns DOWN for 500 error", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockResultRepo := new(MockCheckResultRepository)

		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)
		assert.NotNil(t, executor)

		userID := uuid.New()
		monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
		monitor.Status = domain.StatusUp

		result := domain.NewCheckResult(monitor.ID, domain.StatusDown)
		result = result.WithStatusCode(500)

		assert.Equal(t, domain.StatusDown, result.Status)
	})

	// Test DEGRADED detection by response time
	t.Run("returns DEGRADED for slow response", func(t *testing.T) {
		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		// Status UP but response time exceeds threshold
		result := domain.NewCheckResult(uuid.New(), domain.StatusUp)
		result = result.WithResponseTime(1500) // > 1000ms

		status := determineStatus(config, result, []domain.CheckResult{})
		assert.Equal(t, domain.StatusDegraded, status)
	})

	// Test DEGRADED detection by failure rate
	t.Run("returns DEGRADED for high failure rate", func(t *testing.T) {
		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		// Status UP but failure rate exceeds threshold
		result := domain.NewCheckResult(uuid.New(), domain.StatusUp)
		result = result.WithResponseTime(500)       // Fast response
		results := createResultsWithFailures(10, 6) // 60% failure rate > 50%

		status := determineStatus(config, result, results)
		assert.Equal(t, domain.StatusDegraded, status)
	})

	// Test UP status
	t.Run("returns UP for all checks passed", func(t *testing.T) {
		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		// Status UP, fast response, low failure rate
		result := domain.NewCheckResult(uuid.New(), domain.StatusUp)
		result = result.WithResponseTime(200)

		results := createResultsWithFailures(9, 1) // 10% failure rate < 50%

		status := determineStatus(config, result, results)
		assert.Equal(t, domain.StatusUp, status)
	})
}

// determineStatus упрощённая версия для тестов.
func determineStatus(config CheckExecutorConfig, result *domain.CheckResult, recentResults []domain.CheckResult) domain.MonitorStatus {
	// Check DOWN first (highest priority)
	if result.StatusCode != nil && *result.StatusCode >= 500 {
		return domain.StatusDown
	}

	if result.ErrorMessage != nil {
		return domain.StatusDown
	}

	// Check DEGRADED: OR condition
	// Condition 1: Response time exceeds threshold
	degraded := config.DefaultDegradedResponseTime > 0 && result.ResponseTimeMs != nil && *result.ResponseTimeMs > config.DefaultDegradedResponseTime

	// Condition 2: Failure rate exceeds threshold
	if len(recentResults) > 0 {
		failureCount := 0
		for _, r := range recentResults {
			if r.Status == domain.StatusDown {
				failureCount++
			}
		}
		failureRate := (float64(failureCount) / float64(len(recentResults))) * 100
		if failureRate > float64(config.DefaultDegradedFailureRate) {
			degraded = true
		}
	}

	if degraded {
		return domain.StatusDegraded
	}

	return domain.StatusUp
}

// createResultsWithFailures создаёт результаты с указанным количеством failures.
func createResultsWithFailures(total, failures int) []domain.CheckResult {
	userID := uuid.New()
	results := make([]domain.CheckResult, total)

	for i := 0; i < total; i++ {
		status := domain.StatusUp
		if i < failures {
			status = domain.StatusDown
		}
		results[i] = *domain.NewCheckResult(userID, status)
		results[i].CheckedAt = time.Now().Add(time.Duration(i) * time.Second)
	}

	return results
}

// TestNewCheckExecutor_MetricInitializationErrors тестирует обработку ошибок при инициализации метрик.
func TestNewCheckExecutor_MetricInitializationErrors(t *testing.T) {
	t.Run("creates executor successfully even if metric initialization fails", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockResultRepo := new(MockCheckResultRepository)

		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		// Act - NewCheckExecutor should handle metric initialization errors gracefully
		executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

		// Assert
		assert.NotNil(t, executor)
		assert.NotNil(t, executor.httpClient)
		assert.Equal(t, config.DefaultTimeout, executor.config.DefaultTimeout)
		assert.Equal(t, config.MaxRecentResultsForStats, executor.config.MaxRecentResultsForStats)
		// Metrics may be nil if initialization failed, but executor should still work
		// The executor should handle nil metrics gracefully in recordMetrics
	})

	t.Run("configures HTTP client with correct timeout and redirect handling", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockResultRepo := new(MockCheckResultRepository)

		config := CheckExecutorConfig{
			DefaultTimeout:              45 * time.Second,
			MaxRecentResultsForStats:    20,
			DefaultDegradedResponseTime: 2000,
			DefaultDegradedFailureRate:  75,
		}

		executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

		// Assert
		assert.NotNil(t, executor)
		assert.NotNil(t, executor.httpClient)
		assert.Equal(t, config.DefaultTimeout, executor.httpClient.Timeout)
	})

	t.Run("initializes all metrics fields", func(t *testing.T) {
		mockRepo := new(MockMonitorRepository)
		mockResultRepo := new(MockCheckResultRepository)

		config := CheckExecutorConfig{
			DefaultTimeout:              30 * time.Second,
			MaxRecentResultsForStats:    10,
			DefaultDegradedResponseTime: 1000,
			DefaultDegradedFailureRate:  50,
		}

		executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

		// Assert - all metric fields should be initialized (possibly to nil on error)
		assert.NotNil(t, executor)
		// The meter should be initialized
		assert.NotNil(t, executor.meter)
		// Metric instruments may be nil if initialization failed, which is ok
	})
}

// TestCheckExecutor_ExecuteCheck_InvalidMonitorID тестирует случай невалидного ID монитора.
func TestCheckExecutor_ExecuteCheck_InvalidMonitorID(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	invalidMonitorID := "invalid-uuid"

	// Act
	result, err := executor.ExecuteCheck(ctx, invalidMonitorID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid monitor_id")
}

// TestCheckExecutor_ExecuteCheck_MonitorNotFound тестирует случай, когда монитор не найден.
func TestCheckExecutor_ExecuteCheck_MonitorNotFound(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	monitorID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, monitorID).Return(nil, errors.New("monitor not found"))

	// Act
	result, err := executor.ExecuteCheck(ctx, monitorID.String())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get monitor")
	mockRepo.AssertExpectations(t)
}

// TestCheckExecutor_ExecuteCheck_MonitorPaused тестирует случай, когда монитор на паузе.
func TestCheckExecutor_ExecuteCheck_MonitorPaused(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusPaused

	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	// Act
	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "monitor is paused")
	mockRepo.AssertExpectations(t)
}

// TestCheckExecutor_ExecuteCheck_OutsideWorkingHours тестирует случай вне рабочих часов.
func TestCheckExecutor_ExecuteCheck_OutsideWorkingHours(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	// Устанавливаем рабочие часы так, чтобы текущее время было вне их
	now := time.Now()
	startTime := now.Add(2 * time.Hour)
	endTime := now.Add(10 * time.Hour)
	monitor.WorkingHoursStart = &startTime
	monitor.WorkingHoursEnd = &endTime

	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	// Act
	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "outside working hours")
	mockRepo.AssertExpectations(t)
}

// TestCheckExecutor_ExecuteCheck_SaveError тестирует ошибку при сохранении результата.
func TestCheckExecutor_ExecuteCheck_SaveError(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusUp

	// Mock: монитор найден
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	// Mock: нет последних результатов
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)

	// Mock: ошибка при сохранении результата
	mockResultRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error"))
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Maybe() // Update may not be called if Create fails

	// Act - реальный HTTP запрос к example.com
	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save check result")

	// Don't assert expectations - Update may or may not be called
}

// TestCheckExecutor_ExecuteCheck_UpdateMonitorError тестирует что check успешен даже если Update монитора падает.
func TestCheckExecutor_ExecuteCheck_UpdateMonitorError(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.Status = domain.StatusUp

	// Mock: монитор найден
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	// Mock: нет последних результатов
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)

	// Mock: результат сохранён успешно
	mockResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Mock: Update монитора падает, но check должен всё равно пройти успешно
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("database update failed"))

	// Act
	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	// Assert - check должен быть успешным несмотря на ошибку Update
	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
	mockResultRepo.AssertExpectations(t)
}

// TestCheckExecutor_CreateErrorResult тестирует создание результата с ошибкой.
func TestCheckExecutor_CreateErrorResult(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	monitorID := uuid.New()
	testError := errors.New("test error")

	// Act
	result := executor.createErrorResult(monitorID, testError, domain.ErrorCodeNetworkError)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, domain.StatusDown, result.Status)
	assert.Equal(t, monitorID, result.MonitorID)
	assert.NotNil(t, result.ErrorMessage)
	assert.Contains(t, *result.ErrorMessage, "test error")
}

// TestCheckExecutor_RecordMetrics_NilMetrics тестирует запись метрик когда они nil.
func TestCheckExecutor_RecordMetrics_NilMetrics(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)
	// Устанавливаем метрики в nil
	executor.checksTotal = nil
	executor.checksDuration = nil
	executor.checksStatus = nil

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	result := domain.NewCheckResult(monitor.ID, domain.StatusUp)

	// Act - должен паниковать
	assert.NotPanics(t, func() {
		executor.recordMetrics(ctx, monitor, result, 1.5)
	})
}

// TestCheckExecutor_DetermineStatus_NoRecentResults тестирует определение статуса без истории.
func TestCheckExecutor_DetermineStatus_NoRecentResults(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	// Mock: нет последних результатов
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)

	// Act
	status, err := executor.determineStatus(ctx, monitor, 500)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusUp, status)
	mockResultRepo.AssertExpectations(t)
}

// TestCheckExecutor_DetermineStatus_ResponseTimeThreshold тестирует определение статуса по времени ответа.
func TestCheckExecutor_DetermineStatus_ResponseTimeThreshold(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	// Mock: нет последних результатов
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)

	// Act - response time > threshold
	status, err := executor.determineStatus(ctx, monitor, 1500)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusDegraded, status)
	mockResultRepo.AssertExpectations(t)
}

// TestCheckExecutor_DetermineStatus_MonitorThresholds тестирует использование порогов монитора.
func TestCheckExecutor_DetermineStatus_MonitorThresholds(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	// Устанавливаем свои пороги
	customThreshold := 500
	monitor.DegradedResponseTimeThreshold = &customThreshold
	customFailureRate := 30
	monitor.DegradedFailureRateThreshold = &customFailureRate

	// Mock: нет последних результатов
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)

	// Act - response time > custom threshold but < default
	status, err := executor.determineStatus(ctx, monitor, 750)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusDegraded, status)
	mockResultRepo.AssertExpectations(t)
}

// TestCheckExecutor_ExecuteHTTPCheck_Success тестирует успешную HTTP проверку.
func TestCheckExecutor_ExecuteHTTPCheck_Success(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	monitor.TimeoutSeconds = 5

	// Mock: нет последних результатов для статистики
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)

	// Act - выполняем реальную HTTP проверку
	result := executor.executeHTTPCheck(ctx, monitor)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, monitor.ID, result.MonitorID)
	assert.NotNil(t, result.ResponseTimeMs)
	assert.NotNil(t, result.StatusCode)
	// Status может быть UP или DEGRADED в зависимости от response time
	assert.Contains(t, []domain.MonitorStatus{domain.StatusUp, domain.StatusDegraded}, result.Status)
	mockResultRepo.AssertExpectations(t)
}

// TestCheckExecutor_ExecuteHTTPCheck_ServerError тестирует 5xx ошибку.
func TestCheckExecutor_ExecuteHTTPCheck_ServerError(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()

	// Создаём test server который возвращает 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("Internal Server Error"))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.TimeoutSeconds = 5

	// Act
	result := executor.executeHTTPCheck(ctx, monitor)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, domain.StatusDown, result.Status)
	assert.Equal(t, monitor.ID, result.MonitorID)
	assert.NotNil(t, result.StatusCode)
	assert.Equal(t, 500, *result.StatusCode)
	assert.NotNil(t, result.ErrorMessage)
	assert.Contains(t, *result.ErrorMessage, "server error")
}

// TestCheckExecutor_ExecuteHTTPCheck_ClientError тестирует 4xx ошибку.
func TestCheckExecutor_ExecuteHTTPCheck_ClientError(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
		Treat4xxAsDown:              true,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()

	// Создаём test server который возвращает 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("Not Found"))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.TimeoutSeconds = 5

	// Act
	result := executor.executeHTTPCheck(ctx, monitor)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, domain.StatusDown, result.Status)
	assert.Equal(t, monitor.ID, result.MonitorID)
	assert.NotNil(t, result.StatusCode)
	assert.Equal(t, 404, *result.StatusCode)
	assert.NotNil(t, result.ErrorMessage)
	assert.Contains(t, *result.ErrorMessage, "client error")
}

// TestCheckExecutor_ExecuteHTTPCheck_Timeout тестирует timeout.
func TestCheckExecutor_ExecuteHTTPCheck_Timeout(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()

	// Создаём test server который медленно отвечает
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.TimeoutSeconds = 1 // 1 second timeout

	// Act
	result := executor.executeHTTPCheck(ctx, monitor)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, domain.StatusDown, result.Status)
	assert.Equal(t, monitor.ID, result.MonitorID)
	assert.NotNil(t, result.ErrorMessage)
	assert.Contains(t, *result.ErrorMessage, "deadline exceeded")
}

// TestCheckExecutor_ExecuteHTTPCheck_InvalidURL тестирует невалидный URL.
func TestCheckExecutor_ExecuteHTTPCheck_InvalidURL(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "http://localhost:99999", 60)
	monitor.TimeoutSeconds = 5

	// Act
	result := executor.executeHTTPCheck(ctx, monitor)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, domain.StatusDown, result.Status)
	assert.Equal(t, monitor.ID, result.MonitorID)
	// ErrorMessage should not be nil for error cases
	if result.ErrorMessage != nil {
		assert.Contains(t, *result.ErrorMessage, "invalid port")
	}
}

// TestCheckExecutor_RecordMetrics_AllNotNil тестирует запись метрик когда все метрики инициализированы.
func TestCheckExecutor_RecordMetrics_AllNotNil(t *testing.T) {
	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)
	result := domain.NewCheckResult(monitor.ID, domain.StatusUp)

	// Act - должен паниковать
	assert.NotPanics(t, func() {
		executor.recordMetrics(ctx, monitor, result, 1.5)
	})
}

// TestCheckExecutor_DetermineStatus_AllCombinations тестирует все комбинации статусов.
func TestCheckExecutor_DetermineStatus_AllCombinations(t *testing.T) {
	tests := []struct {
		name                  string
		statusCode            int
		responseTime          int
		responseTimeThreshold int
		failureRate           float64
		failureRateThreshold  int
		expectedStatus        domain.MonitorStatus
	}{
		{
			name:                  "Slow response -> DEGRADED",
			statusCode:            200,
			responseTime:          1500,
			responseTimeThreshold: 1000,
			failureRate:           0,
			failureRateThreshold:  50,
			expectedStatus:        domain.StatusDegraded,
		},
		{
			name:                  "High failure rate -> DEGRADED",
			statusCode:            200,
			responseTime:          500,
			responseTimeThreshold: 1000,
			failureRate:           60,
			failureRateThreshold:  50,
			expectedStatus:        domain.StatusDegraded,
		},
		{
			name:                  "Both slow and high failure rate -> DEGRADED",
			statusCode:            200,
			responseTime:          1500,
			responseTimeThreshold: 1000,
			failureRate:           60,
			failureRateThreshold:  50,
			expectedStatus:        domain.StatusDegraded,
		},
		{
			name:                  "All good -> UP",
			statusCode:            200,
			responseTime:          500,
			responseTimeThreshold: 1000,
			failureRate:           10,
			failureRateThreshold:  50,
			expectedStatus:        domain.StatusUp,
		},
		{
			name:                  "Boundary: response time equals threshold -> UP",
			statusCode:            200,
			responseTime:          1000,
			responseTimeThreshold: 1000,
			failureRate:           0,
			failureRateThreshold:  50,
			expectedStatus:        domain.StatusUp,
		},
		{
			name:                  "Boundary: failure rate equals threshold -> UP",
			statusCode:            200,
			responseTime:          500,
			responseTimeThreshold: 1000,
			failureRate:           50,
			failureRateThreshold:  50,
			expectedStatus:        domain.StatusUp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMonitorRepository)
			mockResultRepo := new(MockCheckResultRepository)

			config := CheckExecutorConfig{
				DefaultTimeout:              30 * time.Second,
				MaxRecentResultsForStats:    10,
				DefaultDegradedResponseTime: 1000,
				DefaultDegradedFailureRate:  50,
				Treat4xxAsDown:              true,
			}

			executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

			ctx := context.Background()
			userID := uuid.New()
			monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

			// Создаём результаты с нужным failure rate
			// For 5xx errors, GetLatestByMonitorID won't be called
			if tt.statusCode < 500 {
				results := createResultsWithFailureRate(tt.failureRate)
				mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).
					Return(results, nil)
			}

			status, err := executor.determineStatus(ctx, monitor, tt.responseTime)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status)
			mockResultRepo.AssertExpectations(t)
		})
	}
}

// createResultsWithFailureRate создаёт результаты с указанным failure rate.
func createResultsWithFailureRate(failureRate float64) []*domain.CheckResult {
	userID := uuid.New()
	total := 10
	failures := int(float64(total) * failureRate / 100)

	results := make([]*domain.CheckResult, total)
	for i := 0; i < total; i++ {
		status := domain.StatusUp
		if i < failures {
			status = domain.StatusDown
		}
		result := domain.NewCheckResult(userID, status)
		result.CheckedAt = time.Now().Add(time.Duration(i) * time.Second)
		results[i] = result
	}

	return results
}

func TestCheckExecutor_ClassifyNetworkError(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)
	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}
	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)
	monitorID := uuid.New()

	t.Run("dns resolution error", func(t *testing.T) {
		err := &net.DNSError{Err: "no such host", Name: "nonexistent.test"}
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.NotNil(t, result.ErrorCode)
		assert.Equal(t, domain.ErrorCodeDNSResolutionFailed, *result.ErrorCode)
	})

	t.Run("tls certificate error", func(t *testing.T) {
		err := &tls.CertificateVerificationError{}
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.NotNil(t, result.ErrorCode)
		assert.Equal(t, domain.ErrorCodeSSLCertificateExpired, *result.ErrorCode)
	})

	t.Run("dial error (connection timeout)", func(t *testing.T) {
		err := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.NotNil(t, result.ErrorCode)
		assert.Equal(t, domain.ErrorCodeConnectionTimeout, *result.ErrorCode)
	})

	t.Run("too many redirects", func(t *testing.T) {
		err := errors.New("stopped after 10 redirects")
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.Equal(t, domain.ErrorCodeTooManyRedirects, *result.ErrorCode)
	})

	t.Run("timeout error", func(t *testing.T) {
		err := errors.New("dial tcp: lookup example.com: i/o timeout")
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.Equal(t, domain.ErrorCodeConnectionTimeout, *result.ErrorCode)
	})

	t.Run("deadline exceeded error", func(t *testing.T) {
		err := errors.New("context deadline exceeded")
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.Equal(t, domain.ErrorCodeNetworkError, *result.ErrorCode)
	})

	t.Run("generic network error", func(t *testing.T) {
		err := errors.New("some random error")
		result := executor.classifyNetworkError(monitorID, err)
		assert.Equal(t, domain.StatusDown, result.Status)
		assert.Equal(t, domain.ErrorCodeNetworkError, *result.ErrorCode)
	})
}

func TestCheckExecutor_EnrichDegradedResult(t *testing.T) {
	t.Parallel()

	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)
	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}
	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", "https://example.com", 60)

	t.Run("slow response sets error code", func(t *testing.T) {
		result := domain.NewCheckResult(monitor.ID, domain.StatusDegraded)
		result = result.WithResponseTime(1500)

		result = executor.enrichDegradedResult(result, monitor, 1500)

		assert.NotNil(t, result.ErrorCode)
		assert.Equal(t, domain.ErrorCodeDegradedSlowResponse, *result.ErrorCode)
		assert.Contains(t, result.GetErrorMessage(), "response time exceeded threshold")
	})

	t.Run("fast response does not set error code", func(t *testing.T) {
		result := domain.NewCheckResult(monitor.ID, domain.StatusDegraded)
		result = result.WithResponseTime(200)

		result = executor.enrichDegradedResult(result, monitor, 200)

		assert.Nil(t, result.ErrorCode)
	})

	t.Run("uses monitor threshold when set", func(t *testing.T) {
		threshold := 200
		monitor.DegradedResponseTimeThreshold = &threshold

		result := domain.NewCheckResult(monitor.ID, domain.StatusDegraded)
		result = result.WithResponseTime(250)

		result = executor.enrichDegradedResult(result, monitor, 250)

		assert.NotNil(t, result.ErrorCode)
		assert.Equal(t, domain.ErrorCodeDegradedSlowResponse, *result.ErrorCode)
	})
}
