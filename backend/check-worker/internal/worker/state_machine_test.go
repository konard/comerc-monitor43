package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "DISCONNECTED"},
		{StateConnecting, "CONNECTING"},
		{StateConnected, "CONNECTED"},
		{StateReconnecting, "RECONNECTING"},
		{StateDegraded, "DEGRADED"},
		{StateShutdown, "SHUTDOWN"},
		{ConnectionState(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.state.String()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNewConnectionStateMachine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)

	assert.Equal(t, StateDisconnected, sm.GetState())
	assert.False(t, sm.IsConnected())
	assert.False(t, sm.IsDegraded())
	assert.False(t, sm.CanExecuteChecks())
}

func TestConnectionStateMachine_OnSchedulerConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Scheduler подключен, monitor нет
	sm.OnSchedulerConnected(ctx)
	assert.Equal(t, StateDegraded, sm.GetState())
	assert.False(t, sm.IsConnected())
	assert.True(t, sm.IsDegraded())
	assert.False(t, sm.CanExecuteChecks()) // Monitor недоступен для выполнения проверок
}

func TestConnectionStateMachine_OnMonitorConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Monitor подключен, scheduler нет
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateDegraded, sm.GetState())
	assert.False(t, sm.IsConnected())
	assert.True(t, sm.IsDegraded())
	assert.True(t, sm.CanExecuteChecks()) // Monitor доступен для выполнения проверок
}

func TestConnectionStateMachine_BothConnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Оба подключены
	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())
	assert.True(t, sm.IsConnected())
	assert.False(t, sm.IsDegraded())
	assert.True(t, sm.CanExecuteChecks())
}

func TestConnectionStateMachine_SchedulerDisconnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Сначала подключаем оба
	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())

	// Scheduler теряет соединение
	err := errors.New("scheduler connection lost")
	sm.OnSchedulerDisconnected(ctx, err)
	assert.Equal(t, StateDegraded, sm.GetState())
	assert.True(t, sm.CanExecuteChecks()) // Monitor доступен
}

func TestConnectionStateMachine_MonitorDisconnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Сначала подключаем оба
	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())

	// Monitor теряет соединение
	err := errors.New("monitor connection lost")
	sm.OnMonitorDisconnected(ctx, err)
	assert.Equal(t, StateDegraded, sm.GetState())
	assert.False(t, sm.CanExecuteChecks()) // Monitor недоступен
}

func TestConnectionStateMachine_BothDisconnected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Сначала подключаем оба
	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())

	// Оба теряют соединение
	sm.OnSchedulerDisconnected(ctx, errors.New("scheduler lost"))
	sm.OnMonitorDisconnected(ctx, errors.New("monitor lost"))
	assert.Equal(t, StateReconnecting, sm.GetState())
	assert.False(t, sm.IsConnected())
	assert.False(t, sm.IsDegraded())
	assert.False(t, sm.CanExecuteChecks())
}

func TestConnectionStateMachine_ReconnectCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Подключаем
	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())

	// Первый disconnect
	sm.OnSchedulerDisconnected(ctx, errors.New("lost"))
	scheduler, _ := sm.GetConnectionInfo()
	assert.Equal(t, 1, scheduler.ReconnectNum)

	// Второй disconnect
	sm.OnSchedulerDisconnected(ctx, errors.New("lost again"))
	scheduler, _ = sm.GetConnectionInfo()
	assert.Equal(t, 2, scheduler.ReconnectNum)
}

func TestConnectionStateMachine_Shutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	// Подключаем
	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())

	// Shutdown
	sm.Shutdown()
	assert.Equal(t, StateShutdown, sm.GetState())
	assert.False(t, sm.IsConnected())
	assert.False(t, sm.CanExecuteChecks())

	// Попытки подключиться не меняют состояние
	sm.OnSchedulerConnected(ctx)
	assert.Equal(t, StateShutdown, sm.GetState())
}

func TestConnectionStateMachine_GetConnectionInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	sm.OnSchedulerConnected(ctx)
	scheduler, monitor := sm.GetConnectionInfo()

	assert.Equal(t, "scheduler", scheduler.Name)
	assert.True(t, scheduler.Connected)
	assert.Nil(t, scheduler.LastError)

	assert.Equal(t, "monitor", monitor.Name)
	assert.False(t, monitor.Connected)
}

func TestConnectionStateMachine_String(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	sm.OnSchedulerConnected(ctx)
	sm.OnMonitorConnected(ctx)

	str := sm.String()
	assert.Contains(t, str, "CONNECTED")
	assert.Contains(t, str, "scheduler")
	assert.Contains(t, str, "monitor")
}

func TestConnectionStateMachine_CanExecuteChecks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	tests := []struct {
		name             string
		setup            func()
		canExecuteChecks bool
		expectedState    ConnectionState
	}{
		{
			name: "both connected",
			setup: func() {
				sm.OnSchedulerConnected(ctx)
				sm.OnMonitorConnected(ctx)
			},
			canExecuteChecks: true,
			expectedState:    StateConnected,
		},
		{
			name: "only monitor connected",
			setup: func() {
				sm.OnMonitorConnected(ctx)
			},
			canExecuteChecks: true,
			expectedState:    StateDegraded,
		},
		{
			name: "only scheduler connected",
			setup: func() {
				sm.OnSchedulerConnected(ctx)
			},
			canExecuteChecks: false,
			expectedState:    StateDegraded,
		},
		{
			name:             "neither connected",
			setup:            func() {},
			canExecuteChecks: false,
			expectedState:    StateDisconnected, // Начальное состояние
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state machine
			sm = NewConnectionStateMachine(logger)

			tt.setup()
			assert.Equal(t, tt.expectedState, sm.GetState())
			assert.Equal(t, tt.canExecuteChecks, sm.CanExecuteChecks())
		})
	}
}

func TestConnectionStateMachine_SetReconnectCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)
	ctx := context.Background()

	sm.SetReconnectCallback(func(reconnectCtx context.Context) error {
		return nil
	})

	// Подключаем scheduler
	sm.OnSchedulerConnected(ctx)

	// Подключаем monitor - должно сработать подключение
	sm.OnMonitorConnected(ctx)
	assert.Equal(t, StateConnected, sm.GetState())

	// Сбрасываем соединения для запуска reconnect
	sm.OnSchedulerDisconnected(ctx, errors.New("connection lost"))
	sm.OnMonitorDisconnected(ctx, errors.New("connection lost"))
	assert.Equal(t, StateReconnecting, sm.GetState())

	// Callback должен быть установлен
	assert.NotNil(t, sm)
}

func TestConnectionStateMachine_SetReconnectCallback_nil(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sm := NewConnectionStateMachine(logger)

	// Установка nil callback не должна паниковать
	sm.SetReconnectCallback(nil)
	assert.NotNil(t, sm)
}
