package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// openBrokenDB открывает БД с заведомо нерабочим адресом для тестирования ошибок.
func openBrokenDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 user=x password=x dbname=x sslmode=disable connect_timeout=1")
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	return db
}

// TestNewPostgresTemplateRepository проверяет создание репозитория.
func TestNewPostgresTemplateRepository(t *testing.T) {
	t.Parallel()

	// Arrange
	db := &sqlx.DB{}

	// Act
	repo := NewPostgresTemplateRepository(db)

	// Assert
	if repo == nil {
		t.Fatal("NewPostgresTemplateRepository() returned nil")
	}
}

// TestCreate_DBError проверяет обработку ошибки БД при создании.
func TestCreate_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)
	tmpl := &model.Template{
		UserID:  "user-1",
		Name:    "Test",
		Body:    "body",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
	}

	// Act
	err := repo.Create(context.Background(), tmpl)

	// Assert
	if err == nil {
		t.Fatal("Create() expected error with broken DB, got nil")
	}
}

// TestGetByID_DBError проверяет обработку ошибки БД при получении по ID.
func TestGetByID_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	// Act
	_, err := repo.GetByID(context.Background(), "test-id")

	// Assert
	if err == nil {
		t.Fatal("GetByID() expected error with broken DB, got nil")
	}
}

// TestGetDefault_DBError проверяет обработку ошибки БД при получении дефолтного шаблона.
func TestGetDefault_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	// Act
	_, err := repo.GetDefault(context.Background(), "user-1", model.ChannelEmail, model.TypeMonitorDown)

	// Assert
	if err == nil {
		t.Fatal("GetDefault() expected error with broken DB, got nil")
	}
}

// TestList_DBError проверяет обработку ошибки БД при получении списка.
func TestList_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	// Act
	_, _, err := repo.List(context.Background(), service.ListFilter{})

	// Assert
	if err == nil {
		t.Fatal("List() expected error with broken DB, got nil")
	}
}

// TestList_DBError_WithAllFilters проверяет все ветки фильтрации при ошибке БД.
func TestList_DBError_WithAllFilters(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	isDefault := true
	isSystem := false

	filter := service.ListFilter{
		UserID:    "user-1",
		Channel:   model.ChannelEmail,
		Type:      model.TypeMonitorDown,
		IsDefault: &isDefault,
		IsSystem:  &isSystem,
		Page:      2,
		PageSize:  10,
		SortBy:    "name",
		SortDesc:  true,
	}

	// Act
	_, _, err := repo.List(context.Background(), filter)

	// Assert
	if err == nil {
		t.Fatal("List() expected error with broken DB, got nil")
	}
}

// TestUpdate_DBError проверяет обработку ошибки БД при обновлении.
func TestUpdate_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)
	tmpl := &model.Template{
		ID:   "test-id",
		Name: "Test",
		Body: "body",
	}

	// Act
	err := repo.Update(context.Background(), tmpl)

	// Assert
	if err == nil {
		t.Fatal("Update() expected error with broken DB, got nil")
	}
}

// TestDelete_DBError проверяет обработку ошибки БД при удалении.
func TestDelete_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	// Act
	err := repo.Delete(context.Background(), "test-id")

	// Assert
	if err == nil {
		t.Fatal("Delete() expected error with broken DB, got nil")
	}
}

// TestSetDefault_DBError проверяет обработку ошибки БД при установке дефолтного шаблона.
func TestSetDefault_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	// Act
	err := repo.SetDefault(context.Background(), "test-id", "user-1")

	// Assert
	if err == nil {
		t.Fatal("SetDefault() expected error with broken DB, got nil")
	}
}

// TestExists_DBError проверяет обработку ошибки БД при проверке существования.
func TestExists_DBError(t *testing.T) {
	t.Parallel()

	// Arrange
	db := openBrokenDB(t)
	closeTestDB(t, db)

	repo := NewPostgresTemplateRepository(db)

	// Act
	_, err := repo.Exists(context.Background(), "test-id")

	// Assert
	if err == nil {
		t.Fatal("Exists() expected error with broken DB, got nil")
	}
}
