package publisher

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestEventPublisher_PublishMonitorCreated_PublishError тестирует ошибку PublishWithContext в PublishMonitorCreated.
func TestEventPublisher_PublishMonitorCreated_PublishError(t *testing.T) {
	t.Parallel()

	mockChannel := new(MockChannel)
	ctx := context.Background()

	mockChannel.On("PublishWithContext",
		ctx,
		"test-exchange",
		"monitor.created",
		false,
		false,
		mock.Anything,
	).Return(errors.New("connection failed"))

	publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

	event := &MonitorCreatedEvent{
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		Name:      "Test Monitor",
		URL:       "https://example.com",
		Timestamp: time.Now().Unix(),
	}

	err := publisher.PublishMonitorCreated(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish event")
	mockChannel.AssertExpectations(t)
}

// TestEventPublisher_PublishMonitorDeleted_PublishError тестирует ошибку PublishWithContext в PublishMonitorDeleted.
func TestEventPublisher_PublishMonitorDeleted_PublishError(t *testing.T) {
	t.Parallel()

	mockChannel := new(MockChannel)
	ctx := context.Background()

	mockChannel.On("PublishWithContext",
		ctx,
		"test-exchange",
		"monitor.deleted",
		false,
		false,
		mock.Anything,
	).Return(errors.New("publish failed"))

	publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

	event := &MonitorDeletedEvent{
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		Name:      "Test Monitor",
		Timestamp: time.Now().Unix(),
	}

	err := publisher.PublishMonitorDeleted(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish event")
	mockChannel.AssertExpectations(t)
}

// TestEventPublisher_PublishStatusChange_PublishError тестирует ошибку PublishWithContext в PublishStatusChange.
func TestEventPublisher_PublishStatusChange_PublishError(t *testing.T) {
	t.Parallel()

	mockChannel := new(MockChannel)
	ctx := context.Background()

	mockChannel.On("PublishWithContext",
		ctx,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		false,
		false,
		mock.Anything,
	).Return(errors.New("amqp error"))

	publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

	event := &MonitorStatusChangedEvent{
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		OldStatus: "up",
		NewStatus: "down",
		Timestamp: time.Now().Unix(),
	}

	err := publisher.PublishStatusChange(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish")
	mockChannel.AssertExpectations(t)
}

// TestMockChannel_PublishWithContext тестирует MockChannel.
func TestMockChannel_PublishWithContext(t *testing.T) {
	t.Parallel()

	m := new(MockChannel)
	ctx := context.Background()
	msg := amqp.Publishing{ContentType: "application/json"}

	m.On("PublishWithContext", ctx, "exchange", "key", false, false, msg).Return(nil)

	err := m.PublishWithContext(ctx, "exchange", "key", false, false, msg)
	assert.NoError(t, err)
	m.AssertExpectations(t)
}
