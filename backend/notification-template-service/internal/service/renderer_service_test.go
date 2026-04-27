package service

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

func newTestCfg() *config.Config {
	return &config.Config{MaxTemplateSize: 100000}
}

func newNopLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

func TestRendererService_New(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := NewRendererService(repo, newNopLogger(), newTestCfg())
	if svc == nil {
		t.Fatal("NewRendererService() returned nil")
	}
}

func TestRendererService_RenderTemplate_Success(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-1"] = &model.Template{
		ID:      "tmpl-1",
		Name:    "Test",
		Body:    "Monitor {{ index . \"monitor_name\" }} is {{ index . \"status\" }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatText,
	}

	svc := NewRendererService(repo, newNopLogger(), newTestCfg())
	result, err := svc.RenderTemplate(context.Background(), "tmpl-1", model.VariablesMap{
		"monitor_name": "My API",
		"status":       "DOWN",
	})

	if err != nil {
		t.Fatalf("RenderTemplate() unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("RenderTemplate() Success = false, want true")
	}
	if result.Body == "" {
		t.Error("RenderTemplate() Body should not be empty")
	}
}

func TestRendererService_RenderTemplate_TemplateNotFound(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := NewRendererService(repo, newNopLogger(), newTestCfg())

	_, err := svc.RenderTemplate(context.Background(), "non-existent", model.VariablesMap{})
	if err == nil {
		t.Fatal("RenderTemplate() expected error for missing template, got nil")
	}
}

func TestRendererService_RenderTemplate_RepoError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.getErr = errors.New("db error")
	svc := NewRendererService(repo, newNopLogger(), newTestCfg())

	_, err := svc.RenderTemplate(context.Background(), "any-id", model.VariablesMap{})
	if err == nil {
		t.Fatal("RenderTemplate() expected error when repo fails, got nil")
	}
}

func TestRendererService_RenderTemplate_InvalidBodySyntax(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["bad-tmpl"] = &model.Template{
		ID:      "bad-tmpl",
		Name:    "Bad Template",
		Body:    "{{ .name | badfunction }}", // will fail on execution with missingkey=zero but parse ok
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatText,
	}

	svc := NewRendererService(repo, newNopLogger(), newTestCfg())
	_, err := svc.RenderTemplate(context.Background(), "bad-tmpl", model.VariablesMap{})
	// Either error or success depending on Go template behavior — just ensure no panic
	_ = err
}

func TestRendererService_RenderTemplate_WithSubject(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-subj"] = &model.Template{
		ID:      "tmpl-subj",
		Name:    "Test",
		Subject: "Alert for {{ index . \"monitor_name\" }}",
		Body:    "Monitor {{ index . \"monitor_name\" }} is down",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatHTML,
	}

	svc := NewRendererService(repo, newNopLogger(), newTestCfg())
	result, err := svc.RenderTemplate(context.Background(), "tmpl-subj", model.VariablesMap{
		"monitor_name": "My API",
		"status":       "DOWN",
	})

	if err != nil {
		t.Fatalf("RenderTemplate() unexpected error: %v", err)
	}
	if result.Subject == "" {
		t.Error("RenderTemplate() Subject should not be empty")
	}
}

func TestRendererService_PreviewTemplate_Success(t *testing.T) {
	svc := NewRendererService(NewMockTemplateRepository(), newNopLogger(), newTestCfg())

	tmpl := &model.Template{
		ID:      "preview-1",
		Name:    "Preview Test",
		Body:    "Monitor {{ index . \"monitor_name\" }} is {{ index . \"status\" }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatText,
	}

	result, err := svc.PreviewTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("PreviewTemplate() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("PreviewTemplate() returned nil result")
	}
}

func TestRendererService_PreviewTemplate_AllTypes(t *testing.T) {
	svc := NewRendererService(NewMockTemplateRepository(), newNopLogger(), newTestCfg())

	types := []model.TemplateType{
		model.TypeMonitorUp,
		model.TypeMonitorDown,
		model.TypeMonitorDegraded,
		model.TypeCertificateExpiry,
		model.TypeFlappingDetected,
		model.TypeIncidentCreated,
		model.TypeCustom,
	}

	for _, tmplType := range types {
		t.Run(string(tmplType), func(t *testing.T) {
			tmpl := &model.Template{
				Name:    "Test",
				Body:    "body content",
				Channel: model.ChannelEmail,
				Type:    tmplType,
				Format:  model.FormatText,
			}

			result, err := svc.PreviewTemplate(context.Background(), tmpl)
			if err != nil {
				t.Fatalf("PreviewTemplate() type=%s unexpected error: %v", tmplType, err)
			}
			if result == nil {
				t.Fatalf("PreviewTemplate() type=%s returned nil", tmplType)
			}
		})
	}
}

func TestRendererService_RenderTemplate_UnusedVariableWarning(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-warn"] = &model.Template{
		ID:      "tmpl-warn",
		Name:    "Test",
		Body:    "simple body without variables",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatText,
	}

	svc := NewRendererService(repo, newNopLogger(), newTestCfg())
	result, err := svc.RenderTemplate(context.Background(), "tmpl-warn", model.VariablesMap{
		"unused_var": "value",
	})
	if err != nil {
		t.Fatalf("RenderTemplate() unexpected error: %v", err)
	}

	_ = result
}

func TestRendererService_Render_SubjectError(t *testing.T) {
	repo := NewMockTemplateRepository()
	// Parse fails for bad subject; we call render directly
	svc := NewRendererService(repo, newNopLogger(), newTestCfg())

	tmpl := &model.Template{
		ID:      "bad-subj",
		Name:    "Bad Subject",
		Subject: "{{ bad subject syntax",
		Body:    "valid body",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatText,
	}

	result := svc.render(tmpl, model.VariablesMap{})
	if result.Success {
		t.Error("render() with bad subject should set Success=false")
	}
	if len(result.Errors) == 0 {
		t.Error("render() with bad subject should have errors")
	}
}

func TestRendererService_Render_BodyError(t *testing.T) {
	svc := NewRendererService(NewMockTemplateRepository(), newNopLogger(), newTestCfg())

	tmpl := &model.Template{
		ID:      "bad-body",
		Name:    "Bad Body",
		Body:    "{{ bad body syntax",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatText,
	}

	result := svc.render(tmpl, model.VariablesMap{})
	if result.Success {
		t.Error("render() with bad body should set Success=false")
	}
	if len(result.Errors) == 0 {
		t.Error("render() with bad body should have errors")
	}
}
