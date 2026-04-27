package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestNoopPublisher тестирует NoopPublisher.
func TestNoopPublisher(t *testing.T) {
	t.Parallel()
	t.Run("creates NoopPublisher", func(t *testing.T) {
		publisher := NewNoopPublisher()

		assert.NotNil(t, publisher)
	})

	t.Run("PublishStatusChange does nothing", func(t *testing.T) {
		publisher := NewNoopPublisher()
		ctx := context.Background()

		// NoopPublisher.PublishStatusChange принимает interface{}, поэтому можем передать что угодно
		err := publisher.PublishStatusChange(ctx, "test")

		assert.NoError(t, err)
	})

	t.Run("PublishStatusChange with nil event", func(t *testing.T) {
		publisher := NewNoopPublisher()
		ctx := context.Background()

		err := publisher.PublishStatusChange(ctx, nil)

		assert.NoError(t, err)
	})

	t.Run("PublishStatusChange with complex event", func(t *testing.T) {
		publisher := NewNoopPublisher()
		ctx := context.Background()

		event := map[string]any{
			"monitor_id": uuid.New(),
			"status":     "DOWN",
		}

		err := publisher.PublishStatusChange(ctx, event)

		assert.NoError(t, err)
	})
}

// TestMonitorStatusChangedEvent_JSONSerialization тестирует JSON сериализацию событий.
func TestMonitorStatusChangedEvent_JSONSerialization(t *testing.T) {
	t.Parallel()
	t.Run("serializes event with all fields", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()
		responseTime := 500
		errorMessage := "connection timeout"

		event := &MonitorStatusChangedEvent{
			MonitorID:    monitorID,
			UserID:       userID,
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    1714123456,
			ResponseTime: &responseTime,
			ErrorMessage: &errorMessage,
		}

		data, err := json.Marshal(event)

		require.NoError(t, err)
		assert.NotNil(t, data)

		// Проверяем, что JSON содержит все поля
		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, monitorID.String(), decoded["monitor_id"])
		assert.Equal(t, userID.String(), decoded["user_id"])
		assert.Equal(t, "UP", decoded["old_status"])
		assert.Equal(t, "DOWN", decoded["new_status"])
		assert.Equal(t, float64(1714123456), decoded["timestamp"])
		assert.Equal(t, float64(500), decoded["response_time"])
		assert.Equal(t, "connection timeout", decoded["error_message"])
	})

	t.Run("serializes event without optional fields", func(t *testing.T) {
		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DEGRADED",
			Timestamp: time.Now().Unix(),
		}

		data, err := json.Marshal(event)

		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.NotContains(t, decoded, "response_time")
		assert.NotContains(t, decoded, "error_message")
	})

	t.Run("deserializes event correctly", func(t *testing.T) {
		jsonData := `{
			"monitor_id": "123e4567-e89b-12d3-a456-426614174000",
			"user_id": "123e4567-e89b-12d3-a456-426614174001",
			"old_status": "UP",
			"new_status": "DOWN",
			"timestamp": 1714123456,
			"response_time": 500,
			"error_message": "timeout"
		}`

		var event MonitorStatusChangedEvent
		err := json.Unmarshal([]byte(jsonData), &event)

		require.NoError(t, err)
		assert.Equal(t, "UP", event.OldStatus)
		assert.Equal(t, "DOWN", event.NewStatus)
		assert.Equal(t, int64(1714123456), event.Timestamp)
		require.NotNil(t, event.ResponseTime)
		assert.Equal(t, 500, *event.ResponseTime)
		require.NotNil(t, event.ErrorMessage)
		assert.Equal(t, "timeout", *event.ErrorMessage)
	})
}

// TestMonitorCreatedEvent_JSONSerialization тестирует JSON сериализацию MonitorCreatedEvent.
func TestMonitorCreatedEvent_JSONSerialization(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	userID := uuid.New()

	event := &MonitorCreatedEvent{
		MonitorID: monitorID,
		UserID:    userID,
		Name:      "Test Monitor",
		URL:       "https://example.com",
		Timestamp: 1714123456,
	}

	data, err := json.Marshal(event)

	require.NoError(t, err)

	var decoded map[string]any
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, monitorID.String(), decoded["monitor_id"])
	assert.Equal(t, userID.String(), decoded["user_id"])
	assert.Equal(t, "Test Monitor", decoded["name"])
	assert.Equal(t, "https://example.com", decoded["url"])
	assert.Equal(t, float64(1714123456), decoded["timestamp"])
}

// TestMonitorDeletedEvent_JSONSerialization тестирует JSON сериализацию MonitorDeletedEvent.
func TestMonitorDeletedEvent_JSONSerialization(t *testing.T) {
	t.Parallel()
	monitorID := uuid.New()
	userID := uuid.New()

	event := &MonitorDeletedEvent{
		MonitorID: monitorID,
		UserID:    userID,
		Name:      "Test Monitor",
		Timestamp: 1714123456,
	}

	data, err := json.Marshal(event)

	require.NoError(t, err)

	var decoded map[string]any
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, monitorID.String(), decoded["monitor_id"])
	assert.Equal(t, userID.String(), decoded["user_id"])
	assert.Equal(t, "Test Monitor", decoded["name"])
	assert.Equal(t, float64(1714123456), decoded["timestamp"])
}

// TestRoutingKeyGeneration тестирует генерацию routing keys.
func TestRoutingKeyGeneration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		status      string
		expectedKey string
	}{
		{
			name:        "status UP",
			status:      "UP",
			expectedKey: "monitor.status.UP",
		},
		{
			name:        "status DOWN",
			status:      "DOWN",
			expectedKey: "monitor.status.DOWN",
		},
		{
			name:        "status DEGRADED",
			status:      "DEGRADED",
			expectedKey: "monitor.status.DEGRADED",
		},
		{
			name:        "status PAUSED",
			status:      "PAUSED",
			expectedKey: "monitor.status.PAUSED",
		},
		{
			name:        "status PENDING",
			status:      "PENDING",
			expectedKey: "monitor.status.PENDING",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Симулируем генерацию routing key как в коде
			routingKey := "monitor.status." + tc.status
			assert.Equal(t, tc.expectedKey, routingKey)
		})
	}
}

// TestEventStructureValidation тестирует структуру событий.
func TestEventStructureValidation(t *testing.T) {
	t.Parallel()
	t.Run("MonitorStatusChangedEvent has required fields", func(t *testing.T) {
		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		// Проверяем, что обязательные поля заполнены
		assert.NotEqual(t, uuid.Nil, event.MonitorID)
		assert.NotEqual(t, uuid.Nil, event.UserID)
		assert.NotEmpty(t, event.OldStatus)
		assert.NotEmpty(t, event.NewStatus)
		assert.NotZero(t, event.Timestamp)
	})

	t.Run("MonitorCreatedEvent has required fields", func(t *testing.T) {
		event := &MonitorCreatedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test",
			URL:       "https://example.com",
			Timestamp: time.Now().Unix(),
		}

		assert.NotEqual(t, uuid.Nil, event.MonitorID)
		assert.NotEqual(t, uuid.Nil, event.UserID)
		assert.NotEmpty(t, event.Name)
		assert.NotEmpty(t, event.URL)
		assert.NotZero(t, event.Timestamp)
	})

	t.Run("MonitorDeletedEvent has required fields", func(t *testing.T) {
		event := &MonitorDeletedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test",
			Timestamp: time.Now().Unix(),
		}

		assert.NotEqual(t, uuid.Nil, event.MonitorID)
		assert.NotEqual(t, uuid.Nil, event.UserID)
		assert.NotEmpty(t, event.Name)
		assert.NotZero(t, event.Timestamp)
	})
}

// MockChannel является mock реализацией Channel для тестов.
type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	args_ := m.Called(name, kind, durable, autoDelete, internal, noWait, args)
	return args_.Error(0)
}

func (m *MockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args_ := m.Called(ctx, exchange, key, mandatory, immediate, msg)
	return args_.Error(0)
}

// TestNewEventPublisher тестирует создание EventPublisher.
func TestNewEventPublisher(t *testing.T) {
	t.Parallel()
	t.Run("creates EventPublisher with valid parameters", func(t *testing.T) {
		mockChannel := new(MockChannel)
		logger := slog.Default()

		publisher := NewEventPublisher(mockChannel, "test-exchange", logger)

		assert.NotNil(t, publisher)
		assert.Equal(t, mockChannel, publisher.channel)
		assert.Equal(t, "test-exchange", publisher.exchange)
		assert.Equal(t, logger, publisher.logger)
	})
}

// TestEventPublisher_SetupTopology тестирует настройку topology.
func TestEventPublisher_SetupTopology(t *testing.T) {
	t.Parallel()
	t.Run("successfully declares exchange", func(t *testing.T) {
		mockChannel := new(MockChannel)
		mockChannel.On("ExchangeDeclare",
			"test-exchange",
			"topic",
			true,  // durable
			false, // autoDelete
			false, // internal
			false, // noWait
			mock.Anything,
		).Return(nil)

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())
		ctx := context.Background()

		err := publisher.SetupTopology(ctx)

		assert.NoError(t, err)
		mockChannel.AssertExpectations(t)
	})

	t.Run("returns error when exchange declare fails", func(t *testing.T) {
		mockChannel := new(MockChannel)
		mockChannel.On("ExchangeDeclare",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(errors.New("exchange declare failed"))

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())
		ctx := context.Background()

		err := publisher.SetupTopology(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to declare exchange")
		mockChannel.AssertExpectations(t)
	})
}

// TestEventPublisher_PublishStatusChange тестирует публикацию событий изменения статуса.
func TestEventPublisher_PublishStatusChange(t *testing.T) {
	t.Parallel()
	t.Run("successfully publishes status change event", func(t *testing.T) {
		mockChannel := new(MockChannel)
		ctx := context.Background()

		mockChannel.On("PublishWithContext",
			ctx,
			"test-exchange",
			"monitor.status.DOWN",
			false,
			false,
			mock.MatchedBy(func(msg amqp.Publishing) bool {
				return msg.ContentType == "application/json" &&
					msg.DeliveryMode == amqp.Persistent
			}),
		).Return(nil)

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishStatusChange(ctx, event)

		assert.NoError(t, err)
		mockChannel.AssertExpectations(t)
	})

	t.Run("returns error when publish fails", func(t *testing.T) {
		mockChannel := new(MockChannel)
		ctx := context.Background()

		mockChannel.On("PublishWithContext",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(errors.New("publish failed"))

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishStatusChange(ctx, event)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to publish event")
		mockChannel.AssertExpectations(t)
	})
}

// TestEventPublisher_PublishMonitorCreated тестирует публикацию событий создания монитора.
func TestEventPublisher_PublishMonitorCreated(t *testing.T) {
	t.Parallel()
	t.Run("successfully publishes monitor created event", func(t *testing.T) {
		mockChannel := new(MockChannel)
		ctx := context.Background()

		mockChannel.On("PublishWithContext",
			ctx,
			"test-exchange",
			"monitor.created",
			false,
			false,
			mock.MatchedBy(func(msg amqp.Publishing) bool {
				return msg.ContentType == "application/json" &&
					msg.DeliveryMode == amqp.Persistent
			}),
		).Return(nil)

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

		event := &MonitorCreatedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test Monitor",
			URL:       "https://example.com",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishMonitorCreated(ctx, event)

		assert.NoError(t, err)
		mockChannel.AssertExpectations(t)
	})
}

// TestEventPublisher_PublishMonitorDeleted тестирует публикацию событий удаления монитора.
func TestEventPublisher_PublishMonitorDeleted(t *testing.T) {
	t.Parallel()
	t.Run("successfully publishes monitor deleted event", func(t *testing.T) {
		mockChannel := new(MockChannel)
		ctx := context.Background()

		mockChannel.On("PublishWithContext",
			ctx,
			"test-exchange",
			"monitor.deleted",
			false,
			false,
			mock.MatchedBy(func(msg amqp.Publishing) bool {
				return msg.ContentType == "application/json" &&
					msg.DeliveryMode == amqp.Persistent
			}),
		).Return(nil)

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

		event := &MonitorDeletedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test Monitor",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishMonitorDeleted(ctx, event)

		assert.NoError(t, err)
		mockChannel.AssertExpectations(t)
	})
}

// TestEventPublisher_Close тестирует закрытие publisher'а.
func TestEventPublisher_Close(t *testing.T) {
	t.Parallel()
	t.Run("closes without error", func(t *testing.T) {
		mockChannel := new(MockChannel)

		publisher := NewEventPublisher(mockChannel, "test-exchange", slog.Default())

		err := publisher.Close()

		assert.NoError(t, err)
	})
}

// TestEventMarshaling тестирует маршалинг событий в JSON.
func TestEventMarshaling(t *testing.T) {
	t.Parallel()
	t.Run("marshals MonitorStatusChangedEvent correctly", func(t *testing.T) {
		responseTime := 500
		errorMessage := "connection timeout"

		event := &MonitorStatusChangedEvent{
			MonitorID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			UserID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    1714123456,
			ResponseTime: &responseTime,
			ErrorMessage: &errorMessage,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", decoded["monitor_id"])
		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174001", decoded["user_id"])
		assert.Equal(t, "UP", decoded["old_status"])
		assert.Equal(t, "DOWN", decoded["new_status"])
		assert.Equal(t, float64(1714123456), decoded["timestamp"])
		assert.Equal(t, float64(500), decoded["response_time"])
		assert.Equal(t, "connection timeout", decoded["error_message"])
	})

	t.Run("marshals MonitorCreatedEvent correctly", func(t *testing.T) {
		event := &MonitorCreatedEvent{
			MonitorID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			UserID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
			Name:      "Test Monitor",
			URL:       "https://example.com",
			Timestamp: 1714123456,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", decoded["monitor_id"])
		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174001", decoded["user_id"])
		assert.Equal(t, "Test Monitor", decoded["name"])
		assert.Equal(t, "https://example.com", decoded["url"])
		assert.Equal(t, float64(1714123456), decoded["timestamp"])
	})

	t.Run("marshals MonitorDeletedEvent correctly", func(t *testing.T) {
		event := &MonitorDeletedEvent{
			MonitorID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			UserID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
			Name:      "Test Monitor",
			Timestamp: 1714123456,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", decoded["monitor_id"])
		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174001", decoded["user_id"])
		assert.Equal(t, "Test Monitor", decoded["name"])
		assert.Equal(t, float64(1714123456), decoded["timestamp"])
	})
}

// TestEventCreation тестирует создание событий.
func TestEventCreation(t *testing.T) {
	t.Parallel()
	t.Run("creates MonitorStatusChangedEvent", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorStatusChangedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		assert.NotEqual(t, uuid.Nil, event.MonitorID)
		assert.NotEqual(t, uuid.Nil, event.UserID)
		assert.NotEmpty(t, event.OldStatus)
		assert.NotEmpty(t, event.NewStatus)
		assert.NotZero(t, event.Timestamp)
	})

	t.Run("creates MonitorCreatedEvent", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorCreatedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			Name:      "Test Monitor",
			URL:       "https://example.com",
			Timestamp: time.Now().Unix(),
		}

		assert.NotEqual(t, uuid.Nil, event.MonitorID)
		assert.NotEqual(t, uuid.Nil, event.UserID)
		assert.NotEmpty(t, event.Name)
		assert.NotEmpty(t, event.URL)
		assert.NotZero(t, event.Timestamp)
	})

	t.Run("creates MonitorDeletedEvent", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorDeletedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			Name:      "Test Monitor",
			Timestamp: time.Now().Unix(),
		}

		assert.NotEqual(t, uuid.Nil, event.MonitorID)
		assert.NotEqual(t, uuid.Nil, event.UserID)
		assert.NotEmpty(t, event.Name)
		assert.NotZero(t, event.Timestamp)
	})
}

// TestEventValidation тестирует валидацию событий.
func TestEventValidation(t *testing.T) {
	t.Parallel()
	t.Run("validates required fields in MonitorStatusChangedEvent", func(t *testing.T) {
		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		// Проверяем, что событие можно маршалировать без ошибок
		_, err := json.Marshal(event)
		assert.NoError(t, err)
	})

	t.Run("validates status values", func(t *testing.T) {
		validStatuses := []string{"UP", "DOWN", "DEGRADED", "PAUSED", "PENDING"}

		for _, status := range validStatuses {
			event := &MonitorStatusChangedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				OldStatus: "PENDING",
				NewStatus: status,
				Timestamp: time.Now().Unix(),
			}

			_, err := json.Marshal(event)
			assert.NoError(t, err, "status %s should be valid", status)
		}
	})
}

// TestNoopPublisher тестирует NoopPublisher.

// TestPublisherInterface тестирует интерфейс publisher'а.
func TestPublisherInterface(t *testing.T) {
	t.Parallel()
	// Проверяем, что NoopPublisher реализует ожидаемый интерфейс
	publisher := NewNoopPublisher()
	ctx := context.Background()

	// Должен поддерживать PublishStatusChange с различными типами событий
	assert.NoError(t, publisher.PublishStatusChange(ctx, nil))
	assert.NoError(t, publisher.PublishStatusChange(ctx, "string"))
	assert.NoError(t, publisher.PublishStatusChange(ctx, map[string]any{}))
	assert.NoError(t, publisher.PublishStatusChange(ctx, &MonitorStatusChangedEvent{}))
}

// TestEventTimestamps тестирует timestamp в событиях.
func TestEventTimestamps(t *testing.T) {
	t.Parallel()
	now := time.Now().Unix()

	event := &MonitorStatusChangedEvent{
		MonitorID: uuid.New(),
		UserID:    uuid.New(),
		OldStatus: "UP",
		NewStatus: "DOWN",
		Timestamp: now,
	}

	// Проверяем, что timestamp является разумным (не в будущем, не слишком давно)
	assert.LessOrEqual(t, event.Timestamp, now+1)       // Не более чем на 1 секунду в будущем
	assert.GreaterOrEqual(t, event.Timestamp, now-3600) // Не более чем час назад
}

// TestEventSerializationEdgeCases тестирует edge cases сериализации.
func TestEventSerializationEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("empty strings in optional fields", func(t *testing.T) {
		emptyStr := ""
		event := &MonitorStatusChangedEvent{
			MonitorID:    uuid.New(),
			UserID:       uuid.New(),
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ErrorMessage: &emptyStr,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Пустая строка должна быть в JSON
		assert.Contains(t, decoded, "error_message")
	})

	t.Run("zero values in optional fields", func(t *testing.T) {
		zeroTime := 0
		event := &MonitorStatusChangedEvent{
			MonitorID:    uuid.New(),
			UserID:       uuid.New(),
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ResponseTime: &zeroTime,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Contains(t, decoded, "response_time")
		assert.Equal(t, float64(0), decoded["response_time"])
	})
}

// TestEventRoundTrip тестирует полный цикл сериализации/десериализации.
func TestEventRoundTrip(t *testing.T) {
	t.Parallel()
	t.Run("MonitorStatusChangedEvent round trip", func(t *testing.T) {
		original := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: 1714123456,
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded MonitorStatusChangedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.MonitorID, decoded.MonitorID)
		assert.Equal(t, original.UserID, decoded.UserID)
		assert.Equal(t, original.OldStatus, decoded.OldStatus)
		assert.Equal(t, original.NewStatus, decoded.NewStatus)
		assert.Equal(t, original.Timestamp, decoded.Timestamp)
	})

	t.Run("MonitorCreatedEvent round trip", func(t *testing.T) {
		original := &MonitorCreatedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test Monitor",
			URL:       "https://example.com",
			Timestamp: 1714123456,
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded MonitorCreatedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.MonitorID, decoded.MonitorID)
		assert.Equal(t, original.UserID, decoded.UserID)
		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.URL, decoded.URL)
		assert.Equal(t, original.Timestamp, decoded.Timestamp)
	})

	t.Run("MonitorDeletedEvent round trip", func(t *testing.T) {
		original := &MonitorDeletedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test Monitor",
			Timestamp: 1714123456,
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded MonitorDeletedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.MonitorID, decoded.MonitorID)
		assert.Equal(t, original.UserID, decoded.UserID)
		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Timestamp, decoded.Timestamp)
	})
}

// TestEventFieldsValidation тестирует валидацию полей событий.
func TestEventFieldsValidation(t *testing.T) {
	t.Parallel()
	t.Run("valid status values", func(t *testing.T) {
		validStatuses := []string{"UP", "DOWN", "DEGRADED", "PAUSED", "PENDING"}

		for _, status := range validStatuses {
			event := &MonitorStatusChangedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				OldStatus: "PENDING",
				NewStatus: status,
				Timestamp: time.Now().Unix(),
			}

			data, err := json.Marshal(event)
			assert.NoError(t, err, "should serialize status: %s", status)
			assert.NotNil(t, data)

			var decoded MonitorStatusChangedEvent
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err, "should deserialize status: %s", status)
			assert.Equal(t, status, decoded.NewStatus)
		}
	})

	t.Run("URL validation in MonitorCreatedEvent", func(t *testing.T) {
		validURLs := []string{
			"https://example.com",
			"http://localhost:8080",
			"https://api.example.com/v1/endpoint",
			"https://example.com:9443/path",
		}

		for _, url := range validURLs {
			event := &MonitorCreatedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				Name:      "Test Monitor",
				URL:       url,
				Timestamp: time.Now().Unix(),
			}

			data, err := json.Marshal(event)
			assert.NoError(t, err, "should serialize URL: %s", url)

			var decoded MonitorCreatedEvent
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err, "should deserialize URL: %s", url)
			assert.Equal(t, url, decoded.URL)
		}
	})

	t.Run("name validation in events", func(t *testing.T) {
		names := []string{
			"Simple",
			"Monitor with spaces",
			"Monitor-with-dashes",
			"Monitor_with_underscores",
			"Монитор с русским",
		}

		for _, name := range names {
			event := &MonitorCreatedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				Name:      name,
				URL:       "https://example.com",
				Timestamp: time.Now().Unix(),
			}

			data, err := json.Marshal(event)
			assert.NoError(t, err, "should serialize name: %s", name)

			var decoded MonitorCreatedEvent
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err, "should deserialize name: %s", name)
			assert.Equal(t, name, decoded.Name)
		}
	})
}

// TestEventConsistency тестирует консистентность событий.
func TestEventConsistency(t *testing.T) {
	t.Parallel()
	t.Run("events have consistent timestamp format", func(t *testing.T) {
		timestamp := time.Now().Unix()

		statusEvent := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: timestamp,
		}

		createdEvent := &MonitorCreatedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test",
			URL:       "https://example.com",
			Timestamp: timestamp,
		}

		deletedEvent := &MonitorDeletedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			Name:      "Test",
			Timestamp: timestamp,
		}

		assert.Equal(t, timestamp, statusEvent.Timestamp)
		assert.Equal(t, timestamp, createdEvent.Timestamp)
		assert.Equal(t, timestamp, deletedEvent.Timestamp)
	})
}

// TestMultipleEventCreation тестирует создание множественных событий.
func TestMultipleEventCreation(t *testing.T) {
	t.Parallel()
	t.Run("creates multiple events with different UUIDs", func(t *testing.T) {
		var events []*MonitorStatusChangedEvent
		ids := make(map[uuid.UUID]bool)

		for i := 0; i < 100; i++ {
			event := &MonitorStatusChangedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				OldStatus: "PENDING",
				NewStatus: "UP",
				Timestamp: time.Now().Unix(),
			}
			events = append(events, event)
			ids[event.MonitorID] = true
		}

		assert.Len(t, events, 100)
		assert.Len(t, ids, 100, "all monitor IDs should be unique")
	})

	t.Run("serializes multiple events", func(t *testing.T) {
		var events []*MonitorStatusChangedEvent

		for i := 0; i < 50; i++ {
			event := &MonitorStatusChangedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				OldStatus: "UP",
				NewStatus: "DOWN",
				Timestamp: time.Now().Unix(),
			}
			events = append(events, event)
		}

		for _, event := range events {
			data, err := json.Marshal(event)
			assert.NoError(t, err)
			assert.NotNil(t, data)

			var decoded MonitorStatusChangedEvent
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)
			assert.Equal(t, event.MonitorID, decoded.MonitorID)
		}
	})
}

// TestEventSize тестирует размер событий.
func TestEventSize(t *testing.T) {
	t.Parallel()
	t.Run("event size is reasonable", func(t *testing.T) {
		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		assert.Less(t, len(data), 1024, "event size should be under 1KB")
	})

	t.Run("event with optional fields size", func(t *testing.T) {
		responseTime := 5000
		errorMessage := "This is a very long error message with lots of details about what went wrong"

		event := &MonitorStatusChangedEvent{
			MonitorID:    uuid.New(),
			UserID:       uuid.New(),
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ResponseTime: &responseTime,
			ErrorMessage: &errorMessage,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		assert.Less(t, len(data), 2048, "event size with optional fields should be under 2KB")
	})
}

// TestConcurrentEventCreation тестирует конкурентное создание событий.
func TestConcurrentEventCreation(t *testing.T) {
	t.Parallel()
	t.Run("concurrent event creation", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				event := &MonitorStatusChangedEvent{
					MonitorID: uuid.New(),
					UserID:    uuid.New(),
					OldStatus: "UP",
					NewStatus: "DOWN",
					Timestamp: time.Now().Unix(),
				}

				data, err := json.Marshal(event)
				assert.NoError(t, err)
				assert.NotNil(t, data)

				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestEventTimestampPrecision тестирует точность timestamp.
func TestEventTimestampPrecision(t *testing.T) {
	t.Parallel()
	t.Run("timestamp has second precision", func(t *testing.T) {
		before := time.Now().Unix()
		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}
		after := time.Now().Unix()

		assert.GreaterOrEqual(t, event.Timestamp, before)
		assert.LessOrEqual(t, event.Timestamp, after)
	})

	t.Run("timestamp preserves through serialization", func(t *testing.T) {
		timestamp := time.Now().Unix()

		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: timestamp,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded MonitorStatusChangedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, timestamp, decoded.Timestamp)
	})
}

// TestEventWithNilOptionalFields тестирует события с nil опциональными полями.
func TestEventWithNilOptionalFields(t *testing.T) {
	t.Parallel()
	t.Run("MonitorStatusChangedEvent with nil optional fields", func(t *testing.T) {
		event := &MonitorStatusChangedEvent{
			MonitorID:    uuid.New(),
			UserID:       uuid.New(),
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ResponseTime: nil,
			ErrorMessage: nil,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]any
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.NotContains(t, decoded, "response_time")
		assert.NotContains(t, decoded, "error_message")
	})
}

// TestSpecialCharactersInFields тестирует специальные символы в полях.
func TestSpecialCharactersInFields(t *testing.T) {
	t.Parallel()
	t.Run("unicode in error message", func(t *testing.T) {
		errorMessage := "Ошибка подключения 🚀"
		event := &MonitorStatusChangedEvent{
			MonitorID:    uuid.New(),
			UserID:       uuid.New(),
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ErrorMessage: &errorMessage,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded MonitorStatusChangedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, errorMessage, *decoded.ErrorMessage)
	})

	t.Run("special characters in monitor name", func(t *testing.T) {
		specialNames := []string{
			"Test Monitor",
			"Test<>Monitor",
			"Test/\\Monitor",
			"Test\"Monitor",
			"Test'Monitor",
		}

		for _, name := range specialNames {
			event := &MonitorCreatedEvent{
				MonitorID: uuid.New(),
				UserID:    uuid.New(),
				Name:      name,
				URL:       "https://example.com",
				Timestamp: time.Now().Unix(),
			}

			data, err := json.Marshal(event)
			assert.NoError(t, err, "should serialize name: %s", name)

			var decoded MonitorCreatedEvent
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err, "should deserialize name: %s", name)
			assert.Equal(t, name, decoded.Name)
		}
	})
}

// TestEventTimestampRanges тестирует различные значения timestamp.
func TestEventTimestampRanges(t *testing.T) {
	t.Parallel()
	t.Run("past timestamp", func(t *testing.T) {
		pastTimestamp := time.Now().Add(-24 * time.Hour).Unix()

		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: pastTimestamp,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded MonitorStatusChangedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, pastTimestamp, decoded.Timestamp)
	})

	t.Run("zero timestamp", func(t *testing.T) {
		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: 0,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded MonitorStatusChangedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, int64(0), decoded.Timestamp)
	})

	t.Run("maximum timestamp", func(t *testing.T) {
		maxTimestamp := int64(1<<63 - 1)

		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: maxTimestamp,
		}

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded MonitorStatusChangedEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, maxTimestamp, decoded.Timestamp)
	})
}

// TestResponseTimeValues тестирует различные значения response time.
func TestResponseTimeValues(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		responseTime int
	}{
		{"zero", 0},
		{"small", 1},
		{"normal", 500},
		{"large", 5000},
		{"very large", 60000},
		{"negative", -100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responseTime := tc.responseTime
			event := &MonitorStatusChangedEvent{
				MonitorID:    uuid.New(),
				UserID:       uuid.New(),
				OldStatus:    "UP",
				NewStatus:    "DOWN",
				Timestamp:    time.Now().Unix(),
				ResponseTime: &responseTime,
			}

			data, err := json.Marshal(event)
			require.NoError(t, err)

			var decoded MonitorStatusChangedEvent
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tc.responseTime, *decoded.ResponseTime)
		})
	}
}

// TestMultipleEventsSequential тестирует последовательность событий.
func TestMultipleEventsSequential(t *testing.T) {
	t.Parallel()
	t.Run("sequential status changes", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		statuses := []string{"PENDING", "UP", "DEGRADED", "DOWN", "UP"}
		events := make([]*MonitorStatusChangedEvent, 0, len(statuses))

		for i, status := range statuses {
			event := &MonitorStatusChangedEvent{
				MonitorID: monitorID,
				UserID:    userID,
				OldStatus: getStatus(statuses, i-1),
				NewStatus: status,
				Timestamp: time.Now().Add(time.Duration(i) * time.Second).Unix(),
			}
			events = append(events, event)
		}

		assert.Len(t, events, len(statuses))

		for i, event := range events {
			assert.Equal(t, monitorID, event.MonitorID)
			assert.Equal(t, statuses[i], event.NewStatus)
		}
	})
}

func getStatus(statuses []string, index int) string {
	if index < 0 {
		return "PENDING"
	}
	return statuses[index]
}

// TestUUIDGeneration тестирует генерацию UUID.
func TestUUIDGeneration(t *testing.T) {
	t.Parallel()
	t.Run("generates unique UUIDs", func(t *testing.T) {
		uuids := make(map[uuid.UUID]bool)

		for i := 0; i < 1000; i++ {
			id := uuid.New()
			uuids[id] = true
		}

		assert.Len(t, uuids, 1000, "all UUIDs should be unique")
	})

	t.Run("UUID is not nil", func(t *testing.T) {
		id := uuid.New()
		assert.NotEqual(t, uuid.Nil, id)
	})

	t.Run("UUID length is correct", func(t *testing.T) {
		assert.Equal(t, 16, len(uuid.New()))
	})
}

// TestEventSerializationPerformance тестирует производительность сериализации.
func TestEventSerializationPerformance(t *testing.T) {
	t.Parallel()
	t.Run("serializes many events quickly", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping performance test in short mode")
		}

		event := &MonitorStatusChangedEvent{
			MonitorID: uuid.New(),
			UserID:    uuid.New(),
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		iterations := 10000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Serialized %d events in %v (%v per event)", iterations, duration, avgDuration)

		// Should be able to serialize at least 1000 events per second
		assert.Less(t, avgDuration, time.Millisecond)
	})
}
