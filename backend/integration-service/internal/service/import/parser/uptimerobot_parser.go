package parser

import (
	"encoding/json"
	"fmt"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// UptimeRobotParser парсит UptimeRobot JSON export.
type UptimeRobotParser struct{}

// NewUptimeRobotParser создаёт новый UptimeRobotParser.
func NewUptimeRobotParser() *UptimeRobotParser {
	return &UptimeRobotParser{}
}

// GetSource возвращает источник импорта.
func (p *UptimeRobotParser) GetSource() model.ImportSource {
	return model.ImportSourceUptimeRobot
}

// Validate валидирует структуру данных.
func (p *UptimeRobotParser) Validate(data []byte) error {
	var export UptimeRobotExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("invalid UptimeRobot JSON format: %w", err)
	}

	if len(export.Monitors) == 0 {
		return fmt.Errorf("no monitors found in export")
	}

	return nil
}

// Parse парсит UptimeRobot JSON export.
func (p *UptimeRobotParser) Parse(data []byte) ([]*model.MonitorImportData, error) {
	var export UptimeRobotExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse UptimeRobot JSON: %w", err)
	}

	monitors := make([]*model.MonitorImportData, 0, len(export.Monitors))

	for _, um := range export.Monitors {
		monitor := p.convertMonitor(um)
		monitors = append(monitors, monitor)
	}

	return monitors, nil
}

// convertMonitor конвертирует UptimeRobot монитор в нашу модель.
func (p *UptimeRobotParser) convertMonitor(um UptimeRobotMonitor) *model.MonitorImportData {
	monitor := &model.MonitorImportData{
		Name:          um.FriendlyName,
		URL:           um.URL,
		Method:        "GET", // UptimeRobot использует GET
		CheckInterval: &um.Interval,
		Enabled:       boolPtr(um.Status == 1 || um.Status == 2), // 1=up, 2=paused, 0=not checked, 9=down
		Tags:          um.Tags,
	}

	// UptimeRobot specific settings
	switch um.Type {
	case 1:
		// HTTP
		monitor.Method = "GET"
	case 2:
		// Keyword checking требует специальной HTTP библиотеки для keyword matching
		// При импорте конвертируем в обычный HTTP монитор
		monitor.Method = "GET"
	}

	// Contact Groups для оповещений
	if len(um.ContactGroups) > 0 {
		// Contact Groups конвертируются в теги для последующей настройки
		// Полная конвертация требует настройки notification channels
		monitor.Tags = append(monitor.Tags, fmt.Sprintf("contact_groups:%v", um.ContactGroups))
	}

	// Status thresholds
	if um.Status == 1 {
		// Active
		monitor.Enabled = boolPtr(true)
	}

	return monitor
}

// UptimeRobotExport структура UptimeRobot JSON export.
type UptimeRobotExport struct {
	Monitors []UptimeRobotMonitor `json:"monitors"`
}

// UptimeRobotMonitor структура монитора из UptimeRobot.
type UptimeRobotMonitor struct {
	ID            int64    `json:"id"`
	UserID        int64    `json:"user_id"`
	FriendlyName  string   `json:"friendly_name"`
	URL           string   `json:"url"`
	Type          int      `json:"type"`
	SubType       *string  `json:"sub_type"`
	Port          *int     `json:"port"`
	Interval      int      `json:"interval"` // seconds
	Timeout       int      `json:"timeout"`  // seconds
	Status        int      `json:"status"`   // 0=not checked, 1=up, 2=paused, 9=down
	ContactGroups []int64  `json:"contact_groups"`
	Tags          []string `json:"tags"`
}
