package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// TestUptimeCalculator_GetMonitorHistory_InvalidUserID тестирует невалидный userID.
func TestUptimeCalculator_GetMonitorHistory_InvalidUserID(t *testing.T) {
	t.Parallel()

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)
	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	req := &dto.GetMonitorHistoryRequest{
		MonitorID: uuid.New().String(),
		UserID:    "not-a-uuid",
		From:      time.Now().Add(-1 * time.Hour),
		To:        time.Now(),
	}

	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid user_id")
}

// TestUptimeCalculator_GetMonitorHistory_InvalidDateRange тестирует From > To.
func TestUptimeCalculator_GetMonitorHistory_InvalidDateRange(t *testing.T) {
	t.Parallel()

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)
	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	req := &dto.GetMonitorHistoryRequest{
		MonitorID: uuid.New().String(),
		UserID:    uuid.New().String(),
		From:      time.Now(),
		To:        time.Now().Add(-1 * time.Hour), // To перед From
	}

	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid date range")
}

// TestUptimeCalculator_GetMonitorHistory_MonitorRepoError тестирует ошибку monitorRepo.
func TestUptimeCalculator_GetMonitorHistory_MonitorRepoError(t *testing.T) {
	t.Parallel()

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)
	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	monitorID := uuid.New()
	userID := uuid.New()

	req := &dto.GetMonitorHistoryRequest{
		MonitorID: monitorID.String(),
		UserID:    userID.String(),
		From:      time.Now().Add(-1 * time.Hour),
		To:        time.Now(),
	}

	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return((*domain.Monitor)(nil), assert.AnError)

	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get monitor")
	mockMonitorRepo.AssertExpectations(t)
}

// TestUptimeCalculator_GetMonitorHistory_StatusFilter тестирует фильтрацию по статусу.
func TestUptimeCalculator_GetMonitorHistory_StatusFilter(t *testing.T) {
	t.Parallel()

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
		Status:    "UP",
		Limit:     10,
		Offset:    0,
	}

	monitor := &domain.Monitor{ID: monitorID, UserID: userID}
	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	results := []*domain.CheckResult{
		domain.NewCheckResult(monitorID, domain.StatusUp),
	}
	mockCheckResultRepo.On("GetByMonitorIDAndPeriodAndStatus", mock.Anything, monitorID, from, to, domain.MonitorStatus("UP"), 10, 0).Return(results, nil)
	mockCheckResultRepo.On("CountByMonitorID", mock.Anything, monitorID).Return(int64(1), nil)

	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
	mockMonitorRepo.AssertExpectations(t)
	mockCheckResultRepo.AssertExpectations(t)
}

// TestUptimeCalculator_GetMonitorHistory_InvalidStatus тестирует невалидный статус фильтра.
func TestUptimeCalculator_GetMonitorHistory_InvalidStatus(t *testing.T) {
	t.Parallel()

	mockMonitorRepo := new(MockMonitorRepository)
	mockCheckResultRepo := new(MockCheckResultRepository)
	mockIncidentRepo := new(MockIncidentRepository)
	calculator := NewUptimeCalculator(mockMonitorRepo, mockCheckResultRepo, mockIncidentRepo)

	userID := uuid.New()
	monitorID := uuid.New()

	req := &dto.GetMonitorHistoryRequest{
		MonitorID: monitorID.String(),
		UserID:    userID.String(),
		From:      time.Now().Add(-1 * time.Hour),
		To:        time.Now(),
		Status:    "INVALID_STATUS",
	}

	monitor := &domain.Monitor{ID: monitorID, UserID: userID}
	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)

	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid status filter")
	mockMonitorRepo.AssertExpectations(t)
}

// TestUptimeCalculator_GetMonitorHistory_StatusFilterError тестирует ошибку при фильтрации.
func TestUptimeCalculator_GetMonitorHistory_StatusFilterError(t *testing.T) {
	t.Parallel()

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
		Status:    "DOWN",
		Limit:     10,
		Offset:    0,
	}

	monitor := &domain.Monitor{ID: monitorID, UserID: userID}
	mockMonitorRepo.On("GetByID", mock.Anything, monitorID).Return(monitor, nil)
	mockCheckResultRepo.On("GetByMonitorIDAndPeriodAndStatus", mock.Anything, monitorID, from, to, domain.MonitorStatus("DOWN"), 10, 0).Return(([]*domain.CheckResult)(nil), assert.AnError)

	resp, err := calculator.GetMonitorHistory(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get check results")
	mockMonitorRepo.AssertExpectations(t)
	mockCheckResultRepo.AssertExpectations(t)
}
