package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

// BenchmarkCheckExecutor_ExecuteCheck benchmarks executing a check.
func BenchmarkCheckExecutor_ExecuteCheck(b *testing.B) {
	// Create test server that responds quickly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(server.Close)

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              5 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockMonitorRepo, mockCheckResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp

	// Mock
	mockMonitorRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockCheckResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockCheckResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockMonitorRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := executor.ExecuteCheck(context.Background(), monitor.ID.String()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckExecutor_ExecuteCheck_WithResponseTime benchmarks executing a check with response time tracking.
func BenchmarkCheckExecutor_ExecuteCheck_WithResponseTime(b *testing.B) {
	// Create test server with simulated response time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some processing time
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(server.Close)

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              5 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockMonitorRepo, mockCheckResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp

	// Mock
	mockMonitorRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockCheckResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockCheckResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockMonitorRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := executor.ExecuteCheck(context.Background(), monitor.ID.String()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckExecutor_ExecuteCheck_WithHistory benchmarks executing a check with historical data.
func BenchmarkCheckExecutor_ExecuteCheck_WithHistory(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(server.Close)

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              5 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockMonitorRepo, mockCheckResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp

	// Create historical check results
	history := make([]*domain.CheckResult, 10)
	for i := 0; i < 10; i++ {
		result := domain.NewCheckResult(monitor.ID, domain.StatusUp)
		result = result.WithResponseTime(100)
		result = result.WithStatusCode(200)
		history[i] = result
	}

	// Mock
	mockMonitorRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockCheckResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockCheckResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return(history, nil)
	mockMonitorRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := executor.ExecuteCheck(context.Background(), monitor.ID.String()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckExecutor_ExecuteCheck_Degraded benchmarks executing a check that results in DEGRADED status.
func BenchmarkCheckExecutor_ExecuteCheck_Degraded(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(server.Close)

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              5 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockMonitorRepo, mockCheckResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp
	threshold := 2000
	monitor.DegradedResponseTimeThreshold = &threshold

	// Create historical check results with high failure rate
	history := make([]*domain.CheckResult, 10)
	for i := 0; i < 10; i++ {
		result := domain.NewCheckResult(monitor.ID, domain.StatusDown)
		result = result.WithResponseTime(100)
		result = result.WithStatusCode(500)
		history[i] = result
	}

	// Mock
	mockMonitorRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockCheckResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockCheckResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return(history, nil)
	mockMonitorRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := executor.ExecuteCheck(context.Background(), monitor.ID.String()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckExecutor_ExecuteCheck_Down benchmarks executing a check that results in DOWN status.
func BenchmarkCheckExecutor_ExecuteCheck_Down(b *testing.B) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	b.Cleanup(server.Close)

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              5 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockMonitorRepo, mockCheckResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp

	// Mock
	mockMonitorRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockCheckResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockMonitorRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := executor.ExecuteCheck(context.Background(), monitor.ID.String()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckExecutor_ExecuteCheck_Parallel benchmarks executing checks in parallel.
func BenchmarkCheckExecutor_ExecuteCheck_Parallel(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(server.Close)

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)

	config := CheckExecutorConfig{
		DefaultTimeout:              5 * time.Second,
		MaxRecentResultsForStats:    10,
		DefaultDegradedResponseTime: 1000,
		DefaultDegradedFailureRate:  50,
	}

	executor := NewCheckExecutor(mockMonitorRepo, mockCheckResultRepo, nil, nil, config)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", server.URL, 60)
	monitor.Status = domain.StatusUp

	// Mock
	mockMonitorRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockCheckResultRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockCheckResultRepo.On("GetLatestByMonitorID", mock.Anything, monitor.ID, 10).Return([]*domain.CheckResult{}, nil)
	mockMonitorRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := executor.ExecuteCheck(context.Background(), monitor.ID.String()); err != nil {
				b.Fatal(err)
			}
		}
	})
}
