package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	integrationv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/service/apikey"
)

// APIKeyHandler обрабатывает gRPC запросы для API ключей.
type APIKeyHandler struct {
	integrationv1.UnimplementedAPIKeyServiceServer
	apiKeyService *apikey.APIKeyService
}

// NewAPIKeyHandler создаёт новый APIKeyHandler.
func NewAPIKeyHandler(apiKeyService *apikey.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// CreateAPIKey создаёт новый API ключ.
func (h *APIKeyHandler) CreateAPIKey(ctx context.Context, req *integrationv1.CreateAPIKeyRequest) (*integrationv1.APIKey, error) {
	// 1. Парсим UUID
	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// 2. Конвертируем scopes
	scopes := make([]string, len(req.Scopes))
	copy(scopes, req.Scopes)

	// 3. Конвертируем key type
	keyType := model.APIKeyType(req.KeyType)

	// 4. Конвертируем expires_at
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		expiresAt = &t
	}

	// 5. Конвертируем proto в DTO
	createReq := &apikey.CreateAPIKeyRequest{
		UserID:             userID,
		Name:               req.Name,
		Description:        &req.Description,
		KeyType:            keyType,
		Scopes:             scopes,
		ExpiresAt:          expiresAt,
		IPWhitelist:        req.IpWhitelist,
		RateLimitPerMinute: int(req.RateLimitPerMinute),
	}

	// 6. Вызываем service
	createdKey, _, err := h.apiKeyService.CreateAPIKey(ctx, createReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	// 7. Конвертируем модель в proto
	return modelToProtoAPIKey(createdKey), nil
}

// GetAPIKey возвращает API ключ по ID.
func (h *APIKeyHandler) GetAPIKey(ctx context.Context, req *integrationv1.GetAPIKeyRequest) (*integrationv1.APIKey, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	apiKey, err := h.apiKeyService.GetAPIKey(ctx, id, userID)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoAPIKey(apiKey), nil
}

// ListAPIKeys возвращает список API ключей.
func (h *APIKeyHandler) ListAPIKeys(ctx context.Context, req *integrationv1.ListAPIKeysRequest) (*integrationv1.ListAPIKeysResponse, error) {
	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	listReq := &apikey.ListAPIKeysRequest{
		UserID:    userID,
		PageSize:  int(req.PageSize),
		PageToken: req.PageToken,
	}

	keys, err := h.apiKeyService.ListAPIKeys(ctx, listReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	protoKeys := make([]*integrationv1.APIKey, len(keys))
	for i, k := range keys {
		protoKeys[i] = modelToProtoAPIKey(k)
	}

	return &integrationv1.ListAPIKeysResponse{
		ApiKeys:    protoKeys,
		TotalCount: int32(len(keys)), //nolint:gosec // G115: len всегда неотрицателен и помещается в int32
	}, nil
}

// UpdateAPIKey обновляет API ключ.
func (h *APIKeyHandler) UpdateAPIKey(ctx context.Context, req *integrationv1.UpdateAPIKeyRequest) (*integrationv1.APIKey, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	updateReq := &apikey.UpdateAPIKeyRequest{
		ID:     id,
		UserID: userID,
	}

	if req.Name != "" {
		updateReq.Name = &req.Name
	}

	if req.Description != "" {
		updateReq.Description = &req.Description
	}

	if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.AsTime()
		updateReq.ExpiresAt = &expiresAt
	}

	if req.IpWhitelist != nil {
		updateReq.IPWhitelist = req.IpWhitelist
	}

	if req.RateLimitPerMinute > 0 {
		rateLimit := int(req.RateLimitPerMinute)
		updateReq.RateLimitPerMinute = &rateLimit
	}

	updatedKey, err := h.apiKeyService.UpdateAPIKey(ctx, updateReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoAPIKey(updatedKey), nil
}

// DeleteAPIKey удаляет API ключ.
func (h *APIKeyHandler) DeleteAPIKey(ctx context.Context, req *integrationv1.DeleteAPIKeyRequest) (*integrationv1.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	if err := h.apiKeyService.DeleteAPIKey(ctx, id, userID); err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.Empty{}, nil
}

// RotateAPIKeySecret ротирует секретный ключ API ключа.
func (h *APIKeyHandler) RotateAPIKeySecret(ctx context.Context, req *integrationv1.RotateAPIKeySecretRequest) (*integrationv1.APIKey, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	rotateReq := &apikey.RotateAPIKeySecretRequest{
		ID:     id,
		UserID: userID,
	}

	response, err := h.apiKeyService.RotateAPIKeySecret(ctx, rotateReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoAPIKey(response.APIKey), nil
}

// GetAPIKeyStats возвращает статистику API ключа.
func (h *APIKeyHandler) GetAPIKeyStats(ctx context.Context, req *integrationv1.GetAPIKeyStatsRequest) (*integrationv1.APIKeyStats, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	statsReq := &apikey.GetAPIKeyStatsRequest{
		ID:     id,
		UserID: userID,
	}

	stats, err := h.apiKeyService.GetAPIKeyStats(ctx, statsReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.APIKeyStats{
		TotalRequests:      stats.TotalRequests,
		SuccessfulRequests: stats.SuccessfulRequests,
		FailedRequests:     stats.FailedRequests,
		SuccessRate:        float32(stats.SuccessRate),
		AvgLatencyMs:       stats.AvgLatencyMs,
		LastUsedAt:         timestampProto(stats.LastUsedAt),
		LastUsedIp:         stats.LastUsedIP,
		MostUsedEndpoint:   stats.MostUsedEndpoint,
	}, nil
}

// ValidateAPIKey валидирует API ключ (используется Gateway).
func (h *APIKeyHandler) ValidateAPIKey(ctx context.Context, req *integrationv1.ValidateAPIKeyRequest) (*integrationv1.ValidateAPIKeyResponse, error) {
	validateReq := &apikey.ValidateAPIKeyRequest{
		APIKey:    req.ApiKey,
		Endpoint:  req.Endpoint,
		Method:    req.Method,
		IPAddress: req.IpAddress,
	}

	response, err := h.apiKeyService.ValidateAPIKey(ctx, validateReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &integrationv1.ValidateAPIKeyResponse{
		Valid:     response.Valid,
		UserId:    response.UserID.String(),
		Scopes:    response.Scopes,
		Status:    string(response.Status),
		ErrorCode: response.ErrorCode,
	}, nil
}

// Helper функции для конвертации model ↔ proto

func modelToProtoAPIKey(key *model.APIKey) *integrationv1.APIKey {
	proto := &integrationv1.APIKey{
		Id:                 key.ID.String(),
		UserId:             key.UserID.String(),
		Name:               key.Name,
		KeyPrefix:          key.KeyPrefix,
		Scopes:             key.Scopes,
		Status:             string(key.Status),
		RateLimitPerMinute: int32(key.RateLimitPerMinute), //nolint:gosec // G115: значение ограничено при создании ключа
		CreatedAt:          timestamppb.New(key.CreatedAt),
		UpdatedAt:          timestamppb.New(key.UpdatedAt),
	}

	if key.Description != nil {
		proto.Description = *key.Description
	}

	if key.ExpiresAt != nil {
		proto.ExpiresAt = timestamppb.New(*key.ExpiresAt)
	}

	if len(key.IPWhitelist) > 0 {
		proto.IpWhitelist = key.IPWhitelist
	}

	if key.LastRotatedAt != nil {
		proto.LastRotatedAt = timestamppb.New(*key.LastRotatedAt)
	}

	return proto
}
