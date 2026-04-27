package handler

import (
	"context"
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

// --- вспомогательные функции ---

func unauthContext() context.Context {
	return context.Background()
}

// --- GetIncidentsReport ---

func TestGetIncidentsReport(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		analyticsSvc := new(MockAnalyticsService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()
		endTime := to.Add(-time.Hour)

		analyticsSvc.On("GetIncidentsReport", mock.Anything, mock.Anything).Return(&dto.GetIncidentsReportResponse{
			Incidents: []*dto.IncidentReportResponse{
				{
					ID:              uuid.New().String(),
					MonitorID:       "monitor-1",
					StartTime:       from,
					EndTime:         &endTime,
					DurationSeconds: 3600,
					Status:          "resolved",
					FailedChecks:    10,
				},
			},
			Total:  1,
			Limit:  100,
			Offset: 0,
		}, nil)

		h := NewReportingHandler(nil, analyticsSvc, nil, logger)
		resp, err := h.GetIncidentsReport(authContext(t), &reportingv1.GetIncidentsReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.NoError(t, err)
		assert.Equal(t, int32(1), resp.Total)
		require.Len(t, resp.Incidents, 1)
		assert.Equal(t, "resolved", resp.Incidents[0].Status)
		assert.Equal(t, int64(3600), resp.Incidents[0].DurationSeconds)
		assert.Equal(t, int32(10), resp.Incidents[0].FailedChecks)
		assert.NotNil(t, resp.Incidents[0].EndTime)
		analyticsSvc.AssertExpectations(t)
	})

	t.Run("success_no_end_time", func(t *testing.T) {
		t.Parallel()

		analyticsSvc := new(MockAnalyticsService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		analyticsSvc.On("GetIncidentsReport", mock.Anything, mock.Anything).Return(&dto.GetIncidentsReportResponse{
			Incidents: []*dto.IncidentReportResponse{
				{
					ID:        uuid.New().String(),
					MonitorID: "monitor-1",
					StartTime: from,
					EndTime:   nil,
					Status:    "ongoing",
				},
			},
			Total:  1,
			Limit:  100,
			Offset: 0,
		}, nil)

		h := NewReportingHandler(nil, analyticsSvc, nil, logger)
		resp, err := h.GetIncidentsReport(authContext(t), &reportingv1.GetIncidentsReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.NoError(t, err)
		require.Len(t, resp.Incidents, 1)
		assert.Nil(t, resp.Incidents[0].EndTime)
		analyticsSvc.AssertExpectations(t)
	})

	t.Run("default_limit_applied", func(t *testing.T) {
		t.Parallel()

		analyticsSvc := new(MockAnalyticsService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		analyticsSvc.On("GetIncidentsReport", mock.Anything, mock.MatchedBy(func(req *dto.GetIncidentsReportRequest) bool {
			return req.Limit == 100
		})).Return(&dto.GetIncidentsReportResponse{
			Incidents: nil, Total: 0, Limit: 100, Offset: 0,
		}, nil)

		h := NewReportingHandler(nil, analyticsSvc, nil, logger)
		_, err := h.GetIncidentsReport(authContext(t), &reportingv1.GetIncidentsReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
			Limit:     0,
		})

		require.NoError(t, err)
		analyticsSvc.AssertExpectations(t)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()

		h := NewReportingHandler(nil, new(MockAnalyticsService), nil, testLogger())
		_, err := h.GetIncidentsReport(unauthContext(), &reportingv1.GetIncidentsReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(time.Now()),
			To:        timestamppb.New(time.Now()),
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		analyticsSvc := new(MockAnalyticsService)
		logger := testLogger()

		from := time.Now().UTC().AddDate(0, 0, -7)
		to := time.Now().UTC()

		analyticsSvc.On("GetIncidentsReport", mock.Anything, mock.Anything).Return(nil, model.ErrMonitorNotFound)

		h := NewReportingHandler(nil, analyticsSvc, nil, logger)
		_, err := h.GetIncidentsReport(authContext(t), &reportingv1.GetIncidentsReportRequest{
			MonitorId: "monitor-1",
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		analyticsSvc.AssertExpectations(t)
	})
}

// --- ExportReportPDF ---

func TestExportReportPDF(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exportSvc := new(MockExportService)
		logger := testLogger()

		reportID := uuid.New().String()
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)

		exportSvc.On("ExportPDF", mock.Anything, &dto.ExportPDFRequest{
			ReportID: reportID,
			UserID:   userID.String(),
		}).Return(&dto.ExportPDFResponse{
			Content:  []byte("%PDF-1.4"),
			Filename: "report.pdf",
		}, nil)

		h := NewReportingHandler(nil, nil, exportSvc, logger)
		resp, err := h.ExportReportPDF(ctx, &reportingv1.ExportReportPDFRequest{ReportId: reportID})

		require.NoError(t, err)
		assert.Equal(t, "report.pdf", resp.Filename)
		assert.Equal(t, []byte("%PDF-1.4"), resp.Content)
		exportSvc.AssertExpectations(t)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		t.Parallel()

		h := NewReportingHandler(nil, nil, new(MockExportService), testLogger())
		_, err := h.ExportReportPDF(unauthContext(), &reportingv1.ExportReportPDFRequest{ReportId: "r1"})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("service_error", func(t *testing.T) {
		t.Parallel()

		exportSvc := new(MockExportService)
		logger := testLogger()

		exportSvc.On("ExportPDF", mock.Anything, mock.Anything).Return(nil, model.ErrReportNotFound)

		h := NewReportingHandler(nil, nil, exportSvc, logger)
		_, err := h.ExportReportPDF(authContext(t), &reportingv1.ExportReportPDFRequest{ReportId: "r1"})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		exportSvc.AssertExpectations(t)
	})
}

// --- уточнённые сценарии handleError ---

func TestHandleError_AllCodes(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(nil, nil, nil, testLogger())

	tests := []struct {
		name     string
		code     string
		wantCode codes.Code
	}{
		{"invalid_period_format", "INVALID_PERIOD_FORMAT", codes.InvalidArgument},
		{"invalid_timezone", "INVALID_TIMEZONE", codes.InvalidArgument},
		{"export_unavailable", "EXPORT_SERVICE_UNAVAILABLE", codes.Unavailable},
		{"timeseries_unavailable", "TIME_SERIES_SERVICE_UNAVAILABLE", codes.Unavailable},
		{"database_unavailable", "DATABASE_UNAVAILABLE", codes.Unavailable},
		{"query_timeout", "QUERY_TIMEOUT", codes.DeadlineExceeded},
		{"pdf_timeout", "PDF_GENERATION_TIMEOUT", codes.DeadlineExceeded},
		{"in_progress", "REPORT_GENERATION_IN_PROGRESS", codes.Aborted},
		{"unknown_code", "SOME_UNKNOWN_CODE", codes.Internal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := model.NewReportError(tc.code, "some message")
			got := h.handleError(err)
			st, ok := status.FromError(got)
			require.True(t, ok)
			assert.Equal(t, tc.wantCode, st.Code())
		})
	}
}

// --- Unauthenticated paths для других хендлеров ---

func TestGenerateSLAReport_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(new(MockSLAService), nil, nil, testLogger())
	_, err := h.GenerateSLAReport(unauthContext(), &reportingv1.GenerateSLAReportRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestListSLAReports_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(new(MockSLAService), nil, nil, testLogger())
	_, err := h.ListSLAReports(unauthContext(), &reportingv1.ListSLAReportsRequest{MonitorId: "m1"})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestGetSLAReport_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(new(MockSLAService), nil, nil, testLogger())
	_, err := h.GetSLAReport(unauthContext(), &reportingv1.GetSLAReportRequest{ReportId: "r1"})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestGetResponseTimeMetrics_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(nil, new(MockAnalyticsService), nil, testLogger())
	_, err := h.GetResponseTimeMetrics(unauthContext(), &reportingv1.GetResponseTimeMetricsRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestGetCheckCountByStatus_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(nil, new(MockAnalyticsService), nil, testLogger())
	_, err := h.GetCheckCountByStatus(unauthContext(), &reportingv1.GetCheckCountByStatusRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestExportReportCSV_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := NewReportingHandler(nil, nil, new(MockExportService), testLogger())
	_, err := h.ExportReportCSV(unauthContext(), &reportingv1.ExportReportCSVRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

// --- сервисные ошибки для покрытия путей ошибок ---

func TestGetResponseTimeMetrics_ServiceError(t *testing.T) {
	t.Parallel()

	analyticsSvc := new(MockAnalyticsService)
	analyticsSvc.On("GetResponseTimeMetrics", mock.Anything, mock.Anything).Return(nil, model.ErrUnauthorizedAccess)

	h := NewReportingHandler(nil, analyticsSvc, nil, testLogger())
	_, err := h.GetResponseTimeMetrics(authContext(t), &reportingv1.GetResponseTimeMetricsRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	analyticsSvc.AssertExpectations(t)
}

func TestGetCheckCountByStatus_ServiceError(t *testing.T) {
	t.Parallel()

	analyticsSvc := new(MockAnalyticsService)
	analyticsSvc.On("GetCheckCountByStatus", mock.Anything, mock.Anything).Return(nil, model.ErrInvalidDateRange)

	h := NewReportingHandler(nil, analyticsSvc, nil, testLogger())
	_, err := h.GetCheckCountByStatus(authContext(t), &reportingv1.GetCheckCountByStatusRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	analyticsSvc.AssertExpectations(t)
}

func TestExportReportCSV_ServiceError(t *testing.T) {
	t.Parallel()

	exportSvc := new(MockExportService)
	exportSvc.On("ExportCSV", mock.Anything, mock.Anything).Return(nil, model.ErrPeriodExceedsRetention)

	h := NewReportingHandler(nil, nil, exportSvc, testLogger())
	_, err := h.ExportReportCSV(authContext(t), &reportingv1.ExportReportCSVRequest{
		MonitorId: "m1",
		From:      timestamppb.New(time.Now()),
		To:        timestamppb.New(time.Now()),
	})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	exportSvc.AssertExpectations(t)
}

func TestListSLAReports_ServiceError(t *testing.T) {
	t.Parallel()

	slaSvc := new(MockSLAService)
	slaSvc.On("ListSLAReports", mock.Anything, mock.Anything).Return(nil, model.ErrReportNotFound)

	h := NewReportingHandler(slaSvc, nil, nil, testLogger())
	_, err := h.ListSLAReports(authContext(t), &reportingv1.ListSLAReportsRequest{MonitorId: "m1"})

	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
	slaSvc.AssertExpectations(t)
}
