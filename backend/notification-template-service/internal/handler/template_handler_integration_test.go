package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	v1 "github.com/raul/monitor/api/proto"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/logging"
	"github.com/raul/monitor/backend/notification-template-service/internal/repository"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// setupTestDB создаёт тестовую БД для интеграционных тестов handler.
func setupTestDBForHandler(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test_templates",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("skipping: docker unavailable: %v", err)
	}
	terminateTestContainer(t, container)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=test_templates sslmode=disable",
		host, port.Port())

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	closeTestDB(t, db)

	// Применяем миграции
	if err := runMigrationsForHandler(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// runMigrationsForHandler применяет миграции.
func runMigrationsForHandler(db *sqlx.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS templates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			channel VARCHAR(50) NOT NULL CHECK (channel IN ('email', 'telegram', 'webhook', 'slack', 'discord', 'sms')),
			type VARCHAR(50) NOT NULL CHECK (type IN ('monitor_up', 'monitor_down', 'monitor_degraded', 'certificate_expiry', 'flapping_detected', 'incident_created', 'incident_resolved', 'maintenance_started', 'maintenance_ended', 'custom')),
			engine VARCHAR(50) NOT NULL DEFAULT 'gotemplate' CHECK (engine IN ('gotemplate', 'jinja2', 'handlebars')),
			subject TEXT,
			body TEXT NOT NULL,
			format VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (format IN ('text', 'html', 'markdown', 'json')),
			is_default BOOLEAN NOT NULL DEFAULT false,
			is_system BOOLEAN NOT NULL DEFAULT false,
			version INT NOT NULL DEFAULT 1,
			parent_id UUID,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

			CONSTRAINT templates_user_id_name_channel_type_unique UNIQUE (user_id, name, channel, type)
		);

		CREATE INDEX idx_templates_user_id ON templates(user_id);
		CREATE INDEX idx_templates_channel_type ON templates(channel, type);
	`

	_, err := db.Exec(schema)
	return err
}

// setupHandler создаёт handler для тестов.
func setupHandler(t *testing.T) (*Handler, *sqlx.DB) {
	t.Helper()

	db := setupTestDBForHandler(t)

	logger := logging.New("info", "test")
	cfg := &config.Config{MaxTemplateSize: 100000}

	repo := repository.NewPostgresTemplateRepository(db)
	templateSvc := service.NewTemplateService(repo, logger.ZLogger(), cfg)
	rendererSvc := service.NewRendererService(repo, logger.ZLogger(), cfg)
	validatorSvc := service.NewValidatorService(logger.ZLogger())

	handler := New(templateSvc, rendererSvc, validatorSvc)

	return handler, db
}

func TestHandler_CreateTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	req := &v1.CreateTemplateRequest{
		UserId:      "00000000-0000-0000-0000-000000000001",
		Name:        "Test Template",
		Description: "Test description",
		Channel:     v1.TemplateChannel_CHANNEL_EMAIL,
		Type:        v1.TemplateType_TYPE_MONITOR_DOWN,
		Subject:     "Test Subject",
		Body:        "Monitor {{ .monitor_name }} is {{ .status }}",
		Format:      "text",
	}

	resp, err := handler.CreateTemplate(ctx, req)
	if err != nil {
		t.Fatalf("CreateTemplate() error = %v", err)
	}

	if resp.Name != req.Name {
		t.Errorf("CreateTemplate() name = %v, want %v", resp.Name, req.Name)
	}

	if resp.Subject != req.Subject {
		t.Errorf("CreateTemplate() subject = %v, want %v", resp.Subject, req.Subject)
	}

	if resp.Body != req.Body {
		t.Errorf("CreateTemplate() body = %v, want %v", resp.Body, req.Body)
	}

	if resp.Id == "" {
		t.Error("CreateTemplate() should return template ID")
	}
}

func TestHandler_GetTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	// Сначала создаём шаблон
	createReq := &v1.CreateTemplateRequest{
		UserId:  "00000000-0000-0000-0000-000000000001",
		Name:    "Test Template",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "Test body",
	}

	created, err := handler.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Получаем шаблон
	getReq := &v1.GetTemplateRequest{Id: created.Id}
	got, err := handler.GetTemplate(ctx, getReq)
	if err != nil {
		t.Fatalf("GetTemplate() error = %v", err)
	}

	if got.Name != created.Name {
		t.Errorf("GetTemplate() name = %v, want %v", got.Name, created.Name)
	}

	if got.Id != created.Id {
		t.Errorf("GetTemplate() id = %v, want %v", got.Id, created.Id)
	}
}

func TestHandler_UpdateTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	// Создаём шаблон
	createReq := &v1.CreateTemplateRequest{
		UserId:  "00000000-0000-0000-0000-000000000001",
		Name:    "Original Name",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "Original body",
	}

	created, err := handler.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Обновляем
	updateReq := &v1.UpdateTemplateRequest{
		Id:          created.Id,
		Name:        "Updated Name",
		Description: "Updated description",
		Body:        "Updated body",
		Format:      "html",
	}

	updated, err := handler.UpdateTemplate(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateTemplate() error = %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("UpdateTemplate() name = %v, want %v", updated.Name, "Updated Name")
	}

	if updated.Body != "Updated body" {
		t.Errorf("UpdateTemplate() body = %v, want %v", updated.Body, "Updated body")
	}
}

func TestHandler_DeleteTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	// Создаём шаблон
	createReq := &v1.CreateTemplateRequest{
		UserId:  "00000000-0000-0000-0000-000000000001",
		Name:    "To Delete",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "Body",
	}

	created, err := handler.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Удаляем
	deleteReq := &v1.DeleteTemplateRequest{Id: created.Id}
	_, err = handler.DeleteTemplate(ctx, deleteReq)
	if err != nil {
		t.Fatalf("DeleteTemplate() error = %v", err)
	}

	// Проверяем, что удалён
	getReq := &v1.GetTemplateRequest{Id: created.Id}
	_, err = handler.GetTemplate(ctx, getReq)
	if err == nil {
		t.Error("after DeleteTemplate(), GetTemplate() should return error")
	}
}

func TestHandler_ListTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000001"

	// Создаём несколько шаблонов
	for i := 0; i < 3; i++ {
		createReq := &v1.CreateTemplateRequest{
			UserId:  userID,
			Name:    fmt.Sprintf("Template %d", i),
			Channel: v1.TemplateChannel_CHANNEL_EMAIL,
			Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
			Body:    fmt.Sprintf("Body %d", i),
		}
		if _, err := handler.CreateTemplate(ctx, createReq); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	// Получаем список
	listReq := &v1.ListTemplatesRequest{
		UserId:   userID,
		Page:     1,
		PageSize: 10,
	}

	resp, err := handler.ListTemplates(ctx, listReq)
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("ListTemplates() total = %v, want %v", resp.Total, 3)
	}

	if len(resp.Templates) != 3 {
		t.Errorf("ListTemplates() len = %v, want %v", len(resp.Templates), 3)
	}
}

func TestHandler_RenderTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	// Создаём шаблон
	createReq := &v1.CreateTemplateRequest{
		UserId:  "00000000-0000-0000-0000-000000000001",
		Name:    "Render Test",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Subject: "Alert: {{ .monitor_name }}",
		Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
	}

	created, err := handler.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Рендерим
	renderReq := &v1.RenderTemplateRequest{
		TemplateId: created.Id,
		Variables: map[string]string{
			"monitor_name": "API Check",
			"status":       "DOWN",
		},
	}

	rendered, err := handler.RenderTemplate(ctx, renderReq)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	if !rendered.Success {
		t.Errorf("RenderTemplate() success = %v, want true", rendered.Success)
	}

	if rendered.Subject != "Alert: API Check" {
		t.Errorf("RenderTemplate() subject = %v, want %v", rendered.Subject, "Alert: API Check")
	}

	if rendered.Body != "Monitor API Check is DOWN" {
		t.Errorf("RenderTemplate() body = %v, want %v", rendered.Body, "Monitor API Check is DOWN")
	}
}

func TestHandler_ValidateTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	// Валидный шаблон
	validReq := &v1.ValidateTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Engine:  v1.TemplateEngine_ENGINE_GOTEMPLATE,
		Subject: "{{ .monitor_name }} is down",
		Body:    "Status: {{ .status }}",
	}

	result, err := handler.ValidateTemplate(ctx, validReq)
	if err != nil {
		t.Fatalf("ValidateTemplate() error = %v", err)
	}

	if !result.Valid {
		t.Errorf("ValidateTemplate() valid = %v, want true", result.Valid)
	}

	if len(result.Errors) > 0 {
		t.Errorf("ValidateTemplate() errors = %v, want empty", result.Errors)
	}

	// Невалидный шаблон (синтаксическая ошибка)
	invalidReq := &v1.ValidateTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Engine:  v1.TemplateEngine_ENGINE_GOTEMPLATE,
		Body:    "Status: {{ .status }", // незакрытый bracket
	}

	result, err = handler.ValidateTemplate(ctx, invalidReq)
	if err != nil {
		t.Fatalf("ValidateTemplate() error = %v", err)
	}

	if result.Valid {
		t.Error("ValidateTemplate() should be invalid for syntax error")
	}

	if len(result.Errors) == 0 {
		t.Error("ValidateTemplate() should have errors for invalid syntax")
	}
}

func TestHandler_PreviewTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	req := &v1.PreviewTemplateRequest{
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Engine:  v1.TemplateEngine_ENGINE_GOTEMPLATE,
		Subject: "{{ .monitor_name }} is down",
		Body:    "Status: {{ .status }}",
		Format:  "text",
	}

	result, err := handler.PreviewTemplate(ctx, req)
	if err != nil {
		t.Fatalf("PreviewTemplate() error = %v", err)
	}

	if !result.Success {
		t.Errorf("PreviewTemplate() success = %v, want true", result.Success)
	}

	// Проверяем, что подставились sample данные
	if result.Subject == "" {
		t.Error("PreviewTemplate() subject should not be empty")
	}

	if result.Body == "" {
		t.Error("PreviewTemplate() body should not be empty")
	}
}

func TestHandler_CloneTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, _ := setupHandler(t)

	ctx := context.Background()

	// Создаём оригинальный шаблон
	createReq := &v1.CreateTemplateRequest{
		UserId:  "00000000-0000-0000-0000-000000000001",
		Name:    "Original",
		Channel: v1.TemplateChannel_CHANNEL_EMAIL,
		Type:    v1.TemplateType_TYPE_MONITOR_DOWN,
		Body:    "Original body",
	}

	original, err := handler.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Клонируем
	cloneReq := &v1.CloneTemplateRequest{
		Id:      original.Id,
		UserId:  "00000000-0000-0000-0000-000000000001",
		NewName: "Clone",
	}

	clone, err := handler.CloneTemplate(ctx, cloneReq)
	if err != nil {
		t.Fatalf("CloneTemplate() error = %v", err)
	}

	if clone.Name != "Clone" {
		t.Errorf("CloneTemplate() name = %v, want %v", clone.Name, "Clone")
	}

	if clone.ParentId != original.Id {
		t.Errorf("CloneTemplate() parentId = %v, want %v", clone.ParentId, original.Id)
	}

	if clone.IsSystem {
		t.Error("CloneTemplate() clone should not be system")
	}
}
