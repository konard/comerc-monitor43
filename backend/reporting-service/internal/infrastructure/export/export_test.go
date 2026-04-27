package export

import (
	"testing"
	"time"

	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewCSVExporter(t *testing.T) {
	t.Parallel()

	e := NewCSVExporter()
	require.NotNil(t, e)
}

func TestCSVExporter_ExportCheckResults(t *testing.T) {
	t.Parallel()

	now := timestamppb.New(time.Now().UTC())

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exporter := NewCSVExporter()

		results := []*monitov1.CheckResult{
			{Id: "1", StatusCode: 200, ResponseTimeMs: 100, Status: "UP", ErrorMessage: "", CheckedAt: now},
			{Id: "2", StatusCode: 503, ResponseTimeMs: 500, Status: "DOWN", ErrorMessage: "timeout", CheckedAt: now},
		}

		content, filename, err := exporter.ExportCheckResults(results, "test-monitor")

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, filename, "test-monitor")
		assert.Contains(t, filename, ".csv")
		assert.Contains(t, string(content), "timestamp")
		assert.Contains(t, string(content), "status")
		assert.Contains(t, string(content), "response_time_ms")
		assert.Contains(t, string(content), "timeout")
	})

	t.Run("success_empty_results", func(t *testing.T) {
		t.Parallel()

		exporter := NewCSVExporter()

		content, filename, err := exporter.ExportCheckResults([]*monitov1.CheckResult{}, "test-monitor")

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, filename, "test-monitor")
	})
}

func TestNewPDFExporter(t *testing.T) {
	t.Parallel()

	e := NewPDFExporter()
	require.NotNil(t, e)
}

func TestPDFExporter_ExportSLAReport(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		exporter := NewPDFExporter()

		report := &reportingv1.SLAReport{
			MonitorName:          "test-monitor",
			PeriodStart:          timestamppb.New(time.Now().UTC()),
			PeriodEnd:            timestamppb.New(time.Now().UTC()),
			Availability:         99.5,
			TotalChecks:          1000,
			UpChecks:             990,
			DownChecks:           5,
			DegradedChecks:       5,
			PausedChecks:         0,
			TotalDowntimeSeconds: 300,
			IncidentsCount:       2,
		}

		content, filename, err := exporter.ExportSLAReport(report)

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, filename, "sla_report")
		assert.Contains(t, filename, "test-monitor")
		assert.Contains(t, filename, ".pdf")
	})
}
