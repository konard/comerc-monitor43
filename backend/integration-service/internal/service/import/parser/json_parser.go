package parser

import (
	"encoding/json"
	"fmt"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// JSONParser парсит JSON массив мониторов.
type JSONParser struct{}

// NewJSONParser создаёт новый JSONParser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// GetSource возвращает источник импорта.
func (p *JSONParser) GetSource() model.ImportSource {
	return model.ImportSourceJSON
}

// Validate валидирует структуру данных.
func (p *JSONParser) Validate(data []byte) error {
	var monitors []model.MonitorImportData
	if err := json.Unmarshal(data, &monitors); err != nil {
		return err
	}

	if len(monitors) == 0 {
		return fmt.Errorf("no monitors found in JSON")
	}

	// Валидируем каждый монитор
	for i, monitor := range monitors {
		if err := monitor.Validate(); err != nil {
			return fmt.Errorf("monitor %d: %w", i, err)
		}
	}

	return nil
}

// Parse парсит JSON массив мониторов.
func (p *JSONParser) Parse(data []byte) ([]*model.MonitorImportData, error) {
	var monitors []*model.MonitorImportData
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, err
	}

	// Валидируем каждый монитор
	validMonitors := make([]*model.MonitorImportData, 0, len(monitors))

	for _, monitor := range monitors {
		if err := monitor.Validate(); err != nil {
			// Добавляем информацию об ошибке в теги
			monitor.Tags = append(monitor.Tags, fmt.Sprintf("validation_error:%v", err))
			continue
		}
		validMonitors = append(validMonitors, monitor)
	}

	return validMonitors, nil
}
