package flapping

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

type mockStatusChangeRepo struct {
	mock.Mock
}

func (m *mockStatusChangeRepo) Create(ctx context.Context, change *model.MonitorStatusChange) error {
	args := m.Called(ctx, change)
	return args.Error(0)
}

func (m *mockStatusChangeRepo) CountInWindow(ctx context.Context, monitorID string, window time.Duration) (int, error) {
	args := m.Called(ctx, monitorID, window)
	return args.Int(0), args.Error(1)
}

func (m *mockStatusChangeRepo) GetLatestByMonitorID(ctx context.Context, monitorID string) (*model.MonitorStatusChange, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MonitorStatusChange), args.Error(1)
}

type mockFlappingAlertRepo struct {
	mock.Mock
}

func (m *mockFlappingAlertRepo) Create(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockFlappingAlertRepo) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockFlappingAlertRepo) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockFlappingAlertRepo) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *mockFlappingAlertRepo) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*model.Alert), args.Int(1), args.Error(2)
}

func (m *mockFlappingAlertRepo) Update(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockFlappingAlertRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockFlappingAlertRepo) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, alert, attempt)
	return args.Error(0)
}

func (m *mockFlappingAlertRepo) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *mockFlappingAlertRepo) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockFlappingAlertRepo) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Get(0).(int), args.Error(1)
}

func (m *mockFlappingAlertRepo) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	args := m.Called(ctx, alertID, userID)
	return args.Error(0)
}

func (m *mockFlappingAlertRepo) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func newTestFlappingService(repo repository.MonitorStatusChangeRepository) *FlappingService {
	tracer := otel.GetTracerProvider().Tracer("test")
	return NewFlappingService(repo, nil, nil, tracer, nil)
}

func newTestFlappingServiceWithAlertRepo(repo repository.MonitorStatusChangeRepository, alertRepo repository.AlertRepository) *FlappingService {
	tracer := otel.GetTracerProvider().Tracer("test")
	return NewFlappingService(repo, alertRepo, nil, tracer, nil)
}

func TestCheckFlapping_NotFlapping(t *testing.T) {
	t.Parallel()

	repo := &mockStatusChangeRepo{}
	service := newTestFlappingService(repo)

	ctx := context.Background()
	monitorID := "mon-123"

	repo.On("CountInWindow", mock.Anything, monitorID, FlappingWindow).Return(3, nil)

	result, err := service.CheckFlapping(ctx, monitorID)

	assert.NoError(t, err)
	assert.False(t, result.IsFlapping)
	assert.Equal(t, 3, result.FlapCount)
	assert.Equal(t, FlappingThreshold, result.Threshold)
}

func TestCheckFlapping_ExactlyAtThreshold(t *testing.T) {
	t.Parallel()

	repo := &mockStatusChangeRepo{}
	service := newTestFlappingService(repo)

	ctx := context.Background()
	monitorID := "mon-123"

	repo.On("CountInWindow", mock.Anything, monitorID, FlappingWindow).Return(FlappingThreshold, nil)

	result, err := service.CheckFlapping(ctx, monitorID)

	assert.NoError(t, err)
	assert.True(t, result.IsFlapping)
	assert.Equal(t, FlappingThreshold, result.FlapCount)
}

func TestCheckFlapping_AboveThreshold(t *testing.T) {
	t.Parallel()

	repo := &mockStatusChangeRepo{}
	service := newTestFlappingService(repo)

	ctx := context.Background()
	monitorID := "mon-123"

	repo.On("CountInWindow", mock.Anything, monitorID, FlappingWindow).Return(12, nil)

	result, err := service.CheckFlapping(ctx, monitorID)

	assert.NoError(t, err)
	assert.True(t, result.IsFlapping)
	assert.Equal(t, 12, result.FlapCount)
}

func TestCheckExitFlapping_CanExit(t *testing.T) {
	t.Parallel()

	repo := &mockStatusChangeRepo{}
	service := newTestFlappingService(repo)

	ctx := context.Background()
	monitorID := "mon-123"

	repo.On("GetLatestByMonitorID", mock.Anything, monitorID).Return(&model.MonitorStatusChange{
		CreatedAt: time.Now().Add(-20 * time.Minute),
	}, nil)

	canExit, err := service.CheckExitFlapping(ctx, monitorID)

	assert.NoError(t, err)
	assert.True(t, canExit)
}

func TestCheckExitFlapping_StayFlapping(t *testing.T) {
	t.Parallel()

	repo := &mockStatusChangeRepo{}
	service := newTestFlappingService(repo)

	ctx := context.Background()
	monitorID := "mon-123"

	repo.On("GetLatestByMonitorID", mock.Anything, monitorID).Return(&model.MonitorStatusChange{
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}, nil)

	canExit, err := service.CheckExitFlapping(ctx, monitorID)

	assert.NoError(t, err)
	assert.False(t, canExit)
}

func TestRecordStatusChange_NoFlapping(t *testing.T) {
	t.Parallel()

	statusRepo := &mockStatusChangeRepo{}
	alertRepo := &mockFlappingAlertRepo{}
	service := newTestFlappingServiceWithAlertRepo(statusRepo, alertRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	statusRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	statusRepo.On("CountInWindow", mock.Anything, monitorID.String(), FlappingWindow).Return(2, nil)

	err := service.RecordStatusChange(ctx, userID, monitorID, "UP", "DOWN")

	assert.NoError(t, err)
	statusRepo.AssertExpectations(t)
}

func TestRecordStatusChange_FlappingDetected(t *testing.T) {
	t.Parallel()

	statusRepo := &mockStatusChangeRepo{}
	alertRepo := &mockFlappingAlertRepo{}
	service := newTestFlappingServiceWithAlertRepo(statusRepo, alertRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	statusRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	statusRepo.On("CountInWindow", mock.Anything, monitorID.String(), FlappingWindow).Return(FlappingThreshold, nil)
	alertRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.RecordStatusChange(ctx, userID, monitorID, "UP", "DOWN")

	assert.NoError(t, err)
	alertRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestRecordStatusChange_CreateError(t *testing.T) {
	t.Parallel()

	statusRepo := &mockStatusChangeRepo{}
	service := newTestFlappingService(statusRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	statusRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	err := service.RecordStatusChange(ctx, userID, monitorID, "UP", "DOWN")

	assert.Error(t, err)
}

func TestRecordStatusChange_CheckFlappingError(t *testing.T) {
	t.Parallel()

	statusRepo := &mockStatusChangeRepo{}
	service := newTestFlappingService(statusRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	statusRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	statusRepo.On("CountInWindow", mock.Anything, monitorID.String(), FlappingWindow).Return(0, assert.AnError)

	err := service.RecordStatusChange(ctx, userID, monitorID, "UP", "DOWN")

	assert.Error(t, err)
}
