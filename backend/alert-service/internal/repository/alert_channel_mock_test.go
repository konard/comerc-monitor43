package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MockAlertChannelRepository implements AlertChannelRepository interface for testing
type MockAlertChannelRepository struct {
	channels      []model.AlertChannel
	channelsMutex sync.RWMutex
}

// NewMockAlertChannelRepository creates a new mock alert channel repository
func NewMockAlertChannelRepository() *MockAlertChannelRepository {
	return &MockAlertChannelRepository{
		channels:      make([]model.AlertChannel, 0),
		channelsMutex: sync.RWMutex{},
	}
}

// Create creates a new alert channel
func (m *MockAlertChannelRepository) Create(ctx context.Context, channel *model.AlertChannel) error {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()

	m.channels = append(m.channels, *channel)
	return nil
}

// GetByID retrieves an alert channel by ID
func (m *MockAlertChannelRepository) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	m.channelsMutex.RLock()
	defer m.channelsMutex.RUnlock()

	for i, channel := range m.channels {
		if channel.ID.String() == id {
			return &m.channels[i], nil
		}
	}
	return nil, model.ErrAlertChannelNotFound
}

// GetByUserIDAndType retrieves an alert channel by user and type
func (m *MockAlertChannelRepository) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	m.channelsMutex.RLock()
	defer m.channelsMutex.RUnlock()

	for i, channel := range m.channels {
		if channel.UserID.String() == userID && channel.Type == channelType {
			return &m.channels[i], nil
		}
	}
	return nil, model.ErrAlertChannelNotFound
}

// ListByUserID retrieves all alert channels for a user
func (m *MockAlertChannelRepository) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	m.channelsMutex.RLock()
	defer m.channelsMutex.RUnlock()

	var results []*model.AlertChannel
	for i := range m.channels {
		if m.channels[i].UserID.String() == userID {
			results = append(results, &m.channels[i])
		}
	}

	if len(results) == 0 {
		return nil, model.ErrAlertChannelNotFound
	}
	return results, nil
}

// Update updates an alert channel
func (m *MockAlertChannelRepository) Update(ctx context.Context, channel *model.AlertChannel) error {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()

	for i, c := range m.channels {
		if c.ID == channel.ID {
			m.channels[i] = *channel
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

// Delete removes an alert channel
func (m *MockAlertChannelRepository) Delete(ctx context.Context, id string) error {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()

	for i, c := range m.channels {
		if c.ID.String() == id {
			m.channels = append(m.channels[:i], m.channels[i+1:]...)
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

// getChannelAddress extracts the address from an AlertChannel based on its type
func getChannelAddress(channel *model.AlertChannel) string {
	switch channel.Type {
	case model.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			return channel.EmailConfig.Email
		}
	case model.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			return channel.TelegramConfig.ChatID
		}
	case model.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			return channel.WebhookConfig.URL
		}
	}
	return ""
}

// ExistsDuplicate checks if a duplicate channel exists
func (m *MockAlertChannelRepository) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	m.channelsMutex.RLock()
	defer m.channelsMutex.RUnlock()

	for _, channel := range m.channels {
		if channel.UserID.String() == userID && channel.Type == channelType && getChannelAddress(&channel) == address {
			return true, nil
		}
	}
	return false, nil
}

// MarkAsFailed marks a channel as failed
func (m *MockAlertChannelRepository) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()

	for i, c := range m.channels {
		if c.ID.String() == id {
			m.channels[i].FailureCount = failureCount
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

// ListPendingForChannel returns pending delivery attempts for a channel
func (m *MockAlertChannelRepository) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	return nil, nil
}

// CountByUserID returns the number of channels for a user
func (m *MockAlertChannelRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	m.channelsMutex.RLock()
	defer m.channelsMutex.RUnlock()

	count := 0
	for _, channel := range m.channels {
		if channel.UserID.String() == userID {
			count++
		}
	}
	return count, nil
}

// IncrementFailureCount increments the failure count for a channel
func (m *MockAlertChannelRepository) IncrementFailureCount(ctx context.Context, id string) (int, error) {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()

	for i, c := range m.channels {
		if c.ID.String() == id {
			m.channels[i].FailureCount++
			return m.channels[i].FailureCount, nil
		}
	}
	return 0, model.ErrAlertChannelNotFound
}

// DisableChannel disables a notification channel
func (m *MockAlertChannelRepository) DisableChannel(ctx context.Context, id string, reason string) error {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()

	for i, c := range m.channels {
		if c.ID.String() == id {
			m.channels[i].Enabled = false
			m.channels[i].Status = model.AlertChannelStatusFailed
			return nil
		}
	}
	return model.ErrAlertChannelNotFound
}

// Clear clears all channels for testing
func (m *MockAlertChannelRepository) Clear() {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()
	m.channels = make([]model.AlertChannel, 0)
}

// AddChannel adds a predefined channel for testing
func (m *MockAlertChannelRepository) AddChannel(channel model.AlertChannel) {
	m.channelsMutex.Lock()
	defer m.channelsMutex.Unlock()
	m.channels = append(m.channels, channel)
}

// TestMockAlertChannelRepository_Create_Success проверяет успешное создание канала
func TestMockAlertChannelRepository_Create_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, channel)
	assert.NoError(t, err)
}

// TestMockAlertChannelRepository_GetByID_Success проверяет успешное получение канала
func TestMockAlertChannelRepository_GetByID_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add channel to repository
	repo.AddChannel(*channel)

	result, err := repo.GetByID(ctx, channel.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, channel.ID, result.ID)
}

// TestMockAlertChannelRepository_GetByID_NotFound проверяет случай когда канал не найден
func TestMockAlertChannelRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	result, err := repo.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, model.ErrAlertChannelNotFound))
}

// TestMockAlertChannelRepository_GetByUserIDAndType_Success проверяет успешное получение канала по типу
func TestMockAlertChannelRepository_GetByUserIDAndType_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add channel to repository
	repo.AddChannel(*channel)

	result, err := repo.GetByUserIDAndType(ctx, channel.UserID.String(), model.AlertChannelTypeEmail)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, channel.ID, result.ID)
}

// TestMockAlertChannelRepository_ListByUserID_Success проверяет успешное получение списка каналов
func TestMockAlertChannelRepository_ListByUserID_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	userID := uuid.New()

	// Add multiple channels for user
	for i := 0; i < 3; i++ {
		channel := &model.AlertChannel{
			ID:      uuid.New(),
			UserID:  userID,
			Type:    model.AlertChannelTypeEmail,
			Status:  model.AlertChannelStatusActive,
			Enabled: true,
			EmailConfig: &model.EmailChannelConfig{
				Email: fmt.Sprintf("test%d@example.com", i),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.AddChannel(*channel)
	}

	channels, err := repo.ListByUserID(ctx, userID.String())
	assert.NoError(t, err)
	assert.Len(t, channels, 3)
}

// TestMockAlertChannelRepository_ListByUserID_NotFound проверяет отсутствие каналов для пользователя
func TestMockAlertChannelRepository_ListByUserID_NotFound(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	channels, err := repo.ListByUserID(ctx, "non-existent-user-id")
	assert.Error(t, err)
	assert.Nil(t, channels)
	assert.True(t, errors.Is(err, model.ErrAlertChannelNotFound))
}

// TestMockAlertChannelRepository_Update_Success проверяет успешное обновление канала
func TestMockAlertChannelRepository_Update_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	originalChannel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	repo.AddChannel(*originalChannel)

	// Update channel status and add failure count
	updatedChannel := &model.AlertChannel{
		ID:           originalChannel.ID,
		UserID:       originalChannel.UserID,
		Type:         originalChannel.Type,
		Status:       model.AlertChannelStatusFailed,
		Enabled:      false,
		FailureCount: 3,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: originalChannel.CreatedAt,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(ctx, updatedChannel)
	assert.NoError(t, err)

	// Verify update
	result, err := repo.GetByID(ctx, originalChannel.ID.String())
	assert.NoError(t, err)
	assert.False(t, result.Enabled)
	assert.Equal(t, 3, result.FailureCount)
}

// TestMockAlertChannelRepository_Delete_Success проверяет успешное удаление канала
func TestMockAlertChannelRepository_Delete_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	repo.AddChannel(*channel)

	err := repo.Delete(ctx, channel.ID.String())
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, channel.ID.String())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertChannelNotFound))
}

// TestMockAlertChannelRepository_Delete_NotFound проверяет удаление несуществующего канала
func TestMockAlertChannelRepository_Delete_NotFound(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	err := repo.Delete(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertChannelNotFound))
}

// TestMockAlertChannelRepository_ExistsDuplicate_Success проверяет обнаружение дубликата
func TestMockAlertChannelRepository_ExistsDuplicate_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	userID := uuid.New()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	repo.AddChannel(*channel)

	exists, err := repo.ExistsDuplicate(ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com")
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestMockAlertChannelRepository_ExistsDuplicate_NotExists проверяет отсутствие дубликата
func TestMockAlertChannelRepository_ExistsDuplicate_NotExists(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	userID := uuid.New()
	exists, err := repo.ExistsDuplicate(ctx, userID.String(), model.AlertChannelTypeEmail, "test@example.com")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestMockAlertChannelRepository_MarkAsFailed_Success проверяет отметку канала как неудачного
func TestMockAlertChannelRepository_MarkAsFailed_Success(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	channel := &model.AlertChannel{
		ID:      uuid.New(),
		UserID:  uuid.New(),
		Type:    model.AlertChannelTypeEmail,
		Status:  model.AlertChannelStatusActive,
		Enabled: true,
		EmailConfig: &model.EmailChannelConfig{
			Email: "test@example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	repo.AddChannel(*channel)

	err := repo.MarkAsFailed(ctx, channel.ID.String(), 5)
	assert.NoError(t, err)

	// Verify failure count
	result, err := repo.GetByID(ctx, channel.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, 5, result.FailureCount)
}

// TestMockAlertChannelRepository_MarkAsFailed_NotFound проверяет отметку несуществующего канала
func TestMockAlertChannelRepository_MarkAsFailed_NotFound(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	err := repo.MarkAsFailed(ctx, "non-existent-id", 5)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrAlertChannelNotFound))
}

// TestMockAlertChannelRepository_Concurrency проверяет конкурентный доступ
func TestMockAlertChannelRepository_Concurrency(t *testing.T) {
	repo := NewMockAlertChannelRepository()

	ctx := context.Background()
	userID := uuid.New()

	// Test concurrent creates
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			channel := &model.AlertChannel{
				ID:      uuid.New(),
				UserID:  userID,
				Type:    model.AlertChannelTypeEmail,
				Status:  model.AlertChannelStatusActive,
				Enabled: true,
				EmailConfig: &model.EmailChannelConfig{
					Email: fmt.Sprintf("test%d@example.com", id),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			require.NoError(t, repo.Create(ctx, channel))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all channels were created
	channels, err := repo.ListByUserID(ctx, userID.String())
	assert.NoError(t, err)
	assert.Len(t, channels, 10)
}
