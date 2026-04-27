package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// MockCheckResultRepository является mock для CheckResultRepository.
type MockCheckResultRepository struct {
	mock.Mock
}

func (m *MockCheckResultRepository) Create(ctx context.Context, result *domain.CheckResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockCheckResultRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CheckResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CheckResult), args.Error(1)
}

func (m *MockCheckResultRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*domain.CheckResult, error) {
	args := m.Called(ctx, monitorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.CheckResult), args.Error(1)
}

func (m *MockCheckResultRepository) GetByMonitorIDAndPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.CheckResult, error) {
	args := m.Called(ctx, monitorID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.CheckResult), args.Error(1)
}

func (m *MockCheckResultRepository) GetByMonitorIDAndPeriodAndStatus(ctx context.Context, monitorID uuid.UUID, from, to time.Time, status domain.MonitorStatus, limit, offset int) ([]*domain.CheckResult, error) {
	args := m.Called(ctx, monitorID, from, to, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.CheckResult), args.Error(1)
}

func (m *MockCheckResultRepository) GetByMonitorIDAndPeriodPaginated(ctx context.Context, monitorID uuid.UUID, from, to time.Time, limit, offset int) ([]*domain.CheckResult, error) {
	args := m.Called(ctx, monitorID, from, to, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.CheckResult), args.Error(1)
}

func (m *MockCheckResultRepository) GetLatestByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.CheckResult, error) {
	args := m.Called(ctx, monitorID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.CheckResult), args.Error(1)
}

func (m *MockCheckResultRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	args := m.Called(ctx, olderThan)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockCheckResultRepository) CountByMonitorID(ctx context.Context, monitorID uuid.UUID) (int64, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), args.Error(1)
}

// MockIncidentRepository является mock для IncidentRepository.
type MockIncidentRepository struct {
	mock.Mock
}

func (m *MockIncidentRepository) Create(ctx context.Context, incident *domain.Incident) error {
	args := m.Called(ctx, incident)
	return args.Error(0)
}

func (m *MockIncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Incident), args.Error(1)
}

func (m *MockIncidentRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*domain.Incident, error) {
	args := m.Called(ctx, monitorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Incident), args.Error(1)
}

func (m *MockIncidentRepository) GetByMonitorIDAndPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Incident, error) {
	args := m.Called(ctx, monitorID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Incident), args.Error(1)
}

func (m *MockIncidentRepository) GetActiveByMonitorID(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Incident), args.Error(1)
}

func (m *MockIncidentRepository) Update(ctx context.Context, incident *domain.Incident) error {
	args := m.Called(ctx, incident)
	return args.Error(0)
}

func (m *MockIncidentRepository) Resolve(ctx context.Context, id uuid.UUID, endTime time.Time) error {
	args := m.Called(ctx, id, endTime)
	return args.Error(0)
}

func (m *MockIncidentRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	args := m.Called(ctx, olderThan)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockIncidentRepository) CountByMonitorID(ctx context.Context, monitorID uuid.UUID) (int64, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), args.Error(1)
}

// MockEventPublisher является mock для EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishStatusChange(ctx context.Context, event *StatusChangeEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockMonitorService является mock для MonitorService.
type MockMonitorService struct {
	mock.Mock
}

func (m *MockMonitorService) CreateMonitor(ctx context.Context, req *dto.CreateMonitorRequest) (*dto.MonitorResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MonitorResponse), args.Error(1)
}

func (m *MockMonitorService) GetMonitor(ctx context.Context, id, userID string) (*dto.MonitorResponse, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MonitorResponse), args.Error(1)
}

func (m *MockMonitorService) ListMonitors(ctx context.Context, req *dto.ListMonitorsRequest) (*dto.ListMonitorsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ListMonitorsResponse), args.Error(1)
}

func (m *MockMonitorService) UpdateMonitor(ctx context.Context, req *dto.UpdateMonitorRequest) (*dto.MonitorResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MonitorResponse), args.Error(1)
}

func (m *MockMonitorService) DeleteMonitor(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMonitorService) PauseMonitor(ctx context.Context, req *dto.PauseMonitorRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockMonitorService) ResumeMonitor(ctx context.Context, req *dto.ResumeMonitorRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// MockUptimeCalculator является mock для UptimeCalculator.
type MockUptimeCalculator struct {
	mock.Mock
}

func (m *MockUptimeCalculator) CalculateUptime(ctx context.Context, req *dto.GetUptimeStatsRequest) (*dto.UptimeStatsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UptimeStatsResponse), args.Error(1)
}

func (m *MockUptimeCalculator) GetMonitorHistory(ctx context.Context, req *dto.GetMonitorHistoryRequest) (*dto.GetMonitorHistoryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetMonitorHistoryResponse), args.Error(1)
}

func (m *MockUptimeCalculator) GetCheckResults(ctx context.Context, req *dto.GetCheckResultsRequest) (*dto.GetCheckResultsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetCheckResultsResponse), args.Error(1)
}

// MockIncidentDetector является mock для IncidentDetector.
type MockIncidentDetector struct {
	mock.Mock
}

func (m *MockIncidentDetector) GetIncidents(ctx context.Context, req *dto.GetIncidentsRequest) (*dto.GetIncidentsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetIncidentsResponse), args.Error(1)
}
