package service

import (
	"context"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

// TemplateRepository определяет интерфейс для работы с хранилищем шаблонов.
type TemplateRepository interface {
	// Create создает новый шаблон.
	Create(ctx context.Context, tmpl *model.Template) error

	// GetByID получает шаблон по ID.
	GetByID(ctx context.Context, id string) (*model.Template, error)

	// GetByUserAndType получает дефолтный шаблон для пользователя, канала и типа.
	GetDefault(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error)

	// List получает список шаблонов с пагинацией и фильтрами.
	List(ctx context.Context, filter ListFilter) ([]*model.Template, int, error)

	// Update обновляет шаблон.
	Update(ctx context.Context, tmpl *model.Template) error

	// Delete удаляет шаблон.
	Delete(ctx context.Context, id string) error

	// SetDefault устанавливает шаблон как дефолтный.
	SetDefault(ctx context.Context, id, userID string) error

	// Exists проверяет существование шаблона по ID.
	Exists(ctx context.Context, id string) (bool, error)
}

// ListFilter параметры для фильтрации шаблонов.
type ListFilter struct {
	UserID    string
	Channel   model.TemplateChannel
	Type      model.TemplateType
	IsDefault *bool
	IsSystem  *bool
	Page      int
	PageSize  int
	SortBy    string
	SortDesc  bool
}

// TemplateService определяет интерфейс сервиса шаблонов.
type TemplateService interface {
	// CreateTemplate создает новый шаблон.
	CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*model.Template, error)

	// GetTemplate получает шаблон по ID.
	GetTemplate(ctx context.Context, id string) (*model.Template, error)

	// ListTemplates получает список шаблонов.
	ListTemplates(ctx context.Context, filter ListFilter) ([]*model.Template, int, error)

	// UpdateTemplate обновляет шаблон.
	UpdateTemplate(ctx context.Context, id string, req *UpdateTemplateRequest) (*model.Template, error)

	// DeleteTemplate удаляет шаблон.
	DeleteTemplate(ctx context.Context, id string) error

	// CloneTemplate клонирует шаблон.
	CloneTemplate(ctx context.Context, id, userID, newName string) (*model.Template, error)

	// SetDefaultTemplate устанавливает шаблон как дефолтный.
	SetDefaultTemplate(ctx context.Context, id, userID string) (*model.Template, error)

	// GetDefaultTemplate получает дефолтный шаблон.
	GetDefaultTemplate(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error)
}

// RendererService определяет интерфейс сервиса рендеринга.
type RendererService interface {
	// RenderTemplate рендерит шаблон с переменными.
	RenderTemplate(ctx context.Context, templateID string, variables model.VariablesMap) (*model.RenderedTemplate, error)

	// PreviewTemplate превью шаблона с тестовыми данными.
	PreviewTemplate(ctx context.Context, tmpl *model.Template) (*model.RenderedTemplate, error)
}

// ValidatorService определяет интерфейс сервиса валидации.
type ValidatorService interface {
	// ValidateTemplate проверяет синтаксис и переменные шаблона.
	ValidateTemplate(ctx context.Context, tmpl *model.Template) (*model.ValidationResult, error)

	// GetAvailableVariables возвращает доступные переменные для типа шаблона.
	GetAvailableVariables(ctx context.Context, tmplType model.TemplateType, channel model.TemplateChannel) []model.TemplateVariable
}

// CreateTemplateRequest запрос на создание шаблона.
type CreateTemplateRequest struct {
	UserID      string
	Name        string
	Description string
	Channel     model.TemplateChannel
	Type        model.TemplateType
	Engine      model.TemplateEngine
	Subject     string
	Body        string
	Format      model.TemplateFormat
}

// UpdateTemplateRequest запрос на обновление шаблона.
type UpdateTemplateRequest struct {
	Name        string
	Description string
	Subject     string
	Body        string
	Format      model.TemplateFormat
}
