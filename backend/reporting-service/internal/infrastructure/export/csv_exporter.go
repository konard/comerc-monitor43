package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/pkg/errors"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
)

// CSVExporter экспортирует данные отчётов в формат CSV.
type CSVExporter interface {
	ExportCheckResults(results []*monitov1.CheckResult, monitorName string) ([]byte, string, error)
}

type csvExporter struct{}

func NewCSVExporter() CSVExporter {
	return &csvExporter{}
}

func (e *csvExporter) ExportCheckResults(results []*monitov1.CheckResult, monitorName string) ([]byte, string, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	header := []string{"timestamp", "status", "response_time_ms", "status_code", "error_message"}
	if err := writer.Write(header); err != nil {
		return nil, "", errors.Wrap(err, "failed to write csv header")
	}

	for _, r := range results {
		row := []string{
			r.CheckedAt.AsTime().UTC().Format(time.RFC3339),
			r.Status,
			fmt.Sprintf("%d", r.ResponseTimeMs),
			fmt.Sprintf("%d", r.StatusCode),
			r.ErrorMessage,
		}
		if err := writer.Write(row); err != nil {
			return nil, "", errors.Wrap(err, "failed to write csv row")
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.Wrap(err, "failed to flush csv")
	}

	filename := fmt.Sprintf("%s_%s.csv", monitorName, time.Now().UTC().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}
