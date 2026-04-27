package executor

import (
	"context"
	"fmt"
	"time"

	api "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// GRPCMaintenanceConfig хранит параметры соединения с maintenance-service.
type GRPCMaintenanceConfig struct {
	// Address — адрес gRPC-сервера maintenance-service (host:port).
	Address string
	// Timeout — таймаут одного RPC-вызова.
	Timeout time.Duration
}

// GRPCMaintenanceClient реализует MaintenanceClient через gRPC API maintenance-service.
type GRPCMaintenanceClient struct {
	cfg    GRPCMaintenanceConfig
	conn   *grpc.ClientConn
	client api.MaintenanceWindowServiceClient
}

// NewGRPCMaintenanceClient создаёт клиент обслуживания без установки соединения.
// Соединение устанавливается при первом вызове или через Connect.
func NewGRPCMaintenanceClient(cfg GRPCMaintenanceConfig) *GRPCMaintenanceClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &GRPCMaintenanceClient{cfg: cfg}
}

// Connect устанавливает gRPC-соединение с maintenance-service.
func (c *GRPCMaintenanceClient) Connect(ctx context.Context) error {
	conn, err := grpc.NewClient(
		c.cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to maintenance service: %w", err)
	}
	c.conn = conn
	c.client = api.NewMaintenanceWindowServiceClient(conn)
	return nil
}

// Close закрывает gRPC-соединение.
func (c *GRPCMaintenanceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetActiveMaintenanceWindows запрашивает у maintenance-service список активных окон обслуживания
// и преобразует их во внутренний формат.
func (c *GRPCMaintenanceClient) GetActiveMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error) {
	if c.client == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	rpcCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	resp, err := c.client.ListMaintenanceWindows(rpcCtx, &api.ListMaintenanceWindowsRequest{
		Status: api.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE,
	})
	if err != nil {
		// При недоступности сервиса возвращаем пустой список, чтобы не блокировать проверки
		if isUnavailable(err) {
			return []MaintenanceWindow{}, nil
		}
		return nil, fmt.Errorf("failed to list active maintenance windows: %w", err)
	}

	return convertProtoWindows(resp.GetWindows()), nil
}

// isUnavailable проверяет, недоступен ли gRPC-сервер.
func isUnavailable(err error) bool {
	if st, ok := status.FromError(err); ok {
		return st.Code() == codes.Unavailable
	}
	return false
}

// convertProtoWindows преобразует список proto-окон во внутренний формат.
// Для глобальных окон создаётся одна запись; для окон с конкретными мониторами —
// по одной записи на каждый монитор из списка monitor_ids.
func convertProtoWindows(windows []*api.MaintenanceWindow) []MaintenanceWindow {
	result := make([]MaintenanceWindow, 0, len(windows))
	for _, w := range windows {
		if w == nil {
			continue
		}

		var startTime, endTime time.Time
		if w.GetStartTime() != nil {
			startTime = w.GetStartTime().AsTime()
		}
		if w.GetEndTime() != nil {
			endTime = w.GetEndTime().AsTime()
		}

		if w.GetIsGlobal() || len(w.GetMonitorIds()) == 0 {
			// Глобальное окно или без привязки к мониторам
			result = append(result, MaintenanceWindow{
				ID:              w.GetId(),
				IsGlobal:        true,
				StartTime:       startTime,
				EndTime:         endTime,
				PauseMonitoring: w.GetPauseMonitoring(),
			})
			continue
		}

		// Одна запись на каждый монитор в списке
		for _, monitorID := range w.GetMonitorIds() {
			result = append(result, MaintenanceWindow{
				ID:              w.GetId(),
				MonitorID:       monitorID,
				IsGlobal:        false,
				StartTime:       startTime,
				EndTime:         endTime,
				PauseMonitoring: w.GetPauseMonitoring(),
			})
		}
	}
	return result
}

// ConfigMaintenanceClient реализует MaintenanceClient на основе предзагруженного списка окон.
// Используется для тестирования или когда список окон известен заранее из конфигурации.
type ConfigMaintenanceClient struct {
	windows []MaintenanceWindow
}

// NewConfigMaintenanceClient создаёт клиент на основе конфигурации.
func NewConfigMaintenanceClient(windows []MaintenanceWindow) *ConfigMaintenanceClient {
	return &ConfigMaintenanceClient{
		windows: windows,
	}
}

// GetActiveMaintenanceWindows возвращает преднастроенные активные окна.
func (c *ConfigMaintenanceClient) GetActiveMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error) {
	active := make([]MaintenanceWindow, 0, len(c.windows))
	for _, window := range c.windows {
		if window.IsActive() {
			active = append(active, window)
		}
	}
	return active, nil
}

// ExampleActiveWindows создаёт пример активных окон для тестирования.
func ExampleActiveWindows() []MaintenanceWindow {
	return []MaintenanceWindow{
		{
			ID:              "global-weekly",
			IsGlobal:        true,
			StartTime:       time.Now().Add(-1 * time.Hour),
			EndTime:         time.Now().Add(1 * time.Hour),
			PauseMonitoring: true,
		},
		{
			ID:              "monitor-1-daily",
			MonitorID:       "monitor-1",
			IsGlobal:        false,
			StartTime:       time.Now().Add(-30 * time.Minute),
			EndTime:         time.Now().Add(30 * time.Minute),
			PauseMonitoring: true,
		},
	}
}

// ErrorMaintenanceClient всегда возвращает ошибку — применяется в тестах для имитации сбоя.
type ErrorMaintenanceClient struct {
	err error
}

// NewErrorMaintenanceClient создаёт клиент, возвращающий заданную ошибку.
func NewErrorMaintenanceClient(err error) *ErrorMaintenanceClient {
	return &ErrorMaintenanceClient{err: err}
}

// GetActiveMaintenanceWindows возвращает ошибку.
func (c *ErrorMaintenanceClient) GetActiveMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error) {
	return nil, fmt.Errorf("maintenance client error: %w", c.err)
}
