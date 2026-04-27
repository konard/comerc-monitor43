package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/integration-service/internal/client"
	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
)

// tracer используется для трассировки операций webhook сервиса.
var tracer = otel.Tracer("github.com/raul/monitor/backend/integration-service/internal/service/webhook")

// Config конфигурация WebhookService.
type WebhookServiceConfig struct {
	DefaultMaxPayloadSizeBytes int
	DefaultTimeout             time.Duration
}

// WebhookService сервис для управления webhook интеграциями.
type WebhookService struct {
	repo          interfaces.WebhookRepository
	encryptor     *security.EncryptionService
	keyGenerator  *security.APIKeyGenerator
	billingClient *client.BillingClient
	config        *WebhookServiceConfig
}

// NewWebhookService создаёт новый WebhookService.
func NewWebhookService(
	webhookRepo interfaces.WebhookRepository,
	encryptor *security.EncryptionService,
	keyGenerator *security.APIKeyGenerator,
	billingClient *client.BillingClient,
	config *WebhookServiceConfig,
) *WebhookService {
	if config == nil {
		config = &WebhookServiceConfig{
			DefaultMaxPayloadSizeBytes: 1048576, // 1MB
			DefaultTimeout:             10 * time.Second,
		}
	}

	return &WebhookService{
		repo:          webhookRepo,
		encryptor:     encryptor,
		keyGenerator:  keyGenerator,
		billingClient: billingClient,
		config:        config,
	}
}

// CreateWebhook создаёт новую webhook интеграцию.
func (s *WebhookService) CreateWebhook(ctx context.Context, req *CreateWebhookRequest) (*model.WebhookIntegration, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.CreateWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID.String()),
		attribute.String("webhook_name", req.Name),
	)

	// 1. Конвертируем DTO в модель
	webhook, err := req.ToModel()
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to create webhook model")
	}

	// 2. Шифруем секретный ключ если указан
	if req.SecretKey != nil && *req.SecretKey != "" {
		encryptedSecret, err := s.encryptor.EncryptWebhookSecret(*req.SecretKey)
		if err != nil {
			span.RecordError(err)
			return nil, errors.Wrap(err, "failed to encrypt webhook secret")
		}
		webhook.SecretKey = &encryptedSecret
	}

	// 3. Проверяем, что webhook с таким именем не существует
	exists, err := s.repo.ExistsByName(ctx, webhook.UserID, webhook.Name)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to check webhook existence")
	}

	if exists {
		return nil, model.ErrWebhookDuplicateName
	}

	// 4. Проверяем лимит webhook перед сохранением
	currentCount, err := s.repo.CountByUserID(ctx, webhook.UserID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to count webhooks")
	}

	canCreate, err := s.billingClient.CheckWebhookLimit(ctx, webhook.UserID, currentCount)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to check webhook limit")
	}
	if !canCreate {
		return nil, model.ErrWebhookLimitReached
	}

	// 5. Сохраняем в БД
	if err := s.repo.Create(ctx, webhook); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to create webhook")
	}

	// 6. Загружаем из БД для получения всех полей
	createdWebhook, err := s.repo.GetByID(ctx, webhook.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to retrieve created webhook")
	}

	span.AddEvent("webhook_created", trace.WithAttributes(
		attribute.String("webhook_id", webhook.ID.String()),
	))

	return createdWebhook, nil
}

// GetWebhook возвращает webhook по ID.
func (s *WebhookService) GetWebhook(ctx context.Context, id, userID uuid.UUID) (*model.WebhookIntegration, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.GetWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	webhook, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	// Проверяем права доступа
	if webhook.UserID != userID {
		return nil, model.ErrWebhookNotFound
	}

	return webhook, nil
}

// ListWebhooks возвращает список webhooks пользователя.
func (s *WebhookService) ListWebhooks(ctx context.Context, req *ListWebhooksRequest) ([]*model.WebhookIntegration, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.ListWebhooks")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", req.UserID.String()),
		attribute.Int("page_size", req.PageSize),
	)

	// Применяем пагинацию (по умолчанию 50 на страницу)
	limit := req.PageSize
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Парсим page token для offset-based pagination
	offset := 0
	if req.PageToken != "" {
		// Page token format: base64("offset:N")
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

	webhooks, err := s.repo.ListByUserID(ctx, req.UserID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to list webhooks")
	}

	span.SetAttributes(attribute.Int("result_count", len(webhooks)))

	return webhooks, nil
}

// UpdateWebhook обновляет webhook.
func (s *WebhookService) UpdateWebhook(ctx context.Context, req *UpdateWebhookRequest) (*model.WebhookIntegration, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.UpdateWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", req.ID.String()),
		attribute.String("user_id", req.UserID.String()),
	)

	// 1. Получаем существующий webhook
	webhook, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	// 2. Проверяем права доступа
	if webhook.UserID != req.UserID {
		return nil, model.ErrWebhookNotFound
	}

	// 3. Применяем изменения
	req.ApplyToModel(webhook)

	// 4. Валидируем обновлённую модель
	if req.Name != nil {
		if err := validateWebhookName(*req.Name); err != nil {
			span.RecordError(err)
			return nil, err
		}
	}

	if req.URL != nil {
		if err := validateWebhookURL(*req.URL); err != nil {
			span.RecordError(err)
			return nil, err
		}
	}

	// 5. Обновляем в БД
	webhook.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, webhook); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to update webhook")
	}

	// 6. Загружаем обновлённый webhook
	updatedWebhook, err := s.repo.GetByID(ctx, webhook.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to retrieve updated webhook")
	}

	return updatedWebhook, nil
}

// DeleteWebhook удаляет webhook.
func (s *WebhookService) DeleteWebhook(ctx context.Context, id, userID uuid.UUID) error {
	ctx, span := tracer.Start(ctx, "WebhookService.DeleteWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	// 1. Проверяем, что webhook существует и принадлежит пользователю
	webhook, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to get webhook")
	}

	if webhook.UserID != userID {
		return model.ErrWebhookNotFound
	}

	// 2. Удаляем
	if err := s.repo.Delete(ctx, id); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to delete webhook")
	}

	return nil
}

// EnableWebhook включает webhook.
func (s *WebhookService) EnableWebhook(ctx context.Context, id, userID uuid.UUID) error {
	ctx, span := tracer.Start(ctx, "WebhookService.EnableWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	// 1. Получаем webhook
	webhook, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to get webhook")
	}

	// 2. Проверяем права доступа
	if webhook.UserID != userID {
		return model.ErrWebhookNotFound
	}

	webhook.MarkAsActive()
	if err := s.repo.Update(ctx, webhook); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to enable webhook")
	}

	return nil
}

// DisableWebhook отключает webhook.
func (s *WebhookService) DisableWebhook(ctx context.Context, id, userID uuid.UUID) error {
	ctx, span := tracer.Start(ctx, "WebhookService.DisableWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	// 1. Получаем webhook
	webhook, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to get webhook")
	}

	// 2. Проверяем права доступа
	if webhook.UserID != userID {
		return model.ErrWebhookNotFound
	}

	webhook.MarkAsDisabled()
	if err := s.repo.Update(ctx, webhook); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to disable webhook")
	}

	return nil
}

// TestWebhook тестирует webhook endpoint путём отправки тестового запроса.
func (s *WebhookService) TestWebhook(ctx context.Context, id, userID uuid.UUID) (*TestWebhookResponse, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.TestWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	// 1. Verify webhook exists and belongs to user
	webhook, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	if webhook.UserID != userID {
		return nil, model.ErrWebhookNotFound
	}

	// 2. Validate webhook is active
	if webhook.Status != model.WebhookStatusActive {
		return &TestWebhookResponse{
			Success:        false,
			StatusCode:     0,
			ResponseTimeMs: 0,
			ErrorMessage:   "webhook is not active",
		}, nil
	}

	// 3. Создаём HTTP клиент с таймаутом
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 4. Формируем тестовый payload
	testPayload := map[string]any{
		"test":       true,
		"timestamp":  time.Now().Unix(),
		"webhook_id": webhook.ID.String(),
		"message":    "Test webhook delivery",
	}
	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal test payload")
	}

	// 5. Создаём HTTP request
	req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return &TestWebhookResponse{
			Success:        false,
			StatusCode:     0,
			ResponseTimeMs: 0,
			ErrorMessage:   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	// 6. Устанавливаем headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Monitor-Integration-Service/1.0 Test")
	for k, v := range webhook.Headers {
		req.Header.Set(k, v)
	}

	// 7. Отправляем запрос и замеряем время
	startTime := time.Now()
	resp, err := httpClient.Do(req)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		return &TestWebhookResponse{
			Success:        false,
			StatusCode:     0,
			ResponseTimeMs: int(responseTime),
			ErrorMessage:   fmt.Sprintf("request failed: %v", err),
		}, nil
	}

	// 8. Читаем response body
	body, err := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		slog.Default().WarnContext(ctx, "failed to close test webhook response body", "error", closeErr)
	}
	if err != nil {
		return &TestWebhookResponse{
			Success:        false,
			StatusCode:     resp.StatusCode,
			ResponseTimeMs: int(responseTime),
			ErrorMessage:   fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	// 9. Проверяем статус
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	errorMsg := ""
	if !success {
		errorMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return &TestWebhookResponse{
		Success:        success,
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: int(responseTime),
		ErrorMessage:   errorMsg,
	}, nil
}

// CloneWebhook клонирует webhook.
func (s *WebhookService) CloneWebhook(ctx context.Context, req *CloneWebhookRequest) (*model.WebhookIntegration, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.CloneWebhook")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", req.ID.String()),
		attribute.String("user_id", req.UserID.String()),
		attribute.String("new_name", req.NewName),
	)

	// 1. Получаем оригинальный webhook
	original, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	if original.UserID != req.UserID {
		return nil, model.ErrWebhookNotFound
	}

	// 2. Создаём новый webhook на основе оригинала
	webhook := &model.WebhookIntegration{
		ID:                      uuid.New(),
		UserID:                  original.UserID,
		Name:                    req.NewName,
		URL:                     req.NewURL,
		Method:                  original.Method,
		Headers:                 original.Headers,
		SecretKey:               original.SecretKey,
		Enabled:                 original.Enabled,
		Status:                  original.Status,
		Priority:                original.Priority,
		SeverityFilter:          original.SeverityFilter,
		MaxPayloadSizeBytes:     original.MaxPayloadSizeBytes,
		PayloadHandlingStrategy: original.PayloadHandlingStrategy,
		TotalSent:               0,
		SuccessfulSent:          0,
		FailedSent:              0,
		FailureCount:            0,
		ConsecutiveFailures:     0,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	// 3. Сохраняем клон
	if err := s.repo.Create(ctx, webhook); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to create webhook clone")
	}

	span.AddEvent("webhook_cloned", trace.WithAttributes(
		attribute.String("new_webhook_id", webhook.ID.String()),
	))

	return webhook, nil
}

// GetWebhookStats возвращает статистику webhook.
func (s *WebhookService) GetWebhookStats(ctx context.Context, id, userID uuid.UUID) (*WebhookStatsDTO, error) {
	ctx, span := tracer.Start(ctx, "WebhookService.GetWebhookStats")
	defer span.End()

	span.SetAttributes(
		attribute.String("webhook_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	// NOTE: Full stats tracking requires delivery history in repository
	// For now, return webhook's built-in statistics

	webhook, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	if webhook.UserID != userID {
		return nil, model.ErrWebhookNotFound
	}

	successRate := 0.0
	if webhook.TotalSent > 0 {
		successRate = float64(webhook.TotalSent-webhook.FailureCount) / float64(webhook.TotalSent) * 100
	}

	return &WebhookStatsDTO{
		TotalSent:         webhook.TotalSent,
		SuccessfulSent:    webhook.TotalSent - webhook.FailureCount,
		FailedSent:        webhook.FailureCount,
		SuccessRate:       successRate,
		AvgResponseTimeMs: derefInt(webhook.AvgResponseTimeMs),
		LastSentAt:        timeToInt64Ptr(webhook.LastSentAt),
		LastSuccessAt:     timeToInt64Ptr(webhook.LastSuccessAt),
		LastFailureAt:     timeToInt64Ptr(webhook.LastFailureAt),
		FailureCount:      webhook.FailureCount,
	}, nil
}

// Helper functions for dereferencing pointers
func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func timeToInt64Ptr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	unix := t.Unix()
	return &unix
}

// Helper функции (копируем из model пакета)
func validateWebhookName(name string) error {
	if name == "" {
		return model.ErrEmptyWebhookName
	}

	if len(name) > 255 {
		return model.ErrWebhookNameTooLong
	}

	return nil
}

func validateWebhookURL(webhookURL string) error {
	if webhookURL == "" {
		return errors.New("webhook URL cannot be empty")
	}

	// Базовая проверка формата URL
	if !regexp.MustCompile(`^https?://`).MatchString(webhookURL) {
		return errors.New("webhook URL must start with http:// or https://")
	}

	// Проверка длины URL (RFC 7230 recommends reasonable limits)
	if len(webhookURL) > 2048 {
		return errors.New("webhook URL too long (max 2048 characters)")
	}

	return nil
}
