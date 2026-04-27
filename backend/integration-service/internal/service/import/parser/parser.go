// Package parser содержит парсеры для импорта мониторов из внешних систем.
//
// Пакет реализует парсеры для:
//   - UptimeRobot JSON export format
//   - Pingdom JSON export format
//   - CSV format
//   - JSON format
package parser

import (
	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// Parser определяет интерфейс для парсинга импорта мониторов.
type Parser interface {
	// Parse парсит данные и возвращает мониторы для импорта.
	Parse(data []byte) ([]*model.MonitorImportData, error)

	// Validate валидирует структуру данных перед парсингом.
	Validate(data []byte) error

	// GetSource возвращает источник импорта.
	GetSource() model.ImportSource
}
