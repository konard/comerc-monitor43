package consumer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

// startRabbitMQContainer запускает временный RabbitMQ контейнер для тестов и
// возвращает AMQP URL и функцию очистки.
func startRabbitMQContainer(t *testing.T) string {
	t.Helper()

	// отключаем ryuk (reaper) — необходимо для colima и некоторых CI окружений
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3-alpine",
		ExposedPorts: []string{"5672/tcp"},
		WaitingFor: wait.ForLog("Server startup complete").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start RabbitMQ container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate rabbitmq container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5672")
	require.NoError(t, err)

	return fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())
}

// TestStart_returns_error_on_bad_url проверяет, что Start возвращает ошибку при неверном URL.
func TestStart_returns_error_on_bad_url(t *testing.T) {
	t.Parallel()

	c := NewEventConsumer(
		"amqp://localhost:1", // порт, на котором точно нет сервера
		&stubStatusRepo{},
		&stubCheckRepo{},
		&stubIncidentRepo{},
		realtime.NewHub(),
		applogger.New("error"),
	)

	err := c.Start(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to RabbitMQ")
	assert.False(t, c.IsConnected())
}

// TestStart_connects_and_close проверяет полный цикл Start/Close с реальным RabbitMQ.
func TestStart_connects_and_close(t *testing.T) {
	amqpURL := startRabbitMQContainer(t)

	hub := realtime.NewHub()
	go hub.Run()
	defer hub.Stop()

	c := NewEventConsumer(
		amqpURL,
		&stubStatusRepo{},
		&stubCheckRepo{},
		&stubIncidentRepo{},
		hub,
		applogger.New("error"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)
	assert.True(t, c.IsConnected())

	err = c.Close()
	require.NoError(t, err)
	assert.False(t, c.IsConnected())
}

// TestStart_context_cancel_stops_consumer проверяет, что отмена контекста останавливает горутину.
func TestStart_context_cancel_stops_consumer(t *testing.T) {
	amqpURL := startRabbitMQContainer(t)

	hub := realtime.NewHub()
	go hub.Run()
	defer hub.Stop()

	c := NewEventConsumer(
		amqpURL,
		&stubStatusRepo{},
		&stubCheckRepo{},
		&stubIncidentRepo{},
		hub,
		applogger.New("error"),
	)

	ctx, cancel := context.WithCancel(context.Background())

	err := c.Start(ctx)
	require.NoError(t, err)
	assert.True(t, c.IsConnected())

	// отменяем контекст — горутина должна завершиться и сбросить connected
	cancel()

	// даём горутине время завершиться
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, c.Close())
}
