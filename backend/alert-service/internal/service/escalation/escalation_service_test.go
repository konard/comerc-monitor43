package escalation

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

type mockEscalationRepo struct {
	mock.Mock
}

func (m *mockEscalationRepo) Create(ctx context.Context, escalation *model.AlertEscalation) error {
	args := m.Called(ctx, mock.AnythingOfType("*model.AlertEscalation"))
	return args.Error(0)
}

func (m *mockEscalationRepo) GetLatestByAlertID(ctx context.Context, alertID string) (*model.AlertEscalation, error) {
	args := m.Called(ctx, alertID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertEscalation), args.Error(1)
}

type mockEscalationAlertRepo struct {
	mock.Mock
}

func (m *mockEscalationAlertRepo) Create(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockEscalationAlertRepo) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockEscalationAlertRepo) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockEscalationAlertRepo) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *mockEscalationAlertRepo) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*model.Alert), args.Int(1), args.Error(2)
}

func (m *mockEscalationAlertRepo) Update(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockEscalationAlertRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockEscalationAlertRepo) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, alert, attempt)
	return args.Error(0)
}

func (m *mockEscalationAlertRepo) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func (m *mockEscalationAlertRepo) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *mockEscalationAlertRepo) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Get(0).(int), args.Error(1)
}

func (m *mockEscalationAlertRepo) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *mockEscalationAlertRepo) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	args := m.Called(ctx, alertID, userID)
	return args.Error(0)
}

type mockEscalationChannelRepo struct {
	mock.Mock
}

func (m *mockEscalationChannelRepo) Create(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *mockEscalationChannelRepo) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *mockEscalationChannelRepo) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	args := m.Called(ctx, userID, channelType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *mockEscalationChannelRepo) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*model.AlertChannel), args.Error(1)
}

func (m *mockEscalationChannelRepo) Update(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *mockEscalationChannelRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockEscalationChannelRepo) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	args := m.Called(ctx, userID, channelType, address)
	return args.Bool(0), args.Error(1)
}

func (m *mockEscalationChannelRepo) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	args := m.Called(ctx, id, failureCount)
	return args.Error(0)
}

func (m *mockEscalationChannelRepo) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, channelID, limit)
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *mockEscalationChannelRepo) CountByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *mockEscalationChannelRepo) IncrementFailureCount(ctx context.Context, id string) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

func (m *mockEscalationChannelRepo) DisableChannel(ctx context.Context, id string, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *mockEscalationChannelRepo) GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannelPriority), args.Error(1)
}

func (m *mockEscalationChannelRepo) SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error {
	args := m.Called(ctx, ruleID, priorities)
	return args.Error(0)
}

func newTestEscalationService(
	escRepo repository.AlertEscalationRepository,
	alertRepo repository.AlertRepository,
	channelRepo repository.AlertChannelRepository,
) *EscalationService {
	tracer := otel.GetTracerProvider().Tracer("test")
	return NewEscalationService(escRepo, alertRepo, channelRepo, nil, tracer, nil)
}

func TestCheckAutoEscalation_FindsExpired(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()
	alertID := uuid.New()

	oldAlert := &model.Alert{
		ID:        alertID,
		Status:    model.AlertStatusTriggered,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	alertRepo.On("ListActiveByMonitorID", mock.Anything, "").Return([]*model.Alert{oldAlert}, nil)
	escRepo.On("GetLatestByAlertID", mock.Anything, alertID.String()).Return(&model.AlertEscalation{Level: 0}, nil)
	escRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.CheckAutoEscalation(ctx)

	assert.NoError(t, err)
	alertRepo.AssertExpectations(t)
	escRepo.AssertExpectations(t)
}

func TestCheckAutoEscalation_NoExpiredAlerts(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()

	recentAlert := &model.Alert{
		ID:        uuid.New(),
		Status:    model.AlertStatusTriggered,
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}

	alertRepo.On("ListActiveByMonitorID", mock.Anything, "").Return([]*model.Alert{recentAlert}, nil)

	err := service.CheckAutoEscalation(ctx)

	assert.NoError(t, err)
	escRepo.AssertNotCalled(t, "GetLatestByAlertID")
	escRepo.AssertNotCalled(t, "Create")
}

func TestGetEscalationLevel(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()
	alertID := uuid.New().String()

	escRepo.On("GetLatestByAlertID", mock.Anything, alertID).Return(&model.AlertEscalation{Level: 3}, nil)

	level, err := service.GetEscalationLevel(ctx, alertID)

	assert.NoError(t, err)
	assert.Equal(t, 3, level)
}

func TestGetEscalationLevel_Error(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()
	alertID := uuid.New().String()

	escRepo.On("GetLatestByAlertID", mock.Anything, alertID).Return(nil, assert.AnError)

	_, err := service.GetEscalationLevel(ctx, alertID)

	assert.Error(t, err)
}

func TestEscalate_Success(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()
	alertID := uuid.New().String()

	escRepo.On("GetLatestByAlertID", mock.Anything, alertID).Return(&model.AlertEscalation{Level: 1}, nil)
	escRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := service.Escalate(ctx, alertID, "manual escalation")

	assert.NoError(t, err)
	escRepo.AssertExpectations(t)
}

func TestEscalate_GetLevelError(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()
	alertID := uuid.New().String()

	escRepo.On("GetLatestByAlertID", mock.Anything, alertID).Return(nil, assert.AnError)

	err := service.Escalate(ctx, alertID, "manual escalation")

	assert.Error(t, err)
}

func TestCheckAutoEscalation_ListError(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()

	alertRepo.On("ListActiveByMonitorID", mock.Anything, "").Return(nil, assert.AnError)

	err := service.CheckAutoEscalation(ctx)

	assert.Error(t, err)
}

func TestCheckAutoEscalation_SkipsNonTriggered(t *testing.T) {
	t.Parallel()

	escRepo := &mockEscalationRepo{}
	alertRepo := &mockEscalationAlertRepo{}
	channelRepo := &mockEscalationChannelRepo{}
	service := newTestEscalationService(escRepo, alertRepo, channelRepo)

	ctx := context.Background()

	acknowledgedAlert := &model.Alert{
		ID:        uuid.New(),
		Status:    model.AlertStatusAcknowledged,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	alertRepo.On("ListActiveByMonitorID", mock.Anything, "").Return([]*model.Alert{acknowledgedAlert}, nil)

	err := service.CheckAutoEscalation(ctx)

	assert.NoError(t, err)
	escRepo.AssertNotCalled(t, "GetLatestByAlertID")
}
