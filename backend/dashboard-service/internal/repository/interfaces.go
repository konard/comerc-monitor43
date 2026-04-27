package repository

import (
	"context"
	"time"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
)

type MonitorStatusRepository interface {
	Upsert(ctx context.Context, status *model.MonitorStatusView) error
	Delete(ctx context.Context, id string) error
	ListByUserID(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error)
	GetByID(ctx context.Context, id string) (*model.MonitorStatusView, error)
	GetOverallUptime(ctx context.Context, userID string, statuses []model.MonitorStatus) (float64, error)
}

type CheckHistoryRepository interface {
	Create(ctx context.Context, entry *model.CheckHistoryEntry) error
	ListByMonitorID(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error)
	GetPeriodMetrics(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error)
}

type IncidentRepository interface {
	Upsert(ctx context.Context, incident *model.Incident) error
	ListByMonitorID(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error)
	GetByID(ctx context.Context, id string) (*model.Incident, error)
	Update(ctx context.Context, incident *model.Incident) error
}
