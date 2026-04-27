package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/export"
	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

func TestNewExportService(t *testing.T) {
	t.Parallel()

	mc := new(MockMonitorClient)
	mr := new(MockReportRepository)
	logger := testLogger()

	svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)

	require.NotNil(t, svc)
}

func TestExportCSV(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor-id"
	from := "2026-01-01T00:00:00Z"
	to := "2026-01-08T00:00:00Z"

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		now := timestamppb.New(parseTime(t, from))
		results := []*monitov1.CheckResult{
			{Id: "1", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 100, Status: "UP", CheckedAt: now},
			{Id: "2", MonitorId: monitorID, StatusCode: 503, ResponseTimeMs: 500, Status: "DOWN", ErrorMessage: "timeout", CheckedAt: now},
		}

		mc.On("GetCheckResults", mock.Anything, monitorID, parseTime(t, from), parseTime(t, to), 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
			Results: results,
			Total:   2,
		}, nil)
		mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(&monitov1.GetMonitorResponse{
			Monitor: &monitov1.Monitor{Id: monitorID, Name: "test-monitor"},
		}, nil)

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		resp, err := svc.ExportCSV(context.Background(), &dto.ExportCSVRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
		assert.Contains(t, string(resp.Content), "timestamp")
		assert.Contains(t, resp.Filename, "test-monitor")
		assert.Equal(t, 2, resp.Rows)
		mc.AssertExpectations(t)
	})

	t.Run("error-invalid-from-format", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportCSV(context.Background(), &dto.ExportCSVRequest{
			MonitorID: monitorID,
			From:      "not-a-date",
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid from time format")
	})

	t.Run("error-invalid-to-format", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportCSV(context.Background(), &dto.ExportCSVRequest{
			MonitorID: monitorID,
			From:      from,
			To:        "not-a-date",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid to time format")
	})

	t.Run("error-get-check-results-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		mc.On("GetCheckResults", mock.Anything, monitorID, parseTime(t, from), parseTime(t, to), 100000, 0).Once().Return(nil, assert.AnError)

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportCSV(context.Background(), &dto.ExportCSVRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get check results for csv export")
		mc.AssertExpectations(t)
	})

	t.Run("error-get-monitor-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		now := timestamppb.New(parseTime(t, from))

		mc.On("GetCheckResults", mock.Anything, monitorID, parseTime(t, from), parseTime(t, to), 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
			Results: []*monitov1.CheckResult{
				{Id: "1", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 100, Status: "UP", CheckedAt: now},
			},
			Total: 1,
		}, nil)
		mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(nil, assert.AnError)

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportCSV(context.Background(), &dto.ExportCSVRequest{
			MonitorID: monitorID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get monitor for csv export")
		mc.AssertExpectations(t)
	})
}

func TestExportPDF(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		reportID := uuid.New()
		userID := uuid.New()

		mr.On("GetByID", mock.Anything, reportID).Once().Return(&model.SLAReport{
			ID:           reportID,
			UserID:       userID,
			MonitorName:  "test-monitor",
			Availability: 99.5,
			TotalChecks:  1000,
		}, nil)

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		resp, err := svc.ExportPDF(context.Background(), &dto.ExportPDFRequest{
			ReportID: reportID.String(),
			UserID:   userID.String(),
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
		assert.Contains(t, resp.Filename, ".pdf")
		mr.AssertExpectations(t)
	})

	t.Run("error-invalid-report-id", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportPDF(context.Background(), &dto.ExportPDFRequest{
			ReportID: "not-a-uuid",
			UserID:   uuid.New().String(),
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid report_id")
	})

	t.Run("error-report-not-found", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		reportID := uuid.New()

		mr.On("GetByID", mock.Anything, reportID).Once().Return(nil, model.ErrReportNotFound)

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportPDF(context.Background(), &dto.ExportPDFRequest{
			ReportID: reportID.String(),
			UserID:   uuid.New().String(),
		})

		assert.ErrorIs(t, err, model.ErrReportNotFound)
		mr.AssertExpectations(t)
	})

	t.Run("error-unauthorized-access", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		reportID := uuid.New()
		ownerID := uuid.New()

		mr.On("GetByID", mock.Anything, reportID).Once().Return(&model.SLAReport{
			ID:     reportID,
			UserID: ownerID,
		}, nil)

		svc := NewExportService(mc, mr, export.NewCSVExporter(), export.NewPDFExporter(), logger, 100000)
		_, err := svc.ExportPDF(context.Background(), &dto.ExportPDFRequest{
			ReportID: reportID.String(),
			UserID:   uuid.New().String(),
		})

		assert.ErrorIs(t, err, model.ErrUnauthorizedAccess)
		mr.AssertExpectations(t)
	})
}

func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tm
}
