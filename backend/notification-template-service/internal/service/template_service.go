package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/metrics"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

// TemplateServiceImpl реализует TemplateService.
type TemplateServiceImpl struct {
	repo   TemplateRepository
	logger *zerolog.Logger
	cfg    *config.Config
}

// NewTemplateService создаёт новый сервис шаблонов.
func NewTemplateService(repo TemplateRepository, logger *zerolog.Logger, cfg *config.Config) *TemplateServiceImpl {
	return &TemplateServiceImpl{
		repo:   repo,
		logger: logger,
		cfg:    cfg,
	}
}

// CreateTemplate создаёт новый шаблон.
func (s *TemplateServiceImpl) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*model.Template, error) {
	tmpl := &model.Template{
		UserID:      req.UserID,
		Name:        req.Name,
		Description: req.Description,
		Channel:     req.Channel,
		Type:        req.Type,
		Engine:      req.Engine,
		Subject:     req.Subject,
		Body:        req.Body,
		Format:      req.Format,
		IsDefault:   false,
		IsSystem:    false,
	}

	// Валидация
	if err := tmpl.Validate(); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	// Проверка размера шаблона
	if len(tmpl.Body) > s.cfg.MaxTemplateSize {
		return nil, fmt.Errorf("template body exceeds maximum size of %d bytes", s.cfg.MaxTemplateSize)
	}

	// Подготовка к созданию
	tmpl.BeforeCreate()

	// Создание в репозитории
	if err := s.repo.Create(ctx, tmpl); err != nil {
		s.logger.Error().Err(err).Str("user_id", req.UserID).Str("name", req.Name).Msg("failed to create template")
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	s.logger.Info().
		Str("template_id", tmpl.ID).
		Str("user_id", req.UserID).
		Str("name", req.Name).
		Msg("template created")

	// Записываем метрику
	metrics.RecordTemplateCreate(string(tmpl.Channel), string(tmpl.Type), req.UserID)

	return tmpl, nil
}

// GetTemplate получает шаблон по ID.
func (s *TemplateServiceImpl) GetTemplate(ctx context.Context, id string) (*model.Template, error) {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to get template")
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return tmpl, nil
}

// ListTemplates получает список шаблонов.
func (s *TemplateServiceImpl) ListTemplates(ctx context.Context, filter ListFilter) ([]*model.Template, int, error) {
	templates, total, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list templates")
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, total, nil
}

// UpdateTemplate обновляет шаблон.
func (s *TemplateServiceImpl) UpdateTemplate(ctx context.Context, id string, req *UpdateTemplateRequest) (*model.Template, error) {
	// Получаем существующий шаблон
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to get template for update")
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Проверяем, что это не системный шаблон
	if tmpl.IsSystem {
		return nil, fmt.Errorf("system templates cannot be modified")
	}

	// Обновляем поля
	tmpl.Name = req.Name
	tmpl.Description = req.Description
	tmpl.Subject = req.Subject
	tmpl.Body = req.Body
	tmpl.Format = req.Format

	// Валидация
	if err := tmpl.Validate(); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	// Проверка размера
	if len(tmpl.Body) > s.cfg.MaxTemplateSize {
		return nil, fmt.Errorf("template body exceeds maximum size of %d bytes", s.cfg.MaxTemplateSize)
	}

	// Подготовка к обновлению
	tmpl.BeforeUpdate()

	// Обновление в репозитории
	if err := s.repo.Update(ctx, tmpl); err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to update template")
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	s.logger.Info().
		Str("template_id", id).
		Str("name", tmpl.Name).
		Msg("template updated")

	// Записываем метрику
	metrics.RecordTemplateUpdate(string(tmpl.Channel), string(tmpl.Type))

	return tmpl, nil
}

// DeleteTemplate удаляет шаблон.
func (s *TemplateServiceImpl) DeleteTemplate(ctx context.Context, id string) error {
	// Получаем шаблон для проверки
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to get template for deletion")
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Проверяем, можно ли удалить
	if !tmpl.CanDelete() {
		return fmt.Errorf("template cannot be deleted (system template)")
	}

	// Удаление
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to delete template")
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.logger.Info().
		Str("template_id", id).
		Str("name", tmpl.Name).
		Msg("template deleted")

	// Записываем метрику
	metrics.RecordTemplateDelete(tmpl.IsSystem, string(tmpl.Channel), string(tmpl.Type))

	return nil
}

// CloneTemplate клонирует шаблон.
func (s *TemplateServiceImpl) CloneTemplate(ctx context.Context, id, userID, newName string) (*model.Template, error) {
	// Получаем оригинальный шаблон
	original, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to get template for cloning")
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Клонируем
	clone := original.Clone(newName)
	clone.UserID = userID

	// Валидация
	if err := clone.Validate(); err != nil {
		return nil, fmt.Errorf("cloned template validation failed: %w", err)
	}

	// Создание клон
	clone.BeforeCreate()

	if err := s.repo.Create(ctx, clone); err != nil {
		s.logger.Error().Err(err).Str("parent_id", id).Str("name", newName).Msg("failed to create cloned template")
		return nil, fmt.Errorf("failed to create cloned template: %w", err)
	}

	s.logger.Info().
		Str("template_id", clone.ID).
		Str("parent_id", id).
		Str("name", newName).
		Msg("template cloned")

	// Записываем метрику
	metrics.RecordTemplateClone()

	return clone, nil
}

// SetDefaultTemplate устанавливает шаблон как дефолтный.
func (s *TemplateServiceImpl) SetDefaultTemplate(ctx context.Context, id, userID string) (*model.Template, error) {
	// Проверяем существование шаблона
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to get template for set default")
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Проверяем, что шаблон принадлежит пользователю
	if tmpl.UserID != userID {
		return nil, fmt.Errorf("template does not belong to user")
	}

	// Устанавливаем как дефолтный
	if err := s.repo.SetDefault(ctx, id, userID); err != nil {
		s.logger.Error().Err(err).Str("template_id", id).Str("user_id", userID).Msg("failed to set default template")
		return nil, fmt.Errorf("failed to set default template: %w", err)
	}

	s.logger.Info().
		Str("template_id", id).
		Str("user_id", userID).
		Str("channel", string(tmpl.Channel)).
		Str("type", string(tmpl.Type)).
		Msg("default template set")

	return tmpl, nil
}

// GetDefaultTemplate получает дефолтный шаблон.
func (s *TemplateServiceImpl) GetDefaultTemplate(
	ctx context.Context,
	userID string,
	channel model.TemplateChannel,
	tmplType model.TemplateType,
) (*model.Template, error) {
	tmpl, err := s.repo.GetDefault(ctx, userID, channel, tmplType)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("channel", string(channel)).
			Str("type", string(tmplType)).
			Msg("failed to get default template")
		return nil, fmt.Errorf("failed to get default template: %w", err)
	}

	return tmpl, nil
}
