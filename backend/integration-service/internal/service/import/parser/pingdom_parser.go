package parser

import (
	"encoding/json"
	"fmt"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// PingdomParser парсит Pingdom JSON export.
type PingdomParser struct{}

// NewPingdomParser создаёт новый PingdomParser.
func NewPingdomParser() *PingdomParser {
	return &PingdomParser{}
}

// GetSource возвращает источник импорта.
func (p *PingdomParser) GetSource() model.ImportSource {
	return model.ImportSourcePingdom
}

// Validate валидирует структуру данных.
func (p *PingdomParser) Validate(data []byte) error {
	var export PingdomExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("invalid Pingdom JSON format: %w", err)
	}

	if len(export.Checks) == 0 {
		return fmt.Errorf("no checks found in export")
	}

	return nil
}

// Parse парсит Pingdom JSON export.
func (p *PingdomParser) Parse(data []byte) ([]*model.MonitorImportData, error) {
	var export PingdomExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse Pingdom JSON: %w", err)
	}

	monitors := make([]*model.MonitorImportData, 0, len(export.Checks))

	for _, check := range export.Checks {
		monitor := p.convertCheck(check)
		monitors = append(monitors, monitor)
	}

	return monitors, nil
}

// convertCheck конвертирует Pingdom check в нашу модель.
func (p *PingdomParser) convertCheck(check PingdomCheck) *model.MonitorImportData {
	monitor := &model.MonitorImportData{
		Name:          check.Name,
		URL:           check.Hostname,
		Method:        "GET",
		CheckInterval: &check.Interval,
		Enabled:       boolPtr(check.Status == "up"),
		Tags:          []string{"pingdom"},
	}

	// Pingdom specific settings
	if check.Type == "http" {
		monitor.Method = "GET"

		// HTTP headers если есть
		if check.RequestHeaders != nil {
			// Конвертируем headers в формат монитора
			// Важные headers (User-Agent и др.) сохраняются в тегах
			for k, v := range check.RequestHeaders {
				if k == "User-Agent" {
					monitor.Tags = append(monitor.Tags, fmt.Sprintf("ua:%s", v))
				}
				// Другие headers можно добавить в будущем при расширении функциональности
			}
		}
	}

	// Tags дляPingdom проверки
	if check.Tags != nil {
		monitor.Tags = append(monitor.Tags, check.Tags...)
	}

	// Status thresholds
	if check.Status == "up" || check.Status == "paused" {
		monitor.Enabled = boolPtr(true)
	}

	// Alert thresholds
	if check.AlertThreshold != nil {
		monitor.Tags = append(monitor.Tags, fmt.Sprintf("alert_threshold:%d", *check.AlertThreshold))
	}

	return monitor
}

// PingdomExport структура Pingdom JSON export.
type PingdomExport struct {
	Checks []PingdomCheck `json:"checks"`
}

// PingdomCheck структура check из Pingdom.
type PingdomCheck struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Hostname       string            `json:"hostname"`
	Resolution     int               `json:"resolution"`
	Interval       int               `json:"interval"` // seconds
	Timeout        int               `json:"timeout"`  // seconds
	Status         string            `json:"status"`
	Tags           []string          `json:"tags"`
	RequestHeaders map[string]string `json:"request_headers"`
	AlertThreshold *int              `json:"alert_threshold"`
}
