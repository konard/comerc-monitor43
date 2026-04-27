package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/raul/monitor/api/proto"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// --- Моки сервисов ---

// mockTemplateService мок сервиса шаблонов.
type mockTemplateService struct {
	createFn     func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error)
	getFn        func(ctx context.Context, id string) (*model.Template, error)
	listFn       func(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error)
	updateFn     func(ctx context.Context, id string, req *service.UpdateTemplateRequest) (*model.Template, error)
	deleteFn     func(ctx context.Context, id string) error
	cloneFn      func(ctx context.Context, id, userID, newName string) (*model.Template, error)
	setDefaultFn func(ctx context.Context, id, userID string) (*model.Template, error)
	getDefaultFn func(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error)
}

func (m *mockTemplateService) CreateTemplate(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
	return m.createFn(ctx, req)
}

func (m *mockTemplateService) GetTemplate(ctx context.Context, id string) (*model.Template, error) {
	return m.getFn(ctx, id)
}

func (m *mockTemplateService) ListTemplates(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error) {
	return m.listFn(ctx, filter)
}

func (m *mockTemplateService) UpdateTemplate(ctx context.Context, id string, req *service.UpdateTemplateRequest) (*model.Template, error) {
	return m.updateFn(ctx, id, req)
}

func (m *mockTemplateService) DeleteTemplate(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func (m *mockTemplateService) CloneTemplate(ctx context.Context, id, userID, newName string) (*model.Template, error) {
	return m.cloneFn(ctx, id, userID, newName)
}

func (m *mockTemplateService) SetDefaultTemplate(ctx context.Context, id, userID string) (*model.Template, error) {
	return m.setDefaultFn(ctx, id, userID)
}

func (m *mockTemplateService) GetDefaultTemplate(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error) {
	return m.getDefaultFn(ctx, userID, channel, tmplType)
}

// mockRendererService мок сервиса рендеринга.
type mockRendererService struct {
	renderFn  func(ctx context.Context, templateID string, variables model.VariablesMap) (*model.RenderedTemplate, error)
	previewFn func(ctx context.Context, tmpl *model.Template) (*model.RenderedTemplate, error)
}

func (m *mockRendererService) RenderTemplate(ctx context.Context, templateID string, variables model.VariablesMap) (*model.RenderedTemplate, error) {
	return m.renderFn(ctx, templateID, variables)
}

func (m *mockRendererService) PreviewTemplate(ctx context.Context, tmpl *model.Template) (*model.RenderedTemplate, error) {
	return m.previewFn(ctx, tmpl)
}

// mockValidatorService мок сервиса валидации.
type mockValidatorService struct {
	validateFn  func(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error)
	variablesFn func(ctx context.Context, tmplType model.TemplateType, channel model.TemplateChannel) []model.TemplateVariable
}

func (m *mockValidatorService) ValidateTemplate(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error) {
	return m.validateFn(ctx, tmpl)
}

func (m *mockValidatorService) GetAvailableVariables(ctx context.Context, tmplType model.TemplateType, channel model.TemplateChannel) []model.TemplateVariable {
	return m.variablesFn(ctx, tmplType, channel)
}

// --- Хелпер: создать шаблон ---

func sampleTemplate() *model.Template {
	return &model.Template{
		ID:          "test-id-0000-0000-0000-000000000001",
		UserID:      "user-1",
		Name:        "Test Template",
		Description: "desc",
		Channel:     model.ChannelEmail,
		Type:        model.TypeMonitorDown,
		Engine:      model.EngineGoTemplate,
		Subject:     "Alert: {{ .monitor_name }}",
		Body:        "Monitor {{ .monitor_name }} is {{ .status }}",
		Format:      model.FormatText,
		IsDefault:   false,
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
}

// --- Тесты New ---

func TestNew(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{}
	rendSvc := &mockRendererService{}
	valSvc := &mockValidatorService{}

	// Act
	h := New(tmplSvc, rendSvc, valSvc)

	// Assert
	if h == nil {
		t.Fatal("New() returned nil")
	}
}

// --- Тесты CreateTemplate ---

func TestCreateTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	tmpl := sampleTemplate()
	tmplSvc := &mockTemplateService{
		createFn: func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
			return tmpl, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.CreateTemplateRequest{
		UserId:  "user-1",
		Name:    "Test Template",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Engine:  v1.TemplateEngine_ENGINE_GOTEMPLATE,
		Subject: "Alert",
		Body:    "Body",
		Format:  "text",
	}

	// Act
	resp, err := h.CreateTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("CreateTemplate() unexpected error: %v", err)
	}
	if resp.Id != tmpl.ID {
		t.Errorf("CreateTemplate() ID = %q, want %q", resp.Id, tmpl.ID)
	}
	if resp.Name != tmpl.Name {
		t.Errorf("CreateTemplate() Name = %q, want %q", resp.Name, tmpl.Name)
	}
}

func TestCreateTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		createFn: func(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
			return nil, errors.New("create failed")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.CreateTemplateRequest{
		UserId:  "user-1",
		Name:    "Test",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "Body",
	}

	// Act
	_, err := h.CreateTemplate(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("CreateTemplate() expected error, got nil")
	}
}

// --- Тесты GetTemplate ---

func TestGetTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	tmpl := sampleTemplate()
	tmplSvc := &mockTemplateService{
		getFn: func(ctx context.Context, id string) (*model.Template, error) {
			return tmpl, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	resp, err := h.GetTemplate(context.Background(), &v1.GetTemplateRequest{Id: tmpl.ID})

	// Assert
	if err != nil {
		t.Fatalf("GetTemplate() unexpected error: %v", err)
	}
	if resp.Id != tmpl.ID {
		t.Errorf("GetTemplate() ID = %q, want %q", resp.Id, tmpl.ID)
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		getFn: func(ctx context.Context, id string) (*model.Template, error) {
			return nil, errors.New("not found")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.GetTemplate(context.Background(), &v1.GetTemplateRequest{Id: "missing"})

	// Assert
	if err == nil {
		t.Fatal("GetTemplate() expected error, got nil")
	}
}

// --- Тесты ListTemplates ---

func TestListTemplates(t *testing.T) {
	t.Parallel()

	// Arrange
	templates := []*model.Template{sampleTemplate(), sampleTemplate()}
	tmplSvc := &mockTemplateService{
		listFn: func(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error) {
			return templates, len(templates), nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.ListTemplatesRequest{
		UserId:   "user-1",
		Page:     1,
		PageSize: 10,
	}

	// Act
	resp, err := h.ListTemplates(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("ListTemplates() unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("ListTemplates() Total = %d, want 2", resp.Total)
	}
	if len(resp.Templates) != 2 {
		t.Errorf("ListTemplates() len(Templates) = %d, want 2", len(resp.Templates))
	}
}

func TestListTemplates_WithFilters(t *testing.T) {
	t.Parallel()

	// Arrange
	var capturedFilter service.ListFilter
	tmplSvc := &mockTemplateService{
		listFn: func(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error) {
			capturedFilter = filter
			return []*model.Template{}, 0, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	isDefault := true
	req := &v1.ListTemplatesRequest{
		UserId:    "user-1",
		Channel:   v1.TemplateChannel_CHANNEL_EMAIL,
		Type:      v1.TemplateType_TYPE_MONITOR_DOWN,
		IsDefault: isDefault,
		IsSystem:  false,
		Page:      2,
		PageSize:  20,
		SortBy:    "name",
		SortDesc:  true,
	}

	// Act
	_, err := h.ListTemplates(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("ListTemplates() unexpected error: %v", err)
	}
	if capturedFilter.UserID != "user-1" {
		t.Errorf("filter.UserID = %q, want %q", capturedFilter.UserID, "user-1")
	}
	if capturedFilter.IsDefault == nil || !*capturedFilter.IsDefault {
		t.Error("filter.IsDefault should be true")
	}
	if capturedFilter.SortDesc != true {
		t.Error("filter.SortDesc should be true")
	}
}

func TestListTemplates_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		listFn: func(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.ListTemplates(context.Background(), &v1.ListTemplatesRequest{})

	// Assert
	if err == nil {
		t.Fatal("ListTemplates() expected error, got nil")
	}
}

// --- Тесты UpdateTemplate ---

func TestUpdateTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	tmpl := sampleTemplate()
	tmpl.Name = "Updated"
	tmplSvc := &mockTemplateService{
		updateFn: func(ctx context.Context, id string, req *service.UpdateTemplateRequest) (*model.Template, error) {
			return tmpl, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.UpdateTemplateRequest{
		Id:   tmpl.ID,
		Name: "Updated",
		Body: "New body",
	}

	// Act
	resp, err := h.UpdateTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("UpdateTemplate() unexpected error: %v", err)
	}
	if resp.Name != "Updated" {
		t.Errorf("UpdateTemplate() Name = %q, want %q", resp.Name, "Updated")
	}
}

func TestUpdateTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		updateFn: func(ctx context.Context, id string, req *service.UpdateTemplateRequest) (*model.Template, error) {
			return nil, errors.New("update failed")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.UpdateTemplate(context.Background(), &v1.UpdateTemplateRequest{Id: "id"})

	// Assert
	if err == nil {
		t.Fatal("UpdateTemplate() expected error, got nil")
	}
}

// --- Тесты DeleteTemplate ---

func TestDeleteTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		deleteFn: func(ctx context.Context, id string) error {
			return nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	resp, err := h.DeleteTemplate(context.Background(), &v1.DeleteTemplateRequest{Id: "test-id"})

	// Assert
	if err != nil {
		t.Fatalf("DeleteTemplate() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("DeleteTemplate() returned nil response")
	}
}

func TestDeleteTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		deleteFn: func(ctx context.Context, id string) error {
			return errors.New("delete failed")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.DeleteTemplate(context.Background(), &v1.DeleteTemplateRequest{Id: "id"})

	// Assert
	if err == nil {
		t.Fatal("DeleteTemplate() expected error, got nil")
	}
}

// --- Тесты CloneTemplate ---

func TestCloneTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	clone := sampleTemplate()
	clone.Name = "Clone"
	parentID := "parent-id"
	clone.ParentID = &parentID
	tmplSvc := &mockTemplateService{
		cloneFn: func(ctx context.Context, id, userID, newName string) (*model.Template, error) {
			return clone, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.CloneTemplateRequest{
		Id:      "parent-id",
		UserId:  "user-1",
		NewName: "Clone",
	}

	// Act
	resp, err := h.CloneTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("CloneTemplate() unexpected error: %v", err)
	}
	if resp.Name != "Clone" {
		t.Errorf("CloneTemplate() Name = %q, want %q", resp.Name, "Clone")
	}
}

func TestCloneTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		cloneFn: func(ctx context.Context, id, userID, newName string) (*model.Template, error) {
			return nil, errors.New("clone failed")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.CloneTemplate(context.Background(), &v1.CloneTemplateRequest{Id: "id"})

	// Assert
	if err == nil {
		t.Fatal("CloneTemplate() expected error, got nil")
	}
}

// --- Тесты RenderTemplate ---

func TestRenderTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	rendered := &model.RenderedTemplate{
		Subject:  "Alert: API Check",
		Body:     "Monitor API Check is DOWN",
		Format:   model.FormatText,
		Success:  true,
		Errors:   nil,
		Warnings: nil,
	}
	rendSvc := &mockRendererService{
		renderFn: func(ctx context.Context, templateID string, variables model.VariablesMap) (*model.RenderedTemplate, error) {
			return rendered, nil
		},
	}
	h := New(&mockTemplateService{}, rendSvc, &mockValidatorService{})

	req := &v1.RenderTemplateRequest{
		TemplateId: "tmpl-1",
		Variables: map[string]string{
			"monitor_name": "API Check",
			"status":       "DOWN",
		},
	}

	// Act
	resp, err := h.RenderTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("RenderTemplate() unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("RenderTemplate() Success should be true")
	}
	if resp.Subject != "Alert: API Check" {
		t.Errorf("RenderTemplate() Subject = %q, want %q", resp.Subject, "Alert: API Check")
	}
}

func TestRenderTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	rendSvc := &mockRendererService{
		renderFn: func(ctx context.Context, templateID string, variables model.VariablesMap) (*model.RenderedTemplate, error) {
			return nil, errors.New("render failed")
		},
	}
	h := New(&mockTemplateService{}, rendSvc, &mockValidatorService{})

	// Act
	_, err := h.RenderTemplate(context.Background(), &v1.RenderTemplateRequest{TemplateId: "id"})

	// Assert
	if err == nil {
		t.Fatal("RenderTemplate() expected error, got nil")
	}
}

// --- Тесты ValidateTemplate ---

func TestValidateTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	result := &model.ValidationResult{
		Valid:            true,
		Errors:           nil,
		Warnings:         nil,
		UsedVariables:    []string{"monitor_name", "status"},
		MissingVariables: nil,
	}
	valSvc := &mockValidatorService{
		validateFn: func(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error) {
			return result, nil
		},
	}
	h := New(&mockTemplateService{}, &mockRendererService{}, valSvc)

	req := &v1.ValidateTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Engine:  v1.TemplateEngine_ENGINE_GOTEMPLATE,
		Subject: "{{ .monitor_name }} is down",
		Body:    "Status: {{ .status }}",
	}

	// Act
	resp, err := h.ValidateTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if !resp.Valid {
		t.Error("ValidateTemplate() should be valid")
	}
}

func TestValidateTemplate_Invalid(t *testing.T) {
	t.Parallel()

	// Arrange
	result := &model.ValidationResult{
		Valid:  false,
		Errors: []string{"syntax error"},
	}
	valSvc := &mockValidatorService{
		validateFn: func(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error) {
			return result, nil
		},
	}
	h := New(&mockTemplateService{}, &mockRendererService{}, valSvc)

	req := &v1.ValidateTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "{{ .status }",
	}

	// Act
	resp, err := h.ValidateTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("ValidateTemplate() unexpected error: %v", err)
	}
	if resp.Valid {
		t.Error("ValidateTemplate() should be invalid")
	}
	if len(resp.Errors) == 0 {
		t.Error("ValidateTemplate() Errors should not be empty")
	}
}

func TestValidateTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	valSvc := &mockValidatorService{
		validateFn: func(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error) {
			return nil, errors.New("validate failed")
		},
	}
	h := New(&mockTemplateService{}, &mockRendererService{}, valSvc)

	// Act
	_, err := h.ValidateTemplate(context.Background(), &v1.ValidateTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "body",
	})

	// Assert
	if err == nil {
		t.Fatal("ValidateTemplate() expected error, got nil")
	}
}

// --- Тесты GetAvailableVariables ---

func TestGetAvailableVariables(t *testing.T) {
	t.Parallel()

	// Arrange
	variables := []model.TemplateVariable{
		{Name: "monitor_name", Description: "Monitor name", Type: "string", Required: true},
		{Name: "status", Description: "Monitor status", Type: "string", Required: true},
	}
	valSvc := &mockValidatorService{
		variablesFn: func(ctx context.Context, tmplType model.TemplateType, channel model.TemplateChannel) []model.TemplateVariable {
			return variables
		},
	}
	h := New(&mockTemplateService{}, &mockRendererService{}, valSvc)

	req := &v1.GetAvailableVariablesRequest{
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
	}

	// Act
	resp, err := h.GetAvailableVariables(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("GetAvailableVariables() unexpected error: %v", err)
	}
	if len(resp.Variables) != 2 {
		t.Errorf("GetAvailableVariables() len(Variables) = %d, want 2", len(resp.Variables))
	}
	if resp.Variables[0].Name != "monitor_name" {
		t.Errorf("GetAvailableVariables() Variables[0].Name = %q, want %q", resp.Variables[0].Name, "monitor_name")
	}
}

func TestGetAvailableVariables_Empty(t *testing.T) {
	t.Parallel()

	// Arrange
	valSvc := &mockValidatorService{
		variablesFn: func(ctx context.Context, tmplType model.TemplateType, channel model.TemplateChannel) []model.TemplateVariable {
			return []model.TemplateVariable{}
		},
	}
	h := New(&mockTemplateService{}, &mockRendererService{}, valSvc)

	req := &v1.GetAvailableVariablesRequest{
		Type:    v1.TemplateType_TYPE_CUSTOM,
		Channel: v1.TemplateChannel_CHANNEL_SMS,
	}

	// Act
	resp, err := h.GetAvailableVariables(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("GetAvailableVariables() unexpected error: %v", err)
	}
	if len(resp.Variables) != 0 {
		t.Errorf("GetAvailableVariables() expected empty, got %d", len(resp.Variables))
	}
}

// --- Тесты SetDefaultTemplate ---

func TestSetDefaultTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	tmpl := sampleTemplate()
	tmpl.IsDefault = true
	tmplSvc := &mockTemplateService{
		setDefaultFn: func(ctx context.Context, id, userID string) (*model.Template, error) {
			return tmpl, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.SetDefaultTemplateRequest{
		TemplateId: tmpl.ID,
		UserId:     "user-1",
	}

	// Act
	resp, err := h.SetDefaultTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("SetDefaultTemplate() unexpected error: %v", err)
	}
	if !resp.IsDefault {
		t.Error("SetDefaultTemplate() IsDefault should be true")
	}
}

func TestSetDefaultTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		setDefaultFn: func(ctx context.Context, id, userID string) (*model.Template, error) {
			return nil, errors.New("set default failed")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.SetDefaultTemplate(context.Background(), &v1.SetDefaultTemplateRequest{TemplateId: "id"})

	// Assert
	if err == nil {
		t.Fatal("SetDefaultTemplate() expected error, got nil")
	}
}

// --- Тесты GetDefaultTemplate ---

func TestGetDefaultTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	tmpl := sampleTemplate()
	tmpl.IsDefault = true
	tmplSvc := &mockTemplateService{
		getDefaultFn: func(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error) {
			return tmpl, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	req := &v1.GetDefaultTemplateRequest{
		UserId:  "user-1",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
	}

	// Act
	resp, err := h.GetDefaultTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("GetDefaultTemplate() unexpected error: %v", err)
	}
	if !resp.IsDefault {
		t.Error("GetDefaultTemplate() IsDefault should be true")
	}
}

func TestGetDefaultTemplate_NotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	tmplSvc := &mockTemplateService{
		getDefaultFn: func(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error) {
			return nil, errors.New("not found")
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	_, err := h.GetDefaultTemplate(context.Background(), &v1.GetDefaultTemplateRequest{
		UserId:  "user-1",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
	})

	// Assert
	if err == nil {
		t.Fatal("GetDefaultTemplate() expected error, got nil")
	}
}

// --- Тесты PreviewTemplate ---

func TestPreviewTemplate(t *testing.T) {
	t.Parallel()

	// Arrange
	rendered := &model.RenderedTemplate{
		Subject:  "Monitor DOWN is down",
		Body:     "Status: DOWN",
		Format:   model.FormatText,
		Success:  true,
		Errors:   nil,
		Warnings: nil,
	}
	rendSvc := &mockRendererService{
		previewFn: func(ctx context.Context, tmpl *model.Template) (*model.RenderedTemplate, error) {
			return rendered, nil
		},
	}
	h := New(&mockTemplateService{}, rendSvc, &mockValidatorService{})

	req := &v1.PreviewTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Engine:  v1.TemplateEngine_ENGINE_GOTEMPLATE,
		Subject: "{{ .monitor_name }} is down",
		Body:    "Status: {{ .status }}",
		Format:  "text",
	}

	// Act
	resp, err := h.PreviewTemplate(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("PreviewTemplate() unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("PreviewTemplate() Success should be true")
	}
	if resp.Subject == "" {
		t.Error("PreviewTemplate() Subject should not be empty")
	}
}

func TestPreviewTemplate_ServiceError(t *testing.T) {
	t.Parallel()

	// Arrange
	rendSvc := &mockRendererService{
		previewFn: func(ctx context.Context, tmpl *model.Template) (*model.RenderedTemplate, error) {
			return nil, errors.New("preview failed")
		},
	}
	h := New(&mockTemplateService{}, rendSvc, &mockValidatorService{})

	// Act
	_, err := h.PreviewTemplate(context.Background(), &v1.PreviewTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
	})

	// Assert
	if err == nil {
		t.Fatal("PreviewTemplate() expected error, got nil")
	}
}

// --- Тесты templateToProto (через публичный метод) ---

func TestTemplateToProto_ZeroTimes(t *testing.T) {
	t.Parallel()

	// Arrange — шаблон с нулевыми временем
	tmpl := &model.Template{
		ID:   "test-id-0000-0000-0000-000000000001",
		Name: "Test",
		Body: "body",
	}
	tmplSvc := &mockTemplateService{
		getFn: func(ctx context.Context, id string) (*model.Template, error) {
			return tmpl, nil
		},
	}
	h := New(tmplSvc, &mockRendererService{}, &mockValidatorService{})

	// Act
	resp, err := h.GetTemplate(context.Background(), &v1.GetTemplateRequest{Id: tmpl.ID})

	// Assert
	if err != nil {
		t.Fatalf("GetTemplate() unexpected error: %v", err)
	}
	if resp.CreatedAt != nil {
		t.Error("templateToProto() CreatedAt should be nil for zero time")
	}
	if resp.UpdatedAt != nil {
		t.Error("templateToProto() UpdatedAt should be nil for zero time")
	}
}
