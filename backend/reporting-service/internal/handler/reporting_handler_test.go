package handler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func authContext(t *testing.T) context.Context {
	t.Helper()
	userID := uuid.New()
	return context.WithValue(context.Background(), auth.UserIDKey, userID)
}

type MockSLAService struct {
	mock.Mock
}

func (m *MockSLAService) GenerateSLAReport(ctx context.Context, req *dto.GenerateSLAReportRequest) (*dto.SLAReportResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.SLAReportResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSLAService) GetSLAReport(ctx context.Context, reportID, userID string) (*dto.SLAReportResponse, error) {
	args := m.Called(ctx, reportID, userID)
	if r := args.Get(0); r != nil {
		return r.(*dto.SLAReportResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSLAService) ListSLAReports(ctx context.Context, req *dto.ListSLAReportsRequest) (*dto.ListSLAReportsResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.ListSLAReportsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockAnalyticsService struct {
	mock.Mock
}

func (m *MockAnalyticsService) GetResponseTimeMetrics(ctx context.Context, req *dto.GetResponseTimeMetricsRequest) (*dto.ResponseTimeMetricsResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.ResponseTimeMetricsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAnalyticsService) GetIncidentsReport(ctx context.Context, req *dto.GetIncidentsReportRequest) (*dto.GetIncidentsReportResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.GetIncidentsReportResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAnalyticsService) GetCheckCountByStatus(ctx context.Context, req *dto.GetCheckCountByStatusRequest) (*dto.GetCheckCountByStatusResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.GetCheckCountByStatusResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockExportService struct {
	mock.Mock
}

func (m *MockExportService) ExportCSV(ctx context.Context, req *dto.ExportCSVRequest) (*dto.ExportCSVResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.ExportCSVResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockExportService) ExportPDF(ctx context.Context, req *dto.ExportPDFRequest) (*dto.ExportPDFResponse, error) {
	args := m.Called(ctx, req)
	if r := args.Get(0); r != nil {
		return r.(*dto.ExportPDFResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestHandleError(t *testing.T) {
	t.Parallel()

	logger := testLogger()
	h := NewReportingHandler(nil, nil, nil, logger)

	tests := []struct {
		name      string
		err       error
		wantCode  codes.Code
		wantMatch string
	}{
		{"invalid_date_range", model.ErrInvalidDateRange, codes.InvalidArgument, "invalid date range"},
		{"period_exceeds_retention", model.ErrPeriodExceedsRetention, codes.InvalidArgument, "period exceeds retention policy"},
		{"report_not_found", model.ErrReportNotFound, codes.NotFound, "report not found"},
		{"monitor_not_found", model.ErrMonitorNotFound, codes.NotFound, "monitor not found"},
		{"unauthorized_access", model.ErrUnauthorizedAccess, codes.PermissionDenied, "no access to this report"},
		{"generic_error", assert.AnError, codes.Internal, "internal server error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := h.handleError(tc.err)
			st, ok := status.FromError(got)
			require.True(t, ok)
			assert.Equal(t, tc.wantCode, st.Code())
			assert.Contains(t, st.Message(), tc.wantMatch)
		})
	}
}

func TestGenerateSLAReport(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		slaSvc := new(MockSLAService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		slaSvc.On("GenerateSLAReport", mock.Anything, mock.Anything).Return(&dto.SLAReportResponse{
			ID:           uuid.New().String(),
			MonitorID:    "monitor-1",
			MonitorName:  "test-monitor",
			Availability: 99.5,
			TotalChecks:  1000,
		}, nil)

		h := NewReportingHandler(slaSvc, nil, nil, logger)
		resp, err := h.GenerateSLAReport(authContext(t), &reportingv1.GenerateSLAReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.NoError(t, err)
		assert.Equal(t, "test-monitor", resp.MonitorName)
		assert.Equal(t, 99.5, resp.Availability)
		slaSvc.AssertExpectations(t)
	})

	t.Run("error-invalid-date-range", func(t *testing.T) {
		t.Parallel()

		slaSvc := new(MockSLAService)
		logger := testLogger()

		from := time.Now().UTC()
		to := time.Now().UTC().AddDate(0, 0, -7)

		slaSvc.On("GenerateSLAReport", mock.Anything, mock.Anything).Return(nil, model.ErrInvalidDateRange)

		h := NewReportingHandler(slaSvc, nil, nil, logger)
		_, err := h.GenerateSLAReport(authContext(t), &reportingv1.GenerateSLAReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		slaSvc.AssertExpectations(t)
	})
}

func TestGetResponseTimeMetrics(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		analyticsSvc := new(MockAnalyticsService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		analyticsSvc.On("GetResponseTimeMetrics", mock.Anything, mock.Anything).Return(&dto.ResponseTimeMetricsResponse{
			MonitorID:   "monitor-1",
			P50:         100,
			P95:         300,
			P99:         500,
			Average:     200,
			Min:         50,
			Max:         500,
			TotalChecks: 100,
		}, nil)

		h := NewReportingHandler(nil, analyticsSvc, nil, logger)
		resp, err := h.GetResponseTimeMetrics(authContext(t), &reportingv1.GetResponseTimeMetricsRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.NoError(t, err)
		assert.Equal(t, int32(100), resp.P50)
		assert.Equal(t, int32(300), resp.P95)
		assert.Equal(t, int32(500), resp.P99)
		assert.Equal(t, int32(100), resp.TotalChecks)
		analyticsSvc.AssertExpectations(t)
	})
}

func TestGetCheckCountByStatus(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		analyticsSvc := new(MockAnalyticsService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		analyticsSvc.On("GetCheckCountByStatus", mock.Anything, mock.Anything).Return(&dto.GetCheckCountByStatusResponse{
			MonitorID: "monitor-1",
			StatusCounts: []*dto.StatusCount{
				{StatusGroup: "2xx", Count: 90},
				{StatusGroup: "5xx", Count: 10},
			},
			Total: 100,
		}, nil)

		h := NewReportingHandler(nil, analyticsSvc, nil, logger)
		resp, err := h.GetCheckCountByStatus(authContext(t), &reportingv1.GetCheckCountByStatusRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.NoError(t, err)
		assert.Equal(t, int32(100), resp.Total)
		require.Len(t, resp.StatusCounts, 2)
		assert.Equal(t, "2xx", resp.StatusCounts[0].StatusGroup)
		assert.Equal(t, int32(90), resp.StatusCounts[0].Count)
		analyticsSvc.AssertExpectations(t)
	})
}

func TestExportReportCSV(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exportSvc := new(MockExportService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		exportSvc.On("ExportCSV", mock.Anything, mock.Anything).Return(&dto.ExportCSVResponse{
			Content:  []byte("csv,data\n"),
			Filename: "test.csv",
			Rows:     10,
		}, nil)

		h := NewReportingHandler(nil, nil, exportSvc, logger)
		resp, err := h.ExportReportCSV(authContext(t), &reportingv1.ExportReportCSVRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.NoError(t, err)
		assert.Equal(t, "test.csv", resp.Filename)
		assert.Equal(t, int32(10), resp.Rows)
		exportSvc.AssertExpectations(t)
	})
}

func TestGetSLAReport(t *testing.T) {
	t.Parallel()

	t.Run("success_with_user_metadata", func(t *testing.T) {
		t.Parallel()

		slaSvc := new(MockSLAService)
		logger := testLogger()

		reportID := uuid.New().String()
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)

		slaSvc.On("GetSLAReport", mock.Anything, reportID, userID.String()).Return(&dto.SLAReportResponse{
			ID:           reportID,
			MonitorName:  "test-monitor",
			Availability: 99.5,
		}, nil)

		h := NewReportingHandler(slaSvc, nil, nil, logger)
		resp, err := h.GetSLAReport(ctx, &reportingv1.GetSLAReportRequest{ReportId: reportID})

		require.NoError(t, err)
		assert.Equal(t, reportID, resp.Id)
		slaSvc.AssertExpectations(t)
	})

	t.Run("error-report-not-found", func(t *testing.T) {
		t.Parallel()

		slaSvc := new(MockSLAService)
		logger := testLogger()

		reportID := uuid.New().String()
		ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

		slaSvc.On("GetSLAReport", mock.Anything, reportID, mock.Anything).Return(nil, model.ErrReportNotFound)

		h := NewReportingHandler(slaSvc, nil, nil, logger)
		_, err := h.GetSLAReport(ctx, &reportingv1.GetSLAReportRequest{ReportId: reportID})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		slaSvc.AssertExpectations(t)
	})
}

func TestListSLAReports(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		slaSvc := new(MockSLAService)
		logger := testLogger()

		slaSvc.On("ListSLAReports", mock.Anything, mock.Anything).Return(&dto.ListSLAReportsResponse{
			Reports: []*dto.SLAReportResponse{
				{ID: "1", MonitorName: "m1"},
				{ID: "2", MonitorName: "m2"},
			},
			Total:  2,
			Limit:  50,
			Offset: 0,
		}, nil)

		h := NewReportingHandler(slaSvc, nil, nil, logger)
		resp, err := h.ListSLAReports(authContext(t), &reportingv1.ListSLAReportsRequest{
			MonitorId: "monitor-1",
		})

		require.NoError(t, err)
		assert.Len(t, resp.Reports, 2)
		assert.Equal(t, int32(2), resp.Total)
		slaSvc.AssertExpectations(t)
	})

	t.Run("default_limit_applied", func(t *testing.T) {
		t.Parallel()

		slaSvc := new(MockSLAService)
		logger := testLogger()

		slaSvc.On("ListSLAReports", mock.Anything, mock.MatchedBy(func(req *dto.ListSLAReportsRequest) bool {
			return req.Limit == 50
		})).Return(&dto.ListSLAReportsResponse{Reports: nil, Total: 0, Limit: 50, Offset: 0}, nil)

		h := NewReportingHandler(slaSvc, nil, nil, logger)
		_, err := h.ListSLAReports(authContext(t), &reportingv1.ListSLAReportsRequest{
			MonitorId: "monitor-1",
		})

		require.NoError(t, err)
		slaSvc.AssertExpectations(t)
	})
}
