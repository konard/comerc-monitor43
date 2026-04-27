package grpc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	dashboardproto "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
)

var fixedTime = time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)

func Test_monitorStatusToProto(t *testing.T) {
	t.Parallel()

	t.Run("success_all_fields", func(t *testing.T) {
		t.Parallel()

		responseTime := 42.5
		view := &domain.MonitorStatusView{
			ID:                 uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			UserID:             uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Name:               "test-monitor",
			URL:                "https://example.com",
			Status:             domain.MonitorStatusUP,
			UptimePercentage:   99.5,
			LastCheckedAt:      &fixedTime,
			LastResponseTimeMs: &responseTime,
			CreatedAt:          fixedTime,
			UpdatedAt:          fixedTime,
		}

		result, err := monitorStatusToProto(view)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", result.Id)
		assert.Equal(t, "22222222-2222-2222-2222-222222222222", result.UserId)
		assert.Equal(t, "test-monitor", result.Name)
		assert.Equal(t, "https://example.com", result.Url)
		assert.Equal(t, dashboardproto.HealthStatus_HEALTH_STATUS_UP, result.Status)
		assert.Equal(t, 99.5, result.UptimePercentage)
		assert.Equal(t, 42.5, result.LastResponseTimeMs)
		assert.True(t, result.LastCheckedAt.AsTime().Equal(fixedTime))
		assert.True(t, result.CreatedAt.AsTime().Equal(fixedTime))
		assert.True(t, result.UpdatedAt.AsTime().Equal(fixedTime))
	})

	t.Run("nil_pointer_returns_nil", func(t *testing.T) {
		t.Parallel()

		result, err := monitorStatusToProto(nil)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("up_down_degraded_paused_status_mapping", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name      string
			status    domain.MonitorStatus
			wantProto dashboardproto.HealthStatus
		}{
			{"up", domain.MonitorStatusUP, dashboardproto.HealthStatus_HEALTH_STATUS_UP},
			{"down", domain.MonitorStatusDOWN, dashboardproto.HealthStatus_HEALTH_STATUS_DOWN},
			{"degraded", domain.MonitorStatusDEGRADED, dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED},
			{"paused", domain.MonitorStatusPAUSED, dashboardproto.HealthStatus_HEALTH_STATUS_PAUSED},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				view := &domain.MonitorStatusView{
					ID:        uuid.New(),
					UserID:    uuid.New(),
					Status:    tc.status,
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
				}

				result, err := monitorStatusToProto(view)

				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tc.wantProto, result.Status)
			})
		}
	})

	t.Run("nil_optional_fields", func(t *testing.T) {
		t.Parallel()

		view := &domain.MonitorStatusView{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}

		result, err := monitorStatusToProto(view)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.LastCheckedAt)
		assert.Equal(t, float64(0), result.LastResponseTimeMs)
	})
}

func Test_checkHistoryToProto(t *testing.T) {
	t.Parallel()

	t.Run("success_all_fields", func(t *testing.T) {
		t.Parallel()

		statusCode := 200
		responseTime := 123.4
		errorMsg := "timeout"
		entry := &domain.CheckHistoryEntry{
			ID:             uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			MonitorID:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Status:         domain.CheckStatusUP,
			StatusCode:     &statusCode,
			ResponseTimeMs: &responseTime,
			ErrorMessage:   &errorMsg,
			CheckedAt:      fixedTime,
		}

		result, err := checkHistoryToProto(entry)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", result.Id)
		assert.Equal(t, "22222222-2222-2222-2222-222222222222", result.MonitorId)
		assert.Equal(t, dashboardproto.HealthStatus_HEALTH_STATUS_UP, result.Status)
		assert.Equal(t, int32(200), result.StatusCode)
		assert.Equal(t, 123.4, result.ResponseTimeMs)
		assert.Equal(t, "timeout", result.ErrorMessage)
		assert.True(t, result.CheckedAt.AsTime().Equal(fixedTime))
	})

	t.Run("nil_pointer_returns_nil", func(t *testing.T) {
		t.Parallel()

		result, err := checkHistoryToProto(nil)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil_optional_fields", func(t *testing.T) {
		t.Parallel()

		entry := &domain.CheckHistoryEntry{
			ID:        uuid.New(),
			MonitorID: uuid.New(),
			Status:    domain.CheckStatusDOWN,
			CheckedAt: fixedTime,
		}

		result, err := checkHistoryToProto(entry)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int32(0), result.StatusCode)
		assert.Equal(t, float64(0), result.ResponseTimeMs)
		assert.Equal(t, "", result.ErrorMessage)
	})
}

func Test_incidentToProto(t *testing.T) {
	t.Parallel()

	t.Run("success_all_fields", func(t *testing.T) {
		t.Parallel()

		endTime := fixedTime.Add(time.Hour)
		duration := int64(3600)
		incident := &domain.Incident{
			ID:              uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			MonitorID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			StartedAt:       fixedTime,
			EndedAt:         &endTime,
			DurationSeconds: &duration,
			CheckCount:      15,
			Status:          domain.IncidentStatusActive,
			CreatedAt:       fixedTime,
			UpdatedAt:       fixedTime,
		}

		result, err := incidentToProto(incident)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", result.Id)
		assert.Equal(t, "22222222-2222-2222-2222-222222222222", result.MonitorId)
		assert.Equal(t, int32(15), result.CheckCount)
		assert.Equal(t, dashboardproto.IncidentStatus_INCIDENT_STATUS_ACTIVE, result.Status)
		assert.True(t, result.StartedAt.AsTime().Equal(fixedTime))
		assert.True(t, result.EndedAt.AsTime().Equal(endTime))
		assert.Equal(t, int64(3600), result.DurationSeconds)
		assert.True(t, result.CreatedAt.AsTime().Equal(fixedTime))
		assert.True(t, result.UpdatedAt.AsTime().Equal(fixedTime))
	})

	t.Run("nil_pointer_returns_nil", func(t *testing.T) {
		t.Parallel()

		result, err := incidentToProto(nil)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("active_and_resolved_status_mapping", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name      string
			status    domain.IncidentStatus
			wantProto dashboardproto.IncidentStatus
		}{
			{"active", domain.IncidentStatusActive, dashboardproto.IncidentStatus_INCIDENT_STATUS_ACTIVE},
			{"resolved", domain.IncidentStatusResolved, dashboardproto.IncidentStatus_INCIDENT_STATUS_RESOLVED},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				incident := &domain.Incident{
					ID:        uuid.New(),
					MonitorID: uuid.New(),
					Status:    tc.status,
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
				}

				result, err := incidentToProto(incident)

				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tc.wantProto, result.Status)
			})
		}
	})
}

func Test_protoToHealthStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proto dashboardproto.HealthStatus
		want  domain.MonitorStatus
	}{
		{"up", dashboardproto.HealthStatus_HEALTH_STATUS_UP, domain.MonitorStatusUP},
		{"down", dashboardproto.HealthStatus_HEALTH_STATUS_DOWN, domain.MonitorStatusDOWN},
		{"degraded", dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED, domain.MonitorStatusDEGRADED},
		{"paused", dashboardproto.HealthStatus_HEALTH_STATUS_PAUSED, domain.MonitorStatusPAUSED},
		{"unspecified", dashboardproto.HealthStatus_HEALTH_STATUS_UNSPECIFIED, domain.MonitorStatusUP},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := protoToHealthStatus(tc.proto)
			assert.Equal(t, tc.want, result)
		})
	}
}

func Test_protoToCheckStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proto dashboardproto.HealthStatus
		want  domain.CheckStatus
	}{
		{"up", dashboardproto.HealthStatus_HEALTH_STATUS_UP, domain.CheckStatusUP},
		{"down", dashboardproto.HealthStatus_HEALTH_STATUS_DOWN, domain.CheckStatusDOWN},
		{"degraded", dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED, domain.CheckStatusDEGRADED},
		{"unspecified", dashboardproto.HealthStatus_HEALTH_STATUS_UNSPECIFIED, domain.CheckStatusUP},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := protoToCheckStatus(tc.proto)
			assert.Equal(t, tc.want, result)
		})
	}
}

func Test_incidentStatusToProto(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status domain.IncidentStatus
		want   dashboardproto.IncidentStatus
	}{
		{"active", domain.IncidentStatusActive, dashboardproto.IncidentStatus_INCIDENT_STATUS_ACTIVE},
		{"resolved", domain.IncidentStatusResolved, dashboardproto.IncidentStatus_INCIDENT_STATUS_RESOLVED},
		{"unspecified", domain.IncidentStatus(""), dashboardproto.IncidentStatus_INCIDENT_STATUS_UNSPECIFIED},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := incidentStatusToProto(tc.status)
			assert.Equal(t, tc.want, result)
		})
	}
}

func Test_periodMetricsToProto(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		metrics := &domain.PeriodMetrics{
			TotalChecks:      1000,
			SuccessCount:     950,
			FailedCount:      30,
			DegradedCount:    20,
			UptimePercentage: 99.5,
			P50ResponseMs:    50.0,
			P95ResponseMs:    200.0,
			P99ResponseMs:    500.0,
		}

		result := periodMetricsToProto(metrics)

		require.NotNil(t, result)
		assert.Equal(t, int64(1000), result.TotalChecks)
		assert.Equal(t, int64(950), result.SuccessCount)
		assert.Equal(t, int64(30), result.FailedCount)
		assert.Equal(t, int64(20), result.DegradedCount)
		assert.Equal(t, 99.5, result.UptimePercentage)
		assert.Equal(t, 50.0, result.P50ResponseTimeMs)
		assert.Equal(t, 200.0, result.P95ResponseTimeMs)
		assert.Equal(t, 500.0, result.P99ResponseTimeMs)
	})

	t.Run("nil_returns_nil", func(t *testing.T) {
		t.Parallel()

		result := periodMetricsToProto(nil)
		assert.Nil(t, result)
	})
}

func Test_protoToDashboardFilter(t *testing.T) {
	t.Parallel()

	t.Run("success_with_all_fields", func(t *testing.T) {
		t.Parallel()

		req := &dashboardproto.GetDashboardRequest{
			Statuses:  []dashboardproto.HealthStatus{dashboardproto.HealthStatus_HEALTH_STATUS_UP},
			Tags:      []string{"web", "prod"},
			Search:    "test",
			SortBy:    "name",
			SortOrder: dashboardproto.SortOrder_SORT_ORDER_DESC,
			Page:      2,
			PageSize:  20,
		}

		result := protoToDashboardFilter(req)

		assert.Equal(t, []domain.MonitorStatus{domain.MonitorStatusUP}, result.Statuses)
		assert.Equal(t, []string{"web", "prod"}, result.Tags)
		assert.Equal(t, "test", result.Search)
		assert.Equal(t, "name", result.SortBy)
		assert.Equal(t, "desc", result.SortOrder)
		assert.Equal(t, 2, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("default_sort_order_asc", func(t *testing.T) {
		t.Parallel()

		req := &dashboardproto.GetDashboardRequest{}

		result := protoToDashboardFilter(req)

		assert.Equal(t, "asc", result.SortOrder)
	})
}

func Test_protoToHistoryFilter(t *testing.T) {
	t.Parallel()

	t.Run("success_with_dates", func(t *testing.T) {
		t.Parallel()

		start := timestamppb.New(fixedTime)
		end := timestamppb.New(fixedTime.Add(24 * time.Hour))
		req := &dashboardproto.GetCheckHistoryRequest{
			MonitorId: uuid.New().String(),
			Status:    dashboardproto.HealthStatus_HEALTH_STATUS_DOWN,
			StartDate: start,
			EndDate:   end,
			SortBy:    "checked_at",
			SortOrder: dashboardproto.SortOrder_SORT_ORDER_DESC,
			Page:      1,
			PageSize:  50,
		}

		result := protoToHistoryFilter(req)

		assert.Equal(t, domain.CheckStatusDOWN, result.Status)
		assert.Equal(t, "desc", result.SortOrder)
		assert.Equal(t, "checked_at", result.SortBy)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 50, result.PageSize)
		require.NotNil(t, result.StartDate)
		assert.True(t, result.StartDate.Equal(fixedTime))
		require.NotNil(t, result.EndDate)
		assert.True(t, result.EndDate.Equal(fixedTime.Add(24*time.Hour)))
	})

	t.Run("nil_dates", func(t *testing.T) {
		t.Parallel()

		req := &dashboardproto.GetCheckHistoryRequest{}

		result := protoToHistoryFilter(req)

		assert.Nil(t, result.StartDate)
		assert.Nil(t, result.EndDate)
	})
}

func Test_protoToIncidentFilter(t *testing.T) {
	t.Parallel()

	t.Run("success_with_dates", func(t *testing.T) {
		t.Parallel()

		start := timestamppb.New(fixedTime)
		end := timestamppb.New(fixedTime.Add(48 * time.Hour))
		req := &dashboardproto.GetIncidentsRequest{
			MonitorId: uuid.New().String(),
			StartDate: start,
			EndDate:   end,
			Page:      3,
			PageSize:  100,
		}

		result := protoToIncidentFilter(req)

		require.NotNil(t, result.StartDate)
		assert.True(t, result.StartDate.Equal(fixedTime))
		require.NotNil(t, result.EndDate)
		assert.True(t, result.EndDate.Equal(fixedTime.Add(48*time.Hour)))
		assert.Equal(t, 3, result.Page)
		assert.Equal(t, 100, result.PageSize)
	})
}

func Test_protoToExportDashboardFilter(t *testing.T) {
	t.Parallel()

	req := &dashboardproto.ExportDashboardRequest{
		Statuses: []dashboardproto.HealthStatus{dashboardproto.HealthStatus_HEALTH_STATUS_UP, dashboardproto.HealthStatus_HEALTH_STATUS_DOWN},
		Tags:     []string{"critical"},
	}

	result := protoToExportDashboardFilter(req)

	assert.Equal(t, []domain.MonitorStatus{domain.MonitorStatusUP, domain.MonitorStatusDOWN}, result.Statuses)
	assert.Equal(t, []string{"critical"}, result.Tags)
}

func Test_protoToExportHistoryFilter(t *testing.T) {
	t.Parallel()

	t.Run("with_dates", func(t *testing.T) {
		t.Parallel()

		start := timestamppb.New(fixedTime)
		end := timestamppb.New(fixedTime.Add(time.Hour))
		req := &dashboardproto.ExportHistoryRequest{
			MonitorId: uuid.New().String(),
			Status:    dashboardproto.HealthStatus_HEALTH_STATUS_DEGRADED,
			StartDate: start,
			EndDate:   end,
		}

		result := protoToExportHistoryFilter(req)

		assert.Equal(t, domain.CheckStatusDEGRADED, result.Status)
		require.NotNil(t, result.StartDate)
		assert.True(t, result.StartDate.Equal(fixedTime))
		require.NotNil(t, result.EndDate)
		assert.True(t, result.EndDate.Equal(fixedTime.Add(time.Hour)))
	})

	t.Run("nil_dates", func(t *testing.T) {
		t.Parallel()

		req := &dashboardproto.ExportHistoryRequest{}

		result := protoToExportHistoryFilter(req)

		assert.Nil(t, result.StartDate)
		assert.Nil(t, result.EndDate)
	})
}
