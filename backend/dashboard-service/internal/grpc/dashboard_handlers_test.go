package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dashboardproto "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/dashboard-service/internal/infrastructure/auth"
	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
)

type mockDashboardService struct {
	getDashboardFunc    func(ctx context.Context, userID string, filter domain.DashboardFilter) ([]*domain.MonitorStatusView, int, float64, error)
	exportDashboardFunc func(ctx context.Context, userID string, filter domain.DashboardFilter, format string) (string, error)
}

func (m *mockDashboardService) GetDashboard(ctx context.Context, userID string, filter domain.DashboardFilter) ([]*domain.MonitorStatusView, int, float64, error) {
	return m.getDashboardFunc(ctx, userID, filter)
}

func (m *mockDashboardService) ExportDashboard(ctx context.Context, userID string, filter domain.DashboardFilter, format string) (string, error) {
	return m.exportDashboardFunc(ctx, userID, filter, format)
}

type mockHistoryService struct {
	getCheckHistoryFunc    func(ctx context.Context, monitorID string, filter domain.HistoryFilter) ([]*domain.CheckHistoryEntry, int, error)
	getIncidentsFunc       func(ctx context.Context, monitorID string, filter domain.IncidentFilter) ([]*domain.Incident, int, error)
	getIncidentDetailsFunc func(ctx context.Context, incidentID string) (*domain.IncidentDetail, error)
	getPeriodMetricsFunc   func(ctx context.Context, monitorID string, start, end time.Time) (*domain.PeriodMetrics, error)
	exportHistoryFunc      func(ctx context.Context, monitorID string, filter domain.HistoryFilter, format string, fields []string) (string, error)
}

func (m *mockHistoryService) GetCheckHistory(ctx context.Context, monitorID string, filter domain.HistoryFilter) ([]*domain.CheckHistoryEntry, int, error) {
	return m.getCheckHistoryFunc(ctx, monitorID, filter)
}

func (m *mockHistoryService) GetIncidents(ctx context.Context, monitorID string, filter domain.IncidentFilter) ([]*domain.Incident, int, error) {
	return m.getIncidentsFunc(ctx, monitorID, filter)
}

func (m *mockHistoryService) GetIncidentDetails(ctx context.Context, incidentID string) (*domain.IncidentDetail, error) {
	return m.getIncidentDetailsFunc(ctx, incidentID)
}

func (m *mockHistoryService) GetPeriodMetrics(ctx context.Context, monitorID string, start, end time.Time) (*domain.PeriodMetrics, error) {
	return m.getPeriodMetricsFunc(ctx, monitorID, start, end)
}

func (m *mockHistoryService) ExportHistory(ctx context.Context, monitorID string, filter domain.HistoryFilter, format string, fields []string) (string, error) {
	return m.exportHistoryFunc(ctx, monitorID, filter, format, fields)
}

func authenticatedCtx() context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
}

func TestNewDashboardServiceServer(t *testing.T) {
	t.Parallel()

	t.Run("nil_dashboard_service", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(nil, &mockHistoryService{}, &auth.AuthMiddleware{})
		assert.NotNil(t, srv)
		assert.Nil(t, srv.dashboardService)
	})

	t.Run("nil_history_service", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, nil, &auth.AuthMiddleware{})
		assert.NotNil(t, srv)
		assert.Nil(t, srv.historyService)
	})

	t.Run("nil_auth_middleware", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, nil)
		assert.NotNil(t, srv)
		assert.Nil(t, srv.authMiddleware)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		assert.NotNil(t, srv)
	})
}

func Test_GetDashboard(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		monitor := &domain.MonitorStatusView{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Name:      "test",
			URL:       "https://example.com",
			Status:    domain.MonitorStatusUP,
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}
		mockDash := &mockDashboardService{
			getDashboardFunc: func(ctx context.Context, userID string, filter domain.DashboardFilter) ([]*domain.MonitorStatusView, int, float64, error) {
				return []*domain.MonitorStatusView{monitor}, 1, 99.5, nil
			},
		}
		srv := NewDashboardServiceServer(mockDash, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetDashboardRequest{}

		resp, err := srv.GetDashboard(authenticatedCtx(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Monitors, 1)
		assert.Equal(t, int32(1), resp.Total)
		assert.Equal(t, 99.5, resp.OverallUptimePercentage)
	})

	t.Run("page_size_exceeds_maximum", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetDashboardRequest{PageSize: 201}

		resp, err := srv.GetDashboard(authenticatedCtx(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("search_too_long", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		longSearch := make([]byte, 201)
		req := &dashboardproto.GetDashboardRequest{Search: string(longSearch)}

		resp, err := srv.GetDashboard(authenticatedCtx(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("service_error_monitor_not_found", func(t *testing.T) {
		t.Parallel()

		mockDash := &mockDashboardService{
			getDashboardFunc: func(ctx context.Context, userID string, filter domain.DashboardFilter) ([]*domain.MonitorStatusView, int, float64, error) {
				return nil, 0, 0, domain.ErrMonitorNotFound
			},
		}
		srv := NewDashboardServiceServer(mockDash, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetDashboardRequest{}

		resp, err := srv.GetDashboard(authenticatedCtx(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("unauthenticated_no_user_id", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetDashboardRequest{}

		resp, err := srv.GetDashboard(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func Test_ExportDashboard(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockDash := &mockDashboardService{
			exportDashboardFunc: func(ctx context.Context, userID string, filter domain.DashboardFilter, format string) (string, error) {
				return "https://storage.example.com/export.csv", nil
			},
		}
		srv := NewDashboardServiceServer(mockDash, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportDashboardRequest{Format: "csv"}

		resp, err := srv.ExportDashboard(authenticatedCtx(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "https://storage.example.com/export.csv", resp.DownloadUrl)
		assert.Equal(t, "csv", resp.Format)
		assert.Greater(t, resp.ExpiresAtUnix, time.Now().Unix())
	})

	t.Run("default_format_csv", func(t *testing.T) {
		t.Parallel()

		mockDash := &mockDashboardService{
			exportDashboardFunc: func(ctx context.Context, userID string, filter domain.DashboardFilter, format string) (string, error) {
				assert.Equal(t, "csv", format)
				return "https://storage.example.com/export.csv", nil
			},
		}
		srv := NewDashboardServiceServer(mockDash, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportDashboardRequest{}

		resp, err := srv.ExportDashboard(authenticatedCtx(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "csv", resp.Format)
	})

	t.Run("invalid_format", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportDashboardRequest{Format: "xml"}

		resp, err := srv.ExportDashboard(authenticatedCtx(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportDashboardRequest{}

		resp, err := srv.ExportDashboard(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func Test_GetCheckHistory(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		entry := &domain.CheckHistoryEntry{
			ID:        uuid.New(),
			MonitorID: monitorID,
			Status:    domain.CheckStatusUP,
			CheckedAt: fixedTime,
		}
		mockHist := &mockHistoryService{
			getCheckHistoryFunc: func(ctx context.Context, mid string, filter domain.HistoryFilter) ([]*domain.CheckHistoryEntry, int, error) {
				assert.Equal(t, monitorID.String(), mid)
				return []*domain.CheckHistoryEntry{entry}, 1, nil
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetCheckHistoryRequest{MonitorId: monitorID.String()}

		resp, err := srv.GetCheckHistory(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Checks, 1)
		assert.Equal(t, int32(1), resp.Total)
	})

	t.Run("empty_monitor_id", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetCheckHistoryRequest{}

		resp, err := srv.GetCheckHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetCheckHistoryRequest{MonitorId: "not-a-uuid"}

		resp, err := srv.GetCheckHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_date_range_start_after_end", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetCheckHistoryRequest{
			MonitorId: monitorID.String(),
			StartDate: timestamppb.New(fixedTime.Add(24 * time.Hour)),
			EndDate:   timestamppb.New(fixedTime),
		}

		resp, err := srv.GetCheckHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		mockHist := &mockHistoryService{
			getCheckHistoryFunc: func(ctx context.Context, mid string, filter domain.HistoryFilter) ([]*domain.CheckHistoryEntry, int, error) {
				return nil, 0, domain.ErrMonitorNotFound
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetCheckHistoryRequest{MonitorId: monitorID.String()}

		resp, err := srv.GetCheckHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func Test_GetIncidents(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		incident := &domain.Incident{
			ID:        uuid.New(),
			MonitorID: monitorID,
			Status:    domain.IncidentStatusActive,
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}
		mockHist := &mockHistoryService{
			getIncidentsFunc: func(ctx context.Context, mid string, filter domain.IncidentFilter) ([]*domain.Incident, int, error) {
				assert.Equal(t, monitorID.String(), mid)
				return []*domain.Incident{incident}, 1, nil
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentsRequest{MonitorId: monitorID.String()}

		resp, err := srv.GetIncidents(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Incidents, 1)
		assert.Equal(t, int32(1), resp.Total)
	})

	t.Run("empty_monitor_id", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentsRequest{}

		resp, err := srv.GetIncidents(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentsRequest{MonitorId: "bad-uuid"}

		resp, err := srv.GetIncidents(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		mockHist := &mockHistoryService{
			getIncidentsFunc: func(ctx context.Context, mid string, filter domain.IncidentFilter) ([]*domain.Incident, int, error) {
				return nil, 0, assert.AnError
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentsRequest{MonitorId: monitorID.String()}

		resp, err := srv.GetIncidents(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func Test_GetIncidentDetails(t *testing.T) {
	t.Parallel()

	incidentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		detail := &domain.IncidentDetail{
			Incident: domain.Incident{
				ID:        incidentID,
				MonitorID: uuid.New(),
				Status:    domain.IncidentStatusActive,
				CreatedAt: fixedTime,
				UpdatedAt: fixedTime,
			},
			Timeline: []domain.CheckHistoryEntry{
				{
					ID:        uuid.New(),
					MonitorID: uuid.New(),
					Status:    domain.CheckStatusDOWN,
					CheckedAt: fixedTime,
				},
			},
		}
		mockHist := &mockHistoryService{
			getIncidentDetailsFunc: func(ctx context.Context, id string) (*domain.IncidentDetail, error) {
				assert.Equal(t, incidentID.String(), id)
				return detail, nil
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentDetailsRequest{Id: incidentID.String()}

		resp, err := srv.GetIncidentDetails(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Incident)
		assert.Equal(t, incidentID.String(), resp.Incident.Id)
		assert.Len(t, resp.Timeline, 1)
	})

	t.Run("empty_id", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentDetailsRequest{}

		resp, err := srv.GetIncidentDetails(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentDetailsRequest{Id: "not-a-uuid"}

		resp, err := srv.GetIncidentDetails(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		mockHist := &mockHistoryService{
			getIncidentDetailsFunc: func(ctx context.Context, id string) (*domain.IncidentDetail, error) {
				return nil, domain.ErrIncidentNotFound
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetIncidentDetailsRequest{Id: incidentID.String()}

		resp, err := srv.GetIncidentDetails(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

func Test_GetPeriodMetrics(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		metrics := &domain.PeriodMetrics{
			TotalChecks:      500,
			SuccessCount:     480,
			FailedCount:      10,
			DegradedCount:    10,
			UptimePercentage: 99.8,
			P50ResponseMs:    30.0,
			P95ResponseMs:    150.0,
			P99ResponseMs:    400.0,
		}
		mockHist := &mockHistoryService{
			getPeriodMetricsFunc: func(ctx context.Context, mid string, start, end time.Time) (*domain.PeriodMetrics, error) {
				assert.Equal(t, monitorID.String(), mid)
				return metrics, nil
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetPeriodMetricsRequest{
			MonitorId: monitorID.String(),
			StartDate: timestamppb.New(fixedTime),
			EndDate:   timestamppb.New(fixedTime.Add(24 * time.Hour)),
		}

		resp, err := srv.GetPeriodMetrics(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(500), resp.TotalChecks)
		assert.Equal(t, int64(480), resp.SuccessCount)
		assert.Equal(t, 99.8, resp.UptimePercentage)
	})

	t.Run("empty_monitor_id", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetPeriodMetricsRequest{}

		resp, err := srv.GetPeriodMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetPeriodMetricsRequest{MonitorId: "bad-uuid"}

		resp, err := srv.GetPeriodMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("missing_dates", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetPeriodMetricsRequest{MonitorId: monitorID.String()}

		resp, err := srv.GetPeriodMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_date_range", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.GetPeriodMetricsRequest{
			MonitorId: monitorID.String(),
			StartDate: timestamppb.New(fixedTime.Add(time.Hour)),
			EndDate:   timestamppb.New(fixedTime),
		}

		resp, err := srv.GetPeriodMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		mockHist := &mockHistoryService{
			getPeriodMetricsFunc: func(ctx context.Context, mid string, start, end time.Time) (*domain.PeriodMetrics, error) {
				return nil, assert.AnError
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.GetPeriodMetricsRequest{
			MonitorId: monitorID.String(),
			StartDate: timestamppb.New(fixedTime),
			EndDate:   timestamppb.New(fixedTime.Add(time.Hour)),
		}

		resp, err := srv.GetPeriodMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func Test_ExportHistory(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockHist := &mockHistoryService{
			exportHistoryFunc: func(ctx context.Context, mid string, filter domain.HistoryFilter, format string, fields []string) (string, error) {
				assert.Equal(t, monitorID.String(), mid)
				assert.Equal(t, "json", format)
				assert.Equal(t, []string{"status", "timestamp"}, fields)
				return "https://storage.example.com/history.json", nil
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportHistoryRequest{
			MonitorId: monitorID.String(),
			Format:    "json",
			Fields:    []string{"status", "timestamp"},
		}

		resp, err := srv.ExportHistory(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "https://storage.example.com/history.json", resp.DownloadUrl)
		assert.Equal(t, "json", resp.Format)
		assert.Greater(t, resp.ExpiresAtUnix, time.Now().Unix())
	})

	t.Run("default_format_csv", func(t *testing.T) {
		t.Parallel()

		var capturedFormat string
		mockHist := &mockHistoryService{
			exportHistoryFunc: func(ctx context.Context, mid string, filter domain.HistoryFilter, format string, fields []string) (string, error) {
				capturedFormat = format
				return "https://storage.example.com/export.csv", nil
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportHistoryRequest{MonitorId: monitorID.String()}

		resp, err := srv.ExportHistory(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "csv", capturedFormat)
		assert.Equal(t, "csv", resp.Format)
	})

	t.Run("empty_monitor_id", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportHistoryRequest{}

		resp, err := srv.ExportHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportHistoryRequest{MonitorId: "not-valid"}

		resp, err := srv.ExportHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("invalid_format", func(t *testing.T) {
		t.Parallel()

		srv := NewDashboardServiceServer(&mockDashboardService{}, &mockHistoryService{}, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportHistoryRequest{
			MonitorId: monitorID.String(),
			Format:    "xlsx",
		}

		resp, err := srv.ExportHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		mockHist := &mockHistoryService{
			exportHistoryFunc: func(ctx context.Context, mid string, filter domain.HistoryFilter, format string, fields []string) (string, error) {
				return "", domain.ErrExportFailed
			},
		}
		srv := NewDashboardServiceServer(&mockDashboardService{}, mockHist, &auth.AuthMiddleware{})
		req := &dashboardproto.ExportHistoryRequest{MonitorId: monitorID.String()}

		resp, err := srv.ExportHistory(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func Test_validateUUID(t *testing.T) {
	t.Parallel()

	t.Run("valid_uuid", func(t *testing.T) {
		t.Parallel()

		err := validateUUID(uuid.New().String())
		assert.NoError(t, err)
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		t.Parallel()

		err := validateUUID("not-a-uuid")
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func Test_validatePagination(t *testing.T) {
	t.Parallel()

	t.Run("defaults_zero_values", func(t *testing.T) {
		t.Parallel()

		page, pageSize := validatePagination(0, 0)
		assert.Equal(t, int32(1), page)
		assert.Equal(t, int32(50), pageSize)
	})

	t.Run("clamps_page_size_above_200", func(t *testing.T) {
		t.Parallel()

		_, pageSize := validatePagination(1, 500)
		assert.Equal(t, int32(200), pageSize)
	})

	t.Run("valid_values_unchanged", func(t *testing.T) {
		t.Parallel()

		page, pageSize := validatePagination(5, 100)
		assert.Equal(t, int32(5), page)
		assert.Equal(t, int32(100), pageSize)
	})
}

func Test_validateDateRange(t *testing.T) {
	t.Parallel()

	t.Run("valid_range", func(t *testing.T) {
		t.Parallel()

		err := validateDateRange(
			timestamppb.New(fixedTime),
			timestamppb.New(fixedTime.Add(time.Hour)),
		)
		assert.NoError(t, err)
	})

	t.Run("start_after_end", func(t *testing.T) {
		t.Parallel()

		err := validateDateRange(
			timestamppb.New(fixedTime.Add(time.Hour)),
			timestamppb.New(fixedTime),
		)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("nil_dates_ok", func(t *testing.T) {
		t.Parallel()

		err := validateDateRange(nil, nil)
		assert.NoError(t, err)
	})
}

func Test_domainErrorToGRPCCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"invalid_filter_parameter", domain.ErrInvalidFilter, codes.InvalidArgument},
		{"invalid_page_size", domain.ErrInvalidPageSize, codes.InvalidArgument},
		{"invalid_date_range", domain.ErrInvalidDateRange, codes.InvalidArgument},
		{"invalid_search_query", domain.ErrInvalidSearchQuery, codes.InvalidArgument},
		{"search_query_too_long", domain.ErrSearchQueryTooLong, codes.InvalidArgument},
		{"filter_too_long", domain.ErrFilterTooLong, codes.InvalidArgument},
		{"monitor_not_found", domain.ErrMonitorNotFound, codes.Internal},
		{"incident_not_found", domain.ErrIncidentNotFound, codes.Internal},
		{"export_failed", domain.ErrExportFailed, codes.Internal},
		{"generic_error", assert.AnError, codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			code := domainErrorToGRPCCode(tc.err)
			assert.Equal(t, tc.wantCode, code)
		})
	}
}
