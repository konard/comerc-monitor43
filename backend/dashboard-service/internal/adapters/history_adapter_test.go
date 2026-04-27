package adapters

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/model"
)

type mockCheckHistoryRepository struct {
	listByMonitorIDFunc  func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error)
	getPeriodMetricsFunc func(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error)
}

func (m *mockCheckHistoryRepository) Create(ctx context.Context, entry *model.CheckHistoryEntry) error {
	return nil
}

func (m *mockCheckHistoryRepository) ListByMonitorID(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
	return m.listByMonitorIDFunc(ctx, monitorID, filter)
}

func (m *mockCheckHistoryRepository) GetPeriodMetrics(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error) {
	return m.getPeriodMetricsFunc(ctx, monitorID, start, end)
}

type mockIncidentRepository struct {
	listByMonitorIDFunc func(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error)
	getByIDFunc         func(ctx context.Context, id string) (*model.Incident, error)
}

func (m *mockIncidentRepository) Upsert(ctx context.Context, incident *model.Incident) error {
	return nil
}

func (m *mockIncidentRepository) ListByMonitorID(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error) {
	return m.listByMonitorIDFunc(ctx, monitorID, filter)
}

func (m *mockIncidentRepository) GetByID(ctx context.Context, id string) (*model.Incident, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *mockIncidentRepository) Update(ctx context.Context, incident *model.Incident) error {
	return nil
}

func makeCheckHistoryEntry(status model.CheckStatus, statusCode int, respTimeMs float64) *model.CheckHistoryEntry {
	sc := statusCode
	rt := respTimeMs
	return &model.CheckHistoryEntry{
		ID:             uuid.New(),
		MonitorID:      uuid.New(),
		Status:         status,
		StatusCode:     &sc,
		ResponseTimeMs: &rt,
		CheckedAt:      time.Now(),
	}
}

func makeIncident(monitorID uuid.UUID, startedAt time.Time) *model.Incident {
	return &model.Incident{
		ID:        uuid.New(),
		MonitorID: monitorID,
		StartedAt: startedAt,
		Status:    model.IncidentStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestGetCheckHistory(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		entries := []*model.CheckHistoryEntry{
			makeCheckHistoryEntry(model.CheckStatusUP, 200, 45.2),
			makeCheckHistoryEntry(model.CheckStatusDOWN, 500, 0),
		}

		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return entries, 2, nil
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, t.TempDir())

		result, total, err := adapter.GetCheckHistory(context.Background(), "monitor-1", model.HistoryFilter{})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return nil, 0, errors.New("db error")
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, t.TempDir())

		result, total, err := adapter.GetCheckHistory(context.Background(), "monitor-1", model.HistoryFilter{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list check history")
		assert.Nil(t, result)
		assert.Equal(t, 0, total)
	})
}

func TestGetIncidents(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		incidents := []*model.Incident{
			makeIncident(uuid.New(), time.Now().Add(-time.Hour)),
			makeIncident(uuid.New(), time.Now().Add(-2*time.Hour)),
		}

		repo := &mockIncidentRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error) {
				return incidents, 2, nil
			},
		}

		adapter := NewHistoryAdapter(&mockCheckHistoryRepository{}, repo, t.TempDir())

		result, total, err := adapter.GetIncidents(context.Background(), "monitor-1", model.IncidentFilter{})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		repo := &mockIncidentRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.IncidentFilter) ([]*model.Incident, int, error) {
				return nil, 0, errors.New("db error")
			},
		}

		adapter := NewHistoryAdapter(&mockCheckHistoryRepository{}, repo, t.TempDir())

		result, total, err := adapter.GetIncidents(context.Background(), "monitor-1", model.IncidentFilter{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list incidents")
		assert.Nil(t, result)
		assert.Equal(t, 0, total)
	})
}

func TestGetIncidentDetails(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		monitorID := uuid.New()
		incident := makeIncident(monitorID, time.Now().Add(-time.Hour))

		timeline := []*model.CheckHistoryEntry{
			makeCheckHistoryEntry(model.CheckStatusDOWN, 500, 0),
			makeCheckHistoryEntry(model.CheckStatusUP, 200, 120.5),
		}

		incidentRepo := &mockIncidentRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.Incident, error) {
				return incident, nil
			},
		}
		checkRepo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return timeline, 2, nil
			},
		}

		adapter := NewHistoryAdapter(checkRepo, incidentRepo, t.TempDir())

		result, err := adapter.GetIncidentDetails(context.Background(), incident.ID.String())

		require.NoError(t, err)
		assert.Equal(t, incident.ID, result.Incident.ID)
		assert.Len(t, result.Timeline, 2)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		incidentRepo := &mockIncidentRepository{
			getByIDFunc: func(ctx context.Context, id string) (*model.Incident, error) {
				return nil, nil
			},
		}

		adapter := NewHistoryAdapter(&mockCheckHistoryRepository{}, incidentRepo, t.TempDir())

		result, err := adapter.GetIncidentDetails(context.Background(), uuid.New().String())

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrIncidentNotFound)
		assert.Nil(t, result)
	})
}

func TestGetPeriodMetrics(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		expected := &model.PeriodMetrics{
			TotalChecks:      100,
			SuccessCount:     95,
			FailedCount:      3,
			DegradedCount:    2,
			UptimePercentage: 95.0,
			P50ResponseMs:    50.0,
			P95ResponseMs:    200.0,
			P99ResponseMs:    500.0,
		}

		repo := &mockCheckHistoryRepository{
			getPeriodMetricsFunc: func(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error) {
				return expected, nil
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, t.TempDir())

		result, err := adapter.GetPeriodMetrics(context.Background(), "monitor-1", time.Now().Add(-24*time.Hour), time.Now())

		require.NoError(t, err)
		assert.Equal(t, expected.TotalChecks, result.TotalChecks)
		assert.Equal(t, expected.UptimePercentage, result.UptimePercentage)
		assert.Equal(t, expected.P95ResponseMs, result.P95ResponseMs)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		repo := &mockCheckHistoryRepository{
			getPeriodMetricsFunc: func(ctx context.Context, monitorID string, start, end time.Time) (*model.PeriodMetrics, error) {
				return nil, errors.New("metrics query failed")
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, t.TempDir())

		result, err := adapter.GetPeriodMetrics(context.Background(), "monitor-1", time.Now().Add(-24*time.Hour), time.Now())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get period metrics")
		assert.Nil(t, result)
	})
}

func TestExportHistory(t *testing.T) {
	t.Parallel()

	entries := []*model.CheckHistoryEntry{
		makeCheckHistoryEntry(model.CheckStatusUP, 200, 45.2),
		makeCheckHistoryEntry(model.CheckStatusDOWN, 503, 0),
	}

	t.Run("csv", func(t *testing.T) {
		t.Parallel()

		exportDir := t.TempDir()
		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return entries, 2, nil
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, exportDir)

		path, err := adapter.ExportHistory(context.Background(), "monitor-1", model.HistoryFilter{}, "csv", nil)

		require.NoError(t, err)
		assert.Contains(t, path, "/exports/history_")
		assert.True(t, strings.HasSuffix(path, ".csv"))

		files, err := os.ReadDir(exportDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)

		data, err := os.ReadFile(exportDir + "/" + files[0].Name())
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "timestamp,status,status_code,response_time_ms,error_message")
		assert.Contains(t, content, "UP")
		assert.Contains(t, content, "DOWN")
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		exportDir := t.TempDir()
		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return entries, 2, nil
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, exportDir)

		path, err := adapter.ExportHistory(context.Background(), "monitor-1", model.HistoryFilter{}, "json", nil)

		require.NoError(t, err)
		assert.Contains(t, path, "/exports/history_")
		assert.True(t, strings.HasSuffix(path, ".json"))

		files, err := os.ReadDir(exportDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)

		data, err := os.ReadFile(exportDir + "/" + files[0].Name())
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, `"Status": "UP"`)
		assert.Contains(t, content, `"Status": "DOWN"`)
	})

	t.Run("custom_fields", func(t *testing.T) {
		t.Parallel()

		exportDir := t.TempDir()
		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return entries, 2, nil
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, exportDir)

		_, err := adapter.ExportHistory(context.Background(), "monitor-1", model.HistoryFilter{}, "csv", []string{"status", "response_time_ms"})

		require.NoError(t, err)

		files, err := os.ReadDir(exportDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)

		data, err := os.ReadFile(exportDir + "/" + files[0].Name())
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "status,response_time_ms")
		assert.NotContains(t, content, "timestamp")
		assert.NotContains(t, content, "status_code")
		assert.NotContains(t, content, "error_message")
	})

	t.Run("invalid_format", func(t *testing.T) {
		t.Parallel()

		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return entries, 2, nil
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, t.TempDir())

		path, err := adapter.ExportHistory(context.Background(), "monitor-1", model.HistoryFilter{}, "xml", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
		assert.Empty(t, path)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		repo := &mockCheckHistoryRepository{
			listByMonitorIDFunc: func(ctx context.Context, monitorID string, filter model.HistoryFilter) ([]*model.CheckHistoryEntry, int, error) {
				return nil, 0, errors.New("query failed")
			},
		}

		adapter := NewHistoryAdapter(repo, &mockIncidentRepository{}, t.TempDir())

		path, err := adapter.ExportHistory(context.Background(), "monitor-1", model.HistoryFilter{}, "csv", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list history for export")
		assert.Empty(t, path)
	})
}
