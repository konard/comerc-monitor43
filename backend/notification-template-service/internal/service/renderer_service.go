package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/metrics"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

// RendererServiceImpl реализует RendererService.
type RendererServiceImpl struct {
	repo   TemplateRepository
	logger *zerolog.Logger
	cfg    *config.Config
}

// NewRendererService создаёт новый сервис рендеринга.
func NewRendererService(repo TemplateRepository, logger *zerolog.Logger, cfg *config.Config) *RendererServiceImpl {
	return &RendererServiceImpl{
		repo:   repo,
		logger: logger,
		cfg:    cfg,
	}
}

// RenderTemplate рендерит шаблон с переменными.
func (s *RendererServiceImpl) RenderTemplate(ctx context.Context, templateID string, variables model.VariablesMap) (*model.RenderedTemplate, error) {
	// Получаем шаблон
	tmpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		s.logger.Error().Err(err).Str("template_id", templateID).Msg("failed to get template for rendering")
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Замеряем время рендеринга
	start := time.Now()
	rendered := s.render(tmpl, variables)
	duration := time.Since(start).Seconds()

	// Записываем метрику
	metrics.RecordTemplateRender(string(tmpl.Channel), string(tmpl.Type), rendered.Success, duration)

	if !rendered.Success {
		s.logger.Error().
			Str("template_id", templateID).
			Strs("errors", rendered.Errors).
			Msg("template rendering failed")
		return rendered, fmt.Errorf("template rendering failed: %v", rendered.Errors)
	}

	return rendered, nil
}

// PreviewTemplate превью шаблона с тестовыми данными.
func (s *RendererServiceImpl) PreviewTemplate(ctx context.Context, tmpl *model.Template) (*model.RenderedTemplate, error) {
	// Получаем переменные для типа шаблона
	templateVars := model.GetRequiredVariables(tmpl.Type, tmpl.Channel)

	// Создаём тестовые данные
	sampleData := make(model.VariablesMap)
	for _, tv := range templateVars {
		if tv.ExampleValue != "" {
			sampleData[tv.Name] = tv.ExampleValue
		} else {
			sampleData[tv.Name] = fmt.Sprintf("sample_%s", tv.Name)
		}
	}

	// Рендерим
	rendered := s.render(tmpl, sampleData)

	return rendered, nil
}

// render выполняет рендеринг шаблона.
func (s *RendererServiceImpl) render(tmpl *model.Template, variables model.VariablesMap) *model.RenderedTemplate {
	result := &model.RenderedTemplate{
		Format:   tmpl.Format,
		Success:  true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Рендер subject (если есть)
	if tmpl.Subject != "" {
		subject, err := s.renderTemplateString(tmpl.Subject, variables)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("subject error: %v", err))
		}
		result.Subject = subject
	}

	// Рендер body
	body, err := s.renderTemplateString(tmpl.Body, variables)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("body error: %v", err))
	}
	result.Body = body

	// Проверяем на неиспользованные переменные
	if len(result.Errors) == 0 {
		s.checkUnusedVariables(tmpl, variables, result)
	}

	return result
}

// renderTemplateString рендерит строку шаблона.
func (s *RendererServiceImpl) renderTemplateString(tmplString string, variables model.VariablesMap) (string, error) {
	// Создаём Go template
	tmpl, err := template.New("template").Option("missingkey=zero").Parse(tmplString)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Рендерим
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// checkUnusedVariables проверяет на неиспользованные переменные.
func (s *RendererServiceImpl) checkUnusedVariables(tmpl *model.Template, variables model.VariablesMap, result *model.RenderedTemplate) {
	// Извлекаем используемые переменные из шаблона
	usedVars := make(map[string]bool)
	for _, tmplStr := range []string{tmpl.Subject, tmpl.Body} {
		if tmplStr == "" {
			continue
		}
		extracted := model.ExtractVariables(tmplStr)
		for _, v := range extracted {
			usedVars[v] = true
		}
	}

	// Проверяем, что все обязательные переменные используются
	templateVars := model.GetRequiredVariables(tmpl.Type, tmpl.Channel)
	for _, tv := range templateVars {
		if tv.Required && !usedVars[tv.Name] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("required variable '%s' is not used in template", tv.Name))
		}
	}

	// Проверяем на переданные но не используемые переменные
	for varName := range variables {
		if !usedVars[varName] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("variable '%s' was provided but not used in template", varName))
		}
	}
}
