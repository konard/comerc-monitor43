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

type HistoryAdapter struct {
	checkRepo    repository.CheckHistoryRepository
	incidentRepo repository.IncidentRepository
	exportDir    string
}

func NewHistoryAdapter(checkRepo repository.CheckHistoryRepository, incidentRepo repository.IncidentRepository, exportDir string) *HistoryAdapter {
	return &HistoryAdapter{checkRepo: checkRepo, incidentRepo: incidentRepo, exportDir: exportDir}
}

func (a *HistoryAdapter) GetCheckHistory(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
	entries, total, err := a.checkRepo.ListByMonitorID(ctx, monitorID, filter)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list check history")
	}
	return entries, total, nil
}

func (a *HistoryAdapter) GetIncidents(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error) {
	incidents, total, err := a.incidentRepo.ListByMonitorID(ctx, monitorID, filter)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list incidents")
	}
	return incidents, total, nil
}

func (a *HistoryAdapter) GetIncidentDetails(ctx context.Context, incidentID string) (*model.IncidentDetail, error) {
	incident, err := a.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get incident")
	}
	if incident == nil {
		return nil, model.ErrIncidentNotFound
	}

	timelineFilter := model.HistoryFilter{
		StartDate: &incident.StartedAt,
		SortBy:    "checked_at",
		SortOrder: "asc",
		Page:      1,
		PageSize:  1000,
	}
	if incident.EndedAt != nil {
		timelineFilter.EndDate = incident.EndedAt
	}

	timeline := make([]*model.CheckHistoryEntry, 0)
	if t, _, err := a.checkRepo.ListByMonitorID(ctx, incident.MonitorID.String(), timelineFilter); err == nil {
		timeline = t
	}
	tl := make([]model.CheckHistoryEntry, 0, len(timeline))
	for _, e := range timeline {
		tl = append(tl, *e)
	}

	return &model.IncidentDetail{
		Incident: *incident,
		Timeline: tl,
	}, nil
}

func (a *HistoryAdapter) GetPeriodMetrics(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error) {
	metrics, err := a.checkRepo.GetPeriodMetrics(ctx, monitorID, start, end)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get period metrics")
	}
	return metrics, nil
}

func (a *HistoryAdapter) ExportHistory(ctx context.Context, monitorID string, filter model.HistoryFilter, format string, fields []string) (result string, err error) {
	entries, _, err := a.checkRepo.ListByMonitorID(ctx, monitorID, filter)
	if err != nil {
		return "", errors.Wrap(err, "failed to list history for export")
	}

	if err := os.MkdirAll(a.exportDir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create export directory")
	}
	filename := fmt.Sprintf("history_%s_%d.%s", uuid.New().String()[:8], time.Now().Unix(), format)
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
		if len(fields) == 0 {
			fields = []string{"timestamp", "status", "status_code", "response_time_ms", "error_message"}
		}
		if err := w.Write(fields); err != nil {
			return "", errors.Wrap(err, "failed to write CSV header")
		}
		for _, e := range entries {
			row := make([]string, len(fields))
			for i, field := range fields {
				switch field {
				case "timestamp":
					row[i] = e.CheckedAt.Format(time.RFC3339)
				case "status":
					row[i] = string(e.Status)
				case "status_code":
					if e.StatusCode != nil {
						row[i] = fmt.Sprintf("%d", *e.StatusCode)
					}
				case "response_time_ms":
					if e.ResponseTimeMs != nil {
						row[i] = fmt.Sprintf("%.2f", *e.ResponseTimeMs)
					}
				case "error_message":
					if e.ErrorMessage != nil {
						row[i] = *e.ErrorMessage
					}
				}
			}
			if err := w.Write(row); err != nil {
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
		if err := enc.Encode(entries); err != nil {
			return "", errors.Wrap(err, "failed to encode JSON")
		}
	default:
		return "", errors.Errorf("unsupported export format: %s", format)
	}

	return "/exports/" + filename, nil
}
