package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	integrationv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/service/webhook"
)

// WebhookHandler обрабатывает gRPC запросы для webhook интеграций.
type WebhookHandler struct {
	integrationv1.UnimplementedWebhookIntegrationServiceServer
	webhookService *webhook.WebhookService
}

// NewWebhookHandler создаёт новый WebhookHandler.
func NewWebhookHandler(webhookService *webhook.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// CreateWebhook создаёт новую webhook интеграцию.
func (h *WebhookHandler) CreateWebhook(ctx context.Context, req *integrationv1.CreateWebhookRequest) (*integrationv1.WebhookIntegration, error) {
	// 1. Парсим UUID
	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// 2. Конвертируем proto в DTO
	createReq := &webhook.CreateWebhookRequest{
		UserID:                  userID,
		Name:                    req.Name,
		URL:                     req.Url,
		Method:                  req.Method,
		Headers:                 req.Headers,
		SecretKey:               &req.SecretKey,
		Priority:                model.WebhookPriority(req.Priority),
		SeverityFilter:          req.SeverityFilter,
		MaxPayloadSizeBytes:     int(req.MaxPayloadSizeBytes),
		PayloadHandlingStrategy: model.PayloadHandlingStrategy(req.PayloadHandlingStrategy),
	}

	// 3. Вызываем service
	createdWebhook, err := h.webhookService.CreateWebhook(ctx, createReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	// 4. Конвертируем модель в proto
	return modelToProtoWebhook(createdWebhook), nil
}

// GetWebhook возвращает webhook по ID.
func (h *WebhookHandler) GetWebhook(ctx context.Context, req *integrationv1.GetWebhookRequest) (*integrationv1.WebhookIntegration, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	wh, err := h.webhookService.GetWebhook(ctx, id, userID)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoWebhook(wh), nil
}

// ListWebhooks возвращает список webhooks.
func (h *WebhookHandler) ListWebhooks(ctx context.Context, req *integrationv1.ListWebhooksRequest) (*integrationv1.ListWebhooksResponse, error) {
	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	listReq := &webhook.ListWebhooksRequest{
		UserID:    userID,
		PageSize:  int(req.PageSize),
		PageToken: req.PageToken,
	}

	webhooks, err := h.webhookService.ListWebhooks(ctx, listReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	protoWebhooks := make([]*integrationv1.WebhookIntegration, len(webhooks))
	for i, w := range webhooks {
		protoWebhooks[i] = modelToProtoWebhook(w)
	}

	return &integrationv1.ListWebhooksResponse{
		Webhooks:   protoWebhooks,
		TotalCount: int32(len(webhooks)), //nolint:gosec // G115: len всегда неотрицателен
	}, nil
}

// UpdateWebhook обновляет webhook.
func (h *WebhookHandler) UpdateWebhook(ctx context.Context, req *integrationv1.UpdateWebhookRequest) (*integrationv1.WebhookIntegration, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	updateReq := &webhook.UpdateWebhookRequest{
		ID:     id,
		UserID: userID,
	}

	if req.Name != "" {
		updateReq.Name = &req.Name
	}

	if req.Url != "" {
		updateReq.URL = &req.Url
	}

	if req.Method != "" {
		updateReq.Method = &req.Method
	}

	if req.Headers != nil {
		updateReq.Headers = req.Headers
	}

	if req.Priority != "" {
		priority := model.WebhookPriority(req.Priority)
		updateReq.Priority = &priority
	}

	if req.SeverityFilter != nil {
		updateReq.SeverityFilter = req.SeverityFilter
	}

	if req.MaxPayloadSizeBytes > 0 {
		size := int(req.MaxPayloadSizeBytes)
		updateReq.MaxPayloadSizeBytes = &size
	}

	if req.PayloadHandlingStrategy != "" {
		strategy := model.PayloadHandlingStrategy(req.PayloadHandlingStrategy)
		updateReq.PayloadHandlingStrategy = &strategy
	}

	updatedWebhook, err := h.webhookService.UpdateWebhook(ctx, updateReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoWebhook(updatedWebhook), nil
}

// DeleteWebhook удаляет webhook.
func (h *WebhookHandler) DeleteWebhook(ctx context.Context, req *integrationv1.DeleteWebhookRequest) (*integrationv1.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	if err := h.webhookService.DeleteWebhook(ctx, id, userID); err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.Empty{}, nil
}

// EnableWebhook включает webhook.
func (h *WebhookHandler) EnableWebhook(ctx context.Context, req *integrationv1.EnableWebhookRequest) (*integrationv1.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	if err := h.webhookService.EnableWebhook(ctx, id, userID); err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.Empty{}, nil
}

// DisableWebhook отключает webhook.
func (h *WebhookHandler) DisableWebhook(ctx context.Context, req *integrationv1.DisableWebhookRequest) (*integrationv1.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	if err := h.webhookService.DisableWebhook(ctx, id, userID); err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.Empty{}, nil
}

// TestWebhook тестирует webhook endpoint.
func (h *WebhookHandler) TestWebhook(ctx context.Context, req *integrationv1.TestWebhookRequest) (*integrationv1.TestWebhookResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// Вызываем service для тестирования webhook
	testResp, err := h.webhookService.TestWebhook(ctx, id, userID)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.TestWebhookResponse{
		Success:        testResp.Success,
		StatusCode:     int32(testResp.StatusCode),     //nolint:gosec // G115: HTTP статус 100-599 помещается в int32
		ResponseTimeMs: int32(testResp.ResponseTimeMs), //nolint:gosec // G115: значение в мс ограничено таймаутом
		ErrorMessage:   testResp.ErrorMessage,
	}, nil
}

// CloneWebhook клонирует webhook.
func (h *WebhookHandler) CloneWebhook(ctx context.Context, req *integrationv1.CloneWebhookRequest) (*integrationv1.WebhookIntegration, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	cloneReq := &webhook.CloneWebhookRequest{
		ID:      id,
		UserID:  userID,
		NewName: req.NewName,
		NewURL:  req.NewUrl,
	}

	clonedWebhook, err := h.webhookService.CloneWebhook(ctx, cloneReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoWebhook(clonedWebhook), nil
}

// GetWebhookStats возвращает статистику webhook.
func (h *WebhookHandler) GetWebhookStats(ctx context.Context, req *integrationv1.GetWebhookStatsRequest) (*integrationv1.WebhookStats, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	stats, err := h.webhookService.GetWebhookStats(ctx, id, userID)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.WebhookStats{
		TotalSent:         int32(stats.TotalSent),         //nolint:gosec // G115
		SuccessfulSent:    int32(stats.SuccessfulSent),    //nolint:gosec // G115
		FailedSent:        int32(stats.FailedSent),        //nolint:gosec // G115
		AvgResponseTimeMs: int32(stats.AvgResponseTimeMs), //nolint:gosec // G115
		LastSentAt:        timestampProto(stats.LastSentAt),
		LastSuccessAt:     timestampProto(stats.LastSuccessAt),
		LastFailureAt:     timestampProto(stats.LastFailureAt),
		FailureCount:      int32(stats.FailureCount), //nolint:gosec // G115
		SuccessRate:       float32(stats.SuccessRate),
	}, nil
}

// Helper функции для конвертации model ↔ proto

func modelToProtoWebhook(w *model.WebhookIntegration) *integrationv1.WebhookIntegration {
	proto := &integrationv1.WebhookIntegration{
		Id:                      w.ID.String(),
		UserId:                  w.UserID.String(),
		Name:                    w.Name,
		Url:                     w.URL,
		Method:                  w.Method,
		Headers:                 w.Headers,
		Enabled:                 w.Enabled,
		Status:                  string(w.Status),
		Priority:                string(w.Priority),
		SeverityFilter:          w.SeverityFilter,
		MaxPayloadSizeBytes:     int32(w.MaxPayloadSizeBytes), //nolint:gosec // G115
		PayloadHandlingStrategy: string(w.PayloadHandlingStrategy),
		CreatedAt:               timestamppb.New(w.CreatedAt),
		UpdatedAt:               timestamppb.New(w.UpdatedAt),
	}

	return proto
}

func timestampProto(ts *int64) *timestamppb.Timestamp {
	if ts == nil {
		return nil
	}

	return timestamppb.New(time.Unix(*ts, 0))
}
