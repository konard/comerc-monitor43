package repository

import (
	"context"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// TestPostgresTemplateRepository_TransactionCommit проверяет commit транзакции.
func TestPostgresTemplateRepository_TransactionCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	ctx := context.Background()
	repo := NewPostgresTemplateRepository(db)

	// Начинаем транзакцию
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Создаём шаблон в транзакции
	tmpl := &model.Template{
		UserID:    "00000000-0000-0000-0000-000000000002",
		Name:      "Transaction Test",
		Channel:   model.ChannelEmail,
		Type:      model.TypeMonitorDown,
		Body:      "Test body",
		IsDefault: true,
	}

	// Создаём напрямую через SQL в транзакции
	_, err = tx.Exec(`
		INSERT INTO templates (user_id, name, channel, type, body, is_default, is_system, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id
	`, tmpl.UserID, tmpl.Name, tmpl.Channel, tmpl.Type, tmpl.Body, tmpl.IsDefault, false, 1)
	if err != nil {
		require.NoError(t, tx.Rollback())
		t.Fatalf("failed to insert template: %v", err)
	}

	// Commit транзакции
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// Проверяем, что данные сохранены
	// Получаем через репозиторий
	templates, total, err := repo.List(ctx, service.ListFilter{
		UserID: "00000000-0000-0000-0000-000000000002",
	})
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}

	if total != 1 {
		t.Errorf("after commit, expected 1 template, got %d", total)
	}

	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}

	if templates[0].Name != "Transaction Test" {
		t.Errorf("expected template name 'Transaction Test', got %v", templates[0].Name)
	}
}

// TestPostgresTemplateRepository_TransactionRollback проверяет rollback транзакции.
func TestPostgresTemplateRepository_TransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	ctx := context.Background()
	repo := NewPostgresTemplateRepository(db)

	// Начинаем транзакцию
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Создаём шаблон в транзакции
	tmpl := &model.Template{
		UserID:  "00000000-0000-0000-0000-000000000003",
		Name:    "Rollback Test",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Body:    "Test body",
	}

	_, err = tx.Exec(`
		INSERT INTO templates (user_id, name, channel, type, body, is_default, is_system, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, false, false, 1, NOW(), NOW())
	`, tmpl.UserID, tmpl.Name, tmpl.Channel, tmpl.Type, tmpl.Body)
	if err != nil {
		require.NoError(t, tx.Rollback())
		t.Fatalf("failed to insert template: %v", err)
	}

	// Rollback транзакции
	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	// Проверяем, что данные НЕ сохранены
	templates, total, err := repo.List(ctx, service.ListFilter{
		UserID: "00000000-0000-0000-0000-000000000003",
	})
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}

	if total != 0 {
		t.Errorf("after rollback, expected 0 templates, got %d", total)
	}

	if len(templates) != 0 {
		t.Errorf("expected no templates after rollback, got %d", len(templates))
	}
}

// TestPostgresTemplateRepository_SetDefaultInTransaction проверяет SetDefault в транзакции.
func TestPostgresTemplateRepository_SetDefaultInTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)

	ctx := context.Background()
	repo := NewPostgresTemplateRepository(db)

	userID := "00000000-0000-0000-0000-000000000004"
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

	// Проверяем, что первый является дефолтным
	beforeDefault, err := repo.GetDefault(ctx, userID, channel, tmplType)
	if err != nil {
		t.Fatalf("failed to get default before: %v", err)
	}

	if beforeDefault.ID != tmpl1.ID {
		t.Errorf("expected default to be template 1, got %v", beforeDefault.ID)
	}

	// Устанавливаем второй как дефолтный (внутри использует транзакцию)
	if err := repo.SetDefault(ctx, tmpl2.ID, userID); err != nil {
		t.Fatalf("failed to set default: %v", err)
	}

	// Проверяем, что второй стал дефолтным
	afterDefault, err := repo.GetDefault(ctx, userID, channel, tmplType)
	if err != nil {
		t.Fatalf("failed to get default after: %v", err)
	}

	if afterDefault.ID != tmpl2.ID {
		t.Errorf("expected default to be template 2, got %v", afterDefault.ID)
	}

	if !afterDefault.IsDefault {
		t.Error("new default template should have is_default=true")
	}

	// Проверяем, что первый больше не дефолтный
	oldTemplates, _, err := repo.List(ctx, service.ListFilter{
		UserID:    userID,
		Channel:   channel,
		Type:      tmplType,
		IsDefault: func() *bool { b := false; return &b }(),
	})
	require.NoError(t, err)

	foundOldDefault := false
	for _, tmpl := range oldTemplates {
		if tmpl.ID == tmpl1.ID {
			foundOldDefault = true
			if tmpl.IsDefault {
				t.Error("old default template should have is_default=false after SetDefault")
			}
		}
	}

	if !foundOldDefault {
		t.Error("old default template should still exist")
	}
}
