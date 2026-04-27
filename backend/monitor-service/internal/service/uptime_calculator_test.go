package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestNewUptimeCalculator тестирует создание UptimeCalculator.
func TestNewUptimeCalculator(t *testing.T) {
	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)

	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	assert.NotNil(t, calculator)
	assert.Equal(t, mockMonitorRepo, calculator.monitorRepo)
	assert.Equal(t, mockCheckResultRepo, calculator.checkResultRepo)
	assert.Equal(t, mockIncidentRepo, calculator.incidentRepo)
}

// TestUptimeCalculator_CalculateUptime тестирует расчет uptime.
func TestUptimeCalculator_CalculateUptime(t *testing.T) {
	t.Run("calculates uptime with all UP checks", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetUptimeStatsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		// Mock: получаем монитор
		monitor := &domain.Monitor{
			ID:     monitorID,
			UserID: userID,
		}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		// Mock: получаем результаты проверок (все UP)
		results := make([]*domain.CheckResult, 10)
		for i := 0; i < 10; i++ {
			result := domain.NewCheckResult(monitorID, domain.StatusUp)
			result = result.WithResponseTime(100)
			results[i] = result
		}
		mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

		// Mock: пустой список инцидентов
		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{}, nil)

		// Выполняем
		resp, err := calculator.CalculateUptime(context.Background(), req)

		// Проверяем
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 100.0, resp.Uptime, "Uptime should be 100% for all UP checks")
		assert.Equal(t, 10, resp.TotalChecks)
		assert.Equal(t, 10, resp.UpChecks)
		assert.Equal(t, 0, resp.DownChecks)
		assert.Equal(t, 0, resp.PausedChecks)

		mockMonitorRepo.AssertExpectations(t)
		mockCheckResultRepo.AssertExpectations(t)
		mockIncidentRepo.AssertExpectations(t)
	})

	t.Run("calculates uptime with PAUSED checks excluded", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetUptimeStatsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		// Mock: получаем монитор
		monitor := &domain.Monitor{
			ID:     monitorID,
			UserID: userID,
		}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		// Mock: 5 UP + 3 PAUSED + 2 DOWN
		// PAUSED должны быть исключены из расчета
		// Total active = 7 (5 UP + 2 DOWN)
		// Uptime = 5/7 = 71.4%
		results := make([]*domain.CheckResult, 10)
		for i := 0; i < 5; i++ {
			results[i] = domain.NewCheckResult(monitorID, domain.StatusUp)
		}
		for i := 5; i < 8; i++ {
			results[i] = domain.NewCheckResult(monitorID, domain.StatusPaused)
		}
		for i := 8; i < 10; i++ {
			results[i] = domain.NewCheckResult(monitorID, domain.StatusDown)
		}
		mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{}, nil)

		// Выполняем
		resp, err := calculator.CalculateUptime(context.Background(), req)

		// Проверяем
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		// Uptime = (5 UP + 0 DEGRADED*0.5) / 7 active checks = 5/7 ≈ 71.4%
		assert.InDelta(t, 71.4, resp.Uptime, 0.1)
		assert.Equal(t, 7, resp.TotalChecks, "Total checks should exclude PAUSED (10 total - 3 paused = 7 active)")
		assert.Equal(t, 5, resp.UpChecks)
		assert.Equal(t, 3, resp.PausedChecks, "PAUSED checks should be counted separately")
		assert.Equal(t, 2, resp.DownChecks)
	})

	t.Run("calculates uptime with DEGRADED checks (50% weight)", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetUptimeStatsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		// 6 UP + 4 DEGRADED
		// Uptime = (6 + 4*0.5) / 10 = (6 + 2) / 10 = 8/10 = 80%
		results := make([]*domain.CheckResult, 10)
		for i := 0; i < 6; i++ {
			results[i] = domain.NewCheckResult(monitorID, domain.StatusUp)
		}
		for i := 6; i < 10; i++ {
			results[i] = domain.NewCheckResult(monitorID, domain.StatusDegraded)
		}
		mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{}, nil)

		resp, err := calculator.CalculateUptime(context.Background(), req)

		assert.NoError(t, err)
		assert.InDelta(t, 80.0, resp.Uptime, 0.1)
		assert.Equal(t, 6, resp.UpChecks)
		assert.Equal(t, 4, resp.DegradedChecks)
	})

	t.Run("returns 100% with note when no active checks", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetUptimeStatsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		// Только PAUSED проверки
		results := []*domain.CheckResult{
			domain.NewCheckResult(monitorID, domain.StatusPaused),
			domain.NewCheckResult(monitorID, domain.StatusPaused),
		}
		mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

		mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{}, nil)

		resp, err := calculator.CalculateUptime(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, 100.0, resp.Uptime, "Uptime should be 100% when no active checks")
		assert.NotEmpty(t, resp.Note, "Should have a note about no active checks")
		assert.Equal(t, 2, resp.PausedChecks)
	})

	t.Run("returns error for invalid monitor_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		req := &dto.GetUptimeStatsRequest{
			MonitorID: "invalid-uuid",
			UserID:    uuid.New().String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		resp, err := calculator.CalculateUptime(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("returns error for invalid user_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		req := &dto.GetUptimeStatsRequest{
			MonitorID: uuid.New().String(),
			UserID:    "invalid-uuid",
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		resp, err := calculator.CalculateUptime(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("returns error when monitor not found", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		monitorID := uuid.New()
		userID := uuid.New()

		req := &dto.GetUptimeStatsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(nil, interfaces.ErrMonitorNotFound)

		resp, err := calculator.CalculateUptime(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to get monitor")
	})

	t.Run("returns error when user does not own monitor", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		monitorID := uuid.New()
		ownerID := uuid.New()
		requesterID := uuid.New()

		req := &dto.GetUptimeStatsRequest{
			MonitorID: monitorID.String(),
			UserID:    requesterID.String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: ownerID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		resp, err := calculator.CalculateUptime(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "monitor not found")
	})
}

// TestUptimeCalculator_GetMonitorHistory тестирует получение истории проверок.
func TestUptimeCalculator_GetMonitorHistory(t *testing.T) {
	t.Run("returns monitor history successfully", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetMonitorHistoryRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
			Limit:     10,
			Offset:    0,
		}

		// Mock: получаем монитор
		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		// Mock: получаем результаты
		results := make([]*domain.CheckResult, 5)
		for i := 0; i < 5; i++ {
			result := domain.NewCheckResult(monitorID, domain.StatusUp)
			result = result.WithResponseTime(100 + i*10)
			results[i] = result
		}
		mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

		// Mock: общее количество
		mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(5), nil)

		// Выполняем
		resp, err := calculator.GetMonitorHistory(context.Background(), req)

		// Проверяем
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 5)
		assert.Equal(t, 5, resp.Total)
		assert.Equal(t, 10, resp.Limit)
		assert.Equal(t, 0, resp.Offset)

		// Проверяем первый результат
		firstResult := resp.Results[0]
		assert.Equal(t, monitorID.String(), firstResult.MonitorID)
		assert.Equal(t, "UP", firstResult.Status)
		assert.NotNil(t, firstResult.ResponseTimeMs)
		assert.Equal(t, 100, *firstResult.ResponseTimeMs)

		mockMonitorRepo.AssertExpectations(t)
		mockCheckResultRepo.AssertExpectations(t)
	})

	t.Run("returns error for invalid monitor_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		req := &dto.GetMonitorHistoryRequest{
			MonitorID: "invalid-uuid",
			UserID:    uuid.New().String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		resp, err := calculator.GetMonitorHistory(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("returns empty history when no results", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()

		req := &dto.GetMonitorHistoryRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, mock.Anything, mock.Anything).Return([]*domain.CheckResult{}, nil)
		mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), nil)

		resp, err := calculator.GetMonitorHistory(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Results)
		assert.Equal(t, 0, resp.Total)
	})
}

// TestUptimeCalculator_CalculateWithIncidents тестирует расчет uptime с инцидентами.
func TestUptimeCalculator_CalculateWithIncidents(t *testing.T) {
	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)

	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	userID := uuid.New()
	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	req := &dto.GetUptimeStatsRequest{
		MonitorID: monitorID.String(),
		UserID:    userID.String(),
		From:      from,
		To:        to,
	}

	monitor := &domain.Monitor{ID: monitorID, UserID: userID}
	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	// 8 UP + 2 DOWN
	results := make([]*domain.CheckResult, 10)
	for i := 0; i < 8; i++ {
		results[i] = domain.NewCheckResult(monitorID, domain.StatusUp)
	}
	for i := 8; i < 10; i++ {
		results[i] = domain.NewCheckResult(monitorID, domain.StatusDown)
	}
	mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

	// Создаем инцидент
	incident := domain.NewIncident(monitorID)
	incident.StartTime = from.Add(12 * time.Hour)
	endTime := to.Add(-6 * time.Hour)
	incident.EndTime = &endTime
	mockIncidentRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return([]*domain.Incident{incident}, nil)

	resp, err := calculator.CalculateUptime(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, resp.Incidents, "Should have 1 incident")
	assert.Equal(t, 80.0, resp.Uptime, "Uptime should be 80% (8 UP out of 10)")
	// Note: TotalDowntime calculation depends on domain.UptimeStats implementation
	assert.True(t, resp.TotalDowntime >= 0, "TotalDowntime should be non-negative")
}

// TestUptimeCalculator_GetMonitorHistory_UnauthorizedUser тестирует доступ к чужому монитору.
func TestUptimeCalculator_GetMonitorHistory_UnauthorizedUser(t *testing.T) {
	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)

	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	ownerID := uuid.New()
	requesterID := uuid.New()
	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	req := &dto.GetMonitorHistoryRequest{
		MonitorID: monitorID.String(),
		UserID:    requesterID.String(),
		From:      from,
		To:        to,
	}

	// Mock: монитор принадлежит другому пользователю
	monitor := &domain.Monitor{ID: monitorID, UserID: ownerID}
	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	// Выполняем
	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	// Проверяем
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "monitor not found")
	mockMonitorRepo.AssertExpectations(t)
}

// TestUptimeCalculator_GetMonitorHistory_CountError тестирует ошибку при подсчёте результатов.
func TestUptimeCalculator_GetMonitorHistory_CountError(t *testing.T) {
	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)

	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	userID := uuid.New()
	monitorID := uuid.New()
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	req := &dto.GetMonitorHistoryRequest{
		MonitorID: monitorID.String(),
		UserID:    userID.String(),
		From:      from,
		To:        to,
	}

	// Mock: получаем монитор
	monitor := &domain.Monitor{ID: monitorID, UserID: userID}
	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	// Mock: получаем результаты
	results := make([]*domain.CheckResult, 5)
	for i := 0; i < 5; i++ {
		result := domain.NewCheckResult(monitorID, domain.StatusUp)
		results[i] = result
	}
	mockCheckResultRepo.On("GetByMonitorIDAndPeriod", mock.Anything, monitorID, from, to).Return(results, nil)

	// Mock: ошибка при подсчёте
	mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), assert.AnError)

	// Выполняем
	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	// Проверяем
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to count check results")
	mockMonitorRepo.AssertExpectations(t)
	mockCheckResultRepo.AssertExpectations(t)
}

func TestValidateDateRange(t *testing.T) {
	t.Parallel()

	t.Run("valid range", func(t *testing.T) {
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()
		assert.NoError(t, validateDateRange(from, to))
	})

	t.Run("from after to", func(t *testing.T) {
		from := time.Now()
		to := time.Now().Add(-24 * time.Hour)
		err := validateDateRange(from, to)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from must be before to")
	})

	t.Run("zero values", func(t *testing.T) {
		assert.NoError(t, validateDateRange(time.Time{}, time.Time{}))
	})

	t.Run("from zero to set", func(t *testing.T) {
		assert.NoError(t, validateDateRange(time.Time{}, time.Now()))
	})

	t.Run("from set to zero", func(t *testing.T) {
		assert.NoError(t, validateDateRange(time.Now(), time.Time{}))
	})
}

// TestUptimeCalculator_GetCheckResults тестирует получение raw check results.
func TestUptimeCalculator_GetCheckResults(t *testing.T) {
	t.Run("returns check results with pagination", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetCheckResultsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
			Limit:     10,
			Offset:    0,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		rtMs := 150
		statusCode := 200
		results := []*domain.CheckResult{
			domain.NewCheckResult(monitorID, domain.StatusUp),
			domain.NewCheckResult(monitorID, domain.StatusDown),
		}
		results[0].ResponseTimeMs = &rtMs
		results[0].StatusCode = &statusCode

		mockCheckResultRepo.On("GetByMonitorIDAndPeriodPaginated", mock.Anything, monitorID, from, to, 10, 0).Return(results, nil)
		mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(42), nil)

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 2)
		assert.Equal(t, 42, resp.Total)
		assert.Equal(t, 10, resp.Limit)
		assert.Equal(t, 0, resp.Offset)

		mockMonitorRepo.AssertExpectations(t)
		mockCheckResultRepo.AssertExpectations(t)
	})

	t.Run("returns error for invalid monitor_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		req := &dto.GetCheckResultsRequest{
			MonitorID: "invalid-uuid",
			UserID:    uuid.New().String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("returns error for invalid user_id", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		req := &dto.GetCheckResultsRequest{
			MonitorID: uuid.New().String(),
			UserID:    "invalid-uuid",
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("returns error when monitor not found", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		monitorID := uuid.New()
		userID := uuid.New()

		req := &dto.GetCheckResultsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(nil, assert.AnError)

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to get monitor")
	})

	t.Run("returns error when monitor belongs to another user", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		monitorID := uuid.New()
		userID := uuid.New()
		otherUserID := uuid.New()

		req := &dto.GetCheckResultsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      time.Now().Add(-24 * time.Hour),
			To:        time.Now(),
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: otherUserID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "monitor not found")
	})

	t.Run("applies default limit when zero", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetCheckResultsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
			Limit:     0,
			Offset:    0,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)
		mockCheckResultRepo.On("GetByMonitorIDAndPeriodPaginated", mock.Anything, monitorID, from, to, 100, 0).Return([]*domain.CheckResult{}, nil)
		mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), nil)

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, 100, resp.Limit)

		mockMonitorRepo.AssertExpectations(t)
		mockCheckResultRepo.AssertExpectations(t)
	})

	t.Run("returns empty results when no data", func(t *testing.T) {
		mockMonitorRepo := new(MockMonitorRepository)
		mockCheckResultRepo := new(MockCheckResultRepository)
		mockIncidentRepo := new(MockIncidentRepository)

		calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

		userID := uuid.New()
		monitorID := uuid.New()
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		req := &dto.GetCheckResultsRequest{
			MonitorID: monitorID.String(),
			UserID:    userID.String(),
			From:      from,
			To:        to,
			Limit:     10,
			Offset:    0,
		}

		monitor := &domain.Monitor{ID: monitorID, UserID: userID}
		mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)
		mockCheckResultRepo.On("GetByMonitorIDAndPeriodPaginated", mock.Anything, monitorID, from, to, 10, 0).Return([]*domain.CheckResult{}, nil)
		mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(0), nil)

		resp, err := calculator.GetCheckResults(context.Background(), req)

		assert.NoError(t, err)
		assert.Empty(t, resp.Results)
		assert.Equal(t, 0, resp.Total)

		mockMonitorRepo.AssertExpectations(t)
		mockCheckResultRepo.AssertExpectations(t)
	})
}
