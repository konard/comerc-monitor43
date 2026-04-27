package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorker_StateMachineIntegration тестирует интеграцию Worker с StateMachine
func TestWorker_StateMachineIntegration(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаём mock clients
	schedulerClient := &mockSchedulerClient{
		connected: false,
	}
	monitorClient := &mockMonitorClient{
		connected: false,
	}

	worker := NewWorker(
		schedulerClient,
		monitorClient,
		&mockExecutor{},
		"test-worker",
		"test-zone",
		1*time.Second,
		1*time.Second,
		5*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// Проверяем начальное состояние
	assert.Equal(t, StateDisconnected, worker.GetState())
	assert.False(t, worker.IsRunning())

	// Подключаем клиентов
	schedulerClient.connected = true
	monitorClient.connected = true

	err := worker.Start(ctx)
	require.NoError(t, err)
	assert.True(t, worker.IsRunning())

	// Проверяем состояние после подключения
	assert.Equal(t, StateConnected, worker.GetState())
	assert.True(t, worker.stateMachine.IsConnected())
	assert.True(t, worker.stateMachine.CanExecuteChecks())

	scheduler, monitor := worker.GetConnectionInfo()
	assert.True(t, scheduler.Connected)
	assert.True(t, monitor.Connected)
	assert.Equal(t, 0, scheduler.ReconnectNum)
	assert.Equal(t, 0, monitor.ReconnectNum)

	worker.Stop()
	assert.False(t, worker.IsRunning())
	assert.Equal(t, StateShutdown, worker.GetState())
}

// TestWorker_GracefulDegradation тестирует graceful degradation при потере соединения
func TestWorker_GracefulDegradation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	schedulerClient := &mockSchedulerClient{
		connected: true,
	}
	monitorClient := &mockMonitorClient{
		connected: true,
	}

	worker := NewWorker(
		schedulerClient,
		monitorClient,
		&mockExecutor{},
		"test-worker",
		"test-zone",
		1*time.Second,
		1*time.Second,
		5*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	err := worker.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, StateConnected, worker.GetState())

	// Симулируем потерю monitor connection через state machine
	worker.stateMachine.OnMonitorDisconnected(ctx, errors.New("connection lost"))

	// Проверяем что worker перешёл в degraded mode
	assert.Equal(t, StateDegraded, worker.GetState())
	assert.True(t, worker.stateMachine.IsDegraded())
	assert.False(t, worker.stateMachine.CanExecuteChecks()) // Monitor недоступен

	worker.Stop()
}

// TestWorker_ReconnectIntegration тестирует интеграцию с reconnect manager
func TestWorker_ReconnectIntegration(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	schedulerClient := &mockSchedulerClient{
		connected: true,
	}
	monitorClient := &mockMonitorClient{
		connected: true,
	}

	worker := NewWorker(
		schedulerClient,
		monitorClient,
		&mockExecutor{},
		"test-worker",
		"test-zone",
		100*time.Millisecond,
		100*time.Millisecond,
		5*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	err := worker.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, StateConnected, worker.GetState())

	// Проверяем что GetState работает
	assert.Equal(t, StateConnected, worker.GetState())
	assert.True(t, worker.stateMachine.IsConnected())

	// Проверяем что reconnect manager создан
	assert.NotNil(t, worker.reconnectMgr)

	worker.Stop()
	assert.Equal(t, StateShutdown, worker.GetState())
}

// TestWorker_ExecuteCheckWithDegradedState тестирует выполнение проверок в degraded state
func TestWorker_ExecuteCheckWithDegradedState(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	schedulerClient := &mockSchedulerClient{
		connected: true,
	}
	monitorClient := &mockMonitorClient{
		connected: true,
	}

	worker := NewWorker(
		schedulerClient,
		monitorClient,
		&mockExecutor{},
		"test-worker",
		"test-zone",
		100*time.Millisecond,
		100*time.Millisecond,
		5*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	err := worker.Start(ctx)
	require.NoError(t, err)

	// Симулируем degraded state через state machine
	worker.stateMachine.OnMonitorDisconnected(ctx, errors.New("connection lost"))

	// Проверяем что проверки не могут выполняться
	assert.Equal(t, StateDegraded, worker.GetState())
	assert.False(t, worker.stateMachine.CanExecuteChecks())

	// Восстанавливаем connection
	worker.stateMachine.OnMonitorConnected(ctx)

	// Теперь проверки должны быть доступны
	assert.True(t, worker.stateMachine.CanExecuteChecks())

	worker.Stop()
}

// TestWorker_ConnectionInfo тестирует GetConnectionInfo
func TestWorker_ConnectionInfo(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	schedulerClient := &mockSchedulerClient{
		connected: true,
	}
	monitorClient := &mockMonitorClient{
		connected: true,
	}

	worker := NewWorker(
		schedulerClient,
		monitorClient,
		&mockExecutor{},
		"test-worker",
		"test-zone",
		1*time.Second,
		1*time.Second,
		5*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	err := worker.Start(ctx)
	require.NoError(t, err)

	// Проверяем GetConnectionInfo
	scheduler, monitor := worker.GetConnectionInfo()
	assert.Equal(t, "scheduler", scheduler.Name)
	assert.Equal(t, "monitor", monitor.Name)
	assert.True(t, scheduler.Connected)
	assert.True(t, monitor.Connected)
	assert.Nil(t, scheduler.LastError)
	assert.Nil(t, monitor.LastError)
	assert.Equal(t, 0, scheduler.ReconnectNum)
	assert.Equal(t, 0, monitor.ReconnectNum)

	worker.Stop()
}
