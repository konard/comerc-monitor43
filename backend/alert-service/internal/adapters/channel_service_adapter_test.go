package adapters

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// Mock implementations for testing

type MockChannelRepository struct {
	mock.Mock
}

func (m *MockChannelRepository) Create(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockChannelRepository) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *MockChannelRepository) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	args := m.Called(ctx, userID, channelType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *MockChannelRepository) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return []*model.AlertChannel{}, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannel), args.Error(1)
}

func (m *MockChannelRepository) Update(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockChannelRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockChannelRepository) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	args := m.Called(ctx, userID, channelType, address)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	return args.Bool(0), args.Error(1)
}

func (m *MockChannelRepository) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	args := m.Called(ctx, id, failureCount)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockChannelRepository) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, channelID, limit)
	if args.Get(0) == nil {
		return []*model.DeliveryAttempt{}, args.Error(1)
	}
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *MockChannelRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockChannelRepository) IncrementFailureCount(ctx context.Context, id string) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

func (m *MockChannelRepository) DisableChannel(ctx context.Context, id string, reason string) error {
	args := m.Called(ctx, id, reason)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockChannelRepository) SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error {
	args := m.Called(ctx, ruleID, priorities)
	return args.Error(0)
}

func (m *MockChannelRepository) GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannelPriority), args.Error(1)
}

// Test functions

func TestNewChannelServiceAdapter(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	assert.NotNil(t, adapter)
	assert.Equal(t, mockRepo, adapter.channelRepo)
}

func TestChannelServiceAdapter_GetUserChannels(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()

	expectedChannels := []*model.AlertChannel{
		{ID: uuid.New(), UserID: userID, Type: model.AlertChannelTypeEmail},
		{ID: uuid.New(), UserID: userID, Type: model.AlertChannelTypeTelegram},
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return(expectedChannels, nil)

	channels, err := adapter.GetUserChannels(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, channels, 2)
	assert.Equal(t, expectedChannels[0].ID, channels[0].ID)
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_GetUserChannels_Error(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, assert.AnError)

	channels, err := adapter.GetUserChannels(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, channels)
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_CreateNew(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()

	// Mock empty current channels
	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	newChannel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(false, nil)
	mockRepo.On("Create", ctx, newChannel).Return(nil)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_ExistingChannel(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	existingChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{existingChannel}, nil)

	updatedChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: false,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(true, nil)
	// Update is NOT called because same ID and same address - it's skipped

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{updatedChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_DeleteOld(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	existingChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{existingChannel}, nil)
	mockRepo.On("Delete", ctx, channelID.String()).Return(nil)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_ErrorGettingCurrent(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, assert.AnError)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current channels")
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_ErrorCheckingDuplicate(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	newChannel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(false, assert.AnError)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for duplicate channels")
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_ErrorCreating(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	newChannel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(false, nil)
	mockRepo.On("Create", ctx, newChannel).Return(assert.AnError)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_ErrorUpdating(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	// First call returns empty channels
	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	newChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(false, nil)
	mockRepo.On("Create", ctx, newChannel).Return(assert.AnError)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_UpdateUserChannels_ErrorDeleting(t *testing.T) {
	mockRepo := &MockChannelRepository{}
	adapter := NewChannelServiceAdapter(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	existingChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{existingChannel}, nil)
	mockRepo.On("Delete", ctx, channelID.String()).Return(assert.AnError)

	err := adapter.UpdateUserChannels(ctx, userID, []*model.AlertChannel{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete channel")
	mockRepo.AssertExpectations(t)
}

func TestChannelServiceAdapter_getChannelAddress_Email(t *testing.T) {
	adapter := NewChannelServiceAdapter(&MockChannelRepository{})

	emailChannel := &model.AlertChannel{
		Type: model.AlertChannelTypeEmail,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	address := adapter.getChannelAddress(emailChannel)

	assert.Equal(t, "test@example.com", address)
}

func TestChannelServiceAdapter_getChannelAddress_Telegram(t *testing.T) {
	adapter := NewChannelServiceAdapter(&MockChannelRepository{})

	telegramChannel := &model.AlertChannel{
		Type: model.AlertChannelTypeTelegram,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "@testuser",
		},
	}

	address := adapter.getChannelAddress(telegramChannel)

	assert.Equal(t, "@testuser", address)
}

func TestChannelServiceAdapter_getChannelAddress_Webhook(t *testing.T) {
	adapter := NewChannelServiceAdapter(&MockChannelRepository{})

	webhookChannel := &model.AlertChannel{
		Type: model.AlertChannelTypeWebhook,
		WebhookConfig: &model.WebhookChannelConfig{
			URL: "https://example.com/webhook",
		},
	}

	address := adapter.getChannelAddress(webhookChannel)

	assert.Equal(t, "https://example.com/webhook", address)
}

func TestChannelServiceAdapter_getChannelAddress_NoConfig(t *testing.T) {
	adapter := NewChannelServiceAdapter(&MockChannelRepository{})

	channel := &model.AlertChannel{
		Type: model.AlertChannelTypeEmail,
	}

	address := adapter.getChannelAddress(channel)

	assert.Equal(t, "", address)
}

func TestChannelServiceAdapter_getChannelAddress_NilConfig(t *testing.T) {
	adapter := NewChannelServiceAdapter(&MockChannelRepository{})

	channel := &model.AlertChannel{
		Type:        model.AlertChannelTypeEmail,
		EmailConfig: nil,
	}

	address := adapter.getChannelAddress(channel)

	assert.Equal(t, "", address)
}
