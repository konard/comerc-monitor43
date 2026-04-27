package service

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

func newTemplateSvc(repo *MockTemplateRepository) *TemplateServiceImpl {
	l := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 100000}
	return NewTemplateService(repo, &l, cfg)
}

// --- GetTemplate ---

func TestTemplateService_GetTemplate_Success(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-1"] = &model.Template{
		ID:   "tmpl-1",
		Name: "Test",
		Body: "body",
	}

	svc := newTemplateSvc(repo)
	tmpl, err := svc.GetTemplate(context.Background(), "tmpl-1")
	if err != nil {
		t.Fatalf("GetTemplate() unexpected error: %v", err)
	}
	if tmpl.ID != "tmpl-1" {
		t.Errorf("GetTemplate() ID = %q, want %q", tmpl.ID, "tmpl-1")
	}
}

func TestTemplateService_GetTemplate_NotFound(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := newTemplateSvc(repo)

	_, err := svc.GetTemplate(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("GetTemplate() expected error for missing template, got nil")
	}
}

func TestTemplateService_GetTemplate_RepoError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.getErr = errors.New("db error")
	svc := newTemplateSvc(repo)

	_, err := svc.GetTemplate(context.Background(), "any-id")
	if err == nil {
		t.Fatal("GetTemplate() expected error when repo fails, got nil")
	}
}

// --- ListTemplates ---

func TestTemplateService_ListTemplates_Empty(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := newTemplateSvc(repo)

	templates, total, err := svc.ListTemplates(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("ListTemplates() unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("ListTemplates() total = %d, want 0", total)
	}
	if len(templates) != 0 {
		t.Errorf("ListTemplates() len = %d, want 0", len(templates))
	}
}

func TestTemplateService_ListTemplates_WithItems(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["a"] = &model.Template{ID: "a", Name: "A", Body: "body"}
	repo.templates["b"] = &model.Template{ID: "b", Name: "B", Body: "body"}

	svc := newTemplateSvc(repo)

	templates, total, err := svc.ListTemplates(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("ListTemplates() unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("ListTemplates() total = %d, want 2", total)
	}
	if len(templates) != 2 {
		t.Errorf("ListTemplates() len = %d, want 2", len(templates))
	}
}

// --- UpdateTemplate ---

func TestTemplateService_UpdateTemplate_Success(t *testing.T) {
	repo := NewMockTemplateRepository()
	// Use valid UUID so Validate() passes on ID field
	const validID = "123e4567-e89b-12d3-a456-426614174000"
	repo.templates[validID] = &model.Template{
		ID:      validID,
		Name:    "Original",
		Body:    "Original body {{ .monitor_name }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatHTML,
	}

	svc := newTemplateSvc(repo)
	updated, err := svc.UpdateTemplate(context.Background(), validID, &UpdateTemplateRequest{
		Name: "Updated",
		Body: "Updated body {{ .monitor_name }}",
	})
	if err != nil {
		t.Fatalf("UpdateTemplate() unexpected error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("UpdateTemplate() Name = %q, want %q", updated.Name, "Updated")
	}
}

func TestTemplateService_UpdateTemplate_NotFound(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := newTemplateSvc(repo)

	_, err := svc.UpdateTemplate(context.Background(), "missing", &UpdateTemplateRequest{
		Name: "New Name",
		Body: "body",
	})
	if err == nil {
		t.Fatal("UpdateTemplate() expected error for missing template, got nil")
	}
}

func TestTemplateService_UpdateTemplate_SystemTemplate(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["sys-1"] = &model.Template{
		ID:       "sys-1",
		Name:     "System Template",
		Body:     "body",
		IsSystem: true,
		Channel:  model.ChannelEmail,
		Type:     model.TypeMonitorDown,
	}

	svc := newTemplateSvc(repo)
	_, err := svc.UpdateTemplate(context.Background(), "sys-1", &UpdateTemplateRequest{
		Name: "New Name",
		Body: "body",
	})
	if err == nil {
		t.Fatal("UpdateTemplate() expected error for system template, got nil")
	}
	if !containsString(err.Error(), "system") {
		t.Errorf("UpdateTemplate() error = %v, should contain 'system'", err)
	}
}

func TestTemplateService_UpdateTemplate_ValidationFails(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-val"] = &model.Template{
		ID:      "tmpl-val",
		Name:    "Valid Template",
		Body:    "body",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	svc := newTemplateSvc(repo)
	_, err := svc.UpdateTemplate(context.Background(), "tmpl-val", &UpdateTemplateRequest{
		Name: "", // empty name fails validation
		Body: "body",
	})
	if err == nil {
		t.Fatal("UpdateTemplate() expected error for empty name, got nil")
	}
}

func TestTemplateService_UpdateTemplate_BodyTooLarge(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-big"] = &model.Template{
		ID:      "tmpl-big",
		Name:    "Valid Template",
		Body:    "body",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	l := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 10} // very small limit
	svc := NewTemplateService(repo, &l, cfg)

	_, err := svc.UpdateTemplate(context.Background(), "tmpl-big", &UpdateTemplateRequest{
		Name: "Test",
		Body: "this body is way too long for limit of 10 bytes",
	})
	if err == nil {
		t.Fatal("UpdateTemplate() expected error for oversized body, got nil")
	}
}

func TestTemplateService_UpdateTemplate_RepoUpdateError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-err"] = &model.Template{
		ID:      "tmpl-err",
		Name:    "Test",
		Body:    "body",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}
	repo.updateErr = errors.New("update failed")

	svc := newTemplateSvc(repo)
	_, err := svc.UpdateTemplate(context.Background(), "tmpl-err", &UpdateTemplateRequest{
		Name: "Updated",
		Body: "new body",
	})
	if err == nil {
		t.Fatal("UpdateTemplate() expected error when repo update fails, got nil")
	}
}

// --- DeleteTemplate ---

func TestTemplateService_DeleteTemplate_RepoDeleteError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-del"] = &model.Template{
		ID:       "tmpl-del",
		Name:     "Test",
		Body:     "body",
		IsSystem: false,
	}
	repo.deleteErr = errors.New("delete failed")

	svc := newTemplateSvc(repo)
	err := svc.DeleteTemplate(context.Background(), "tmpl-del")
	if err == nil {
		t.Fatal("DeleteTemplate() expected error when repo delete fails, got nil")
	}
}

// --- CloneTemplate ---

func TestTemplateService_CloneTemplate_NotFound(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := newTemplateSvc(repo)

	_, err := svc.CloneTemplate(context.Background(), "missing-id", "user-1", "Clone")
	if err == nil {
		t.Fatal("CloneTemplate() expected error for missing template, got nil")
	}
}

func TestTemplateService_CloneTemplate_RepoCreateError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["orig-id"] = &model.Template{
		ID:      "orig-id",
		Name:    "Original",
		Body:    "body {{ .monitor_name }}",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Format:  model.FormatHTML,
	}
	repo.createErr = errors.New("create failed")

	svc := newTemplateSvc(repo)
	_, err := svc.CloneTemplate(context.Background(), "orig-id", "user-1", "Clone")
	if err == nil {
		t.Fatal("CloneTemplate() expected error when repo create fails, got nil")
	}
}

// --- CreateTemplate body size ---

func TestTemplateService_CreateTemplate_BodyTooLarge(t *testing.T) {
	repo := NewMockTemplateRepository()
	l := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 10}
	svc := NewTemplateService(repo, &l, cfg)

	_, err := svc.CreateTemplate(context.Background(), &CreateTemplateRequest{
		UserID:  "user-1",
		Name:    "Test",
		Body:    "this body is way too long for max size of 10",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	})
	if err == nil {
		t.Fatal("CreateTemplate() expected error for oversized body, got nil")
	}
}

// --- SetDefaultTemplate ---

func TestTemplateService_SetDefaultTemplate_Success(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-def"] = &model.Template{
		ID:      "tmpl-def",
		Name:    "Test",
		Body:    "body",
		UserID:  "user-1",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	svc := newTemplateSvc(repo)
	tmpl, err := svc.SetDefaultTemplate(context.Background(), "tmpl-def", "user-1")
	if err != nil {
		t.Fatalf("SetDefaultTemplate() unexpected error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("SetDefaultTemplate() returned nil template")
	}
}

func TestTemplateService_SetDefaultTemplate_NotFound(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := newTemplateSvc(repo)

	_, err := svc.SetDefaultTemplate(context.Background(), "missing-id", "user-1")
	if err == nil {
		t.Fatal("SetDefaultTemplate() expected error for missing template, got nil")
	}
}

func TestTemplateService_SetDefaultTemplate_WrongUser(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-owned"] = &model.Template{
		ID:     "tmpl-owned",
		Name:   "Test",
		Body:   "body",
		UserID: "owner-user",
	}

	svc := newTemplateSvc(repo)
	_, err := svc.SetDefaultTemplate(context.Background(), "tmpl-owned", "other-user")
	if err == nil {
		t.Fatal("SetDefaultTemplate() expected error for wrong user, got nil")
	}
	if !containsString(err.Error(), "belong") {
		t.Errorf("SetDefaultTemplate() error = %v, should mention 'belong'", err)
	}
}

func TestTemplateService_SetDefaultTemplate_RepoError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["tmpl-sd"] = &model.Template{
		ID:     "tmpl-sd",
		Name:   "Test",
		Body:   "body",
		UserID: "user-1",
	}

	// We inject a setDefaultErr by replacing with a custom mock
	errRepo := &errSetDefaultMockRepo{MockTemplateRepository: repo, setDefaultErr: errors.New("set default failed")}
	l := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 100000}
	svcErr := NewTemplateService(errRepo, &l, cfg)

	_, err := svcErr.SetDefaultTemplate(context.Background(), "tmpl-sd", "user-1")
	if err == nil {
		t.Fatal("SetDefaultTemplate() expected error when repo fails, got nil")
	}
}

// errSetDefaultMockRepo wraps MockTemplateRepository to inject a SetDefault error.
type errSetDefaultMockRepo struct {
	*MockTemplateRepository
	setDefaultErr error
}

func (m *errSetDefaultMockRepo) SetDefault(ctx context.Context, id, userID string) error {
	return m.setDefaultErr
}

// --- GetDefaultTemplate ---

func TestTemplateService_GetDefaultTemplate_Success(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.templates["def-tmpl"] = &model.Template{
		ID:        "def-tmpl",
		Name:      "Default",
		Body:      "body",
		UserID:    "user-1",
		Channel:   model.ChannelEmail,
		Type:      model.TypeMonitorDown,
		IsDefault: true,
	}

	svc := newTemplateSvc(repo)
	tmpl, err := svc.GetDefaultTemplate(context.Background(), "user-1", model.ChannelEmail, model.TypeMonitorDown)
	if err != nil {
		t.Fatalf("GetDefaultTemplate() unexpected error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("GetDefaultTemplate() returned nil")
	}
	if tmpl.ID != "def-tmpl" {
		t.Errorf("GetDefaultTemplate() ID = %q, want %q", tmpl.ID, "def-tmpl")
	}
}

func TestTemplateService_GetDefaultTemplate_NotFound(t *testing.T) {
	repo := NewMockTemplateRepository()
	svc := newTemplateSvc(repo)

	_, err := svc.GetDefaultTemplate(context.Background(), "user-1", model.ChannelEmail, model.TypeMonitorDown)
	if err == nil {
		t.Fatal("GetDefaultTemplate() expected error when no default exists, got nil")
	}
}

func TestTemplateService_GetDefaultTemplate_RepoError(t *testing.T) {
	repo := NewMockTemplateRepository()
	repo.getErr = errors.New("db error")
	svc := newTemplateSvc(repo)

	_, err := svc.GetDefaultTemplate(context.Background(), "user-1", model.ChannelEmail, model.TypeMonitorDown)
	if err == nil {
		t.Fatal("GetDefaultTemplate() expected error when repo fails, got nil")
	}
}

// errListMockRepo оборачивает MockTemplateRepository для инъекции ошибки в List.
type errListMockRepo struct {
	*MockTemplateRepository
	listErr error
}

func (m *errListMockRepo) List(ctx context.Context, filter ListFilter) ([]*model.Template, int, error) {
	return nil, 0, m.listErr
}

// TestTemplateService_ListTemplates_RepoError проверяет обработку ошибки репозитория при листинге.
func TestTemplateService_ListTemplates_RepoError(t *testing.T) {
	repo := NewMockTemplateRepository()
	errRepo := &errListMockRepo{MockTemplateRepository: repo, listErr: errors.New("list failed")}

	l := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 100000}
	svc := NewTemplateService(errRepo, &l, cfg)

	_, _, err := svc.ListTemplates(context.Background(), ListFilter{})
	if err == nil {
		t.Fatal("ListTemplates() expected error when repo fails, got nil")
	}
}
