package service

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

func newValidatorService() *ValidatorServiceImpl {
	l := zerolog.Nop()
	return NewValidatorService(&l)
}

func TestValidatorService_New(t *testing.T) {
	svc := newValidatorService()
	if svc == nil {
		t.Fatal("NewValidatorService() returned nil")
	}
}

func TestValidatorService_ValidateTemplate_ValidTemplate(t *testing.T) {
	svc := newValidatorService()

	tmpl := &model.Template{
		Name:    "Test",
		Subject: "Alert: {{ .monitor_name }}",
		Body:    "Monitor {{ .monitor_name }} is {{ .status }} at {{ .timestamp }} - {{ .monitor_url }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatHTML,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("ValidateTemplate() returned nil result")
	}
}

func TestValidatorService_ValidateTemplate_InvalidSyntax(t *testing.T) {
	svc := newValidatorService()

	tmpl := &model.Template{
		Name:    "Bad Template",
		Body:    "{{ bad syntax",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("ValidateTemplate() with bad syntax should return Valid=false")
	}
	if len(result.Errors) == 0 {
		t.Error("ValidateTemplate() with bad syntax should have errors")
	}
}

func TestValidatorService_ValidateTemplate_MissingRequiredVars(t *testing.T) {
	svc := newValidatorService()

	// Template that doesn't use any required variables
	tmpl := &model.Template{
		Name:    "Missing Vars",
		Body:    "Static text without any variables",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Error("ValidateTemplate() missing required vars should produce warnings")
	}
	if len(result.MissingVariables) == 0 {
		t.Error("ValidateTemplate() should list missing variables")
	}
}

func TestValidatorService_ValidateTemplate_LargeBody(t *testing.T) {
	svc := newValidatorService()

	// Build a body > 100KB
	largeBody := strings.Repeat("a", 100001)

	tmpl := &model.Template{
		Name:    "Large Template",
		Body:    largeBody,
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Error("ValidateTemplate() with large body should have a size warning")
	}

	hasLargeWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "large") || strings.Contains(w, "100KB") {
			hasLargeWarning = true
		}
	}
	if !hasLargeWarning {
		t.Errorf("ValidateTemplate() warnings = %v, expected large body warning", result.Warnings)
	}
}

func TestValidatorService_ValidateTemplate_ExtractsUsedVars(t *testing.T) {
	svc := newValidatorService()

	tmpl := &model.Template{
		Name:    "Test",
		Body:    "Monitor {{ .monitor_name }} is {{ .status }} at {{ .timestamp }} - {{ .monitor_url }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if len(result.UsedVariables) == 0 {
		t.Error("ValidateTemplate() should populate UsedVariables")
	}
}

func TestValidatorService_ValidateTemplate_WithSubjectVars(t *testing.T) {
	svc := newValidatorService()

	tmpl := &model.Template{
		Name:    "Test",
		Subject: "Alert: {{ .monitor_name }}",
		Body:    "Status: {{ .status }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	// monitor_name extracted from subject, status from body
	foundMonitorName := false
	for _, v := range result.UsedVariables {
		if v == "monitor_name" {
			foundMonitorName = true
		}
	}
	if !foundMonitorName {
		t.Errorf("ValidateTemplate() UsedVariables = %v, missing 'monitor_name'", result.UsedVariables)
	}
}

func TestValidatorService_ValidateTemplate_SyntaxErrorEarlyReturn(t *testing.T) {
	svc := newValidatorService()

	// Bad subject — should return early without checking variables
	tmpl := &model.Template{
		Name:    "Test",
		Subject: "{{ bad",
		Body:    "valid body",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	result, err := svc.ValidateTemplate(context.Background(), tmpl)
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("ValidateTemplate() with bad subject syntax should be invalid")
	}
}

func TestValidatorService_GetAvailableVariables(t *testing.T) {
	svc := newValidatorService()

	tests := []struct {
		tmplType model.TemplateType
		channel  model.TemplateChannel
	}{
		{model.TypeMonitorDown, model.ChannelEmail},
		{model.TypeMonitorDegraded, model.ChannelTelegram},
		{model.TypeCertificateExpiry, model.ChannelSlack},
		{model.TypeFlappingDetected, model.ChannelWebhook},
		{model.TypeMonitorUp, model.ChannelDiscord},
		{model.TypeCustom, model.ChannelSMS},
	}

	for _, tt := range tests {
		t.Run(string(tt.tmplType), func(t *testing.T) {
			vars := svc.GetAvailableVariables(context.Background(), tt.tmplType, tt.channel)
			if len(vars) == 0 {
				t.Errorf("GetAvailableVariables(%s, %s) returned empty list", tt.tmplType, tt.channel)
			}
			// All vars should have a name
			for _, v := range vars {
				if v.Name == "" {
					t.Error("GetAvailableVariables() returned variable with empty name")
				}
			}
		})
	}
}
