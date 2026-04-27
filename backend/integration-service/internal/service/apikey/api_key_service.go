package apikey

import (
	"context"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/raul/monitor/backend/integration-service/internal/client"
	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
)

// tracer используется для трассировки операций сервиса API-ключей.
var tracer = otel.Tracer("github.com/raul/monitor/backend/integration-service/internal/service/apikey")

// APIKeyService сервис для управления API ключами.
type APIKeyService struct {
	apiKeyRepo    interfaces.APIKeyRepository
	usageRepo     interfaces.APIKeyUsageRepository
	encryptor     *security.EncryptionService
	keyGenerator  *security.APIKeyGenerator
	billingClient *client.BillingClient
	rateLimiter   RateLimiter
}

// NewAPIKeyService создаёт новый APIKeyService.
func NewAPIKeyService(
	apiKeyRepo interfaces.APIKeyRepository,
	usageRepo interfaces.APIKeyUsageRepository,
	encryptor *security.EncryptionService,
	keyGenerator *security.APIKeyGenerator,
	billingClient *client.BillingClient,
	rateLimiter RateLimiter,
) *APIKeyService {

	return &APIKeyService{
		apiKeyRepo:    apiKeyRepo,
		usageRepo:     usageRepo,
		encryptor:     encryptor,
		keyGenerator:  keyGenerator,
		billingClient: billingClient,
		rateLimiter:   rateLimiter,
	}
}

// CreateAPIKey создаёт новый API ключ.
func (s *APIKeyService) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*model.APIKey, string, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.CreateAPIKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID.String()),
		attribute.String("key_name", req.Name),
	)

	// 1. Валидируем запрос
	if err := req.Validate(); err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to validate create API key request")
	}

	// 2. Проверяем лимит API ключей в billing service
	currentCount, err := s.apiKeyRepo.CountByUserID(ctx, req.UserID)
	if err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to count API keys")
	}

	canCreate, err := s.billingClient.CheckAPIKeyLimit(ctx, req.UserID, currentCount)
	if err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to check API key limit")
	}
	if !canCreate {
		return nil, "", model.ErrAPIKeyLimitReached
	}

	// 3. Генерируем API ключ
	apiKey, fullKey, err := model.NewAPIKey(req.UserID, req.Name, req.KeyType, req.Scopes)
	if err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to create API key model")
	}

	// 3. Применяем дополнительные поля
	if req.Description != nil {
		apiKey.Description = req.Description
	}

	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}

	if len(req.IPWhitelist) > 0 {
		apiKey.IPWhitelist = req.IPWhitelist
	}

	if req.RateLimitPerMinute > 0 {
		apiKey.RateLimitPerMinute = req.RateLimitPerMinute
	}

	// 4. Шифруем полный ключ для хранения в БД
	encryptedKey, err := s.encryptor.EncryptAPIKey(fullKey)
	if err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to encrypt API key")
	}
	apiKey.SecretKey = &encryptedKey

	// 5. Проверяем, что ключ с таким именем не существует
	exists, err := s.apiKeyRepo.ExistsByName(ctx, apiKey.UserID, apiKey.Name)
	if err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to check API key existence")
	}

	if exists {
		return nil, "", model.ErrAPIKeyDuplicateName
	}

	// 6. Сохраняем в БД
	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to create API key")
	}

	// 7. Загружаем из БД для получения всех полей
	createdKey, err := s.apiKeyRepo.GetByID(ctx, apiKey.ID)
	if err != nil {
		span.RecordError(err)
		return nil, "", errors.Wrap(err, "failed to retrieve created API key")
	}

	return createdKey, fullKey, nil
}

// GetAPIKey возвращает API ключ по ID.
func (s *APIKeyService) GetAPIKey(ctx context.Context, id, userID uuid.UUID) (*model.APIKey, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.GetAPIKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("api_key_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get API key")
	}

	// Проверяем права доступа
	if apiKey.UserID != userID {
		return nil, model.ErrAPIKeyNotFound
	}

	return apiKey, nil
}

// ListAPIKeys возвращает список API ключей пользователя.
func (s *APIKeyService) ListAPIKeys(ctx context.Context, req *ListAPIKeysRequest) ([]*model.APIKey, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.ListAPIKeys")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID.String()),
		attribute.Int("page_size", req.PageSize),
	)

	// Применяем пагинацию
	limit := req.PageSize
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset := 0
	if req.PageToken != "" {
		// Page token - это base64-encoded offset
		// Формат: "offset:<number>" закодированный в base64
		decoded, err := base64.StdEncoding.DecodeString(req.PageToken)
		if err == nil {
			tokenStr := string(decoded)
			if len(tokenStr) > 7 && tokenStr[:7] == "offset:" {
				offsetStr := tokenStr[7:]
				if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
					offset = parsedOffset
				}
			}
		}
	}

	keys, err := s.apiKeyRepo.ListByUserID(ctx, req.UserID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to list API keys")
	}

	span.SetAttributes(attribute.Int("result_count", len(keys)))

	return keys, nil
}

// UpdateAPIKey обновляет API ключ.
func (s *APIKeyService) UpdateAPIKey(ctx context.Context, req *UpdateAPIKeyRequest) (*model.APIKey, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.UpdateAPIKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("api_key_id", req.ID.String()),
		attribute.String("user_id", req.UserID.String()),
	)

	// 1. Получаем существующий ключ
	apiKey, err := s.apiKeyRepo.GetByID(ctx, req.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get API key")
	}

	// 2. Проверяем права доступа
	if apiKey.UserID != req.UserID {
		return nil, model.ErrAPIKeyNotFound
	}

	// 3. Валидируем изменения
	if req.Name != nil {
		if err := validateAPIKeyName(*req.Name); err != nil {
			span.RecordError(err)
			return nil, err
		}
	}

	// 4. Применяем изменения
	req.ApplyToModel(apiKey)

	// 5. Обновляем в БД
	apiKey.UpdatedAt = time.Now()
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to update API key")
	}

	// 6. Загружаем обновлённый ключ
	updatedKey, err := s.apiKeyRepo.GetByID(ctx, apiKey.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to retrieve updated API key")
	}

	return updatedKey, nil
}

// DeleteAPIKey удаляет (отзывает) API ключ.
func (s *APIKeyService) DeleteAPIKey(ctx context.Context, id, userID uuid.UUID) error {
	ctx, span := tracer.Start(ctx, "APIKeyService.DeleteAPIKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("api_key_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	// 1. Проверяем, что ключ существует и принадлежит пользователю
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to get API key")
	}

	if apiKey.UserID != userID {
		return model.ErrAPIKeyNotFound
	}

	// 2. Удаляем
	if err := s.apiKeyRepo.Delete(ctx, id); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to delete API key")
	}

	// 3. Очищаем rate limiter
	s.rateLimiter.Reset(id.String())

	return nil
}

// RotateAPIKeySecret ротирует секретный ключ API ключа.
func (s *APIKeyService) RotateAPIKeySecret(ctx context.Context, req *RotateAPIKeySecretRequest) (*RotateAPIKeySecretResponse, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.RotateAPIKeySecret")
	defer span.End()

	span.SetAttributes(
		attribute.String("api_key_id", req.ID.String()),
		attribute.String("user_id", req.UserID.String()),
	)

	// 1. Получаем существующий ключ
	apiKey, err := s.apiKeyRepo.GetByID(ctx, req.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get API key")
	}

	// 2. Проверяем права доступа
	if apiKey.UserID != req.UserID {
		return nil, model.ErrAPIKeyNotFound
	}

	// 3. Ротируем секрет
	newFullKey, err := apiKey.RotateSecret()
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to rotate API key secret")
	}

	// 4. Шифруем новый секрет
	encryptedKey, err := s.encryptor.EncryptAPIKey(newFullKey)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to encrypt new API key secret")
	}
	apiKey.SecretKey = &encryptedKey

	// 5. Обновляем в БД
	apiKey.UpdatedAt = time.Now()
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to update API key")
	}

	// 6. Загружаем обновлённый ключ
	updatedKey, err := s.apiKeyRepo.GetByID(ctx, apiKey.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to retrieve updated API key")
	}

	return &RotateAPIKeySecretResponse{
		APIKey:     updatedKey,
		NewFullKey: newFullKey,
	}, nil
}

// GetAPIKeyStats возвращает статистику API ключа.
func (s *APIKeyService) GetAPIKeyStats(ctx context.Context, req *GetAPIKeyStatsRequest) (*APIKeyStatsDTO, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.GetAPIKeyStats")
	defer span.End()

	span.SetAttributes(
		attribute.String("api_key_id", req.ID.String()),
		attribute.String("user_id", req.UserID.String()),
	)

	apiKey, err := s.apiKeyRepo.GetByID(ctx, req.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get API key")
	}

	if apiKey.UserID != req.UserID {
		return nil, model.ErrAPIKeyNotFound
	}

	// Вычисляем среднюю задержку (упрощённо)
	var avgLatencyMs int32
	if apiKey.TotalRequests > 0 {
		// Примечание: Точный расчёт на основе логов использования требует
		// дополнительной аналитики. Сейчас используем approximation.
		avgLatencyMs = 100 // Placeholder
	}

	return APIKeyStatsFromModel(apiKey, avgLatencyMs), nil
}

// ValidateAPIKey валидирует API ключ для запроса (используется Gateway).
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, req *ValidateAPIKeyRequest) (*ValidateAPIKeyResponse, error) {
	ctx, span := tracer.Start(ctx, "APIKeyService.ValidateAPIKey")
	defer span.End()

	span.SetAttributes(
		attribute.String("endpoint", req.Endpoint),
		attribute.String("method", req.Method),
	)

	// 1. Валидируем формат ключа
	if err := model.ValidateAPIKey(req.APIKey); err != nil {
		return &ValidateAPIKeyResponse{
			Valid:     false,
			ErrorCode: "INVALID_API_KEY",
			ErrorMsg:  "Invalid API key format",
		}, nil
	}

	// 2. Вычисляем хеш ключа
	keyHash := model.HashAPIKey(req.APIKey)

	// 3. Ищем ключ в БД
	apiKey, err := s.apiKeyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		if errors.Is(err, interfaces.ErrAPIKeyNotFound) {
			return &ValidateAPIKeyResponse{
				Valid:     false,
				ErrorCode: "API_KEY_NOT_FOUND",
				ErrorMsg:  "API key not found",
			}, nil
		}
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get API key by hash")
	}

	// 4. Проверяем статус
	if !apiKey.IsValid() {
		errorCode := "API_KEY_DISABLED"
		errorMsg := "API key disabled"

		switch {
		case apiKey.IsExpired():
			errorCode = "API_KEY_EXPIRED"
			errorMsg = "API key expired"
		case apiKey.Status == model.APIKeyStatusInactive:
			errorCode = "API_KEY_DEACTIVATED"
			errorMsg = "API key deactivated"
		case apiKey.Status == model.APIKeyStatusMaintenance:
			errorCode = "API_KEY_UNDER_MAINTENANCE"
			errorMsg = "API key under maintenance"
		}

		return &ValidateAPIKeyResponse{
			Valid:     false,
			UserID:    apiKey.UserID,
			Scopes:    apiKey.Scopes,
			Status:    apiKey.Status,
			ErrorCode: errorCode,
			ErrorMsg:  errorMsg,
		}, nil
	}

	// 5. Проверяем IP whitelist
	if !apiKey.IsIPAllowed(req.IPAddress) {
		return &ValidateAPIKeyResponse{
			Valid:     false,
			UserID:    apiKey.UserID,
			Scopes:    apiKey.Scopes,
			Status:    apiKey.Status,
			ErrorCode: "IP_NOT_ALLOWED",
			ErrorMsg:  "IP address not allowed",
		}, nil
	}

	// 6. Проверяем rate limit
	if !s.rateLimiter.Allow(ctx, apiKey.ID.String(), apiKey.RateLimitPerMinute) {
		return &ValidateAPIKeyResponse{
			Valid:     false,
			UserID:    apiKey.UserID,
			Scopes:    apiKey.Scopes,
			Status:    apiKey.Status,
			ErrorCode: "RATE_LIMIT_EXCEEDED",
			ErrorMsg:  "Rate limit exceeded",
		}, nil
	}

	// 7. Ключ валиден
	return &ValidateAPIKeyResponse{
		Valid:  true,
		UserID: apiKey.UserID,
		Scopes: apiKey.Scopes,
		Status: apiKey.Status,
	}, nil
}

// LogAPIKeyUsage логирует использование API ключа.
func (s *APIKeyService) LogAPIKeyUsage(ctx context.Context, apiKeyID uuid.UUID, endpoint, method string, statusCode int, responseTimeMs int, ipAddress, userAgent string, requestID uuid.UUID, rateLimited bool) error {
	ctx, span := tracer.Start(ctx, "APIKeyService.LogAPIKeyUsage")
	defer span.End()

	span.SetAttributes(
		attribute.String("api_key_id", apiKeyID.String()),
		attribute.String("endpoint", endpoint),
		attribute.String("method", method),
		attribute.Int("status_code", statusCode),
	)

	log := model.NewAPIKeyUsageLog(apiKeyID, endpoint, method, statusCode)

	log.SetResponseDetails(responseTimeMs, ipAddress, userAgent, requestID)

	if rateLimited {
		log.MarkRateLimited()
	}

	if err := s.usageRepo.Create(ctx, log); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to create API key usage log")
	}

	return nil
}

// Helper функции

func validateAPIKeyName(name string) error {
	if name == "" {
		return model.ErrEmptyAPIKeyName
	}
	if len(name) > 255 {
		return model.ErrAPIKeyNameTooLong
	}
	return nil
}
