package dto

// ExportCSVRequest содержит параметры запроса на экспорт в CSV.
type ExportCSVRequest struct {
	MonitorID string
	UserID    string
	From      string
	To        string
}

// ExportCSVResponse содержит результат экспорта в CSV.
type ExportCSVResponse struct {
	Content  []byte
	Filename string
	Rows     int
}

// ExportPDFRequest содержит параметры запроса на экспорт в PDF.
type ExportPDFRequest struct {
	ReportID string
	UserID   string
}

// ExportPDFResponse содержит результат экспорта в PDF.
type ExportPDFResponse struct {
	Content  []byte
	Filename string
}
