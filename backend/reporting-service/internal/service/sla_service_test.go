package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/reporting-service/internal/model"
	"github.com/raul/monitor/backend/reporting-service/internal/service/dto"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestNewSLAService(t *testing.T) {
	t.Parallel()

	mc := new(MockMonitorClient)
	mr := new(MockReportRepository)
	logger := testLogger()

	svc := NewSLAService(mc, mr, logger, 90)

	require.NotNil(t, svc)
}

func TestGenerateSLAReport(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New().String()
	userID := uuid.New().String()
	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(&monitov1.GetMonitorResponse{
			Monitor: &monitov1.Monitor{Id: monitorID, Name: "test-monitor"},
		}, nil)
		mc.On("GetUptimeStats", mock.Anything, monitorID, from, to).Once().Return(&monitov1.GetUptimeStatsResponse{
			Uptime:         99.5,
			TotalChecks:    1000,
			UpChecks:       990,
			DownChecks:     5,
			DegradedChecks: 5,
			PausedChecks:   0,
			TotalDowntime:  300,
			Incidents:      2,
		}, nil)
		mr.On("Create", mock.Anything, mock.Anything).Once().Return(nil)

		svc := NewSLAService(mc, mr, logger, 90)
		resp, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      from,
			To:        to,
		})

		require.NoError(t, err)
		assert.Equal(t, "test-monitor", resp.MonitorName)
		assert.Equal(t, 99.5, resp.Availability)
		assert.Equal(t, 1000, resp.TotalChecks)
		assert.Equal(t, 990, resp.UpChecks)
		assert.Equal(t, 5, resp.DownChecks)
		assert.Equal(t, 2, resp.IncidentsCount)
		assert.NotEmpty(t, resp.ID)
		mc.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("error-invalid-monitor-id", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: "not-a-uuid",
			UserID:    userID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("error-invalid-user-id", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    "not-a-uuid",
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user_id")
	})

	t.Run("error-invalid-date-range-empty", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      time.Time{},
			To:        to,
		})

		assert.ErrorIs(t, err, model.ErrInvalidDateRange)
	})

	t.Run("error-to-before-from", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      to,
			To:        from,
		})

		assert.ErrorIs(t, err, model.ErrInvalidDateRange)
	})

	t.Run("error-future-period", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      time.Now().UTC().Add(24 * time.Hour),
			To:        time.Now().UTC().Add(48 * time.Hour),
		})

		assert.ErrorIs(t, err, model.ErrFuturePeriod)
	})

	t.Run("error-period-exceeds-retention", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      time.Now().UTC().AddDate(0, 0, -120),
			To:        time.Now().UTC(),
		})

		assert.ErrorIs(t, err, model.ErrPeriodExceedsRetention)
	})

	t.Run("error-get-monitor-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(nil, assert.AnError)

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get monitor")
		mc.AssertExpectations(t)
	})

	t.Run("error-get-uptime-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(&monitov1.GetMonitorResponse{
			Monitor: &monitov1.Monitor{Id: monitorID, Name: "test-monitor"},
		}, nil)
		mc.On("GetUptimeStats", mock.Anything, monitorID, from, to).Once().Return(nil, assert.AnError)

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get uptime stats")
		mc.AssertExpectations(t)
	})

	t.Run("error-report-repo-create-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		mc.On("GetMonitor", mock.Anything, monitorID).Once().Return(&monitov1.GetMonitorResponse{
			Monitor: &monitov1.Monitor{Id: monitorID, Name: "test-monitor"},
		}, nil)
		mc.On("GetUptimeStats", mock.Anything, monitorID, from, to).Once().Return(&monitov1.GetUptimeStatsResponse{
			Uptime:      100.0,
			TotalChecks: 100,
			UpChecks:    100,
		}, nil)
		mr.On("Create", mock.Anything, mock.Anything).Once().Return(assert.AnError)

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GenerateSLAReport(context.Background(), &dto.GenerateSLAReportRequest{
			MonitorID: monitorID,
			UserID:    userID,
			From:      from,
			To:        to,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save sla report")
		mc.AssertExpectations(t)
		mr.AssertExpectations(t)
	})
}

func TestGetSLAReport(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		userID := uuid.New()
		reportID := uuid.New()

		mr.On("GetByID", mock.Anything, reportID).Once().Return(&model.SLAReport{
			ID:           reportID,
			MonitorID:    uuid.New(),
			UserID:       userID,
			MonitorName:  "test-monitor",
			Availability: 99.5,
		}, nil)

		svc := NewSLAService(mc, mr, logger, 90)
		resp, err := svc.GetSLAReport(context.Background(), reportID.String(), userID.String())

		require.NoError(t, err)
		assert.Equal(t, reportID.String(), resp.ID)
		assert.Equal(t, "test-monitor", resp.MonitorName)
		mr.AssertExpectations(t)
	})

	t.Run("error-invalid-report-id", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GetSLAReport(context.Background(), "bad-uuid", uuid.New().String())

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

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GetSLAReport(context.Background(), reportID.String(), uuid.New().String())

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

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.GetSLAReport(context.Background(), reportID.String(), uuid.New().String())

		assert.ErrorIs(t, err, model.ErrUnauthorizedAccess)
		mr.AssertExpectations(t)
	})
}

func TestListSLAReports(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		monitorID := uuid.New()
		reports := []*model.SLAReport{
			{ID: uuid.New(), MonitorID: monitorID, MonitorName: "m1"},
			{ID: uuid.New(), MonitorID: monitorID, MonitorName: "m2"},
		}

		mr.On("ListByMonitorID", mock.Anything, monitorID, 10, 0).Once().Return(reports, 2, nil)

		svc := NewSLAService(mc, mr, logger, 90)
		resp, err := svc.ListSLAReports(context.Background(), &dto.ListSLAReportsRequest{
			MonitorID: monitorID.String(),
			Limit:     10,
			Offset:    0,
		})

		require.NoError(t, err)
		assert.Len(t, resp.Reports, 2)
		assert.Equal(t, 2, resp.Total)
		assert.Equal(t, 10, resp.Limit)
		mr.AssertExpectations(t)
	})

	t.Run("error-invalid-monitor-id", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.ListSLAReports(context.Background(), &dto.ListSLAReportsRequest{
			MonitorID: "bad-uuid",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitor_id")
	})

	t.Run("error-repo-fails", func(t *testing.T) {
		t.Parallel()

		mc := new(MockMonitorClient)
		mr := new(MockReportRepository)
		logger := testLogger()

		monitorID := uuid.New()

		mr.On("ListByMonitorID", mock.Anything, monitorID, 10, 0).Once().Return(nil, 0, assert.AnError)

		svc := NewSLAService(mc, mr, logger, 90)
		_, err := svc.ListSLAReports(context.Background(), &dto.ListSLAReportsRequest{
			MonitorID: monitorID.String(),
			Limit:     10,
			Offset:    0,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list sla reports")
		mr.AssertExpectations(t)
	})
}

func TestValidatePeriod(t *testing.T) {
	t.Parallel()

	mc := new(MockMonitorClient)
	mr := new(MockReportRepository)
	logger := testLogger()
	svc := NewSLAService(mc, mr, logger, 90).(*SLAService)

	now := time.Now().UTC()

	tests := []struct {
		name    string
		from    time.Time
		to      time.Time
		wantErr error
	}{
		{"valid_period", now.AddDate(0, 0, -7), now, nil},
		{"zero_from", time.Time{}, now, model.ErrInvalidDateRange},
		{"zero_to", now.AddDate(0, 0, -7), time.Time{}, model.ErrInvalidDateRange},
		{"to_before_from", now, now.AddDate(0, 0, -7), model.ErrInvalidDateRange},
		{"future_period", now.Add(24 * time.Hour), now.Add(48 * time.Hour), model.ErrFuturePeriod},
		{"exceeds_retention", now.AddDate(0, 0, -100), now, model.ErrPeriodExceedsRetention},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := svc.validatePeriod(tc.from, tc.to, 90)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculatePercentileIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		length     int
		percentile int
		want       int
	}{
		{"p50_of_100", 100, 50, 49},
		{"p95_of_100", 100, 95, 94},
		{"p99_of_100", 100, 99, 98},
		{"p50_of_1", 1, 50, 0},
		{"p99_of_10", 10, 99, 8},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := calculatePercentileIndex(tc.length, tc.percentile)
			assert.Equal(t, tc.want, got)
		})
	}
}
