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

type mockMonitorStatusRepository struct {
	listByUserIDFunc     func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error)
	getOverallUptimeFunc func(ctx context.Context, userID string, statuses []model.MonitorStatus) (float64, error)
}

func (m *mockMonitorStatusRepository) Upsert(ctx context.Context, status *model.MonitorStatusView) error {
	return nil
}

func (m *mockMonitorStatusRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMonitorStatusRepository) ListByUserID(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
	return m.listByUserIDFunc(ctx, userID, filter)
}

func (m *mockMonitorStatusRepository) GetByID(ctx context.Context, id string) (*model.MonitorStatusView, error) {
	return nil, nil
}

func (m *mockMonitorStatusRepository) GetOverallUptime(ctx context.Context, userID string, statuses []model.MonitorStatus) (float64, error) {
	return m.getOverallUptimeFunc(ctx, userID, statuses)
}

func makeMonitorStatusView(name, url string, status model.MonitorStatus, uptime float64) *model.MonitorStatusView {
	now := time.Now()
	return &model.MonitorStatusView{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Name:             name,
		URL:              url,
		Status:           status,
		UptimePercentage: uptime,
		LastCheckedAt:    &now,
	}
}

func TestGetDashboard(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		views := []*model.MonitorStatusView{
			makeMonitorStatusView("m1", "http://m1.com", model.MonitorStatusUP, 99.5),
			makeMonitorStatusView("m2", "http://m2.com", model.MonitorStatusDOWN, 50.0),
			makeMonitorStatusView("m3", "http://m3.com", model.MonitorStatusDEGRADED, 80.0),
		}

		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return views, 3, nil
			},
			getOverallUptimeFunc: func(ctx context.Context, userID string, statuses []model.MonitorStatus) (float64, error) {
				return 76.5, nil
			},
		}

		adapter := NewDashboardAdapter(repo, t.TempDir())

		result, total, uptime, err := adapter.GetDashboard(context.Background(), "user-1", model.DashboardFilter{})

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, 3, total)
		assert.Equal(t, 76.5, uptime)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return nil, 0, errors.New("db connection lost")
			},
		}

		adapter := NewDashboardAdapter(repo, t.TempDir())

		result, total, uptime, err := adapter.GetDashboard(context.Background(), "user-1", model.DashboardFilter{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list monitors")
		assert.Nil(t, result)
		assert.Equal(t, 0, total)
		assert.Equal(t, 0.0, uptime)
	})

	t.Run("uptime_error", func(t *testing.T) {
		t.Parallel()

		views := []*model.MonitorStatusView{
			makeMonitorStatusView("m1", "http://m1.com", model.MonitorStatusUP, 99.5),
		}

		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return views, 1, nil
			},
			getOverallUptimeFunc: func(ctx context.Context, userID string, statuses []model.MonitorStatus) (float64, error) {
				return 0, errors.New("uptime calculation failed")
			},
		}

		adapter := NewDashboardAdapter(repo, t.TempDir())

		result, total, uptime, err := adapter.GetDashboard(context.Background(), "user-1", model.DashboardFilter{})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, total)
		assert.Equal(t, 0.0, uptime)
	})
}

func TestExportDashboard(t *testing.T) {
	t.Parallel()

	views := []*model.MonitorStatusView{
		makeMonitorStatusView("m1", "http://m1.com", model.MonitorStatusUP, 99.5),
		makeMonitorStatusView("m2", "http://m2.com", model.MonitorStatusDOWN, 50.0),
	}

	t.Run("csv", func(t *testing.T) {
		t.Parallel()

		exportDir := t.TempDir()
		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return views, 2, nil
			},
		}

		adapter := NewDashboardAdapter(repo, exportDir)

		path, err := adapter.ExportDashboard(context.Background(), "user-1", model.DashboardFilter{}, "csv")

		require.NoError(t, err)
		assert.Contains(t, path, "/exports/dashboard_")
		assert.True(t, strings.HasSuffix(path, ".csv"))

		files, err := os.ReadDir(exportDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)

		data, err := os.ReadFile(exportDir + "/" + files[0].Name())
		require.NoError(t, err)
		assert.Contains(t, string(data), "id,name,url,status,uptime_percentage,last_checked_at")
		assert.Contains(t, string(data), "m1")
		assert.Contains(t, string(data), "m2")
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		exportDir := t.TempDir()
		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return views, 2, nil
			},
		}

		adapter := NewDashboardAdapter(repo, exportDir)

		path, err := adapter.ExportDashboard(context.Background(), "user-1", model.DashboardFilter{}, "json")

		require.NoError(t, err)
		assert.Contains(t, path, "/exports/dashboard_")
		assert.True(t, strings.HasSuffix(path, ".json"))

		files, err := os.ReadDir(exportDir)
		require.NoError(t, err)
		assert.Len(t, files, 1)

		data, err := os.ReadFile(exportDir + "/" + files[0].Name())
		require.NoError(t, err)
		assert.Contains(t, string(data), `"Name": "m1"`)
		assert.Contains(t, string(data), `"Name": "m2"`)
	})

	t.Run("invalid_format", func(t *testing.T) {
		t.Parallel()

		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return views, 2, nil
			},
		}

		adapter := NewDashboardAdapter(repo, t.TempDir())

		path, err := adapter.ExportDashboard(context.Background(), "user-1", model.DashboardFilter{}, "xml")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
		assert.Empty(t, path)
	})

	t.Run("repo_error", func(t *testing.T) {
		t.Parallel()

		repo := &mockMonitorStatusRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, filter model.DashboardFilter) ([]*model.MonitorStatusView, int, error) {
				return nil, 0, errors.New("query failed")
			},
		}

		adapter := NewDashboardAdapter(repo, t.TempDir())

		path, err := adapter.ExportDashboard(context.Background(), "user-1", model.DashboardFilter{}, "csv")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list monitors for export")
		assert.Empty(t, path)
	})
}
