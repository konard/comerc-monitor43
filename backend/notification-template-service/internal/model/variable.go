package model

import "strings"

// VariableType представляет тип переменной.
type VariableType string

const (
	VarTypeString    VariableType = "string"
	VarTypeInt       VariableType = "int"
	VarTypeFloat     VariableType = "float"
	VarTypeBool      VariableType = "bool"
	VarTypeTimestamp VariableType = "timestamp"
	VarTypeDuration  VariableType = "duration"
	VarTypeObject    VariableType = "object"
	VarTypeArray     VariableType = "array"
)

// TemplateVariable представляет переменную шаблона.
type TemplateVariable struct {
	Name         string
	Description  string
	Type         VariableType
	Required     bool
	ExampleValue string

	// Для вложенных объектов
	NestedFields []string
}

// VariablesMap представляет мапу переменных для рендеринга.
type VariablesMap map[string]string

// RenderedTemplate представляет отрендеренный шаблон.
type RenderedTemplate struct {
	Subject  string
	Body     string
	Format   TemplateFormat
	Success  bool
	Errors   []string
	Warnings []string
}

// ValidationResult представляет результат валидации шаблона.
type ValidationResult struct {
	Valid            bool
	Errors           []string
	Warnings         []string
	UsedVariables    []string
	MissingVariables []string
	UnusedVariables  []string // Переменные, переданные но не использованные
}

// HasErrors проверяет наличие ошибок.
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings проверяет наличие предупреждений.
func (vr *ValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// AddError добавляет ошибку.
func (vr *ValidationResult) AddError(err string) {
	vr.Errors = append(vr.Errors, err)
	vr.Valid = false
}

// AddWarning добавляет предупреждение.
func (vr *ValidationResult) AddWarning(warning string) {
	vr.Warnings = append(vr.Warnings, warning)
}

// ExtractVariables извлекает переменные из шаблона.
// Для Go templates ищет {{ .VariableName }}.
func ExtractVariables(tmpl string) []string {
	var vars []string
	seen := make(map[string]bool)

	// Простое извлечение для Go template syntax
	// Ищет паттерны вида {{ .VarName }} или {{.VarName}}
	parts := strings.Split(tmpl, "{{")
	for _, part := range parts[1:] { // Пропускаем первую часть (до первого {{)
		end := strings.Index(part, "}}")
		if end == -1 {
			continue
		}

		expr := strings.TrimSpace(part[:end])
		expr = strings.TrimPrefix(expr, ".")
		expr = strings.TrimSpace(expr)

		// Простая проверка: если это слово (не содержит пробелов и специальных символов)
		if expr != "" && !strings.ContainsAny(expr, " \t\n\r+-*/%()[]{}") {
			if !seen[expr] {
				seen[expr] = true
				vars = append(vars, expr)
			}
		}
	}

	return vars
}

// GetRequiredVariables возвращает обязательные переменные для типа шаблона.
func GetRequiredVariables(tmplType TemplateType, channel TemplateChannel) []TemplateVariable {
	// Базовый набор переменных для всех типов
	baseVars := []TemplateVariable{
		{
			Name:         "monitor_name",
			Description:  "Имя монитора",
			Type:         VarTypeString,
			Required:     true,
			ExampleValue: "My API Endpoint",
		},
		{
			Name:         "monitor_url",
			Description:  "URL монитора",
			Type:         VarTypeString,
			Required:     true,
			ExampleValue: "https://api.example.com/health",
		},
		{
			Name:         "status",
			Description:  "Текущий статус",
			Type:         VarTypeString,
			Required:     true,
			ExampleValue: "DOWN",
		},
		{
			Name:         "timestamp",
			Description:  "Время события",
			Type:         VarTypeTimestamp,
			Required:     true,
			ExampleValue: "2026-03-27T12:34:56Z",
		},
	}

	// Специфичные переменные для типов
	switch tmplType {
	case TypeMonitorDown:
		return append(baseVars, TemplateVariable{
			Name:         "error_message",
			Description:  "Сообщение об ошибке",
			Type:         VarTypeString,
			Required:     true,
			ExampleValue: "Connection refused",
		})

	case TypeMonitorDegraded:
		return append(baseVars, []TemplateVariable{
			{
				Name:         "response_time_ms",
				Description:  "Время отклика в миллисекундах",
				Type:         VarTypeInt,
				Required:     true,
				ExampleValue: "2500",
			},
			{
				Name:         "threshold_ms",
				Description:  "Пороговое значение",
				Type:         VarTypeInt,
				Required:     true,
				ExampleValue: "1000",
			},
		}...)

	case TypeCertificateExpiry:
		return append(baseVars, []TemplateVariable{
			{
				Name:         "days_until_expiry",
				Description:  "Дней до истечения сертификата",
				Type:         VarTypeInt,
				Required:     true,
				ExampleValue: "7",
			},
			{
				Name:         "expiry_date",
				Description:  "Дата истечения сертификата",
				Type:         VarTypeTimestamp,
				Required:     true,
				ExampleValue: "2026-04-03T00:00:00Z",
			},
		}...)

	case TypeFlappingDetected:
		return append(baseVars, []TemplateVariable{
			{
				Name:         "state_changes",
				Description:  "Количество изменений состояния",
				Type:         VarTypeInt,
				Required:     true,
				ExampleValue: "5",
			},
			{
				Name:         "time_window",
				Description:  "Временное окно",
				Type:         VarTypeDuration,
				Required:     true,
				ExampleValue: "5m",
			},
		}...)

	default:
		return baseVars
	}
}
