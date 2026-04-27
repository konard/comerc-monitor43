package delivery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/channels"
	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MockDeliveryAttemptRepository - mock implementation
type MockDeliveryAttemptRepository struct {
	mock.Mock
}

func (m *MockDeliveryAttemptRepository) Create(ctx context.Context, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepository) GetByID(ctx context.Context, id string) (*model.DeliveryAttempt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepository) ListPending(ctx context.Context, limit int) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return []*model.DeliveryAttempt{}, args.Error(1)
	}
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepository) Update(ctx context.Context, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, attempt)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepository) List(ctx context.Context, alertID string) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, alertID)
	if args.Get(0) == nil {
		return []*model.DeliveryAttempt{}, args.Error(1)
	}
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *MockDeliveryAttemptRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepository) DeleteOldAttempts(ctx context.Context, olderThan int64) error {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockDeliveryAttemptRepository) CountRecentByMonitorAndChannel(ctx context.Context, monitorID string, channelType string, since time.Duration) (int, error) {
	args := m.Called(ctx, monitorID, channelType, since)
	return args.Int(0), args.Error(1)
}

func (m *MockDeliveryAttemptRepository) GetLastDeliveryTimeForMonitorAndStatus(ctx context.Context, monitorID string, status string) (*time.Time, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

// MockAlertChannelRepository - mock implementation
type MockAlertChannelRepository struct {
	mock.Mock
}

func (m *MockAlertChannelRepository) Create(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertChannelRepository) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *MockAlertChannelRepository) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	args := m.Called(ctx, userID, channelType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *MockAlertChannelRepository) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return []*model.AlertChannel{}, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannel), args.Error(1)
}

func (m *MockAlertChannelRepository) Update(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertChannelRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertChannelRepository) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	args := m.Called(ctx, userID, channelType, address)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	return args.Bool(0), args.Error(1)
}

func (m *MockAlertChannelRepository) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	args := m.Called(ctx, id, failureCount)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertChannelRepository) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, channelID, limit)
	if args.Get(0) == nil {
		return []*model.DeliveryAttempt{}, args.Error(1)
	}
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *MockAlertChannelRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockAlertChannelRepository) IncrementFailureCount(ctx context.Context, id string) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

func (m *MockAlertChannelRepository) DisableChannel(ctx context.Context, id string, reason string) error {
	args := m.Called(ctx, id, reason)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertChannelRepository) SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error {
	args := m.Called(ctx, ruleID, priorities)
	return args.Error(0)
}

func (m *MockAlertChannelRepository) GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannelPriority), args.Error(1)
}

// MockAlertRepository - mock implementation
type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) GetByID(ctx context.Context, id string) (*model.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) ListActiveByMonitorID(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return []*model.Alert{}, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) List(ctx context.Context, userID string, filter model.AlertFilter) ([]*model.Alert, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return []*model.Alert{}, 0, args.Error(2)
	}
	return args.Get(0).([]*model.Alert), args.Int(1), args.Error(2)
}

func (m *MockAlertRepository) Update(ctx context.Context, alert *model.Alert) error {
	args := m.Called(ctx, alert)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) GetLastAlertTimeAnyStatus(ctx context.Context, monitorID string) (*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) CreateWithDeliveryAttempt(ctx context.Context, alert *model.Alert, attempt *model.DeliveryAttempt) error {
	args := m.Called(ctx, alert, attempt)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) DeleteResolvedOlderThan(ctx context.Context, olderThan time.Duration) error {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAlertRepository) GetLastAlertByMonitorIDAndStatus(ctx context.Context, monitorID string, status model.AlertStatus) (*model.Alert, error) {
	args := m.Called(ctx, monitorID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) CountUniqueMonitorsWithAlertsSince(ctx context.Context, since time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Int(0), args.Error(1)
}

func (m *MockAlertRepository) GetActiveAlertsForMonitor(ctx context.Context, monitorID string) ([]*model.Alert, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Alert), args.Error(1)
}

func (m *MockAlertRepository) AcknowledgeAlert(ctx context.Context, alertID, userID string) error {
	args := m.Called(ctx, alertID, userID)
	return args.Error(0)
}

// TestNewAlertDeliveryService проверяет создание сервиса
func TestNewAlertDeliveryService(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	telegramClient := channels.NewTelegramClient("test-token", "")
	emailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	webhookClient := channels.NewWebhookClient(30 * time.Second)
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		telegramClient,
		emailClient,
		webhookClient,
		tracer,
		nil,
		WithMaxRetries(5),
		WithRetryInterval(10*time.Minute),
	)

	assert.NotNil(t, service)
	assert.Equal(t, mockDeliveryRepo, service.deliveryRepo)
	assert.Equal(t, mockChannelRepo, service.channelRepo)
	assert.Equal(t, mockAlertRepo, service.alertRepo)
	assert.Equal(t, 5, service.maxRetries)
	assert.Equal(t, 10*time.Minute, service.retryInterval)
}

// TestAlertDeliveryService_DeliverAlert_ChannelDisabled проверяет доставку в отключенный канал
func TestAlertDeliveryService_DeliverAlert_ChannelDisabled(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
	}

	channel := &model.AlertChannel{
		ID:      uuid.New(),
		Enabled: false,
	}

	err := service.DeliverAlert(ctx, alert, channel)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel is disabled")
}

// TestAlertDeliveryService_DeliverAlert_ChannelNotVerified проверяет доставку в неверифицированный канал
func TestAlertDeliveryService_DeliverAlert_ChannelNotVerified(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
	}

	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Enabled:  true,
		Verified: false,
	}

	err := service.DeliverAlert(ctx, alert, channel)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel is not verified")
}

// TestAlertDeliveryService_ScheduleRetry_MaxRetriesReached проверяет планирование retry при достижении лимита
func TestAlertDeliveryService_ScheduleRetry_MaxRetriesReached(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
		WithMaxRetries(3),
	)

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		AlertID:    uuid.New(),
		RetryCount: 2, // Following retry will make it 3, reaching max
	}

	mockDeliveryRepo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)

	err := service.ScheduleRetry(ctx, attempt, &model.DeliveryError{
		Category:   model.DeliveryErrorTransient,
		Retryable:  true,
		MaxRetries: 3,
		BaseDelay:  5 * time.Minute,
	})

	assert.NoError(t, err)
}

// TestAlertDeliveryService_ScheduleRetry_ScheduleNextRetry проверяет планирование следующего retry
func TestAlertDeliveryService_ScheduleRetry_ScheduleNextRetry(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
		WithMaxRetries(3),
		WithRetryInterval(5*time.Minute),
	)

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		AlertID:    uuid.New(),
		RetryCount: 0,
	}

	mockDeliveryRepo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)

	err := service.ScheduleRetry(ctx, attempt, &model.DeliveryError{
		Category:   model.DeliveryErrorTransient,
		Retryable:  true,
		MaxRetries: 3,
		BaseDelay:  5 * time.Minute,
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempt.RetryCount)
	assert.NotEqual(t, model.DeliveryAttemptStatusFailed, attempt.Status)
	assert.NotNil(t, attempt.NextRetryAt)
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_ProcessPendingAttempts проверяет обработку pending попыток
func TestAlertDeliveryService_ProcessPendingAttempts(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	alertID := uuid.New()
	channelID := uuid.New()
	attempts := []*model.DeliveryAttempt{
		{
			ID:             uuid.New(),
			AlertID:        alertID,
			AlertChannelID: channelID,
		},
	}

	// Mock the delivery repository to return pending attempts
	mockDeliveryRepo.On("ListPending", mock.AnythingOfType("*context.valueCtx"), 100).Return(attempts, nil)

	// Mock the alert repository to return an alert
	mockAlertRepo.On("GetByID", mock.AnythingOfType("*context.valueCtx"), alertID.String()).Return(&model.Alert{
		ID:        alertID,
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
	}, nil)

	// Mock the channel repository to return a channel
	mockChannelRepo.On("GetByID", mock.AnythingOfType("*context.valueCtx"), channelID.String()).Return(&model.AlertChannel{
		ID:       channelID,
		Type:     model.AlertChannelTypeTelegram,
		Enabled:  true,
		Verified: true,
	}, nil)

	// Mock the telegram client (will be nil so we expect error)
	// Mock the channel failure handling
	mockChannelRepo.On("IncrementFailureCount", mock.AnythingOfType("*context.valueCtx"), channelID.String()).Return(1, nil)

	// Mock the update of the delivery attempt
	mockDeliveryRepo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)

	err := service.ProcessPendingAttempts(ctx)

	// We don't expect an error because the error is logged and the method continues
	assert.NoError(t, err)
	mockDeliveryRepo.AssertExpectations(t)
	mockAlertRepo.AssertExpectations(t)
	mockChannelRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_ProcessPendingAttempts_Error проверяет обработку ошибок при получении списка
func TestAlertDeliveryService_ProcessPendingAttempts_Error(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	mockDeliveryRepo.On("ListPending", mock.AnythingOfType("*context.valueCtx"), 100).Return([]*model.DeliveryAttempt{}, assert.AnError)

	err := service.ProcessPendingAttempts(ctx)

	assert.Error(t, err)
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverAlert_MissingConfig проверяет отсутствие конфигурации канала
func TestAlertDeliveryService_DeliverAlert_MissingConfig(t *testing.T) {
	tests := []struct {
		name    string
		channel *model.AlertChannel
	}{
		{
			name: "missing telegram config",
			channel: &model.AlertChannel{
				ID:             uuid.New(),
				Type:           model.AlertChannelTypeTelegram,
				Enabled:        true,
				Verified:       true,
				TelegramConfig: nil,
			},
		},
		{
			name: "missing email config",
			channel: &model.AlertChannel{
				ID:          uuid.New(),
				Type:        model.AlertChannelTypeEmail,
				Enabled:     true,
				Verified:    true,
				EmailConfig: nil,
			},
		},
		{
			name: "missing webhook config",
			channel: &model.AlertChannel{
				ID:            uuid.New(),
				Type:          model.AlertChannelTypeWebhook,
				Enabled:       true,
				Verified:      true,
				WebhookConfig: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDeliveryRepo := &MockDeliveryAttemptRepository{}
			mockChannelRepo := &MockAlertChannelRepository{}
			mockAlertRepo := &MockAlertRepository{}
			tracer := otel.GetTracerProvider().Tracer("test")

			service := NewAlertDeliveryService(
				mockDeliveryRepo,
				mockChannelRepo,
				mockAlertRepo,
				nil,
				nil,
				nil,
				tracer,
				nil,
			)

			ctx := context.Background()
			alert := &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
				Type:      model.AlertTypeStatusCode,
			}

			mockDeliveryRepo.On("Create", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)
			mockDeliveryRepo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)
			mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
			mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)

			err := service.DeliverAlert(ctx, alert, tt.channel)

			// Error should be logged but not returned from DeliverAlert
			assert.NoError(t, err)
			mockDeliveryRepo.AssertExpectations(t)
		})
	}
}

// TestAlertDeliveryService_DeliverAlert_AllChannelTypes проверяет все типы каналов
func TestAlertDeliveryService_DeliverAlert_AllChannelTypes(t *testing.T) {
	tests := []struct {
		name    string
		channel *model.AlertChannel
	}{
		{
			name: "telegram channel with config",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeTelegram,
				Enabled:  true,
				Verified: true,
				TelegramConfig: &model.TelegramChannelConfig{
					ChatID: "@testuser",
				},
			},
		},
		{
			name: "email channel with config",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeEmail,
				Enabled:  true,
				Verified: true,
				EmailConfig: &model.EmailChannelConfig{
					Email: "test@example.com",
				},
			},
		},
		{
			name: "webhook channel with config",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeWebhook,
				Enabled:  true,
				Verified: true,
				WebhookConfig: &model.WebhookChannelConfig{
					URL: "https://example.com/webhook",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDeliveryRepo := &MockDeliveryAttemptRepository{}
			mockChannelRepo := &MockAlertChannelRepository{}
			mockAlertRepo := &MockAlertRepository{}
			tracer := otel.GetTracerProvider().Tracer("test")

			// Provide dummy clients to avoid nil pointer dereference
			dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
			dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
			dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

			service := NewAlertDeliveryService(
				mockDeliveryRepo,
				mockChannelRepo,
				mockAlertRepo,
				dummyTelegramClient,
				dummyEmailClient,
				dummyWebhookClient,
				tracer,
				nil,
			)

			ctx := context.Background()
			alert := &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
				Type:      model.AlertTypeStatusCode,
			}

			mockDeliveryRepo.On("Create", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)
			mockDeliveryRepo.On("Update", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)
			mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
			mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)

			err := service.DeliverAlert(ctx, alert, tt.channel)

			// Error is logged but not returned from DeliverAlert
			assert.NoError(t, err)
			mockDeliveryRepo.AssertExpectations(t)
		})
	}
}

// TestAlertDeliveryService_DeliverAlert_CreationError проверяет ошибку при создании попытки доставки
func TestAlertDeliveryService_DeliverAlert_CreationError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
		Type:      model.AlertTypeStatusCode,
	}

	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Type:     model.AlertChannelTypeTelegram,
		Enabled:  true,
		Verified: true,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "@testuser",
		},
	}

	mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	mockDeliveryRepo.On("Create", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*model.DeliveryAttempt")).Return(assert.AnError)

	err := service.DeliverAlert(ctx, alert, channel)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create delivery attempt")
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverRecovery проверяет доставку уведомления о восстановлении
func TestAlertDeliveryService_DeliverRecovery(t *testing.T) {
	tests := []struct {
		name          string
		channel       *model.AlertChannel
		expectError   bool
		errorContains string
	}{
		{
			name: "successful recovery via telegram",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeTelegram,
				Enabled:  true,
				Verified: true,
				TelegramConfig: &model.TelegramChannelConfig{
					ChatID: "@testuser",
				},
			},
			expectError: false,
		},
		{
			name: "successful recovery via email",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeEmail,
				Enabled:  true,
				Verified: true,
				EmailConfig: &model.EmailChannelConfig{
					Email: "test@example.com",
				},
			},
			expectError: false,
		},
		{
			name: "successful recovery via webhook",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeWebhook,
				Enabled:  true,
				Verified: true,
				WebhookConfig: &model.WebhookChannelConfig{
					URL: "https://example.com/webhook",
				},
			},
			expectError: false,
		},
		{
			name: "channel disabled",
			channel: &model.AlertChannel{
				ID:      uuid.New(),
				Type:    model.AlertChannelTypeTelegram,
				Enabled: false,
			},
			expectError:   true,
			errorContains: "channel is disabled",
		},
		{
			name: "channel not verified",
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeTelegram,
				Enabled:  true,
				Verified: false,
			},
			expectError:   true,
			errorContains: "channel is not verified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDeliveryRepo := &MockDeliveryAttemptRepository{}
			mockChannelRepo := &MockAlertChannelRepository{}
			mockAlertRepo := &MockAlertRepository{}
			tracer := otel.GetTracerProvider().Tracer("test")

			dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
			dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
			dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

			service := NewAlertDeliveryService(
				mockDeliveryRepo,
				mockChannelRepo,
				mockAlertRepo,
				dummyTelegramClient,
				dummyEmailClient,
				dummyWebhookClient,
				tracer,
				nil,
			)

			ctx := context.Background()
			alert := &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusResolved,
			}

			// Only set up Create and Update mocks for enabled and verified channels
			if tt.channel.Enabled && tt.channel.Verified {
				mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)
				mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.DeliveryAttempt")).Return(nil)
				mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)
			}

			err := service.DeliverRecovery(ctx, alert, tt.channel)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
			mockDeliveryRepo.AssertExpectations(t)
		})
	}
}

// TestAlertDeliveryService_formatAlertMessage проверяет форматирование сообщений
func TestAlertDeliveryService_formatAlertMessage(t *testing.T) {
	tests := []struct {
		name         string
		alert        *model.Alert
		isRecovery   bool
		expectStatus string
	}{
		{
			name: "triggered alert message",
			alert: &model.Alert{
				ID:        uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
			},
			isRecovery:   false,
			expectStatus: "🔴 Monitor",
		},
		{
			name: "resolved alert message",
			alert: &model.Alert{
				ID:        uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusResolved,
			},
			isRecovery:   false,
			expectStatus: "🟢 Monitor",
		},
		{
			name: "unknown alert status",
			alert: &model.Alert{
				ID:        uuid.New(),
				MonitorID: uuid.New(),
				Status:    "UNKNOWN_STATUS",
			},
			isRecovery:   false,
			expectStatus: "⚠️ Monitor",
		},
		{
			name: "recovery message",
			alert: &model.Alert{
				ID:        uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
			},
			isRecovery:   true,
			expectStatus: "🟢 Monitor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDeliveryRepo := &MockDeliveryAttemptRepository{}
			mockChannelRepo := &MockAlertChannelRepository{}
			mockAlertRepo := &MockAlertRepository{}
			tracer := otel.GetTracerProvider().Tracer("test")

			dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
			dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
			dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

			service := NewAlertDeliveryService(
				mockDeliveryRepo,
				mockChannelRepo,
				mockAlertRepo,
				dummyTelegramClient,
				dummyEmailClient,
				dummyWebhookClient,
				tracer,
				nil,
			)

			message := service.formatAlertMessage(tt.alert, tt.isRecovery)

			assert.Contains(t, message, tt.expectStatus)
			assert.Contains(t, message, tt.alert.MonitorID.String())
		})
	}
}

// TestAlertDeliveryService_processAttempt проверяет обработку одной попытки
func TestAlertDeliveryService_processAttempt(t *testing.T) {
	tests := []struct {
		name                   string
		attempt                *model.DeliveryAttempt
		alert                  *model.Alert
		channel                *model.AlertChannel
		expectError            bool
		errorContains          string
		shouldScheduleRetry    bool
		expectIncrementFailure bool
		updateCalls            int
	}{
		{
			name: "successful delivery",
			attempt: &model.DeliveryAttempt{
				ID:         uuid.New(),
				AlertID:    uuid.New(),
				RetryCount: 0,
			},
			alert: &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
			},
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeTelegram,
				Enabled:  true,
				Verified: true,
				TelegramConfig: &model.TelegramChannelConfig{
					ChatID: "@testuser",
				},
			},
			expectError:            false,
			shouldScheduleRetry:    false,
			expectIncrementFailure: true,
			updateCalls:            2, // ScheduleRetry + final Update
		},
		{
			name: "delivery with max retries",
			attempt: &model.DeliveryAttempt{
				ID:         uuid.New(),
				AlertID:    uuid.New(),
				RetryCount: 3, // At max retries
			},
			alert: &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
			},
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeEmail,
				Enabled:  true,
				Verified: true,
				EmailConfig: &model.EmailChannelConfig{
					Email: "test@example.com",
				},
			},
			expectError:            false,
			shouldScheduleRetry:    false,
			expectIncrementFailure: true,
			updateCalls:            1, // Only final Update (max retries reached)
		},
		{
			name: "delivery with below max retries",
			attempt: &model.DeliveryAttempt{
				ID:         uuid.New(),
				AlertID:    uuid.New(),
				RetryCount: 1, // Below max retries
			},
			alert: &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
			},
			channel: &model.AlertChannel{
				ID:       uuid.New(),
				Type:     model.AlertChannelTypeWebhook,
				Enabled:  true,
				Verified: true,
				WebhookConfig: &model.WebhookChannelConfig{
					URL: "https://example.com/webhook",
				},
			},
			expectError:            false,
			shouldScheduleRetry:    true,
			expectIncrementFailure: true,
			updateCalls:            2, // ScheduleRetry + final Update
		},
		{
			name: "missing channel config",
			attempt: &model.DeliveryAttempt{
				ID:         uuid.New(),
				AlertID:    uuid.New(),
				RetryCount: 0,
			},
			alert: &model.Alert{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				MonitorID: uuid.New(),
				Status:    model.AlertStatusTriggered,
			},
			channel: &model.AlertChannel{
				ID:             uuid.New(),
				Type:           model.AlertChannelTypeTelegram,
				Enabled:        true,
				Verified:       true,
				TelegramConfig: nil, // Missing config
			},
			expectError:            false,
			errorContains:          "",
			shouldScheduleRetry:    false,
			expectIncrementFailure: true,
			updateCalls:            2, // ScheduleRetry + final Update
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDeliveryRepo := &MockDeliveryAttemptRepository{}
			mockChannelRepo := &MockAlertChannelRepository{}
			mockAlertRepo := &MockAlertRepository{}
			tracer := otel.GetTracerProvider().Tracer("test")

			dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
			dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
			dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

			service := NewAlertDeliveryService(
				mockDeliveryRepo,
				mockChannelRepo,
				mockAlertRepo,
				dummyTelegramClient,
				dummyEmailClient,
				dummyWebhookClient,
				tracer,
				nil,
				WithMaxRetries(3),
			)

			ctx := context.Background()

			mockAlertRepo.On("GetByID", mock.Anything, tt.attempt.AlertID.String()).Return(tt.alert, nil)
			mockChannelRepo.On("GetByID", mock.Anything, tt.attempt.AlertChannelID.String()).Return(tt.channel, nil)
			for i := 0; i < tt.updateCalls; i++ {
				mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
			}
			if tt.expectIncrementFailure {
				mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)
			}

			err := service.processAttempt(ctx, tt.attempt)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}

			mockDeliveryRepo.AssertExpectations(t)
			mockAlertRepo.AssertExpectations(t)
			mockChannelRepo.AssertExpectations(t)
		})
	}
}

// TestAlertDeliveryService_DeliverAlert_UpdateError проверяет ошибку при обновлении попытки
func TestAlertDeliveryService_DeliverAlert_UpdateError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
	}

	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Type:     model.AlertChannelTypeTelegram,
		Enabled:  true,
		Verified: true,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "@testuser",
		},
	}

	// Mock create to succeed, update to fail
	mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(assert.AnError)
	mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)

	err := service.DeliverAlert(ctx, alert, channel)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update delivery attempt")
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverRecovery_UpdateError проверяет ошибку при обновлении попытки восстановления
func TestAlertDeliveryService_DeliverRecovery_UpdateError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusResolved,
	}

	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Type:     model.AlertChannelTypeTelegram,
		Enabled:  true,
		Verified: true,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "@testuser",
		},
	}

	// Mock create to succeed, update to fail
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(assert.AnError)
	mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)

	err := service.DeliverRecovery(ctx, alert, channel)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update delivery attempt")
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverRecovery_CreationError проверяет ошибку при создании попытки восстановления
func TestAlertDeliveryService_DeliverRecovery_CreationError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusResolved,
	}

	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Type:     model.AlertChannelTypeTelegram,
		Enabled:  true,
		Verified: true,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "@testuser",
		},
	}

	// Mock create to fail
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(assert.AnError)

	err := service.DeliverRecovery(ctx, alert, channel)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create delivery attempt")
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_processAttempt_GetAlertError проверяет ошибку при получении алерта
func TestAlertDeliveryService_processAttempt_GetAlertError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		AlertID:    uuid.New(),
		RetryCount: 0,
	}

	// Mock alert get to fail
	mockAlertRepo.On("GetByID", mock.Anything, attempt.AlertID.String()).Return(nil, assert.AnError)
	mockChannelRepo.On("GetByID", mock.Anything, attempt.AlertChannelID.String()).Return(nil, nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err := service.processAttempt(ctx, attempt)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert")
	mockAlertRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_processAttempt_GetChannelError проверяет ошибку при получении канала
func TestAlertDeliveryService_processAttempt_GetChannelError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		AlertID:    uuid.New(),
		RetryCount: 0,
	}

	// Mock alert get to succeed, channel get to fail
	mockAlertRepo.On("GetByID", mock.Anything, attempt.AlertID.String()).Return(&model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
	}, nil)
	mockChannelRepo.On("GetByID", mock.Anything, attempt.AlertChannelID.String()).Return(nil, assert.AnError)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err := service.processAttempt(ctx, attempt)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get channel")
	mockAlertRepo.AssertExpectations(t)
	mockChannelRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverAlert_UnsupportedChannelType проверяет неподдерживаемый тип канала
func TestAlertDeliveryService_DeliverAlert_UnsupportedChannelType(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
	}

	// Create channel with unsupported type
	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Type:     "UNSUPPORTED_TYPE", // Invalid type
		Enabled:  true,
		Verified: true,
	}

	mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)

	err := service.DeliverAlert(ctx, alert, channel)

	// For unsupported channel types, the error is logged but not returned
	assert.NoError(t, err)
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverRecovery_UnsupportedChannelType проверяет неподдерживаемый тип канала
func TestAlertDeliveryService_DeliverRecovery_UnsupportedChannelType(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	dummyTelegramClient := channels.NewTelegramClient("dummy-token", "")
	dummyEmailClient := channels.NewEmailClient("smtp.example.com", 587, "from@example.com")
	dummyWebhookClient := channels.NewWebhookClient(30 * time.Second)

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		dummyTelegramClient,
		dummyEmailClient,
		dummyWebhookClient,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusResolved,
	}

	// Create channel with unsupported type
	channel := &model.AlertChannel{
		ID:       uuid.New(),
		Type:     "UNSUPPORTED_TYPE", // Invalid type
		Enabled:  true,
		Verified: true,
	}

	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockChannelRepo.On("IncrementFailureCount", mock.Anything, mock.Anything).Return(1, nil)

	err := service.DeliverRecovery(ctx, alert, channel)

	// For unsupported channel types, the error is logged but not returned
	assert.NoError(t, err)
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_ScheduleRetry_UpdateError проверяет ошибку при планировании retry
func TestAlertDeliveryService_ScheduleRetry_UpdateError(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		AlertID:    uuid.New(),
		RetryCount: 0,
	}

	// Mock update to fail
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(assert.AnError)

	err := service.ScheduleRetry(ctx, attempt, &model.DeliveryError{
		Category:   model.DeliveryErrorTransient,
		Retryable:  true,
		MaxRetries: 3,
		BaseDelay:  5 * time.Minute,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update delivery attempt")
	mockDeliveryRepo.AssertExpectations(t)
}

// MockMaintenanceWindowRepository - mock for maintenance window repo
type MockMaintenanceWindowRepository struct {
	mock.Mock
}

func (m *MockMaintenanceWindowRepository) Create(ctx context.Context, mw *model.MaintenanceWindow) error {
	args := m.Called(ctx, mw)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) GetActiveByMonitorID(ctx context.Context, monitorID string) (*model.MaintenanceWindow, error) {
	args := m.Called(ctx, monitorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) IsUnderMaintenance(ctx context.Context, monitorID string) (bool, error) {
	args := m.Called(ctx, monitorID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) GetByID(ctx context.Context, id string) (*model.MaintenanceWindow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MaintenanceWindow), args.Error(1)
}

func (m *MockMaintenanceWindowRepository) Update(ctx context.Context, mw *model.MaintenanceWindow) error {
	args := m.Called(ctx, mw)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMaintenanceWindowRepository) List(ctx context.Context, userID string, filter model.MaintenanceWindowFilter) ([]*model.MaintenanceWindow, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.MaintenanceWindow), args.Int(1), args.Error(2)
}

func (m *MockMaintenanceWindowRepository) CheckOverlapping(ctx context.Context, monitorIDs []string, startsAt, endsAt time.Time, excludeID string) (bool, *model.MaintenanceWindow, error) {
	args := m.Called(ctx, monitorIDs, startsAt, endsAt, excludeID)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*model.MaintenanceWindow), args.Error(2)
}

// TestAlertDeliveryService_DeliverAlert_MaintenanceSuppressed проверяет создание SUPPRESSED записи (uc_02_03_20)
func TestAlertDeliveryService_DeliverAlert_MaintenanceSuppressed(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	mockMaintenanceRepo := &MockMaintenanceWindowRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
		WithMaintenanceRepo(mockMaintenanceRepo),
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
	}
	channel := &model.AlertChannel{
		ID:          uuid.New(),
		Type:        model.AlertChannelTypeEmail,
		Enabled:     true,
		Verified:    true,
		EmailConfig: &model.EmailChannelConfig{Email: "test@example.com"},
	}

	mockMaintenanceRepo.On("IsUnderMaintenance", mock.Anything, alert.MonitorID.String()).Return(true, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *model.DeliveryAttempt) bool {
		return a.Status == model.DeliveryAttemptStatusSuppressed
	})).Return(nil)

	err := service.DeliverAlert(ctx, alert, channel)

	assert.NoError(t, err)
	mockMaintenanceRepo.AssertExpectations(t)
	mockDeliveryRepo.AssertExpectations(t)
}

// TestAlertDeliveryService_DeliverToAllChannels_Success проверяет доставку во все каналы
func TestAlertDeliveryService_DeliverToAllChannels_Success(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil,
		nil,
		nil,
		tracer,
		nil,
	)

	ctx := context.Background()
	alert := &model.Alert{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		MonitorID: uuid.New(),
		Status:    model.AlertStatusTriggered,
	}
	// Канал отключён — DeliverAlert вернёт ошибку
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Enabled: false,
	}

	result := service.DeliverToAllChannels(ctx, alert, []*model.AlertChannel{channel})

	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalChannels)
}

// TestAlertDeliveryService_DeliverToAllChannels_Empty проверяет доставку без каналов
func TestAlertDeliveryService_DeliverToAllChannels_Empty(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	mockChannelRepo := &MockAlertChannelRepository{}
	mockAlertRepo := &MockAlertRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewAlertDeliveryService(
		mockDeliveryRepo,
		mockChannelRepo,
		mockAlertRepo,
		nil, nil, nil,
		tracer, nil,
	)

	ctx := context.Background()
	alert := &model.Alert{ID: uuid.New()}

	result := service.DeliverToAllChannels(ctx, alert, []*model.AlertChannel{})

	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalChannels)
	assert.Equal(t, 0, result.SuccessfulDeliveries)
}

// TestAlertDeliveryService_ScheduleRetryWithoutClassification проверяет планирование ретрая без классификации
func TestAlertDeliveryService_ScheduleRetryWithoutClassification(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		deliveryRepo:  mockDeliveryRepo,
		maxRetries:    3,
		retryInterval: 30 * time.Second,
		tracer:        tracer,
	}

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		RetryCount: 0,
	}

	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err := service.ScheduleRetryWithoutClassification(ctx, attempt)

	assert.NoError(t, err)
	mockDeliveryRepo.AssertExpectations(t)
}

// TestMaskErrorSecrets проверяет маскировку секретов в сообщениях об ошибках
func TestMaskErrorSecrets(t *testing.T) {
	cases := []struct {
		input    string
		contains string
	}{
		{"token=abc123", "***"},
		{"key=secret_value", "***"},
		{"normal error message", "normal error message"},
		{"auth=Bearer xyz", "***"},
	}

	for _, c := range cases {
		result := maskErrorSecrets(c.input)
		assert.Contains(t, result, c.contains)
	}
}

// TestHandleChannelFailure_AuthError проверяет немедленное отключение при ошибке авторизации
func TestHandleChannelFailure_AuthError(t *testing.T) {
	mockChannelRepo := &MockAlertChannelRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		channelRepo: mockChannelRepo,
		tracer:      tracer,
	}

	ctx := context.Background()
	channel := &model.AlertChannel{ID: uuid.New()}

	mockChannelRepo.On("DisableChannel", mock.Anything, channel.ID.String(), mock.Anything).Return(nil)

	service.handleChannelFailure(ctx, channel, &model.DeliveryError{
		Category:   model.DeliveryErrorAuth,
		StatusCode: 401,
	})

	mockChannelRepo.AssertCalled(t, "DisableChannel", mock.Anything, channel.ID.String(), mock.Anything)
}

// TestHandleChannelFailure_MaxFailures проверяет отключение после maxChannelFailures
func TestHandleChannelFailure_MaxFailures(t *testing.T) {
	mockChannelRepo := &MockAlertChannelRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		channelRepo: mockChannelRepo,
		tracer:      tracer,
	}

	ctx := context.Background()
	channel := &model.AlertChannel{ID: uuid.New()}

	mockChannelRepo.On("IncrementFailureCount", mock.Anything, channel.ID.String()).Return(maxChannelFailures, nil)
	mockChannelRepo.On("DisableChannel", mock.Anything, channel.ID.String(), mock.Anything).Return(nil)

	service.handleChannelFailure(ctx, channel, &model.DeliveryError{
		Category: model.DeliveryErrorPermanent,
	})

	mockChannelRepo.AssertExpectations(t)
}

// TestMaskErrorSecrets_ColonFormat проверяет маскировку секретов в формате key:value
func TestMaskErrorSecrets_ColonFormat(t *testing.T) {
	result := maskErrorSecrets("token: mysecrettoken123")
	assert.NotContains(t, result, "mysecrettoken123")
	assert.Contains(t, result, "token")
}

// TestMaskErrorSecrets_NoMatch проверяет что обычные строки не изменяются
func TestMaskErrorSecrets_NoMatch(t *testing.T) {
	input := "connection refused to host"
	result := maskErrorSecrets(input)
	assert.Equal(t, input, result)
}

// TestHandleChannelFailure_IncrementError проверяет ранний выход при ошибке IncrementFailureCount
func TestHandleChannelFailure_IncrementError(t *testing.T) {
	mockChannelRepo := &MockAlertChannelRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		channelRepo: mockChannelRepo,
		tracer:      tracer,
	}

	ctx := context.Background()
	channel := &model.AlertChannel{ID: uuid.New()}

	mockChannelRepo.On("IncrementFailureCount", mock.Anything, channel.ID.String()).Return(0, assert.AnError)

	// Не должно паниковать и не вызывать DisableChannel
	service.handleChannelFailure(ctx, channel, &model.DeliveryError{
		Category: model.DeliveryErrorPermanent,
	})

	mockChannelRepo.AssertNotCalled(t, "DisableChannel")
}

// TestIsRateLimited_Positive проверяет что count >= maxDeliveriesPerRateLimitWindow возвращает true
func TestIsRateLimited_Positive(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		deliveryRepo: mockDeliveryRepo,
		tracer:       tracer,
	}

	ctx := context.Background()
	alert := &model.Alert{ID: uuid.New(), MonitorID: uuid.New()}

	mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, alert.MonitorID.String(), "", RateLimitPerMonitorPerStatus).Return(3, nil)

	result := service.isRateLimited(ctx, alert)
	assert.True(t, result)
}

// TestIsRateLimited_BelowThreshold проверяет что count < maxDeliveriesPerRateLimitWindow не блокирует доставку
func TestIsRateLimited_BelowThreshold(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		deliveryRepo: mockDeliveryRepo,
		tracer:       tracer,
	}

	ctx := context.Background()
	alert := &model.Alert{ID: uuid.New(), MonitorID: uuid.New()}

	// count=2 ниже порога (maxDeliveriesPerRateLimitWindow=3) — не блокирует
	mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, alert.MonitorID.String(), "", RateLimitPerMonitorPerStatus).Return(2, nil)

	result := service.isRateLimited(ctx, alert)
	assert.False(t, result)
}

// TestIsRateLimited_Error проверяет что ошибка репозитория возвращает false (не блокирует доставку)
func TestIsRateLimited_Error(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		deliveryRepo: mockDeliveryRepo,
		tracer:       tracer,
	}

	ctx := context.Background()
	alert := &model.Alert{ID: uuid.New(), MonitorID: uuid.New()}

	mockDeliveryRepo.On("CountRecentByMonitorAndChannel", mock.Anything, alert.MonitorID.String(), "", RateLimitPerMonitorPerStatus).Return(0, assert.AnError)

	result := service.isRateLimited(ctx, alert)
	assert.False(t, result)
}

// TestHandleChannelFailure_AuthError_DisableError проверяет warn-лог при ошибке DisableChannel (auth path)
func TestHandleChannelFailure_AuthError_DisableError(t *testing.T) {
	mockChannelRepo := &MockAlertChannelRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		channelRepo: mockChannelRepo,
		tracer:      tracer,
	}

	ctx := context.Background()
	channel := &model.AlertChannel{ID: uuid.New()}

	mockChannelRepo.On("DisableChannel", mock.Anything, channel.ID.String(), mock.Anything).Return(assert.AnError)

	service.handleChannelFailure(ctx, channel, &model.DeliveryError{
		Category:   model.DeliveryErrorAuth,
		StatusCode: 401,
	})

	mockChannelRepo.AssertCalled(t, "DisableChannel", mock.Anything, channel.ID.String(), mock.Anything)
}

// TestHandleChannelFailure_MaxFailures_DisableError проверяет warn-лог при ошибке DisableChannel (max failures path)
func TestHandleChannelFailure_MaxFailures_DisableError(t *testing.T) {
	mockChannelRepo := &MockAlertChannelRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		channelRepo: mockChannelRepo,
		tracer:      tracer,
	}

	ctx := context.Background()
	channel := &model.AlertChannel{ID: uuid.New()}

	mockChannelRepo.On("IncrementFailureCount", mock.Anything, channel.ID.String()).Return(maxChannelFailures, nil)
	mockChannelRepo.On("DisableChannel", mock.Anything, channel.ID.String(), mock.Anything).Return(assert.AnError)

	service.handleChannelFailure(ctx, channel, &model.DeliveryError{
		Category: model.DeliveryErrorPermanent,
	})

	mockChannelRepo.AssertExpectations(t)
}

// TestCalculateRetryDelay_ExceedsMaxDelay проверяет что задержка ограничена 10 минутами
func TestCalculateRetryDelay_ExceedsMaxDelay(t *testing.T) {
	service := &AlertDeliveryService{
		tracer: otel.GetTracerProvider().Tracer("test"),
	}

	// attempt=20 → factor=2^20=1048576, baseDelay*factor >> 10min → clamps to maxDelay
	delay := service.calculateRetryDelay(20, time.Minute)
	assert.LessOrEqual(t, delay.Seconds(), (12 * time.Minute).Seconds()) // 10min + max 20% jitter = 12min
	assert.Greater(t, delay.Seconds(), 0.0)
}

// TestAlertDeliveryService_ScheduleRetry_RateLimitFixedDelay проверяет фиксированную задержку для 429 (uc_02_03_13)
func TestAlertDeliveryService_ScheduleRetry_RateLimitFixedDelay(t *testing.T) {
	mockDeliveryRepo := &MockDeliveryAttemptRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := &AlertDeliveryService{
		deliveryRepo: mockDeliveryRepo,
		maxRetries:   5,
		tracer:       tracer,
	}

	ctx := context.Background()
	attempt := &model.DeliveryAttempt{
		ID:         uuid.New(),
		RetryCount: 0,
	}

	classifiedErr := &model.DeliveryError{
		Category:   model.DeliveryErrorRateLimit,
		Retryable:  true,
		MaxRetries: 5,
		BaseDelay:  5 * time.Minute,
	}

	mockDeliveryRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.DeliveryAttempt) bool {
		return a.NextRetryAt != nil && a.RetryCount == 1
	})).Return(nil)

	err := service.ScheduleRetry(ctx, attempt, classifiedErr)

	assert.NoError(t, err)
	assert.NotNil(t, attempt.NextRetryAt)
	// Фиксированная задержка 5 минут (с небольшой погрешностью в тесте)
	delay := time.Until(*attempt.NextRetryAt)
	assert.InDelta(t, (5 * time.Minute).Seconds(), delay.Seconds(), 10)
	mockDeliveryRepo.AssertExpectations(t)
}
