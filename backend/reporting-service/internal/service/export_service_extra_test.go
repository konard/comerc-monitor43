package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/export"
	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

// mockCSVExporter реализует CSVExporter с возможностью задать ошибку.
type mockCSVExporter struct {
	mock.Mock
}

func (m *mockCSVExporter) ExportCheckResults(results []*monitov1.CheckResult, monitorName string) ([]byte, string, error) {
	args := m.Called(results, monitorName)
	if b := args.Get(0); b != nil {
		return b.([]byte), args.String(1), args.Error(2)
	}
	return nil, args.String(1), args.Error(2)
}

// mockPDFExporter реализует PDFExporter с возможностью задать ошибку.
type mockPDFExporter struct {
	mock.Mock
}

func (m *mockPDFExporter) ExportSLAReport(report *reportingv1.SLAReport) ([]byte, string, error) {
	args := m.Called(report)
	if b := args.Get(0); b != nil {
		return b.([]byte), args.String(1), args.Error(2)
	}
	return nil, args.String(1), args.Error(2)
}

func TestExportCSV_CSVExporterError(t *testing.T) {
	t.Parallel()

	monitorID := "test-monitor-id"
	from := "2026-01-01T00:00:00Z"
	to := "2026-01-08T00:00:00Z"

	mc := new(MockMonitorClient)
	mr := new(MockReportRepository)
	logger := testLogger()
	csvExp := new(mockCSVExporter)
	pdfExp := export.NewPDFExporter()

	fromTime, err := time.Parse(time.RFC3339, from)
	require.NoError(t, err)
	toTime, err := time.Parse(time.RFC3339, to)
	require.NoError(t, err)

	now := timestamppb.New(fromTime)
	results := []*monitov1.CheckResult{
		{Id: "1", MonitorId: monitorID, StatusCode: 200, ResponseTimeMs: 100, Status: "UP", CheckedAt: now},
	}

	mc.On("GetCheckResults", mock.Anything, monitorID, fromTime, toTime, 100000, 0).Once().Return(&monitov1.GetCheckResultsResponse{
		Results: results, Total: 1,
	}, nil)
	mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(&monitov1.GetMonitorResponse{
		Monitor: &monitov1.Monitor{Id: monitorID, Name: "test-monitor"},
	}, nil)
	csvExp.On("ExportCheckResults", mock.Anything, "test-monitor").Once().Return(nil, "", errors.New("csv write error"))

	svc := NewExportService(mc, mr, csvExp, pdfExp, logger, 100000)
	_, err = svc.ExportCSV(context.Background(), &dto.ExportCSVRequest{
		MonitorID: monitorID,
		From:      from,
		To:        to,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate csv")
	mc.AssertExpectations(t)
	csvExp.AssertExpectations(t)
}

func TestExportPDF_PDFExporterError(t *testing.T) {
	t.Parallel()

	mc := new(MockMonitorClient)
	mr := new(MockReportRepository)
	logger := testLogger()
	csvExp := export.NewCSVExporter()
	pdfExp := new(mockPDFExporter)

	reportID := uuid.New()
	userID := uuid.New()

	mr.On("GetByID", mock.Anything, reportID).Once().Return(&model.SLAReport{
		ID:          reportID,
		UserID:      userID,
		MonitorName: "test-monitor",
	}, nil)
	pdfExp.On("ExportSLAReport", mock.Anything).Once().Return(nil, "", errors.New("pdf error"))

	svc := NewExportService(mc, mr, csvExp, pdfExp, logger, 100000)
	_, err := svc.ExportPDF(context.Background(), &dto.ExportPDFRequest{
		ReportID: reportID.String(),
		UserID:   userID.String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate pdf")
	mr.AssertExpectations(t)
	pdfExp.AssertExpectations(t)
}

// TestMockMonitorClient_Close проверяет Close() мока.
func TestMockMonitorClient_Close(t *testing.T) {
	t.Parallel()

	mc := new(MockMonitorClient)
	mc.On("Close").Return(nil)

	err := mc.Close()
	assert.NoError(t, err)
	mc.AssertExpectations(t)
}
