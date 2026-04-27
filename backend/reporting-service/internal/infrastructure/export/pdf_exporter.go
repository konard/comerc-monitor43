package export

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/pkg/errors"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
)

// PDFExporter экспортирует данные отчётов в формат PDF.
type PDFExporter interface {
	ExportSLAReport(report *reportingv1.SLAReport) ([]byte, string, error)
}

type pdfExporter struct{}

func NewPDFExporter() PDFExporter {
	return &pdfExporter{}
}

func (e *pdfExporter) ExportSLAReport(report *reportingv1.SLAReport) ([]byte, string, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 20)
	pdf.Cell(0, 15, "SLA Report")
	pdf.Ln(20)

	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 8, fmt.Sprintf("Monitor: %s", report.MonitorName))
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("Period: %s - %s",
		report.PeriodStart.AsTime().UTC().Format("2006-01-02 15:04 UTC"),
		report.PeriodEnd.AsTime().UTC().Format("2006-01-02 15:04 UTC"),
	))
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 10, "Summary")
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "", 11)
	data := [][]string{
		{"Availability", fmt.Sprintf("%.2f%%", report.Availability)},
		{"Total Checks", fmt.Sprintf("%d", report.TotalChecks)},
		{"Up Checks", fmt.Sprintf("%d", report.UpChecks)},
		{"Down Checks", fmt.Sprintf("%d", report.DownChecks)},
		{"Degraded Checks", fmt.Sprintf("%d", report.DegradedChecks)},
		{"Paused Checks", fmt.Sprintf("%d", report.PausedChecks)},
		{"Total Downtime", fmt.Sprintf("%d seconds", report.TotalDowntimeSeconds)},
		{"Incidents", fmt.Sprintf("%d", report.IncidentsCount)},
	}

	for _, row := range data {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(60, 7, row[0])
		pdf.SetFont("Helvetica", "", 10)
		pdf.Cell(0, 7, row[1])
		pdf.Ln(7)
	}

	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.Cell(0, 5, fmt.Sprintf("Generated at: %s UTC", time.Now().UTC().Format("2006-01-02 15:04:05")))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", errors.Wrap(err, "failed to generate pdf")
	}

	filename := fmt.Sprintf("sla_report_%s_%s.pdf",
		report.MonitorName,
		time.Now().UTC().Format("20060102_150405"),
	)

	return buf.Bytes(), filename, nil
}
