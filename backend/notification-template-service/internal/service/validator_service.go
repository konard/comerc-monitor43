package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

// ValidatorServiceImpl реализует ValidatorService.
type ValidatorServiceImpl struct {
	logger *zerolog.Logger
}

// NewValidatorService создаёт новый сервис валидации.
func NewValidatorService(logger *zerolog.Logger) *ValidatorServiceImpl {
	return &ValidatorServiceImpl{
		logger: logger,
	}
}

// ValidateTemplate проверяет синтаксис и переменные шаблона.
func (s *ValidatorServiceImpl) ValidateTemplate(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error) {
	result := &model.ValidationResult{
		Valid:            true,
		Errors:           make([]string, 0),
		Warnings:         make([]string, 0),
		UsedVariables:    make([]string, 0),
		MissingVariables: make([]string, 0),
	}

	// Проверяем синтаксис
	syntaxErrors := tmpl.ValidateSyntax()
	if len(syntaxErrors) > 0 {
		for _, err := range syntaxErrors {
			result.AddError(err.Error())
		}
	}

	// Если есть ошибки синтаксиса, возвращаем результат
	if result.HasErrors() {
		result.Valid = false
		return result, nil
	}

	// Извлекаем используемые переменные
	allVars := make(map[string]bool)
	for _, tmplStr := range []string{tmpl.Subject, tmpl.Body} {
		if tmplStr == "" {
			continue
		}
		extracted := model.ExtractVariables(tmplStr)
		for _, v := range extracted {
			allVars[v] = true
		}
	}

	// Заполняем список используемых переменных
	for v := range allVars {
		result.UsedVariables = append(result.UsedVariables, v)
	}

	// Проверяем, что все обязательные переменные присутствуют
	requiredVars := model.GetRequiredVariables(tmpl.Type, tmpl.Channel)
	requiredMap := make(map[string]bool)
	for _, rv := range requiredVars {
		if rv.Required {
			requiredMap[rv.Name] = true
		}
	}

	for varName := range requiredMap {
		if !allVars[varName] {
			result.MissingVariables = append(result.MissingVariables, varName)
			result.AddWarning(fmt.Sprintf("missing required variable: %s", varName))
		}
	}

	// Проверяем размер шаблона
	if len(tmpl.Body) > 100000 { // 100KB
		result.AddWarning("template body is very large (>100KB), consider reducing size")
	}

	return result, nil
}

// GetAvailableVariables возвращает доступные переменные для типа шаблона.
func (s *ValidatorServiceImpl) GetAvailableVariables(
	ctx context.Context,
	tmplType model.TemplateType,
	channel model.TemplateChannel,
) []model.TemplateVariable {
	return model.GetRequiredVariables(tmplType, channel)
}
