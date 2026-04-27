package parser

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// CSVParser парсит CSV файл с мониторами.
type CSVParser struct {
	// Ожидаемые колонки
	expectedColumns []string
}

// NewCSVParser создаёт новый CSVParser.
func NewCSVParser() *CSVParser {
	return &CSVParser{
		expectedColumns: []string{
			"name", "url", "method", "interval",
			"timeout", "expected_status", "enabled",
		},
	}
}

// GetSource возвращает источник импорта.
func (p *CSVParser) GetSource() model.ImportSource {
	return model.ImportSourceCSV
}

// Validate валидирует структуру данных.
func (p *CSVParser) Validate(data []byte) error {
	reader := csv.NewReader(strings.NewReader(string(data)))

	// Читаем первую строку (headers)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("CSV file is empty")
	}

	if len(records[0]) < 2 {
		return fmt.Errorf("CSV must have at least 2 columns (name, url)")
	}

	// Проверяем, что есть обязательные колонки
	hasName := false
	hasURL := false

	for _, col := range records[0] {
		lowerCol := strings.ToLower(strings.TrimSpace(col))
		if lowerCol == "name" || lowerCol == "monitor_name" {
			hasName = true
		}
		if lowerCol == "url" || lowerCol == "monitor_url" {
			hasURL = true
		}
	}

	if !hasName || !hasURL {
		return fmt.Errorf("CSV must have 'name' and 'url' columns")
	}

	return nil
}

// Parse парсит CSV файл.
func (p *CSVParser) Parse(data []byte) ([]*model.MonitorImportData, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))

	// Читаем все записи
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Первая строка - заголовки
	headers := records[0]
	monitors := make([]*model.MonitorImportData, 0)

	// Остальные строки - данные
	for i, record := range records[1:] {
		if len(record) == 0 {
			continue // Пропускаем пустые строки
		}

		monitor, err := p.parseRecord(i+1, headers, record)
		if err != nil {
			// Возвращаем мониторы, которые удалось распарсить + ошибку
			return monitors, fmt.Errorf("row %d: %w", i+1, err)
		}

		if monitor != nil {
			monitors = append(monitors, monitor)
		}
	}

	return monitors, nil
}

// parseRecord парсит одну строку CSV.
func (p *CSVParser) parseRecord(rowNum int, headers, record []string) (*model.MonitorImportData, error) {
	monitor := &model.MonitorImportData{
		Tags: []string{"csv"},
	}

	// Создаём map для удобного доступа
	colMap := make(map[string]string)
	for i, value := range record {
		if i < len(headers) {
			colName := strings.ToLower(strings.TrimSpace(headers[i]))
			colMap[colName] = strings.TrimSpace(value)
		}
	}

	// Обязательные поля
	name, ok := p.getColumn(colMap, "name", "monitor_name")
	if !ok || name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}

	url, ok := p.getColumn(colMap, "url", "monitor_url")
	if !ok || url == "" {
		return nil, fmt.Errorf("missing required field: url")
	}

	monitor.Name = name
	monitor.URL = url

	// Опциональные поля
	if method, ok := p.getColumn(colMap, "method", "http_method"); ok && method != "" {
		monitor.Method = method
	} else {
		monitor.Method = "GET" // default
	}

	if interval, ok := p.getColumn(colMap, "interval", "check_interval"); ok && interval != "" {
		// Парсим интервал (может быть "60s", "1m", "60")
		parsedInterval, err := parseInterval(interval)
		if err == nil {
			monitor.CheckInterval = &parsedInterval
		}
	}

	if timeout, ok := p.getColumn(colMap, "timeout", "request_timeout"); ok && timeout != "" {
		parsedTimeout, err := parseInterval(timeout)
		if err == nil {
			monitor.Timeout = &parsedTimeout
		}
	}

	if expectedStatus, ok := p.getColumn(colMap, "expected_status", "status_code"); ok && expectedStatus != "" {
		monitor.ExpectedPattern = &expectedStatus
	}

	if enabled, ok := p.getColumn(colMap, "enabled", "is_enabled"); ok && enabled != "" {
		parsedEnabled := strings.ToLower(enabled) == "true" || enabled == "1"
		monitor.Enabled = &parsedEnabled
	} else {
		monitor.Enabled = boolPtr(true) // default
	}

	// Добавляем row number в tags для отладки
	monitor.Tags = append(monitor.Tags, fmt.Sprintf("csv_row:%d", rowNum))

	// Валидируем
	if err := monitor.Validate(); err != nil {
		return nil, err
	}

	return monitor, nil
}

// getColumn получает значение колонки по нескольким возможным именам.
func (p *CSVParser) getColumn(colMap map[string]string, names ...string) (string, bool) {
	for _, name := range names {
		lowerName := strings.ToLower(name)
		for colName, value := range colMap {
			lowerColName := strings.ToLower(colName)
			if lowerColName == lowerName {
				return value, true
			}
		}
	}
	return "", false
}

// parseInterval парсит интервал в формате (60s, 1m, и т.д.)
func parseInterval(intervalStr string) (int, error) {
	intervalStr = strings.ToLower(strings.TrimSpace(intervalStr))

	// Проверяем суффикс
	var multiplier int
	switch {
	case strings.HasSuffix(intervalStr, "s"):
		multiplier = 1
		intervalStr = strings.TrimSuffix(intervalStr, "s")
	case strings.HasSuffix(intervalStr, "m"):
		multiplier = 60
		intervalStr = strings.TrimSuffix(intervalStr, "m")
	case strings.HasSuffix(intervalStr, "h"):
		multiplier = 3600
		intervalStr = strings.TrimSuffix(intervalStr, "h")
	default:
		// Без суффикса считаем секунды
		multiplier = 1
	}

	// Парсим число
	var interval int
	_, err := fmt.Sscanf(intervalStr, "%d", &interval)
	if err != nil {
		return 0, fmt.Errorf("invalid interval format: %s", intervalStr)
	}

	return interval * multiplier, nil
}
