package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/reporting-service/internal/model"
)

type MockMonitorClient struct {
	mock.Mock
}

func (m *MockMonitorClient) GetUptimeStats(ctx context.Context, monitorID string, from, to time.Time) (*monitov1.GetUptimeStatsResponse, error) {
	args := m.Called(ctx, monitorID, from, to)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetUptimeStatsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMonitorClient) GetCheckResults(ctx context.Context, monitorID string, from, to time.Time, limit, offset int) (*monitov1.GetCheckResultsResponse, error) {
	args := m.Called(ctx, monitorID, from, to, limit, offset)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetCheckResultsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMonitorClient) GetIncidents(ctx context.Context, monitorID string, from, to time.Time, limit, offset int) (*monitov1.GetIncidentsResponse, error) {
	args := m.Called(ctx, monitorID, from, to, limit, offset)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetIncidentsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMonitorClient) GetMonitor(ctx context.Context, monitorID string) (*monitov1.GetMonitorResponse, error) {
	args := m.Called(ctx, monitorID)
	if r := args.Get(0); r != nil {
		return r.(*monitov1.GetMonitorResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMonitorClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) Create(ctx context.Context, report *model.SLAReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SLAReport, error) {
	args := m.Called(ctx, id)
	if r := args.Get(0); r != nil {
		return r.(*model.SLAReport), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReportRepository) ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*model.SLAReport, int, error) {
	args := m.Called(ctx, monitorID, limit, offset)
	if r := args.Get(0); r != nil {
		return r.([]*model.SLAReport), args.Get(1).(int), args.Error(2)
	}
	total := 0
	if v := args.Get(1); v != nil {
		total = v.(int)
	}
	return nil, total, args.Error(2)
}

func (m *MockReportRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.SLAReport, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	if r := args.Get(0); r != nil {
		return r.([]*model.SLAReport), args.Get(1).(int), args.Error(2)
	}
	total := 0
	if v := args.Get(1); v != nil {
		total = v.(int)
	}
	return nil, total, args.Error(2)
}
