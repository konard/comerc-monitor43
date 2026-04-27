package main

import (
	"context"
	"testing"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/logging"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// TestLoadConfig_ValidConfig проверяет Validate для корректного конфига.
func TestLoadConfig_ValidConfig(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := &config.Config{
		ServerAddress: ":50051",
		DatabaseHost:  "localhost",
		DatabaseName:  "test",
	}

	// Act
	err := cfg.Validate()

	// Assert
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

// TestLoadConfig_InvalidConfig проверяет Validate для некорректного конфига.
func TestLoadConfig_InvalidConfig(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := &config.Config{
		ServerAddress: "",
		DatabaseHost:  "localhost",
		DatabaseName:  "test",
	}

	// Act
	err := cfg.Validate()

	// Assert
	if err == nil {
		t.Fatal("Validate() expected error for empty ServerAddress, got nil")
	}
}

// TestConnectDB_Fails проверяет что connectDB возвращает ошибку при недоступной БД.
func TestConnectDB_Fails(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := &config.Config{
		DatabaseHost:     "127.0.0.1",
		DatabasePort:     1,
		DatabaseUser:     "x",
		DatabasePassword: "x",
		DatabaseName:     "x",
		DatabaseSSLMode:  "disable",
	}

	// Act
	_, err := connectDB(cfg)

	// Assert
	if err == nil {
		t.Fatal("connectDB() expected error with unavailable DB, got nil")
	}
}

// TestNewServer_CreatesServer проверяет создание gRPC сервера.
func TestNewServer_CreatesServer(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := &config.Config{
		ServerAddress: ":0",
	}
	logger := logging.New("error", "test")

	// Используем нулевой репозиторий для создания сервисов
	repo := &noopTemplateRepo{}
	l := logger.ZLogger()
	templateSvc := service.NewTemplateService(repo, l, cfg)
	rendererSvc := service.NewRendererService(repo, l, cfg)
	validatorSvc := service.NewValidatorService(l)

	// Act
	srv := newServer(cfg, templateSvc, rendererSvc, validatorSvc, logger)

	// Assert
	if srv == nil {
		t.Fatal("newServer() returned nil")
	}
	srv.Stop()
}

// noopTemplateRepo нулевая реализация TemplateRepository.
type noopTemplateRepo struct{}

func (r *noopTemplateRepo) Create(ctx context.Context, tmpl *model.Template) error {
	return nil
}

func (r *noopTemplateRepo) GetByID(ctx context.Context, id string) (*model.Template, error) {
	return nil, nil
}

func (r *noopTemplateRepo) GetDefault(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error) {
	return nil, nil
}

func (r *noopTemplateRepo) List(ctx context.Context, filter service.ListFilter) ([]*model.Template, int, error) {
	return nil, 0, nil
}

func (r *noopTemplateRepo) Update(ctx context.Context, tmpl *model.Template) error {
	return nil
}

func (r *noopTemplateRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *noopTemplateRepo) SetDefault(ctx context.Context, id, userID string) error {
	return nil
}

func (r *noopTemplateRepo) Exists(ctx context.Context, id string) (bool, error) {
	return false, nil
}
