package throttling

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/config"
	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

type mockAlertRepo struct {
	mock.Mock
}

func (m *mockAlertRepo) Create(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockAlertRepo) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockAlertRepo) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockAlertRepo) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *mockAlertRepo) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*model.Alert), args.Int(1), args.Error(2)
}

func (m *mockAlertRepo) Update(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockAlertRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAlertRepo) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, alert, attempt)
	return args.Error(0)
}

func (m *mockAlertRepo) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *mockAlertRepo) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockAlertRepo) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Get(0).(int), args.Error(1)
}

func (m *mockAlertRepo) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *mockAlertRepo) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	args := m.Called(ctx, alertID, userID)
	return args.Error(0)
}

type mockDeliveryAttemptRepo struct {
	mock.Mock
}

func (m *mockDeliveryAttemptRepo) Create(ctx context.Context, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *mockDeliveryAttemptRepo) GetByID(ctx context.Context, id string) (*model.DeliveryAttempt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DeliveryAttempt), args.Error(1)
}

func (m *mockDeliveryAttemptRepo) List(ctx context.Context, alertID string) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, alertID)
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *mockDeliveryAttemptRepo) ListPending(ctx context.Context, limit int) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *mockDeliveryAttemptRepo) Update(ctx context.Context, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *mockDeliveryAttemptRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockDeliveryAttemptRepo) DeleteOldAttempts(ctx context.Context, olderThan int64) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *mockDeliveryAttemptRepo) CountRecentByMonitorAndChannel(ctx context.Context, monitorID string, channelType string, since time.Duration) (int, error) {
	args := m.Called(ctx, monitorID, channelType, since)
	return args.Int(0), args.Error(1)
}

func (m *mockDeliveryAttemptRepo) GetLastDeliveryTimeForMonitorAndStatus(ctx context.Context, monitorID string, status string) (*time.Time, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func newTestService(alertRepo repository.AlertRepository, deliveryRepo repository.DeliveryAttemptRepository) *AlertThrottlingService {
	tracer := otel.GetTracerProvider().Tracer("test")
	return NewAlertThrottlingService(
		config.AlertThrottlingConfig{
			CooldownPeriod:  15 * time.Minute,
			RateLimitPeriod: 5 * time.Minute,
			StormWindow:     5 * time.Minute,
			StormThreshold:  5,
			StormDuration:   10 * time.Minute,
		},
		alertRepo,
		deliveryRepo,
		tracer,
		nil,
	)
}

func TestCheckThrottle_NoRecentAlerts(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, nil)
	deliveryRepo.On("GetLastDeliveryTimeForMonitorAndStatus", mock.Anything, monitorID.String(), "DOWN").Return(nil, nil)
	alertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, monitorID.String()).Return(nil, nil)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ThrottleActionDeliver, result.Action)
}

func TestCheckThrottle_RateLimited(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	recentDelivery := time.Now().Add(-2 * time.Minute)

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, nil)
	deliveryRepo.On("GetLastDeliveryTimeForMonitorAndStatus", mock.Anything, monitorID.String(), "DOWN").Return(&recentDelivery, nil)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ThrottleActionQueue, result.Action)
	assert.Contains(t, result.Reason, "rate limit")
}

func TestCheckThrottle_Cooldown(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, nil)
	deliveryRepo.On("GetLastDeliveryTimeForMonitorAndStatus", mock.Anything, monitorID.String(), "DOWN").Return(nil, nil)
	alertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, monitorID.String()).Return(&model.Alert{
		CreatedAt: time.Now().Add(-2 * time.Minute),
	}, nil)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ThrottleActionSuppress, result.Action)
	assert.Contains(t, result.Reason, "cooldown")
}

func TestCheckThrottle_StormDetected(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(6, nil)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ThrottleActionStorm, result.Action)
	assert.True(t, result.StormGroup)
	assert.Contains(t, result.Reason, "alert storm")
}

func TestCheckThrottle_StormPriorityOverRateLimit(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(10, nil)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ThrottleActionStorm, result.Action)

	deliveryRepo.AssertNotCalled(t, "GetLastDeliveryTimeForMonitorAndStatus")
	alertRepo.AssertNotCalled(t, "GetLastAlertTimeAnyStatus")
}

func TestCheckThrottle_RateLimitPriorityOverCooldown(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	recentDelivery := time.Now().Add(-1 * time.Minute)

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, nil)
	deliveryRepo.On("GetLastDeliveryTimeForMonitorAndStatus", mock.Anything, monitorID.String(), "DOWN").Return(&recentDelivery, nil)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ThrottleActionQueue, result.Action)

	alertRepo.AssertNotCalled(t, "GetLastAlertTimeAnyStatus")
}

func TestCheckThrottle_StormCheckError(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, assert.AnError)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCheckThrottle_RateLimitCheckError(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, nil)
	deliveryRepo.On("GetLastDeliveryTimeForMonitorAndStatus", mock.Anything, monitorID.String(), "DOWN").Return(nil, assert.AnError)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCheckThrottle_CooldownCheckError(t *testing.T) {
	t.Parallel()

	alertRepo := &mockAlertRepo{}
	deliveryRepo := &mockDeliveryAttemptRepo{}
	service := newTestService(alertRepo, deliveryRepo)

	ctx := context.Background()
	userID := uuid.New()
	monitorID := uuid.New()

	alertRepo.On("CountUniqueMonitorsWithAlertsSince", mock.Anything, mock.AnythingOfType("time.Time")).Return(0, nil)
	deliveryRepo.On("GetLastDeliveryTimeForMonitorAndStatus", mock.Anything, monitorID.String(), "DOWN").Return(nil, nil)
	alertRepo.On("GetLastAlertTimeAnyStatus", mock.Anything, monitorID.String()).Return(nil, assert.AnError)

	result, err := service.CheckThrottle(ctx, userID, monitorID, "DOWN")

	assert.Error(t, err)
	assert.Nil(t, result)
}
