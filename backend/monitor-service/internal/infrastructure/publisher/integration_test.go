package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventPublisher_Integration тестирует EventPublisher с реальным RabbitMQ.
func TestEventPublisher_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Получаем URL RabbitMQ из переменной окружения или используем дефолтный
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: тестовые учётные данные RabbitMQ по умолчанию
	}

	ctx := context.Background()

	// Подключаемся к RabbitMQ
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		t.Skipf("failed to connect to RabbitMQ: %v (skip integration tests)", err)
		return
	}
	closeAMQPConnection(t, conn)

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}
	closeAMQPChannel(t, channel)

	// Создаём logger для тестов
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	exchangeName := fmt.Sprintf("test-exchange-%s", uuid.New())
	publisher := NewEventPublisher(channel, exchangeName, logger)

	t.Run("SetupTopology", func(t *testing.T) {
		err := publisher.SetupTopology(ctx)
		require.NoError(t, err)

		// Проверяем, что exchange создан
		// (проверка косвенная - если бы exchange не создался, следующая публикация бы失败了)
	})

	t.Run("PublishStatusChange", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorStatusChangedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			OldStatus: "PENDING",
			NewStatus: "UP",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishStatusChange(ctx, event)
		require.NoError(t, err)
	})

	t.Run("PublishStatusChange with optional fields", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()
		responseTime := 500
		errorMessage := "connection timeout"

		event := &MonitorStatusChangedEvent{
			MonitorID:    monitorID,
			UserID:       userID,
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ResponseTime: &responseTime,
			ErrorMessage: &errorMessage,
		}

		err := publisher.PublishStatusChange(ctx, event)
		require.NoError(t, err)
	})

	t.Run("PublishMonitorCreated", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorCreatedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			Name:      "Test Monitor",
			URL:       "https://example.com",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishMonitorCreated(ctx, event)
		require.NoError(t, err)
	})

	t.Run("PublishMonitorDeleted", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorDeletedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			Name:      "Test Monitor",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishMonitorDeleted(ctx, event)
		require.NoError(t, err)
	})

	t.Run("Publish multiple status changes", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		statuses := []string{"UP", "DOWN", "DEGRADED", "UP"}

		for _, status := range statuses {
			event := &MonitorStatusChangedEvent{
				MonitorID: monitorID,
				UserID:    userID,
				OldStatus: "PENDING",
				NewStatus: status,
				Timestamp: time.Now().Unix(),
			}

			err := publisher.PublishStatusChange(ctx, event)
			assert.NoError(t, err)
		}
	})

	t.Run("Publish with cancelled context", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()

		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Сразу отменяем контекст

		event := &MonitorStatusChangedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		// Публикация может вернуть ошибку или завершиться успешно
		// (зависит от того, когда контекст проверяется в amqp библиотеке)
		err := publisher.PublishStatusChange(cancelledCtx, event)
		if err != nil {
			t.Logf("cancelled context publish returned error (expected): %v", err)
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := publisher.Close()
		assert.NoError(t, err)
	})
}

// TestEventPublisher_ConsumeAndVerify тестирует публикацию и потребление сообщений.
func TestEventPublisher_ConsumeAndVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: тестовые учётные данные RabbitMQ по умолчанию
	}

	ctx := context.Background()

	// Подключаемся к RabbitMQ
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		t.Skipf("failed to connect to RabbitMQ: %v (skip integration tests)", err)
		return
	}
	closeAMQPConnection(t, conn)

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}
	closeAMQPChannel(t, channel)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	exchangeName := fmt.Sprintf("test-exchange-consume-%s", uuid.New())
	queueName := fmt.Sprintf("test-queue-%s", uuid.New())

	publisher := NewEventPublisher(channel, exchangeName, logger)

	// Настраиваем topology
	err = publisher.SetupTopology(ctx)
	require.NoError(t, err)

	// Создаём очередь и биндим её к exchange
	_, err = channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	require.NoError(t, err)

	// Биндим очередь ко всем статусам
	err = channel.QueueBind(
		queueName,
		"monitor.status.*", // routing key pattern
		exchangeName,
		false,
		nil,
	)
	require.NoError(t, err)

	t.Run("publish and consume status change", func(t *testing.T) {
		monitorID := uuid.New()
		userID := uuid.New()
		responseTime := 500

		event := &MonitorStatusChangedEvent{
			MonitorID:    monitorID,
			UserID:       userID,
			OldStatus:    "UP",
			NewStatus:    "DOWN",
			Timestamp:    time.Now().Unix(),
			ResponseTime: &responseTime,
		}

		// Публикуем событие
		err := publisher.PublishStatusChange(ctx, event)
		require.NoError(t, err)

		// Получаем сообщение из очереди
		msgs, err := channel.Consume(
			queueName,
			"",    // consumer tag
			true,  // auto-ack
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // args
		)
		require.NoError(t, err)

		// Ждём сообщение с таймаутом
		select {
		case msg := <-msgs:
			require.NotNil(t, msg)

			// Проверяем, что получили то же событие
			var receivedEvent MonitorStatusChangedEvent
			err := json.Unmarshal(msg.Body, &receivedEvent)
			require.NoError(t, err)

			assert.Equal(t, event.MonitorID, receivedEvent.MonitorID)
			assert.Equal(t, event.UserID, receivedEvent.UserID)
			assert.Equal(t, event.OldStatus, receivedEvent.OldStatus)
			assert.Equal(t, event.NewStatus, receivedEvent.NewStatus)
			assert.Equal(t, event.ResponseTime, receivedEvent.ResponseTime)

		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})
}

// TestEventPublisher_ErrorHandling тестирует обработку ошибок.
func TestEventPublisher_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: тестовые учётные данные RabbitMQ по умолчанию
	}

	ctx := context.Background()

	// Подключаемся к RabbitMQ
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		t.Skipf("failed to connect to RabbitMQ: %v (skip integration tests)", err)
		return
	}
	closeAMQPConnection(t, conn)

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}
	closeAMQPChannel(t, channel)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	t.Run("publish with closed channel", func(t *testing.T) {
		// Закрываем канал
		require.NoError(t, channel.Close())

		exchangeName := fmt.Sprintf("test-exchange-error-%s", uuid.New())
		publisher := NewEventPublisher(channel, exchangeName, logger)

		monitorID := uuid.New()
		userID := uuid.New()

		event := &MonitorStatusChangedEvent{
			MonitorID: monitorID,
			UserID:    userID,
			OldStatus: "UP",
			NewStatus: "DOWN",
			Timestamp: time.Now().Unix(),
		}

		err := publisher.PublishStatusChange(ctx, event)
		assert.Error(t, err)
	})
}

// TestEventPublisher_TableDrivenStatuses тестирует все статусы.
func TestEventPublisher_TableDrivenStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: тестовые учётные данные RabbitMQ по умолчанию
	}

	ctx := context.Background()

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		t.Skipf("failed to connect to RabbitMQ: %v (skip integration tests)", err)
		return
	}
	closeAMQPConnection(t, conn)

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}
	closeAMQPChannel(t, channel)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	exchangeName := fmt.Sprintf("test-exchange-statuses-%s", uuid.New())
	publisher := NewEventPublisher(channel, exchangeName, logger)

	err = publisher.SetupTopology(ctx)
	require.NoError(t, err)

	statuses := []string{"UP", "DOWN", "DEGRADED", "PAUSED", "PENDING"}

	for _, status := range statuses {
		t.Run("status_"+status, func(t *testing.T) {
			monitorID := uuid.New()
			userID := uuid.New()

			event := &MonitorStatusChangedEvent{
				MonitorID: monitorID,
				UserID:    userID,
				OldStatus: "PENDING",
				NewStatus: status,
				Timestamp: time.Now().Unix(),
			}

			err := publisher.PublishStatusChange(ctx, event)
			assert.NoError(t, err)
		})
	}
}
