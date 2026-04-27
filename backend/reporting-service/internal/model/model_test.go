package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSLAReport(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()
	userID := uuid.New()
	monitorName := "test-monitor"
	periodStart := time.Now().UTC().AddDate(0, 0, -7)
	periodEnd := time.Now().UTC()

	before := time.Now()
	report := NewSLAReport(monitorID, userID, monitorName, periodStart, periodEnd)
	after := time.Now()

	require.NotNil(t, report)
	assert.NotEqual(t, uuid.Nil, report.ID)
	assert.Equal(t, monitorID, report.MonitorID)
	assert.Equal(t, userID, report.UserID)
	assert.Equal(t, monitorName, report.MonitorName)
	assert.Equal(t, periodStart, report.PeriodStart)
	assert.Equal(t, periodEnd, report.PeriodEnd)
	assert.True(t, report.CreatedAt.After(before) || report.CreatedAt.Equal(before))
	assert.True(t, report.CreatedAt.Before(after) || report.CreatedAt.Equal(after))
	// числовые поля должны быть нулевыми по умолчанию
	assert.Equal(t, 0, report.TotalChecks)
	assert.Equal(t, 0, report.UpChecks)
	assert.Equal(t, 0, report.DownChecks)
	assert.Equal(t, 0, report.DegradedChecks)
	assert.Equal(t, 0, report.PausedChecks)
	assert.Equal(t, int64(0), report.TotalDowntimeSeconds)
	assert.Equal(t, 0, report.IncidentsCount)
	assert.Equal(t, float64(0), report.Availability)
}

func TestNewSLAReport_UniqueIDs(t *testing.T) {
	t.Parallel()

	monitorID := uuid.New()
	userID := uuid.New()
	period := time.Now().UTC()

	r1 := NewSLAReport(monitorID, userID, "m", period, period)
	r2 := NewSLAReport(monitorID, userID, "m", period, period)

	assert.NotEqual(t, r1.ID, r2.ID)
}

func TestNewReportError(t *testing.T) {
	t.Parallel()

	err := NewReportError("TEST_CODE", "test message")

	require.NotNil(t, err)
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Equal(t, "test message", err.Message)
}

func TestReportError_Error(t *testing.T) {
	t.Parallel()

	err := &ReportError{Code: "MY_CODE", Message: "my message"}
	assert.Equal(t, "MY_CODE: my message", err.Error())
}

func TestPredefinedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     *ReportError
		code    string
		message string
	}{
		{"ErrInvalidDateRange", ErrInvalidDateRange, "INVALID_DATE_RANGE", "invalid date range"},
		{"ErrPeriodExceedsRetention", ErrPeriodExceedsRetention, "PERIOD_EXCEEDS_RETENTION", "period exceeds retention policy"},
		{"ErrReportNotFound", ErrReportNotFound, "REPORT_NOT_FOUND", "report not found"},
		{"ErrUnauthorizedAccess", ErrUnauthorizedAccess, "UNAUTHORIZED_ACCESS", "no access to this report"},
		{"ErrMonitorNotFound", ErrMonitorNotFound, "MONITOR_NOT_FOUND", "monitor not found"},
		{"ErrFuturePeriod", ErrFuturePeriod, "INVALID_DATE_RANGE", "cannot generate report for future period"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.code, tc.err.Code)
			assert.Equal(t, tc.message, tc.err.Message)
			assert.Contains(t, tc.err.Error(), tc.code)
			assert.Contains(t, tc.err.Error(), tc.message)
		})
	}
}
