package adapters

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
	"github.com/raul/monitor/backend/dashboard-service/internal/repository"
)

type DashboardAdapter struct {
	statusRepo repository.MonitorStatusRepository
	exportDir  string
}

func NewDashboardAdapter(statusRepo repository.MonitorStatusRepository, exportDir string) *DashboardAdapter {
	return &DashboardAdapter{statusRepo: statusRepo, exportDir: exportDir}
}

func (a *DashboardAdapter) GetDashboard(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, float64, error) {
	views, total, err := a.statusRepo.ListByUserID(ctx, userID, filter)
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "failed to list monitors")
	}
	uptime := 0.0
	if u, err := a.statusRepo.GetOverallUptime(ctx, userID, filter.Statuses); err == nil {
		uptime = u
	}
	return views, total, uptime, nil
}

func (a *DashboardAdapter) ExportDashboard(ctx context.Context, userID string, filter model.DashboardFilter, format string) (result string, err error) {
	views, _, err := a.statusRepo.ListByUserID(ctx, userID, filter)
	if err != nil {
		return "", errors.Wrap(err, "failed to list monitors for export")
	}

	if err := os.MkdirAll(a.exportDir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create export directory")
	}
	filename := fmt.Sprintf("dashboard_%s_%d.%s", uuid.New().String()[:8], time.Now().Unix(), format)
	filePath := filepath.Join(a.exportDir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to create export file")
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, "failed to close export file")
		}
	}()

	switch format {
	case "csv":
		w := csv.NewWriter(f)
		if err := w.Write([]string{"id", "name", "url", "status", "uptime_percentage", "last_checked_at"}); err != nil {
			return "", errors.Wrap(err, "failed to write CSV header")
		}
		for _, v := range views {
			lastChecked := ""
			if v.LastCheckedAt != nil {
				lastChecked = v.LastCheckedAt.Format(time.RFC3339)
			}
			if err := w.Write([]string{v.ID.String(), v.Name, v.URL, string(v.Status), fmt.Sprintf("%.2f", v.UptimePercentage), lastChecked}); err != nil {
				return "", errors.Wrap(err, "failed to write CSV row")
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return "", errors.Wrap(err, "failed to flush CSV")
		}
	case "json":
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(views); err != nil {
			return "", errors.Wrap(err, "failed to encode JSON")
		}
	default:
		return "", errors.Errorf("unsupported export format: %s", format)
	}

	return "/exports/" + filename, nil
}
