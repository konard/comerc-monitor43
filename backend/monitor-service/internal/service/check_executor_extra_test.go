package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
)

// TestCheckExecutor_ExecuteCheck_WithAuditRepo_Success тестирует что recordAuditLog вызывается при успешном check.
func TestCheckExecutor_ExecuteCheck_WithAuditRepo_Success(t *testing.T) {
	t.Parallel()

	// Создаём тест-сервер вместо реального HTTP запроса
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)
	mockAuditRepo := new(MockAuditRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
		MaxRetries:                  1,
		RetryBaseDelay:              1 * time.Millisecond,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, mockAuditRepo, config)

	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp
	monitor.TimeoutSeconds = 5

	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	// Audit log должен вызываться после успешного check
	mockAuditRepo.On("Create", mock.Anything, mock.AnythingOfType("*interfaces.AuditLogEntry")).Return(nil)

	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Проверяем что audit был вызван
	mockAuditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*interfaces.AuditLogEntry"))
}

// TestCheckExecutor_ExecuteCheck_WithAuditRepo_AuditError тестирует что ошибка audit log не прерывает check.
func TestCheckExecutor_ExecuteCheck_WithAuditRepo_AuditError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)
	mockAuditRepo := new(MockAuditRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
		MaxRetries:                  1,
		RetryBaseDelay:              1 * time.Millisecond,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, mockAuditRepo, config)

	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp
	monitor.TimeoutSeconds = 5

	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	// Audit log отдаёт ошибку, но check должен пройти успешно
	mockAuditRepo.On("Create", mock.Anything, mock.AnythingOfType("*interfaces.AuditLogEntry")).Return(errors.New("audit error"))

	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	// Check должен быть успешным даже если audit log сломан
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestCheckExecutor_ExecuteCheck_WithAuditRepo_FailedCheck тестирует recordAuditLog для упавшего check (ActionCheckFailed).
func TestCheckExecutor_ExecuteCheck_WithAuditRepo_FailedCheck(t *testing.T) {
	t.Parallel()

	// Сервер возвращает 500 — DOWN статус
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)
	mockAuditRepo := new(MockAuditRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
		MaxRetries:                  1,
		RetryBaseDelay:              1 * time.Millisecond,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, mockAuditRepo, config)

	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp
	monitor.TimeoutSeconds = 5

	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	// Ожидаем вызов audit с ActionCheckFailed
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(entry *interfaces.AuditLogEntry) bool {
		return entry.Action == interfaces.ActionCheckFailed
	})).Return(nil)

	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.StatusDown, result.Status)
	mockAuditRepo.AssertExpectations(t)
}

// TestCheckExecutor_ExecuteCheck_WithAuditRepo_WithErrorCode тестирует recordAuditLog с error code.
func TestCheckExecutor_ExecuteCheck_WithAuditRepo_WithErrorCode(t *testing.T) {
	t.Parallel()

	// Сервер возвращает 500 с ошибкой — генерирует error code в result
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("server error"))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	mockRepo := new(MockMonitorRepository)
	mockResultRepo := new(MockCheckResultRepository)
	mockAuditRepo := new(MockAuditRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              30 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
		MaxRetries:                  1,
		RetryBaseDelay:              1 * time.Millisecond,
	}

	executor := NewCheckExecutor(mockRepo, mockResultRepo, nil, mockAuditRepo, config)

	ctx := context.Background()
	userID := uuid.New()
	monitor := mustNewMonitor(t, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp
	monitor.TimeoutSeconds = 5

	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAuditRepo.On("Create", mock.Anything, mock.AnythingOfType("*interfaces.AuditLogEntry")).Return(nil)

	result, err := executor.ExecuteCheck(ctx, monitor.ID.String())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockAuditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*interfaces.AuditLogEntry"))
}
