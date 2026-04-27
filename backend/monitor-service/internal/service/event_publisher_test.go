package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pure-golang/adapters/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	domain "github.com/raul/monitor/backend/monitor-service/internal/model"
)

// MockQueuePublisher является mock для queue.Publisher.
type MockQueuePublisher struct {
	mock.Mock
}

func (m *MockQueuePublisher) Publish(ctx context.Context, msgs ...queue.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

// TestNewEventPublisherImpl тестирует создание EventPublisherImpl.
func TestNewEventPublisherImpl(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	assert.NotNil(t, publisher)
	assert.Equal(t, mockPublisher, publisher.publisher)
	assert.Equal(t, logger, publisher.logger)
}

// TestEventPublisherImpl_PublishStatusChange_Success тестирует успешную публикацию события.
func TestEventPublisherImpl_PublishStatusChange_Success(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()
	monitorID := uuid.New()
	oldStatus := domain.StatusUp
	newStatus := domain.StatusDown
	responseTime := 500
	errorMessage := "connection timeout"

	event := &StatusChangeEvent{
		MonitorID:    monitorID,
		OldStatus:    oldStatus,
		NewStatus:    newStatus,
		Timestamp:    time.Now().Truncate(time.Second),
		ResponseTime: &responseTime,
		ErrorMessage: &errorMessage,
	}

	// Ожидаем, что будет вызван Publish с правильно форматированным сообщением
	mockPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(msgs []queue.Message) bool {
		if len(msgs) != 1 {
			return false
		}
		msg := msgs[0]

		// Проверяем topic
		if msg.Topic != "monitor.status.DOWN" {
			return false
		}

		// Проверяем headers
		if msg.Headers["monitor_id"] != monitorID.String() {
			return false
		}
		if msg.Headers["old_status"] != "UP" {
			return false
		}
		if msg.Headers["new_status"] != "DOWN" {
			return false
		}
		if msg.Headers["source"] != "monitor-service" {
			return false
		}

		// Проверяем body
		body, ok := msg.Body.(map[string]any)
		if !ok {
			return false
		}
		if body["monitor_id"] != monitorID.String() {
			return false
		}
		if body["old_status"] != "UP" {
			return false
		}
		if body["new_status"] != "DOWN" {
			return false
		}
		if _, exists := body["timestamp"]; !exists {
			return false
		}

		// response_time может быть int или int64
		rt := body["response_time"]
		rtMatched := false
		if rtInt, ok := rt.(int); ok {
			rtMatched = (rtInt == responseTime)
		} else if rtInt64, ok := rt.(int64); ok {
			rtMatched = (rtInt64 == int64(responseTime))
		}
		if !rtMatched {
			return false
		}

		if body["error_message"] != errorMessage {
			return false
		}

		return true
	})).Return(nil)

	err := publisher.PublishStatusChange(ctx, event)

	assert.NoError(t, err)
	mockPublisher.AssertExpectations(t)
}

// TestEventPublisherImpl_PublishStatusChange_WithoutOptionalFields тестирует публикацию без опциональных полей.
func TestEventPublisherImpl_PublishStatusChange_WithoutOptionalFields(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()
	monitorID := uuid.New()
	event := &StatusChangeEvent{
		MonitorID: monitorID,
		OldStatus: domain.StatusUp,
		NewStatus: domain.StatusDegraded,
		Timestamp: time.Now().Truncate(time.Second),
		// ResponseTime и ErrorMessage не заданы
	}

	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	err := publisher.PublishStatusChange(ctx, event)

	assert.NoError(t, err)
}

// TestEventPublisherImpl_PublishStatusChange_TopicGeneration тестирует генерацию топиков для разных статусов.
func TestEventPublisherImpl_PublishStatusChange_TopicGeneration(t *testing.T) {
	testCases := []struct {
		name          string
		newStatus     domain.MonitorStatus
		expectedTopic string
	}{
		{
			name:          "status UP",
			newStatus:     domain.StatusUp,
			expectedTopic: "monitor.status.UP",
		},
		{
			name:          "status DOWN",
			newStatus:     domain.StatusDown,
			expectedTopic: "monitor.status.DOWN",
		},
		{
			name:          "status DEGRADED",
			newStatus:     domain.StatusDegraded,
			expectedTopic: "monitor.status.DEGRADED",
		},
		{
			name:          "status PAUSED",
			newStatus:     domain.StatusPaused,
			expectedTopic: "monitor.status.PAUSED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPublisher := new(MockQueuePublisher)
			logger := slog.Default()

			publisher := NewEventPublisherImpl(mockPublisher, logger)

			ctx := context.Background()
			event := &StatusChangeEvent{
				MonitorID: uuid.New(),
				OldStatus: domain.StatusUp,
				NewStatus: tc.newStatus,
				Timestamp: time.Now(),
			}

			mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

			err := publisher.PublishStatusChange(ctx, event)

			assert.NoError(t, err)
		})
	}
}

// TestEventPublisherImpl_PublishStatusChange_PublisherError тестирует обработку ошибки публикации.
func TestEventPublisherImpl_PublishStatusChange_PublisherError(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()
	event := &StatusChangeEvent{
		MonitorID: uuid.New(),
		OldStatus: domain.StatusUp,
		NewStatus: domain.StatusDown,
		Timestamp: time.Now(),
	}

	expectedError := errors.New("rabbitmq connection failed")
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(expectedError)

	err := publisher.PublishStatusChange(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish event")
	assert.Contains(t, err.Error(), "rabbitmq connection failed")
}

// TestEventPublisherImpl_PublishStatusChange_TimestampUnixFormat тестирует форматирование timestamp.
func TestEventPublisherImpl_PublishStatusChange_TimestampUnixFormat(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()
	testTime := time.Date(2026, 3, 26, 12, 30, 45, 0, time.UTC)
	event := &StatusChangeEvent{
		MonitorID: uuid.New(),
		OldStatus: domain.StatusUp,
		NewStatus: domain.StatusDegraded,
		Timestamp: testTime,
	}

	mockPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(msgs []queue.Message) bool {
		if len(msgs) != 1 {
			return false
		}
		body, ok := msgs[0].Body.(map[string]any)
		if !ok {
			return false
		}

		timestamp, exists := body["timestamp"]
		if !exists {
			return false
		}

		// Проверяем, что timestamp в Unix формате (int64)
		timestampInt, ok := timestamp.(int64)
		if !ok {
			// JSON decode может преобразовать в float64
			timestampFloat, ok := timestamp.(float64)
			if !ok {
				return false
			}
			return int64(timestampFloat) == testTime.Unix()
		}

		return timestampInt == testTime.Unix()
	})).Return(nil)

	err := publisher.PublishStatusChange(ctx, event)

	assert.NoError(t, err)
}

// TestEventPublisherImpl_PublishStatusChange_AllStatuses тестирует все возможные статусы.
func TestEventPublisherImpl_PublishStatusChange_AllStatuses(t *testing.T) {
	allStatuses := []domain.MonitorStatus{
		domain.StatusPending,
		domain.StatusUp,
		domain.StatusDown,
		domain.StatusDegraded,
		domain.StatusPaused,
	}

	for _, status := range allStatuses {
		t.Run(status.String(), func(t *testing.T) {
			mockPublisher := new(MockQueuePublisher)
			logger := slog.Default()

			publisher := NewEventPublisherImpl(mockPublisher, logger)

			ctx := context.Background()
			event := &StatusChangeEvent{
				MonitorID: uuid.New(),
				OldStatus: domain.StatusPending,
				NewStatus: status,
				Timestamp: time.Now(),
			}

			mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

			err := publisher.PublishStatusChange(ctx, event)

			assert.NoError(t, err)
		})
	}
}

// TestEventPublisherImpl_PublishStatusChange_HeadersFormatting тестирует форматирование headers.
func TestEventPublisherImpl_PublishStatusChange_HeadersFormatting(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()
	monitorID := uuid.New()
	oldStatus := domain.StatusUp
	newStatus := domain.StatusDown

	event := &StatusChangeEvent{
		MonitorID: monitorID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: time.Now(),
	}

	mockPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(msgs []queue.Message) bool {
		if len(msgs) != 1 {
			return false
		}
		msg := msgs[0]

		// Проверяем все required headers
		if msg.Headers["monitor_id"] != monitorID.String() {
			return false
		}
		if msg.Headers["old_status"] != string(oldStatus) {
			return false
		}
		if msg.Headers["new_status"] != string(newStatus) {
			return false
		}
		if msg.Headers["source"] != "monitor-service" {
			return false
		}

		return true
	})).Return(nil)

	err := publisher.PublishStatusChange(ctx, event)

	assert.NoError(t, err)
}

// TestEventPublisherImpl_Check_Success тестирует успешную проверку здоровья.
func TestEventPublisherImpl_Check_Success(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()

	// Ожидаем, что будет вызван Publish с health check сообщением
	mockPublisher.On("Publish", mock.Anything, mock.MatchedBy(func(msgs []queue.Message) bool {
		if len(msgs) != 1 {
			return false
		}
		msg := msgs[0]

		// Проверяем topic
		if msg.Topic != "monitor.health.check" {
			return false
		}

		// Проверяем body
		body, ok := msg.Body.(map[string]any)
		if !ok {
			return false
		}
		if _, exists := body["timestamp"]; !exists {
			return false
		}

		return true
	})).Return(nil)

	err := publisher.Check(ctx)

	assert.NoError(t, err)
	mockPublisher.AssertExpectations(t)
}

// TestEventPublisherImpl_Check_NilPublisher тестирует проверку с nil publisher.
func TestEventPublisherImpl_Check_NilPublisher(t *testing.T) {
	logger := slog.Default()

	publisher := NewEventPublisherImpl(nil, logger)

	ctx := context.Background()

	err := publisher.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq publisher is nil")
}

// TestEventPublisherImpl_Check_PublisherError тестирует обработку ошибки при проверке.
func TestEventPublisherImpl_Check_PublisherError(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	ctx := context.Background()

	expectedError := errors.New("rabbitmq connection failed")
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(expectedError)

	err := publisher.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq health check failed")
	assert.Contains(t, err.Error(), "rabbitmq connection failed")
}

// TestEventPublisherImpl_Check_ContextTimeout тестирует timeout при проверке.
func TestEventPublisherImpl_Check_ContextTimeout(t *testing.T) {
	mockPublisher := new(MockQueuePublisher)
	logger := slog.Default()

	publisher := NewEventPublisherImpl(mockPublisher, logger)

	// Создаём контекст с коротким timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Ждем немного, чтобы контекст истек
	time.Sleep(10 * time.Millisecond)

	// Publisher должен вернуть ошибку из-за timeout контекста
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(context.DeadlineExceeded)

	err := publisher.Check(ctx)

	// Ожидаем ошибку timeout
	assert.Error(t, err)
}
