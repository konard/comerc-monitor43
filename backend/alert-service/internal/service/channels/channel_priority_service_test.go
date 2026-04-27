package channels

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// mockAlertChannelRepositoryForPriority - mock для AlertChannelRepository в тестах приоритетов
type mockAlertChannelRepositoryForPriority struct {
	mock.Mock
}

func (m *mockAlertChannelRepositoryForPriority) Create(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *mockAlertChannelRepositoryForPriority) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	args := m.Called(ctx, userID, channelType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertChannel), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return []*model.AlertChannel{}, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannel), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	args := m.Called(ctx, userID, channelType, address)
	return args.Bool(0), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	args := m.Called(ctx, id, failureCount)
	return args.Error(0)
}

func (m *mockAlertChannelRepositoryForPriority) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	args := m.Called(ctx, channelID, limit)
	if args.Get(0) == nil {
		return []*model.DeliveryAttempt{}, args.Error(1)
	}
	return args.Get(0).([]*model.DeliveryAttempt), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) CountByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) Update(ctx context.Context, channel *model.AlertChannel) error {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *mockAlertChannelRepositoryForPriority) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *mockAlertChannelRepositoryForPriority) VerifyChannel(ctx context.Context, channelID string, code string) error {
	args := m.Called(ctx, channelID, code)
	return args.Error(0)
}

func (m *mockAlertChannelRepositoryForPriority) IncrementFailureCount(ctx context.Context, channelID string) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Int(0), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) DisableChannel(ctx context.Context, channelID string, reason string) error {
	args := m.Called(ctx, channelID, reason)
	return args.Error(0)
}

func (m *mockAlertChannelRepositoryForPriority) GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return []*model.AlertChannelPriority{}, args.Error(1)
	}
	return args.Get(0).([]*model.AlertChannelPriority), args.Error(1)
}

func (m *mockAlertChannelRepositoryForPriority) SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error {
	args := m.Called(ctx, ruleID, priorities)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

// TestNewChannelPriorityService проверяет создание сервиса приоритетов
func TestNewChannelPriorityService(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}

	service := NewChannelPriorityService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.channelRepo)
	assert.Nil(t, service.tracer)
}

// TestWithChannelPriorityTracer проверяет установку tracer
func TestWithChannelPriorityTracer(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	assert.NotNil(t, service)
	assert.Equal(t, tracer, service.tracer)
}

// TestChannelPriorityService_SetPriorities_Success проверяет успешную установку приоритетов
func TestChannelPriorityService_SetPriorities_Success(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()
	channelIDs := []uuid.UUID{uuid.New(), uuid.New()}
	priorities := []int{1, 2}

	mockRepo.On("SetChannelPriorities", mock.Anything, ruleID.String(), mock.AnythingOfType("[]model.AlertChannelPriority")).Return(nil)

	err := service.SetPriorities(ctx, ruleID, channelIDs, priorities)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestChannelPriorityService_SetPriorities_LengthMismatch проверяет ошибку при несовпадении длин
func TestChannelPriorityService_SetPriorities_LengthMismatch(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()
	channelIDs := []uuid.UUID{uuid.New(), uuid.New()}
	priorities := []int{1} // Разная длина

	err := service.SetPriorities(ctx, ruleID, channelIDs, priorities)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel_ids and priorities must have the same length")
}

// TestChannelPriorityService_SetPriorities_RepoError проверяет ошибку репозитория
func TestChannelPriorityService_SetPriorities_RepoError(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()
	channelIDs := []uuid.UUID{uuid.New()}
	priorities := []int{1}

	mockRepo.On("SetChannelPriorities", mock.Anything, ruleID.String(), mock.AnythingOfType("[]model.AlertChannelPriority")).Return(assert.AnError)

	err := service.SetPriorities(ctx, ruleID, channelIDs, priorities)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set channel priorities")
}

// TestChannelPriorityService_GetPrioritizedChannels_Success проверяет получение приоритетных каналов
func TestChannelPriorityService_GetPrioritizedChannels_Success(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()

	priority1 := &model.AlertChannelPriority{
		ID:             uuid.New(),
		AlertRuleID:    ruleID,
		AlertChannelID: uuid.New(),
		Priority:       1,
		CreatedAt:      time.Now(),
	}
	priority2 := &model.AlertChannelPriority{
		ID:             uuid.New(),
		AlertRuleID:    ruleID,
		AlertChannelID: uuid.New(),
		Priority:       2,
		CreatedAt:      time.Now(),
	}

	mockRepo.On("GetChannelPriorities", mock.Anything, ruleID.String()).Return([]*model.AlertChannelPriority{priority1, priority2}, nil)

	priorities, err := service.GetPrioritizedChannels(ctx, ruleID)

	assert.NoError(t, err)
	assert.Len(t, priorities, 2)
	assert.Equal(t, 1, priorities[0].Priority)
	assert.Equal(t, 2, priorities[1].Priority)
	mockRepo.AssertExpectations(t)
}

// TestChannelPriorityService_GetPrioritizedChannels_RepoError проверяет ошибку репозитория
func TestChannelPriorityService_GetPrioritizedChannels_RepoError(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()

	mockRepo.On("GetChannelPriorities", mock.Anything, ruleID.String()).Return(nil, assert.AnError)

	priorities, err := service.GetPrioritizedChannels(ctx, ruleID)

	assert.Error(t, err)
	assert.Nil(t, priorities)
	assert.Contains(t, err.Error(), "failed to get prioritized channels")
}

// TestChannelPriorityService_GetPrioritizedChannels_Empty проверяет пустой список
func TestChannelPriorityService_GetPrioritizedChannels_Empty(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()

	mockRepo.On("GetChannelPriorities", mock.Anything, ruleID.String()).Return([]*model.AlertChannelPriority{}, nil)

	priorities, err := service.GetPrioritizedChannels(ctx, ruleID)

	assert.NoError(t, err)
	assert.Empty(t, priorities)
}

// TestChannelPriorityService_SetPriorities_WithTracer проверяет работу с tracer
func TestChannelPriorityService_SetPriorities_WithTracer(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()
	channelIDs := []uuid.UUID{uuid.New()}
	priorities := []int{1}

	mockRepo.On("SetChannelPriorities", mock.Anything, ruleID.String(), mock.AnythingOfType("[]model.AlertChannelPriority")).Return(nil)

	err := service.SetPriorities(ctx, ruleID, channelIDs, priorities)

	assert.NoError(t, err)
	// Tracer должен быть вызван для создания span
	assert.NotNil(t, service.tracer)
}

// TestChannelPriorityService_GetPrioritizedChannels_WithTracer проверяет GetPrioritizedChannels с tracer
func TestChannelPriorityService_GetPrioritizedChannels_WithTracer(t *testing.T) {
	t.Parallel()

	mockRepo := &mockAlertChannelRepositoryForPriority{}
	tracer := otel.GetTracerProvider().Tracer("test")

	service := NewChannelPriorityService(mockRepo)
	WithChannelPriorityTracer(tracer)(service)

	ctx := context.Background()
	ruleID := uuid.New()

	mockRepo.On("GetChannelPriorities", mock.Anything, ruleID.String()).Return([]*model.AlertChannelPriority{}, nil)

	priorities, err := service.GetPrioritizedChannels(ctx, ruleID)

	assert.NoError(t, err)
	assert.Empty(t, priorities)
	// Tracer должен быть вызван
	assert.NotNil(t, service.tracer)
}
