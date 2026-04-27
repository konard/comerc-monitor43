package webhook

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	httpclient "github.com/raul/monitor/backend/integration-service/internal/infrastructure/httpclient"
	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
	"github.com/raul/monitor/backend/integration-service/internal/service/webhook/dto"
)

// WebhookDeliveryService сервис для асинхронной доставки webhook.
type WebhookDeliveryService struct {
	webhookRepo  interfaces.WebhookRepository
	deliveryRepo interfaces.WebhookDeliveryRepository
	encryptor    *security.EncryptionService
	httpClient   *httpclient.Client
	config       *WebhookDeliveryConfig
	workers      *WorkerPool
	retryQueue   chan *model.WebhookDeliveryAttempt
	log          *slog.Logger
}

// WebhookDeliveryConfig конфигурация сервиса доставки.
type WebhookDeliveryConfig struct {
	MaxWorkers         int
	WorkerQueueSize    int
	MaxRetries         int
	RetryIntervals     []time.Duration
	RetryCheckInterval time.Duration
	HTTPTimeout        time.Duration
}

// NewWebhookDeliveryService создаёт новый WebhookDeliveryService.
func NewWebhookDeliveryService(
	webhookRepo interfaces.WebhookRepository,
	deliveryRepo interfaces.WebhookDeliveryRepository,
	encryptor *security.EncryptionService,
	config *WebhookDeliveryConfig,
) *WebhookDeliveryService {

	if config == nil {
		config = &WebhookDeliveryConfig{
			MaxWorkers:         10,
			WorkerQueueSize:    1000,
			MaxRetries:         3,
			RetryIntervals:     []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second},
			RetryCheckInterval: 30 * time.Second,
			HTTPTimeout:        10 * time.Second,
		}
	}

	service := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		encryptor:    encryptor,
		httpClient:   httpclient.NewClient(config.HTTPTimeout),
		config:       config,
		retryQueue:   make(chan *model.WebhookDeliveryAttempt, config.WorkerQueueSize),
		log:          slog.Default(),
	}

	// Создаём worker pool
	service.workers = NewWorkerPool(service, config.MaxWorkers, config.WorkerQueueSize)

	return service
}

// Start запускает сервис доставки.
func (s *WebhookDeliveryService) Start(ctx context.Context) error {
	s.log.InfoContext(ctx, "starting webhook delivery service",
		"max_workers", s.config.MaxWorkers,
		"max_retries", s.config.MaxRetries,
	)

	// Запускаем worker pool
	if err := s.workers.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start worker pool")
	}

	// Запускаем retry scheduler
	go s.retryScheduler(ctx)

	s.log.InfoContext(ctx, "webhook delivery service started")

	return nil
}

// Stop останавливает сервис доставки.
func (s *WebhookDeliveryService) Stop(ctx context.Context) error {
	s.log.InfoContext(ctx, "stopping webhook delivery service")

	// Останавливаем worker pool
	if err := s.workers.Stop(ctx); err != nil {
		return errors.Wrap(err, "failed to stop worker pool")
	}

	s.log.InfoContext(ctx, "webhook delivery service stopped")

	return nil
}

// ProcessAlertEvent обрабатывает событие алерта от Alert Service.
func (s *WebhookDeliveryService) ProcessAlertEvent(ctx context.Context, event *dto.AlertEvent) error {
	ctx, span := tracer.Start(ctx, "WebhookDeliveryService.ProcessAlertEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("alert_id", event.AlertID.String()),
		attribute.String("monitor_id", event.MonitorID.String()),
		attribute.String("severity", event.Severity),
	)

	s.log.InfoContext(ctx, "processing alert event",
		"alert_id", event.AlertID,
		"monitor_id", event.MonitorID,
		"severity", event.Severity,
	)

	// 1. Получаем активные webhooks для пользователя
	webhooks, err := s.webhookRepo.ListActiveByUserID(ctx, event.UserID)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to get webhooks for user")
	}

	if len(webhooks) == 0 {
		s.log.DebugContext(ctx, "no active webhooks for user", "user_id", event.UserID)
		return nil
	}

	// 2. Фильтруем по severity
	var filteredWebhooks []*model.WebhookIntegration
	for _, webhook := range webhooks {
		if webhook.ShouldDeliverForSeverity(event.Severity) {
			filteredWebhooks = append(filteredWebhooks, webhook)
		}
	}

	if len(filteredWebhooks) == 0 {
		s.log.DebugContext(ctx, "no webhooks matching severity", "severity", event.Severity)
		return nil
	}

	// 3. Сортируем по priority (high → normal → low)
	sortByPriority(filteredWebhooks)

	// 4. Создаём delivery attempts и отправляем в worker pool
	for _, webhook := range filteredWebhooks {
		attempt := model.NewWebhookDeliveryAttempt(webhook.ID, event.AlertID)

		// Сохраняем в БД
		if err := s.deliveryRepo.Create(ctx, attempt); err != nil {
			s.log.ErrorContext(ctx, "failed to create delivery attempt",
				"webhook_id", webhook.ID,
				"error", err,
			)
			continue
		}

		// Отправляем в worker pool
		job := &WebhookDeliveryJob{
			Webhook: webhook,
			Attempt: attempt,
			Event:   event,
		}

		if err := s.workers.Submit(ctx, job); err != nil {
			s.log.ErrorContext(ctx, "failed to submit webhook job",
				"webhook_id", webhook.ID,
				"error", err,
			)
			// Помечаем как failed
			attempt.MarkAsFailed("WORKER_QUEUE_FULL", "Failed to submit to worker pool")
			if updateErr := s.deliveryRepo.Update(ctx, attempt); updateErr != nil {
				s.log.ErrorContext(ctx, "failed to persist worker queue full status", "error", updateErr)
			}
		}
	}

	span.AddEvent("alert_event_processed", trace.WithAttributes(
		attribute.Int("webhooks_dispatched", len(filteredWebhooks)),
	))

	s.log.InfoContext(ctx, "alert event processed",
		"alert_id", event.AlertID,
		"webhooks_sent", len(filteredWebhooks),
	)

	return nil
}

// DeliverWebhook доставляет webhook (выполняется worker pool).
func (s *WebhookDeliveryService) DeliverWebhook(ctx context.Context, job *WebhookDeliveryJob) error {
	ctx, span := tracer.Start(ctx, "WebhookDeliveryService.DeliverWebhook")
	defer span.End()

	webhook := job.Webhook
	attempt := job.Attempt

	span.SetAttributes(
		attribute.String("webhook_id", webhook.ID.String()),
		attribute.Int("retry_count", attempt.RetryCount),
	)

	s.log.DebugContext(ctx, "delivering webhook",
		"webhook_id", webhook.ID,
		"alert_id", attempt.AlertID,
		"retry_count", attempt.RetryCount,
	)

	// 1. Подготавливаем payload
	payload := s.buildWebhookPayload(job.Event)

	// 2. Расшифровываем секретный ключ если есть
	var secretKey *string
	if webhook.SecretKey != nil {
		decryptedSecret, err := s.encryptor.DecryptWebhookSecret(*webhook.SecretKey)
		if err != nil {
			s.log.WarnContext(ctx, "failed to decrypt webhook secret",
				"webhook_id", webhook.ID,
				"error", err,
			)
			// Продолжаем без секретного ключа
			secretKey = nil
		} else {
			secretKey = &decryptedSecret
		}
	}

	// 3. Формируем запрос
	req := &httpclient.WebhookRequest{
		URL:                 webhook.URL,
		Method:              webhook.Method,
		Headers:             webhook.Headers,
		Payload:             payload,
		SecretKey:           secretKey,
		MaxPayloadSizeBytes: webhook.MaxPayloadSizeBytes,
		TruncateOnOverflow:  webhook.PayloadHandlingStrategy == model.PayloadHandlingStrategyTruncate,
	}

	// 4. Отправляем webhook
	startTime := time.Now()
	resp, err := s.httpClient.SendWebhook(ctx, req)
	responseTimeMs := int(time.Since(startTime).Milliseconds())

	if err != nil {
		span.RecordError(err)
		s.log.WarnContext(ctx, "webhook delivery failed",
			"webhook_id", webhook.ID,
			"alert_id", attempt.AlertID,
			"error", err,
		)

		// Проверяем, нужно ли retry
		if s.shouldRetry(err) {
			return s.scheduleRetry(ctx, webhook, attempt, err.Error(), err.Error())
		}

		// Permanent error - помечаем как failed
		attempt.MarkAsFailed("PERMANENT_ERROR", err.Error())
		if updateErr := s.deliveryRepo.Update(ctx, attempt); updateErr != nil {
			s.log.ErrorContext(ctx, "failed to persist permanent delivery failure", "error", updateErr)
		}

		// Обновляем статистику webhook
		s.updateWebhookFailureStats(ctx, webhook)

		return err
	}

	// 5. Успешная доставка
	attempt.MarkAsSent(resp.StatusCode, responseTimeMs)
	if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
		s.log.ErrorContext(ctx, "failed to update delivery attempt", "error", err)
	}

	// 6. Обновляем статистику webhook
	webhook.MarkAsSuccessful(responseTimeMs)
	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		s.log.ErrorContext(ctx, "failed to update webhook stats", "error", err)
	}

	span.AddEvent("webhook_delivered", trace.WithAttributes(
		attribute.Int("status_code", resp.StatusCode),
		attribute.Int("response_time_ms", responseTimeMs),
	))

	s.log.InfoContext(ctx, "webhook delivered successfully",
		"webhook_id", webhook.ID,
		"alert_id", attempt.AlertID,
		"status_code", resp.StatusCode,
		"response_time_ms", responseTimeMs,
	)

	return nil
}

// retryScheduler проверяет и планирует retry для неудачных попыток.
func (s *WebhookDeliveryService) retryScheduler(ctx context.Context) {
	ticker := time.NewTicker(s.config.RetryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processRetryQueue(ctx)
		}
	}
}

// processRetryQueue обрабатывает очередь retry.
func (s *WebhookDeliveryService) processRetryQueue(ctx context.Context) {
	// Получаем attempts, готовые для retry
	attempts, err := s.deliveryRepo.ListPendingForRetry(ctx, 100)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to list pending retries", "error", err)
		return
	}

	s.log.DebugContext(ctx, "processing retry queue", "count", len(attempts))

	now := time.Now()
	for _, attempt := range attempts {
		// Проверяем, что время retry наступило
		if attempt.NextRetryAt != nil && attempt.NextRetryAt.After(now) {
			continue
		}

		// Проверяем, что ещё не превышен лимит retry
		if attempt.RetryCount >= s.config.MaxRetries {
			// Помечаем как failed
			attempt.MarkAsFailed("MAX_RETRIES_EXCEEDED", "Exceeded maximum retry attempts")
			if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
				s.log.ErrorContext(ctx, "failed to update attempt", "error", err)
			}
			continue
		}

		// Получаем webhook
		webhook, err := s.webhookRepo.GetByID(ctx, attempt.WebhookID)
		if err != nil {
			s.log.ErrorContext(ctx, "failed to get webhook for retry",
				"webhook_id", attempt.WebhookID,
				"error", err,
			)
			continue
		}

		// Отправляем в retry
		attempt.RetryCount++
		attempt.Status = "pending"

		job := &WebhookDeliveryJob{
			Webhook: webhook,
			Attempt: attempt,
			Event:   nil, // Event не хранится в delivery attempt, требуется изменение структуры БД для восстановления
		}

		if err := s.workers.Submit(ctx, job); err != nil {
			s.log.ErrorContext(ctx, "failed to submit retry job", "error", err)
		}
	}
}

// scheduleRetry планирует retry для webhook.
func (s *WebhookDeliveryService) scheduleRetry(ctx context.Context, webhook *model.WebhookIntegration, attempt *model.WebhookDeliveryAttempt, errorCode, errorMessage string) error {
	// Вычисляем задержку
	retryIndex := attempt.RetryCount
	if retryIndex >= len(s.config.RetryIntervals) {
		retryIndex = len(s.config.RetryIntervals) - 1
	}

	delay := s.config.RetryIntervals[retryIndex]
	nextRetryAt := time.Now().Add(delay)

	// Обновляем attempt
	attempt.ScheduleRetry(nextRetryAt, attempt.RetryCount+1)
	attempt.MarkAsFailed(errorCode, errorMessage)

	if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
		return errors.Wrap(err, "failed to update delivery attempt")
	}

	s.log.InfoContext(ctx, "webhook retry scheduled",
		"webhook_id", webhook.ID,
		"alert_id", attempt.AlertID,
		"retry_count", attempt.RetryCount,
		"next_retry_at", nextRetryAt,
	)

	return nil
}

// shouldRetry проверяет, нужно ли retry на основе ошибки.
func (s *WebhookDeliveryService) shouldRetry(err error) bool {
	if errors.Is(err, model.ErrWebhookTimeout) {
		return true // Timeout - transient
	}

	if errors.Is(err, model.ErrWebhook5xxError) {
		return true // 5xx - transient
	}

	// Остальные ошибки - permanent
	return false
}

// updateWebhookFailureStats обновляет статистику при неудачной доставке.
func (s *WebhookDeliveryService) updateWebhookFailureStats(ctx context.Context, webhook *model.WebhookIntegration) {
	webhook.MarkAsFailed()
	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		s.log.ErrorContext(ctx, "failed to update webhook failure stats", "error", err)
	}
}

// buildWebhookPayload строит payload для webhook.
// Если event равен nil (например, при повторной доставке без сохранённого события), возвращает пустой payload.
func (s *WebhookDeliveryService) buildWebhookPayload(event *dto.AlertEvent) map[string]any {
	if event == nil {
		return map[string]any{}
	}

	payload := map[string]any{
		"alert_id":             event.AlertID.String(),
		"monitor_id":           event.MonitorID.String(),
		"monitor_name":         event.MonitorName,
		"monitor_status":       event.MonitorStatus,
		"severity":             event.Severity,
		"triggered_at":         event.TriggeredAt.Format(time.RFC3339),
		"consecutive_failures": event.ConsecutiveFailures,
		"is_flapping":          event.IsFlapping,
	}

	if event.IsFlapping {
		payload["flap_count"] = event.FlapCount
	}

	return payload
}

// sortByPriority сортирует webhooks по приоритету.
func sortByPriority(webhooks []*model.WebhookIntegration) {
	// Priority order: high > normal > low
	priorityOrder := map[string]int{
		string(model.WebhookPriorityHigh):   0,
		string(model.WebhookPriorityNormal): 1,
		string(model.WebhookPriorityLow):    2,
	}

	// Простая bubble sort (для небольших списков)
	n := len(webhooks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			p1 := priorityOrder[string(webhooks[j].Priority)]   //nolint:gosec // G602: j < n-i-1 гарантирует j+1 < n
			p2 := priorityOrder[string(webhooks[j+1].Priority)] //nolint:gosec // G602: j+1 < n по инварианту цикла

			if p1 > p2 {
				webhooks[j], webhooks[j+1] = webhooks[j+1], webhooks[j] //nolint:gosec // G602: индексы проверены выше
			}
		}
	}
}

// WebhookDeliveryJob задача для доставки webhook.
type WebhookDeliveryJob struct {
	Webhook *model.WebhookIntegration
	Attempt *model.WebhookDeliveryAttempt
	Event   *dto.AlertEvent
}
