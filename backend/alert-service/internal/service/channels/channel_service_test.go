package channels

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/alert-service/internal/model"
)

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

func TestNewChannelService(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.channelRepo)
	assert.Equal(t, 10, service.cfg.ChannelLimitFree)
}

func TestChannelService_GetUserChannels(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	expectedChannels := []*model.AlertChannel{
		{
			ID:      uuid.New(),
			UserID:  userID,
			Type:    model.AlertChannelTypeEmail,
			Enabled: true,
		},
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return(expectedChannels, nil)

	channels, err := service.GetUserChannels(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, channels, 1)
	assert.Equal(t, expectedChannels[0].ID, channels[0].ID)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_GetUserChannels_Error(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, assert.AnError)

	channels, err := service.GetUserChannels(ctx, userID)

	assert.Error(t, err)
	assert.Len(t, channels, 0) // Return empty slice on error, not nil
	mockRepo.AssertExpectations(t)
}

func TestChannelService_CreateChannel(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, channel.UserID.String(), channel.Type, "test@example.com").Return(false, nil)
	mockRepo.On("CountByUserID", ctx, channel.UserID.String()).Return(5, nil)
	mockRepo.On("Create", ctx, channel).Return(nil)

	resultChannel, err := service.CreateChannel(ctx, channel)

	assert.NoError(t, err)
	assert.Equal(t, channel.ID, resultChannel.ID)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_CreateChannel_Error(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
	}

	mockRepo.On("ExistsDuplicate", ctx, channel.UserID.String(), channel.Type, "").Return(false, nil)
	mockRepo.On("CountByUserID", ctx, channel.UserID.String()).Return(5, nil)
	mockRepo.On("Create", ctx, channel).Return(assert.AnError)

	resultChannel, err := service.CreateChannel(ctx, channel)

	assert.Error(t, err)
	assert.Nil(t, resultChannel)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_Success(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()
	verificationCode := "123456"

	channel := &model.AlertChannel{
		ID:       channelID,
		Enabled:  true,
		Verified: false,
		Status:   model.AlertChannelStatusUnverified,
	}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.AlertChannel")).Return(nil)

	err := service.VerifyChannel(ctx, channelID, verificationCode)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_AlreadyVerified(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()
	verificationCode := "123456"

	channel := &model.AlertChannel{
		ID:       channelID,
		Enabled:  true,
		Verified: true, // Already verified
		Status:   model.AlertChannelStatusActive,
	}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)

	err := service.VerifyChannel(ctx, channelID, verificationCode)

	assert.NoError(t, err)
	// Should not call Update since channel is already verified
	mockRepo.AssertNotCalled(t, "Update")
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_InvalidCode(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()
	verificationCode := "" // Empty code

	channel := &model.AlertChannel{
		ID:       channelID,
		Enabled:  true,
		Verified: false,
	}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)

	err := service.VerifyChannel(ctx, channelID, verificationCode)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidVerificationCode, err)
	mockRepo.AssertNotCalled(t, "Update")
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_NotFound(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()
	verificationCode := "123456"

	mockRepo.On("GetByID", ctx, channelID.String()).Return(nil, assert.AnError)

	err := service.VerifyChannel(ctx, channelID, verificationCode)

	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "Update")
	mockRepo.AssertExpectations(t)
}

func TestChannelService_DeleteChannel(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()
	channel := &model.AlertChannel{ID: channelID, UserID: userID}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)
	mockRepo.On("Delete", ctx, channelID.String()).Return(nil)

	err := service.DeleteChannel(ctx, userID, channelID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_DeleteChannel_Forbidden(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	channelID := uuid.New()
	channel := &model.AlertChannel{ID: channelID, UserID: ownerID}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)

	err := service.DeleteChannel(ctx, otherUserID, channelID)

	assert.ErrorIs(t, err, model.ErrForbidden)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_DeleteChannel_Error(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()
	channel := &model.AlertChannel{ID: channelID, UserID: userID}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)
	mockRepo.On("Delete", ctx, channelID.String()).Return(assert.AnError)

	err := service.DeleteChannel(ctx, userID, channelID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_NewChannel(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	// No current channels
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

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_UpdateExisting(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	channelID := uuid.New()

	// Existing channel with same ID and same email
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

	// New channel with same ID but different email
	updatedChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: false, // Updated to disabled
		EmailConfig: &model.EmailChannelConfig{
			Email: "new@example.com", // Different email
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "new@example.com").Return(false, nil)
	mockRepo.On("Update", ctx, updatedChannel).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{updatedChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_DeleteOld(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	channelID := uuid.New()

	// Existing channel
	existingChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "old@example.com",
		},
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{existingChannel}, nil)

	// Empty new channels list - should delete old one
	mockRepo.On("Delete", ctx, channelID.String()).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_Error(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, assert.AnError)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{})

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_DuplicateChannel_Skip(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	// Existing channel with same email
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

	// New channel with same ID and same email (duplicate)
	newChannel := &model.AlertChannel{
		ID:      channelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com", // Same email as existing
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(true, nil)
	// No Update should be called since the logic continues (skips) when it's the same channel with same address

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_DuplicateChannel_DifferentID(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()
	existingChannelID := uuid.New()
	newChannelID := uuid.New()

	// Existing channel
	existingChannel := &model.AlertChannel{
		ID:      existingChannelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{existingChannel}, nil)

	// New channel with different ID but same email (should create/update)
	newChannel := &model.AlertChannel{
		ID:      newChannelID,
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com", // Same email
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com").Return(true, nil)
	// Since it's a different channel ID, it should still try to create/update
	mockRepo.On("Create", ctx, newChannel).Return(nil)
	// The old channel should be deleted since it's not in the new list
	mockRepo.On("Delete", ctx, existingChannelID.String()).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_ExistsDuplicateError(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

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

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{newChannel})

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_MultipleChannels(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()
	channelID1 := uuid.New()
	channelID2 := uuid.New()

	// Existing channels
	existingChannels := []*model.AlertChannel{
		{
			ID:      channelID1,
			UserID:  userID,
			Type:    model.AlertChannelTypeEmail,
			Enabled: true,
			EmailConfig: &model.EmailChannelConfig{
				Email: "email1@example.com",
			},
		},
	}

	mockRepo.On("ListByUserID", ctx, userID.String()).Return(existingChannels, nil)

	// New channels - keep existing, add new one
	newChannels := []*model.AlertChannel{
		{
			ID:      channelID1,
			UserID:  userID,
			Type:    model.AlertChannelTypeEmail,
			Enabled: false, // Updated
			EmailConfig: &model.EmailChannelConfig{
				Email: "email1@example.com",
			},
		},
		{
			ID:      channelID2,
			UserID:  userID,
			Type:    model.AlertChannelTypeTelegram,
			Enabled: true,
			TelegramConfig: &model.TelegramChannelConfig{
				ChatID: "123456789",
			},
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "email1@example.com").Return(false, nil)
	mockRepo.On("Update", ctx, newChannels[0]).Return(nil)
	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeTelegram, "123456789").Return(false, nil)
	mockRepo.On("Create", ctx, newChannels[1]).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, newChannels)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_GetByIDError(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()

	mockRepo.On("GetByID", ctx, channelID.String()).Return(nil, assert.AnError)

	err := service.VerifyChannel(ctx, channelID, "123456")

	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "Update")
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_UpdateError(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()

	channel := &model.AlertChannel{
		ID:       channelID,
		Verified: false,
	}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.AlertChannel")).Return(assert.AnError)

	err := service.VerifyChannel(ctx, channelID, "123456")

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_VerifiedAlready(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()

	channel := &model.AlertChannel{
		ID:       channelID,
		Verified: true, // Already verified
		Status:   model.AlertChannelStatusActive,
	}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)

	err := service.VerifyChannel(ctx, channelID, "any_code")

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Update") // Should not update if already verified
	mockRepo.AssertExpectations(t)
}

func TestChannelService_VerifyChannel_EmptyCode(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channelID := uuid.New()

	channel := &model.AlertChannel{
		ID:       channelID,
		Verified: false,
	}

	mockRepo.On("GetByID", ctx, channelID.String()).Return(channel, nil)

	err := service.VerifyChannel(ctx, channelID, "") // Empty verification code

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidVerificationCode, err)
	mockRepo.AssertNotCalled(t, "Update") // Should not update with invalid code
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_TelegramChannel(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	telegramChannel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    model.AlertChannelTypeTelegram,
		Enabled: true,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "123456789",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeTelegram, "123456789").Return(false, nil)
	mockRepo.On("Create", ctx, telegramChannel).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{telegramChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_WebhookChannel(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	webhookChannel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    model.AlertChannelTypeWebhook,
		Enabled: true,
		WebhookConfig: &model.WebhookChannelConfig{
			URL: "https://example.com/webhook",
		},
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeWebhook, "https://example.com/webhook").Return(false, nil)
	mockRepo.On("Create", ctx, webhookChannel).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{webhookChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_EmailChannel_NilConfig(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	// Email channel with nil config (edge case)
	emailChannel := &model.AlertChannel{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        model.AlertChannelTypeEmail,
		Enabled:     true,
		EmailConfig: nil, // Nil config
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "").Return(false, nil)
	mockRepo.On("Create", ctx, emailChannel).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{emailChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_TelegramChannel_NilConfig(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	// Telegram channel with nil config (edge case)
	telegramChannel := &model.AlertChannel{
		ID:             uuid.New(),
		UserID:         userID,
		Type:           model.AlertChannelTypeTelegram,
		Enabled:        true,
		TelegramConfig: nil, // Nil config
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeTelegram, "").Return(false, nil)
	mockRepo.On("Create", ctx, telegramChannel).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{telegramChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_WebhookChannel_NilConfig(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	// Webhook channel with nil config (edge case)
	webhookChannel := &model.AlertChannel{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          model.AlertChannelTypeWebhook,
		Enabled:       true,
		WebhookConfig: nil, // Nil config
	}

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeWebhook, "").Return(false, nil)
	mockRepo.On("Create", ctx, webhookChannel).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{webhookChannel})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_UpdateUserChannels_MixedChannelTypes(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	userID := uuid.New()
	emailChannelID := uuid.New()
	telegramChannelID := uuid.New()
	webhookChannelID := uuid.New()

	mockRepo.On("ListByUserID", ctx, userID.String()).Return([]*model.AlertChannel{}, nil)

	// Mix of all channel types
	channels := []*model.AlertChannel{
		{
			ID:      emailChannelID,
			UserID:  userID,
			Type:    model.AlertChannelTypeEmail,
			Enabled: true,
			EmailConfig: &model.EmailChannelConfig{
				Email: "email@example.com",
			},
		},
		{
			ID:      telegramChannelID,
			UserID:  userID,
			Type:    model.AlertChannelTypeTelegram,
			Enabled: true,
			TelegramConfig: &model.TelegramChannelConfig{
				ChatID: "telegram_chat_id",
			},
		},
		{
			ID:      webhookChannelID,
			UserID:  userID,
			Type:    model.AlertChannelTypeWebhook,
			Enabled: true,
			WebhookConfig: &model.WebhookChannelConfig{
				URL: "https://webhook.example.com",
			},
		},
	}

	// Set up expectations for all channel types
	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeEmail, "email@example.com").Return(false, nil)
	mockRepo.On("Create", ctx, channels[0]).Return(nil)

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeTelegram, "telegram_chat_id").Return(false, nil)
	mockRepo.On("Create", ctx, channels[1]).Return(nil)

	mockRepo.On("ExistsDuplicate", ctx, userID.String(), model.AlertChannelTypeWebhook, "https://webhook.example.com").Return(false, nil)
	mockRepo.On("Create", ctx, channels[2]).Return(nil)

	err := service.UpdateUserChannels(ctx, userID, channels)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestChannelService_CreateChannel_GuestForbidden проверяет, что GUEST не может создать канал (uc_02_01_18)
func TestChannelService_CreateChannel_GuestForbidden(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.WithValue(context.Background(), auth.RoleKey, auth.RoleGuest)
	channel := &model.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   model.AlertChannelTypeEmail,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
	}

	_, err := service.CreateChannel(ctx, channel)

	assert.ErrorIs(t, err, model.ErrGuestNotAllowed)
	mockRepo.AssertNotCalled(t, "Create")
}

// TestChannelService_UpdateUserChannels_GuestForbidden проверяет, что GUEST не может обновить каналы
func TestChannelService_UpdateUserChannels_GuestForbidden(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.WithValue(context.Background(), auth.RoleKey, auth.RoleGuest)
	userID := uuid.New()

	err := service.UpdateUserChannels(ctx, userID, []*model.AlertChannel{})

	assert.ErrorIs(t, err, model.ErrGuestNotAllowed)
	mockRepo.AssertNotCalled(t, "ListByUserID")
}

// TestChannelService_WithAuditService проверяет установку auditService
func TestChannelService_WithAuditService(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{})

	result := service.WithAuditService(nil)

	assert.Equal(t, service, result)
}

// TestChannelService_WithLogger проверяет установку логгера
func TestChannelService_WithLogger(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{})

	result := service.WithLogger(nil)

	assert.Equal(t, service, result)
}

// TestChannelService_ValidateChannel_TelegramInvalidChatID проверяет валидацию неверного chatID
func TestChannelService_ValidateChannel_TelegramInvalidChatID(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   model.AlertChannelTypeTelegram,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "invalid-chat-id!",
		},
	}

	mockRepo.On("CountByUserID", ctx, channel.UserID.String()).Return(0, nil)

	_, err := service.CreateChannel(ctx, channel)

	assert.ErrorIs(t, err, model.ErrInvalidChatID)
}

// TestChannelService_ValidateChannel_WebhookInvalidURL проверяет валидацию неверного URL
func TestChannelService_ValidateChannel_WebhookInvalidURL(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   model.AlertChannelTypeWebhook,
		WebhookConfig: &model.WebhookChannelConfig{
			URL: "not-a-valid-url",
		},
	}

	mockRepo.On("CountByUserID", ctx, channel.UserID.String()).Return(0, nil)

	_, err := service.CreateChannel(ctx, channel)

	assert.ErrorIs(t, err, model.ErrInvalidURL)
}

// TestChannelService_ValidateChannel_EmailInvalidAddress проверяет валидацию неверного email.
func TestChannelService_ValidateChannel_EmailInvalidAddress(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   model.AlertChannelTypeEmail,
		EmailConfig: &model.EmailChannelConfig{
			Email: "not-an-email",
		},
	}

	_, err := service.CreateChannel(ctx, channel)

	assert.ErrorIs(t, err, model.ErrInvalidEmail)
	mockRepo.AssertNotCalled(t, "CountByUserID")
}

// TestChannelService_ValidateChannel_WebhookInvalidMethod проверяет валидацию HTTP-метода webhook.
func TestChannelService_ValidateChannel_WebhookInvalidMethod(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   model.AlertChannelTypeWebhook,
		WebhookConfig: &model.WebhookChannelConfig{
			URL:    "https://example.com/webhook",
			Method: "INVALID",
		},
	}

	_, err := service.CreateChannel(ctx, channel)

	assert.ErrorIs(t, err, model.ErrInvalidFormat)
	mockRepo.AssertNotCalled(t, "CountByUserID")
}

// TestChannelService_ValidateChannel_NameTooLong проверяет ограничение длины имени канала.
func TestChannelService_ValidateChannel_NameTooLong(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Name:   strings.Repeat("a", MaxChannelNameLength+1),
		Type:   model.AlertChannelTypeTelegram,
		TelegramConfig: &model.TelegramChannelConfig{
			ChatID: "-1001234567890",
		},
	}

	_, err := service.CreateChannel(ctx, channel)

	assert.ErrorIs(t, err, model.ErrFieldTooLong)
	mockRepo.AssertNotCalled(t, "CountByUserID")
}

// TestChannelService_CreateChannel_CountError проверяет обработку ошибки CountByUserID
func TestChannelService_CreateChannel_CountError(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Type:        model.AlertChannelTypeEmail,
		EmailConfig: &model.EmailChannelConfig{Email: "test@example.com"},
	}

	mockRepo.On("CountByUserID", ctx, channel.UserID.String()).Return(0, assert.AnError)

	_, err := service.CreateChannel(ctx, channel)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

// TestChannelService_CreateChannel_DuplicateCheckError проверяет ошибку при проверке дубликата
func TestChannelService_CreateChannel_DuplicateCheckError(t *testing.T) {
	mockRepo := &MockAlertChannelRepository{}
	service := NewChannelService(mockRepo, ChannelServiceConfig{ChannelLimitFree: 10})

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Type:        model.AlertChannelTypeEmail,
		EmailConfig: &model.EmailChannelConfig{Email: "test@example.com"},
	}

	mockRepo.On("CountByUserID", ctx, channel.UserID.String()).Return(0, nil)
	mockRepo.On("ExistsDuplicate", ctx, channel.UserID.String(), channel.Type, "test@example.com").Return(false, assert.AnError)

	_, err := service.CreateChannel(ctx, channel)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}
