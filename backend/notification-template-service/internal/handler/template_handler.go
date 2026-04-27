package handler

import (
	"context"
	"time"

	v1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

// Handler реализует gRPC handler для NotificationTemplateService.
type Handler struct {
	v1.UnimplementedNotificationTemplateServiceServer

	templateSvc  service.TemplateService
	rendererSvc  service.RendererService
	validatorSvc service.ValidatorService
}

// New создаёт новый handler.
func New(
	templateSvc service.TemplateService,
	rendererSvc service.RendererService,
	validatorSvc service.ValidatorService,
) *Handler {
	return &Handler{
		templateSvc:  templateSvc,
		rendererSvc:  rendererSvc,
		validatorSvc: validatorSvc,
	}
}

// CreateTemplate создаёт новый шаблон.
func (h *Handler) CreateTemplate(ctx context.Context, req *v1.CreateTemplateRequest) (*v1.Template, error) {
	tmpl, err := h.templateSvc.CreateTemplate(ctx, &service.CreateTemplateRequest{
		UserID:      req.UserId,
		Name:        req.Name,
		Description: req.Description,
		Channel:     protoChannelToModel(req.Channel),
		Type:        protoTypeToModel(req.Type),
		Engine:      protoEngineToModel(req.Engine),
		Subject:     req.Subject,
		Body:        req.Body,
		Format:      model.TemplateFormat(req.Format),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return h.templateToProto(tmpl), nil
}

// GetTemplate получает шаблон по ID.
func (h *Handler) GetTemplate(ctx context.Context, req *v1.GetTemplateRequest) (*v1.Template, error) {
	tmpl, err := h.templateSvc.GetTemplate(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return h.templateToProto(tmpl), nil
}

// ListTemplates получает список шаблонов.
func (h *Handler) ListTemplates(ctx context.Context, req *v1.ListTemplatesRequest) (*v1.ListTemplatesResponse, error) {
	var isDefault, isSystem *bool
	if req.IsDefault {
		isDefault = &req.IsDefault
	}
	if req.IsSystem {
		isSystem = &req.IsSystem
	}

	templates, total, err := h.templateSvc.ListTemplates(ctx, service.ListFilter{
		UserID:    req.UserId,
		Channel:   protoChannelToModel(req.Channel),
		Type:      protoTypeToModel(req.Type),
		IsDefault: isDefault,
		IsSystem:  isSystem,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortDesc:  req.SortDesc,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoTemplates := make([]*v1.Template, len(templates))
	for i, tmpl := range templates {
		protoTemplates[i] = h.templateToProto(tmpl)
	}

	return &v1.ListTemplatesResponse{
		Templates: protoTemplates,
		Total:     int32(total), //nolint:gosec // G115: total из БД в допустимом диапазоне
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// UpdateTemplate обновляет шаблон.
func (h *Handler) UpdateTemplate(ctx context.Context, req *v1.UpdateTemplateRequest) (*v1.Template, error) {
	tmpl, err := h.templateSvc.UpdateTemplate(ctx, req.Id, &service.UpdateTemplateRequest{
		Name:        req.Name,
		Description: req.Description,
		Subject:     req.Subject,
		Body:        req.Body,
		Format:      model.TemplateFormat(req.Format),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return h.templateToProto(tmpl), nil
}

// DeleteTemplate удаляет шаблон.
func (h *Handler) DeleteTemplate(ctx context.Context, req *v1.DeleteTemplateRequest) (*v1.Empty, error) {
	if err := h.templateSvc.DeleteTemplate(ctx, req.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.Empty{}, nil
}

// CloneTemplate клонирует шаблон.
func (h *Handler) CloneTemplate(ctx context.Context, req *v1.CloneTemplateRequest) (*v1.Template, error) {
	tmpl, err := h.templateSvc.CloneTemplate(ctx, req.Id, req.UserId, req.NewName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return h.templateToProto(tmpl), nil
}

// RenderTemplate рендерит шаблон с переменными.
func (h *Handler) RenderTemplate(ctx context.Context, req *v1.RenderTemplateRequest) (*v1.RenderedTemplate, error) {
	rendered, err := h.rendererSvc.RenderTemplate(ctx, req.TemplateId, req.Variables)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.RenderedTemplate{
		Subject:  rendered.Subject,
		Body:     rendered.Body,
		Format:   string(rendered.Format),
		Success:  rendered.Success,
		Errors:   rendered.Errors,
		Warnings: rendered.Warnings,
	}, nil
}

// ValidateTemplate проверяет синтаксис шаблона.
func (h *Handler) ValidateTemplate(ctx context.Context, req *v1.ValidateTemplateRequest) (*v1.ValidationResult, error) {
	tmpl := &model.Template{
		Channel: protoChannelToModel(req.Channel),
		Type:    protoTypeToModel(req.Type),
		Engine:  protoEngineToModel(req.Engine),
		Subject: req.Subject,
		Body:    req.Body,
	}

	result, err := h.validatorSvc.ValidateTemplate(ctx, tmpl)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.ValidationResult{
		Valid:            result.Valid,
		Errors:           result.Errors,
		Warnings:         result.Warnings,
		UsedVariables:    result.UsedVariables,
		MissingVariables: result.MissingVariables,
	}, nil
}

// GetAvailableVariables возвращает доступные переменные.
func (h *Handler) GetAvailableVariables(ctx context.Context, req *v1.GetAvailableVariablesRequest) (*v1.AvailableVariables, error) {
	variables := h.validatorSvc.GetAvailableVariables(
		ctx,
		protoTypeToModel(req.Type),
		protoChannelToModel(req.Channel),
	)

	protoVariables := make([]*v1.TemplateVariable, len(variables))
	for i, v := range variables {
		protoVariables[i] = &v1.TemplateVariable{
			Name:         v.Name,
			Description:  v.Description,
			Type:         string(v.Type),
			Required:     v.Required,
			ExampleValue: v.ExampleValue,
			NestedFields: v.NestedFields,
		}
	}

	return &v1.AvailableVariables{
		Type:      req.Type,
		Variables: protoVariables,
	}, nil
}

// SetDefaultTemplate устанавливает шаблон как дефолтный.
func (h *Handler) SetDefaultTemplate(ctx context.Context, req *v1.SetDefaultTemplateRequest) (*v1.Template, error) {
	tmpl, err := h.templateSvc.SetDefaultTemplate(ctx, req.TemplateId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return h.templateToProto(tmpl), nil
}

// GetDefaultTemplate получает дефолтный шаблон.
func (h *Handler) GetDefaultTemplate(ctx context.Context, req *v1.GetDefaultTemplateRequest) (*v1.Template, error) {
	tmpl, err := h.templateSvc.GetDefaultTemplate(
		ctx,
		req.UserId,
		protoChannelToModel(req.Channel),
		protoTypeToModel(req.Type),
	)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return h.templateToProto(tmpl), nil
}

// PreviewTemplate превью шаблона с тестовыми данными.
func (h *Handler) PreviewTemplate(ctx context.Context, req *v1.PreviewTemplateRequest) (*v1.RenderedTemplate, error) {
	tmpl := &model.Template{
		Channel: protoChannelToModel(req.Channel),
		Type:    protoTypeToModel(req.Type),
		Engine:  protoEngineToModel(req.Engine),
		Subject: req.Subject,
		Body:    req.Body,
		Format:  model.TemplateFormat(req.Format),
	}

	rendered, err := h.rendererSvc.PreviewTemplate(ctx, tmpl)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.RenderedTemplate{
		Subject:  rendered.Subject,
		Body:     rendered.Body,
		Format:   string(rendered.Format),
		Success:  rendered.Success,
		Errors:   rendered.Errors,
		Warnings: rendered.Warnings,
	}, nil
}

// templateToProto конвертирует domain модель в proto.
func (h *Handler) templateToProto(tmpl *model.Template) *v1.Template {
	return &v1.Template{
		Id:          tmpl.ID,
		UserId:      tmpl.UserID,
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Channel:     modelChannelToProto(tmpl.Channel),
		Type:        modelTypeToProto(tmpl.Type),
		Engine:      modelEngineToProto(tmpl.Engine),
		Subject:     tmpl.Subject,
		Body:        tmpl.Body,
		Format:      string(tmpl.Format),
		IsDefault:   tmpl.IsDefault,
		IsSystem:    tmpl.IsSystem,
		CreatedAt:   mustTimestamp(tmpl.CreatedAt),
		UpdatedAt:   mustTimestamp(tmpl.UpdatedAt),
		Version:     int32(tmpl.Version), //nolint:gosec // G115: version из БД в допустимом диапазоне
		ParentId: func() string {
			if tmpl.ParentID != nil {
				return *tmpl.ParentID
			}
			return ""
		}(),
	}
}

// mustTimestamp конвертирует time.Time в protobuf Timestamp.
func mustTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
