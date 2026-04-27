package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/scheduler-service/internal/model"
	"github.com/raul/monitor/backend/scheduler-service/internal/monitor"
)

func TestCheckScheduler_ReassignOverdueChecks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 5*time.Minute, logger)

	// Создаём просроченную проверку
	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.CheckPriorityNormal, time.Now().Add(-10*time.Minute))
	check.WorkerID = uuid.New()

	worker1 := model.NewWorker("worker-msk-01", "moscow")
	worker2 := model.NewWorker("worker-spb-01", "spb")

	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{check}, nil)
	workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{worker1, worker2}, nil)
	checkRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	count, err := svc.ReassignOverdueChecks(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, model.CheckPriorityHigh, check.Priority)
	checkRepo.AssertExpectations(t)
}

func TestCheckScheduler_ReassignOverdueChecks_no_workers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 5*time.Minute, logger)

	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.CheckPriorityNormal, time.Now().Add(-10*time.Minute))

	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{check}, nil)
	workerRepo.On("ListIdleByZone", mock.Anything, "").Return([]*model.Worker{}, nil)

	count, err := svc.ReassignOverdueChecks(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCheckScheduler_ReassignOverdueChecks_list_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 5*time.Minute, logger)

	checkRepo.On("ListOverdue", mock.Anything, mock.Anything).Return([]*model.ScheduledCheck{}, errors.New("db error"))

	_, err := svc.ReassignOverdueChecks(ctx)

	assert.Error(t, err)
}

func TestCheckScheduler_CompleteCheck_success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 5*time.Minute, logger)

	check := model.NewScheduledCheck(uuid.New(), model.CheckPriorityNormal, time.Now())
	check.Assign(uuid.New())

	checkRepo.On("GetByID", mock.Anything, mock.Anything).Return(check, nil)
	checkRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err := svc.CompleteCheck(ctx, check.ID, false, "")

	require.NoError(t, err)
	assert.Equal(t, model.CheckStatusCompleted, check.Status)
	checkRepo.AssertExpectations(t)
}

func TestCheckScheduler_CompleteCheck_failed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 5*time.Minute, logger)

	check := model.NewScheduledCheck(uuid.New(), model.CheckPriorityNormal, time.Now())
	check.Assign(uuid.New())

	checkRepo.On("GetByID", mock.Anything, mock.Anything).Return(check, nil)
	checkRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err := svc.CompleteCheck(ctx, check.ID, true, "connection timeout")

	require.NoError(t, err)
	assert.Equal(t, model.CheckStatusFailed, check.Status)
	assert.Equal(t, "connection timeout", check.ErrorMessage)
}

func TestCheckScheduler_CompleteCheck_not_found(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 5*time.Minute, logger)

	checkRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	err := svc.CompleteCheck(ctx, uuid.New(), false, "")

	assert.Error(t, err)
}

func TestCheckScheduler_TriggerScheduledCheck_paused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()

	mockMonitorClient := &mockMonitorClient{}
	mockMonitorClient.On("IsMonitorPaused", mock.Anything, mock.Anything).Return(true, nil)

	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, mockMonitorClient, 5*time.Minute, logger)

	monitorID := uuid.New()
	_, err := svc.TriggerScheduledCheck(ctx, monitorID, model.CheckPriorityHigh)

	assert.ErrorIs(t, err, model.ErrCheckAlreadyPending)
}

func TestCheckScheduler_TriggerScheduledCheck_monitor_not_found(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()

	mockMonitorClient := &mockMonitorClient{}
	mockMonitorClient.On("IsMonitorPaused", mock.Anything, mock.Anything).Return(false, errors.New("not found"))

	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, mockMonitorClient, 5*time.Minute, logger)

	monitorID := uuid.New()
	_, err := svc.TriggerScheduledCheck(ctx, monitorID, model.CheckPriorityHigh)

	assert.ErrorIs(t, err, model.ErrMonitorNotFound)
}

// Mock для MonitorClient
type mockMonitorClient struct {
	mock.Mock
}

func (m *mockMonitorClient) IsMonitorPaused(ctx context.Context, monitorID string) (bool, error) {
	args := m.Called(ctx, monitorID)
	if len(args) > 1 {
		return args.Bool(0), args.Error(1)
	}
	// Default implementation
	return false, nil
}

func (m *mockMonitorClient) GetMonitor(ctx context.Context, monitorID string) (*monitor.MonitorInfo, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*monitor.MonitorInfo), args.Error(1)
}

func (m *mockMonitorClient) ListActiveMonitors(ctx context.Context, pageSize int) ([]*monitor.MonitorInfo, error) {
	args := m.Called(ctx, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*monitor.MonitorInfo), args.Error(1)
}

func TestGetScheduledChecks_success(t *testing.T) {
	t.Parallel()

	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 2*time.Minute, logger)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now()
	checks := []*model.ScheduledCheck{
		model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now()),
	}
	checkRepo.On("ListByTimeRange", mock.Anything, from, to, "", 1, 10).Return(checks, 1, nil)

	result, total, err := svc.GetScheduledChecks(context.Background(), from, to, "", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, len(result))
}

func TestGetScheduledChecks_with_status_filter(t *testing.T) {
	t.Parallel()

	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 2*time.Minute, logger)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now()
	checks := []*model.ScheduledCheck{
		model.NewScheduledCheck(uuid.New(), model.PriorityNormal, time.Now()),
	}
	checkRepo.On("ListByTimeRange", mock.Anything, from, to, "pending", 1, 10).Return(checks, 1, nil)

	result, total, err := svc.GetScheduledChecks(context.Background(), from, to, "pending", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, len(result))
}

func TestGetNextCheck_success(t *testing.T) {
	t.Parallel()

	checkRepo := &mockCheckRepo{}
	workerRepo := &mockWorkerRepo{}
	auditRepo := &mockAuditRepo{}
	logger := slog.Default()
	svc := NewCheckScheduler(checkRepo, workerRepo, auditRepo, nil, 2*time.Minute, logger)

	monitorID := uuid.New()
	check := model.NewScheduledCheck(monitorID, model.PriorityNormal, time.Now())
	checkRepo.On("GetByMonitorID", mock.Anything, monitorID).Return(check, nil)

	result, err := svc.GetNextCheck(context.Background(), monitorID)

	require.NoError(t, err)
	assert.Equal(t, monitorID, result.MonitorID)
}
