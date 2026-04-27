package monitor

import (
	"context"
	"log/slog"
	"time"

	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
)

type MonitorInfo struct {
	ID              string
	Name            string
	Status          string
	IntervalSeconds int32
	LastCheckedAt   time.Time
}

type Adapter struct {
	client *Client
	logger *slog.Logger
}

func NewAdapter(client *Client, logger *slog.Logger) *Adapter {
	return &Adapter{client: client, logger: logger}
}

func (a *Adapter) GetMonitor(ctx context.Context, monitorID string) (*MonitorInfo, error) {
	mon, err := a.client.GetMonitor(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	return toMonitorInfo(mon), nil
}

func (a *Adapter) ListActiveMonitors(ctx context.Context, pageSize int) ([]*MonitorInfo, error) {
	monitors, err := a.client.ListActiveMonitors(ctx, pageSize)
	if err != nil {
		return nil, err
	}

	result := make([]*MonitorInfo, 0, len(monitors))
	for _, m := range monitors {
		result = append(result, toMonitorInfo(m))
	}

	return result, nil
}

func (a *Adapter) IsMonitorPaused(ctx context.Context, monitorID string) (bool, error) {
	return a.client.IsMonitorPaused(ctx, monitorID)
}

func toMonitorInfo(m *monitov1.Monitor) *MonitorInfo {
	info := &MonitorInfo{
		ID:              m.Id,
		Name:            m.Name,
		Status:          m.Status,
		IntervalSeconds: m.IntervalSeconds,
	}

	if m.LastCheckAt != nil {
		info.LastCheckedAt = m.LastCheckAt.AsTime()
	}

	return info
}
