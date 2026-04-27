package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/reporting-service/internal/model"
)

// ReportRepository определяет интерфейс хранения SLA-отчётов.
type ReportRepository interface {
	Create(ctx context.Context, report *model.SLAReport) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.SLAReport, error)
	ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*model.SLAReport, int, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.SLAReport, int, error)
}
