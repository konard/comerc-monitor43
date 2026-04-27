package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ConnectionState представляет состояние подключения
type ConnectionState int

const (
	// StateDisconnected - начальное состояние или после потери соединения
	StateDisconnected ConnectionState = iota
	// StateConnecting -正在进行 подключение
	StateConnecting
	// StateConnected - успешно подключено
	StateConnected
	// StateReconnecting - переподключение после потери соединения
	StateReconnecting
	// StateDegraded - частично работоспособен (одно соединение потеряно)
	StateDegraded
	// StateShutdown - остановлен
	StateShutdown
)

// String возвращает строковое представление состояния
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "DISCONNECTED"
	case StateConnecting:
		return "CONNECTING"
	case StateConnected:
		return "CONNECTED"
	case StateReconnecting:
		return "RECONNECTING"
	case StateDegraded:
		return "DEGRADED"
	case StateShutdown:
		return "SHUTDOWN"
	default:
		return "UNKNOWN"
	}
}

// ConnectionInfo содержит информацию о соединении
type ConnectionInfo struct {
	Name         string
	Connected    bool
	LastError    error
	LastAttempt  time.Time
	ReconnectNum int
}

// ConnectionStateMachine управляет состояниями подключений Worker
type ConnectionStateMachine struct {
	mu                sync.RWMutex
	state             ConnectionState
	schedulerConn     ConnectionInfo
	monitorConn       ConnectionInfo
	logger            *slog.Logger
	reconnectCallback func(ctx context.Context) error
}

// NewConnectionStateMachine создаёт новый state machine
func NewConnectionStateMachine(logger *slog.Logger) *ConnectionStateMachine {
	return &ConnectionStateMachine{
		state: StateDisconnected,
		schedulerConn: ConnectionInfo{
			Name:      "scheduler",
			Connected: false,
		},
		monitorConn: ConnectionInfo{
			Name:      "monitor",
			Connected: false,
		},
		logger: logger,
	}
}

// SetReconnectCallback устанавливает callback для переподключения
func (sm *ConnectionStateMachine) SetReconnectCallback(cb func(ctx context.Context) error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.reconnectCallback = cb
}

// GetState возвращает текущее состояние
func (sm *ConnectionStateMachine) GetState() ConnectionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// setState устанавливает новое состояние с логированием
func (sm *ConnectionStateMachine) setState(newState ConnectionState) {
	oldState := sm.state
	sm.state = newState

	sm.logger.Info("connection state changed",
		"old_state", oldState,
		"new_state", newState,
	)
}

// OnSchedulerConnected вызывается при успешном подключении к scheduler
func (sm *ConnectionStateMachine) OnSchedulerConnected(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.schedulerConn.Connected = true
	sm.schedulerConn.LastError = nil
	sm.schedulerConn.ReconnectNum = 0

	sm.updateState(ctx)
}

// OnSchedulerDisconnected вызывается при потере соединения с scheduler
func (sm *ConnectionStateMachine) OnSchedulerDisconnected(ctx context.Context, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.schedulerConn.Connected = false
	sm.schedulerConn.LastError = err
	sm.schedulerConn.LastAttempt = time.Now()
	sm.schedulerConn.ReconnectNum++

	sm.logger.Warn("scheduler connection lost",
		"error", err,
		"reconnect_num", sm.schedulerConn.ReconnectNum,
	)

	sm.updateState(ctx)
}

// OnMonitorConnected вызывается при успешном подключении к monitor service
func (sm *ConnectionStateMachine) OnMonitorConnected(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.monitorConn.Connected = true
	sm.monitorConn.LastError = nil
	sm.monitorConn.ReconnectNum = 0

	sm.updateState(ctx)
}

// OnMonitorDisconnected вызывается при потере соединения с monitor service
func (sm *ConnectionStateMachine) OnMonitorDisconnected(ctx context.Context, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.monitorConn.Connected = false
	sm.monitorConn.LastError = err
	sm.monitorConn.LastAttempt = time.Now()
	sm.monitorConn.ReconnectNum++

	sm.logger.Warn("monitor connection lost",
		"error", err,
		"reconnect_num", sm.monitorConn.ReconnectNum,
	)

	sm.updateState(ctx)
}

// updateState обновляет состояние на основе статов подключений
func (sm *ConnectionStateMachine) updateState(ctx context.Context) {
	// Не меняем состояние если уже shutdown
	if sm.state == StateShutdown {
		return
	}

	// Оба соединения активны
	if sm.schedulerConn.Connected && sm.monitorConn.Connected {
		if sm.state != StateConnected {
			sm.setState(StateConnected)
		}
		return
	}

	// Одно соединение потеряно - degraded mode
	if sm.schedulerConn.Connected || sm.monitorConn.Connected {
		if sm.state != StateDegraded {
			sm.setState(StateDegraded)
		}
		return
	}

	// Оба соединения потеряны
	if !sm.schedulerConn.Connected && !sm.monitorConn.Connected {
		// Если это первая попытка подключения
		if sm.state == StateDisconnected {
			sm.setState(StateConnecting)
			return
		}

		// Если переподключение
		sm.setState(StateReconnecting)
		return
	}
}

// IsConnected проверяет, полностью ли подключен Worker
func (sm *ConnectionStateMachine) IsConnected() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state == StateConnected
}

// IsDegraded проверяет, находится ли Worker в degraded mode
func (sm *ConnectionStateMachine) IsDegraded() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state == StateDegraded
}

// CanExecuteChecks проверяет, может ли Worker выполнять проверки
func (sm *ConnectionStateMachine) CanExecuteChecks() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// В degraded mode можно выполнять проверки если monitor доступен
	if sm.state == StateDegraded {
		return sm.monitorConn.Connected
	}

	// В подключенном состоянии можно выполнять проверки
	return sm.state == StateConnected
}

// Shutdown переводит state machine в shutdown состояние
func (sm *ConnectionStateMachine) Shutdown() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.setState(StateShutdown)

	// Сбрасываем соединения
	sm.schedulerConn.Connected = false
	sm.monitorConn.Connected = false
}

// GetConnectionInfo возвращает информацию о соединениях
func (sm *ConnectionStateMachine) GetConnectionInfo() (scheduler, monitor ConnectionInfo) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Копируем ConnectionInfo чтобы избежать race conditions
	scheduler = ConnectionInfo{
		Name:         sm.schedulerConn.Name,
		Connected:    sm.schedulerConn.Connected,
		LastError:    sm.schedulerConn.LastError,
		LastAttempt:  sm.schedulerConn.LastAttempt,
		ReconnectNum: sm.schedulerConn.ReconnectNum,
	}

	monitor = ConnectionInfo{
		Name:         sm.monitorConn.Name,
		Connected:    sm.monitorConn.Connected,
		LastError:    sm.monitorConn.LastError,
		LastAttempt:  sm.monitorConn.LastAttempt,
		ReconnectNum: sm.monitorConn.ReconnectNum,
	}

	return scheduler, monitor
}

// String возвращает детальную информацию о состоянии
func (sm *ConnectionStateMachine) String() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	scheduler, monitor := sm.GetConnectionInfo()

	return fmt.Sprintf("State: %s, Scheduler: %s (connected=%v, reconnects=%d), Monitor: %s (connected=%v, reconnects=%d)",
		sm.state,
		scheduler.Name, scheduler.Connected, scheduler.ReconnectNum,
		monitor.Name, monitor.Connected, monitor.ReconnectNum,
	)
}
