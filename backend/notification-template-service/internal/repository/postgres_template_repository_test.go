package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// setupTestDB создаёт testcontainer с PostgreSQL для тестов.
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := context.Background()

	// Запускаем PostgreSQL контейнер
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

	// Получаем хост и порт
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Подключаемся к БД
	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=test_templates sslmode=disable",
		host, port.Port())

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	closeTestDB(t, db)

	// Применяем миграции
	if err := runMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// runMigrations применяет миграции к тестовой БД.
func runMigrations(db *sqlx.DB) error {
	// Создаём таблицу templates
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

func TestPostgresTemplateRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	tmpl := &model.Template{
		UserID:      "00000000-0000-0000-0000-000000000001",
		Name:        "Test Template",
		Description: "Test description",
		Channel:     model.ChannelEmail,
		Type:        model.TypeMonitorDown,
		Subject:     "Test Subject",
		Body:        "Monitor {{ .monitor_name }} is {{ .status }}",
		Format:      model.FormatText,
	}

	err := repo.Create(ctx, tmpl)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if tmpl.ID == "" {
		t.Error("Create() should set template ID")
	}

	if tmpl.CreatedAt.IsZero() {
		t.Error("Create() should set created_at")
	}

	if tmpl.UpdatedAt.IsZero() {
		t.Error("Create() should set updated_at")
	}
}

func TestPostgresTemplateRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	// Создаём шаблон
	created := &model.Template{
		UserID:  "00000000-0000-0000-0000-000000000001",
		Name:    "Test Template",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Body:    "Test body",
	}

	if err := repo.Create(ctx, created); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Получаем шаблон
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != created.Name {
		t.Errorf("GetByID() name = %v, want %v", got.Name, created.Name)
	}

	if got.Body != created.Body {
		t.Errorf("GetByID() body = %v, want %v", got.Body, created.Body)
	}
}

func TestPostgresTemplateRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if err != ErrTemplateNotFound {
		t.Errorf("GetByID() error = %v, want %v", err, ErrTemplateNotFound)
	}
}

func TestPostgresTemplateRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	// Создаём несколько шаблонов
	userID := "00000000-0000-0000-0000-000000000001"
	for i := 0; i < 3; i++ {
		tmpl := &model.Template{
			UserID:  userID,
			Name:    fmt.Sprintf("Template %d", i),
			Channel: model.ChannelEmail,
			Type:    model.TypeMonitorDown,
			Body:    fmt.Sprintf("Body %d", i),
		}
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	// Получаем список
	templates, total, err := repo.List(ctx, service.ListFilter{
		UserID:   userID,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if total != 3 {
		t.Errorf("List() total = %v, want %v", total, 3)
	}

	if len(templates) != 3 {
		t.Errorf("List() len = %v, want %v", len(templates), 3)
	}
}

func TestPostgresTemplateRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	// Создаём шаблон
	created := &model.Template{
		UserID:  "00000000-0000-0000-0000-000000000001",
		Name:    "Original Name",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Body:    "Original body",
	}

	if err := repo.Create(ctx, created); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Обновляем
	created.Name = "Updated Name"
	created.Body = "Updated body"
	created.Version = 2

	if err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Проверяем обновление
	updated, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Update() name = %v, want %v", updated.Name, "Updated Name")
	}

	if updated.Body != "Updated body" {
		t.Errorf("Update() body = %v, want %v", updated.Body, "Updated body")
	}
}

func TestPostgresTemplateRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	// Создаём шаблон
	created := &model.Template{
		UserID:  "00000000-0000-0000-0000-000000000001",
		Name:    "To Delete",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Body:    "Body",
	}

	if err := repo.Create(ctx, created); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Удаляем
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Проверяем, что удалён
	_, err := repo.GetByID(ctx, created.ID)
	if err != ErrTemplateNotFound {
		t.Errorf("after Delete(), GetByID() error = %v, want %v", err, ErrTemplateNotFound)
	}
}

func TestPostgresTemplateRepository_SetDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000001"
	channel := model.ChannelEmail
	tmplType := model.TypeMonitorDown

	// Создаём два шаблона
	tmpl1 := &model.Template{
		UserID:    userID,
		Name:      "Template 1",
		Channel:   channel,
		Type:      tmplType,
		Body:      "Body 1",
		IsDefault: true,
	}

	tmpl2 := &model.Template{
		UserID:    userID,
		Name:      "Template 2",
		Channel:   channel,
		Type:      tmplType,
		Body:      "Body 2",
		IsDefault: false,
	}

	if err := repo.Create(ctx, tmpl1); err != nil {
		t.Fatalf("failed to create template 1: %v", err)
	}

	if err := repo.Create(ctx, tmpl2); err != nil {
		t.Fatalf("failed to create template 2: %v", err)
	}

	// Устанавливаем второй как дефолтный
	if err := repo.SetDefault(ctx, tmpl2.ID, userID); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}

	// Проверяем, что второй стал дефолтным
	defaultTmpl, err := repo.GetDefault(ctx, userID, channel, tmplType)
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}

	if defaultTmpl.ID != tmpl2.ID {
		t.Errorf("GetDefault() id = %v, want %v", defaultTmpl.ID, tmpl2.ID)
	}

	if !defaultTmpl.IsDefault {
		t.Error("GetDefault() is_default should be true")
	}
}

func TestPostgresTemplateRepository_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	repo := NewPostgresTemplateRepository(db)
	ctx := context.Background()

	// Создаём шаблон
	created := &model.Template{
		UserID:  "00000000-0000-0000-0000-000000000001",
		Name:    "Test",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Body:    "Body",
	}

	if err := repo.Create(ctx, created); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Проверяем существование
	exists, err := repo.Exists(ctx, created.ID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Exists() should return true for existing template")
	}

	// Проверяем несуществующий
	exists, err = repo.Exists(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if exists {
		t.Error("Exists() should return false for non-existing template")
	}
}
