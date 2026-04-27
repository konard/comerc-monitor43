package delivery

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/channels"
	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

var (
	reSecretPattern = regexp.MustCompile(`(?i)(key|token|secret|auth)[=:]\s*(\S+)`)
	reWebhookError  = regexp.MustCompile(`webhook returned status (\d+): (.+)`)
)

const (
	defaultMaxRetries    = 3
	defaultRetryInterval = 5 * time.Minute

	RateLimitPerMonitorPerStatus    = 5 * time.Minute
	maxDeliveriesPerRateLimitWindow = 3 // максимальное число доставок в одно окно для одного монитора
	maxChannelFailures              = 5
)

// AlertDeliveryService управляет доставкой уведомлений
type AlertDeliveryService struct {
	deliveryRepo       repository.DeliveryAttemptRepository
	channelRepo        repository.AlertChannelRepository
	alertRepo          repository.AlertRepository
	maintenanceRepo    repository.MaintenanceWindowRepository
	telegramClient     *channels.TelegramClient
	emailClient        *channels.EmailClient
	webhookClient      *channels.WebhookClient
	maxRetries         int
	retryInterval      time.Duration
	rateLimitWindow    time.Duration
	rateLimitRetryWait time.Duration
	retryMaxDelay      time.Duration
	retryBackoffBase   time.Duration
	tracer             trace.Tracer
	metrics            *apptelemetry.Metrics
}

// ServiceOption определяет опцию для конфигурации сервиса
type ServiceOption func(*AlertDeliveryService)

// WithMaxRetries устанавливает максимальное количество retry
func WithMaxRetries(maxRetries int) ServiceOption {
	return func(s *AlertDeliveryService) {
		s.maxRetries = maxRetries
	}
}

// WithRetryInterval устанавливает интервал между retry
func WithRetryInterval(interval time.Duration) ServiceOption {
	return func(s *AlertDeliveryService) {
		s.retryInterval = interval
	}
}

// WithMaintenanceRepo добавляет репозиторий окон обслуживания
func WithMaintenanceRepo(repo repository.MaintenanceWindowRepository) ServiceOption {
	return func(s *AlertDeliveryService) {
		s.maintenanceRepo = repo
	}
}

// WithRateLimitWindow задаёт окно для подсчёта доставок per-monitor (rate limiting).
func WithRateLimitWindow(window time.Duration) ServiceOption {
	return func(s *AlertDeliveryService) {
		if window > 0 {
			s.rateLimitWindow = window
		}
	}
}

// WithRateLimitRetryWait задаёт фиксированную задержку retry для rate-limit ошибок.
func WithRateLimitRetryWait(d time.Duration) ServiceOption {
	return func(s *AlertDeliveryService) {
		if d > 0 {
			s.rateLimitRetryWait = d
		}
	}
}

// WithRetryMaxDelay ограничивает максимальную задержку exponential backoff.
func WithRetryMaxDelay(d time.Duration) ServiceOption {
	return func(s *AlertDeliveryService) {
		if d > 0 {
			s.retryMaxDelay = d
		}
	}
}

// WithRetryBackoffBase переопределяет базовую задержку exponential backoff,
// игнорируя BaseDelay из классификации ошибки.
func WithRetryBackoffBase(d time.Duration) ServiceOption {
	return func(s *AlertDeliveryService) {
		if d > 0 {
			s.retryBackoffBase = d
		}
	}
}

// NewAlertDeliveryService создаёт новый сервис доставки
func NewAlertDeliveryService(
	deliveryRepo repository.DeliveryAttemptRepository,
	channelRepo repository.AlertChannelRepository,
	alertRepo repository.AlertRepository,
	telegramClient *channels.TelegramClient,
	emailClient *channels.EmailClient,
	webhookClient *channels.WebhookClient,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
	opts ...ServiceOption,
) *AlertDeliveryService {
	service := &AlertDeliveryService{
		deliveryRepo:       deliveryRepo,
		channelRepo:        channelRepo,
		alertRepo:          alertRepo,
		telegramClient:     telegramClient,
		emailClient:        emailClient,
		webhookClient:      webhookClient,
		maxRetries:         defaultMaxRetries,
		retryInterval:      defaultRetryInterval,
		rateLimitWindow:    RateLimitPerMonitorPerStatus,
		rateLimitRetryWait: 5 * time.Minute,
		retryMaxDelay:      10 * time.Minute,
		tracer:             tracer,
		metrics:            metrics,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service
}

// DeliveryResult содержит агрегированный результат доставки по всем каналам
type DeliveryResult struct {
	SuccessfulDeliveries int
	FailedDeliveries     int
	TotalChannels        int
	ChannelResults       map[uuid.UUID]*ChannelDeliveryResult
}

// ChannelDeliveryResult содержит результат доставки по одному каналу
type ChannelDeliveryResult struct {
	ChannelID   uuid.UUID
	ChannelType string
	Status      string
	Error       string
	MessageID   string
}

// DeliverToAllChannels доставляет уведомление параллельно по всем каналам
func (s *AlertDeliveryService) DeliverToAllChannels(
	ctx context.Context,
	alert *model.Alert,
	alertChannels []*model.AlertChannel,
) *DeliveryResult {
	ctx, span := s.tracer.Start(ctx, "AlertDeliveryService.DeliverToAllChannels")
	defer span.End()

	span.SetAttributes(
		attribute.String("alert_id", alert.ID.String()),
		attribute.Int("channel_count", len(alertChannels)),
	)

	result := &DeliveryResult{
		TotalChannels:  len(alertChannels),
		ChannelResults: make(map[uuid.UUID]*ChannelDeliveryResult, len(alertChannels)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, ch := range alertChannels {
		wg.Add(1)
		go func(ch *model.AlertChannel) {
			defer wg.Done()

			channelResult := &ChannelDeliveryResult{
				ChannelID:   ch.ID,
				ChannelType: string(ch.Type),
			}

			err := s.DeliverAlert(ctx, alert, ch)
			if err != nil {
				channelResult.Status = "failed"
				channelResult.Error = maskErrorSecrets(err.Error())
				mu.Lock()
				result.FailedDeliveries++
				mu.Unlock()
			} else {
				channelResult.Status = "success"
				mu.Lock()
				result.SuccessfulDeliveries++
				mu.Unlock()
			}

			mu.Lock()
			result.ChannelResults[ch.ID] = channelResult
			mu.Unlock()
		}(ch)
	}

	wg.Wait()

	span.SetAttributes(
		attribute.Int("successful", result.SuccessfulDeliveries),
		attribute.Int("failed", result.FailedDeliveries),
	)

	return result
}

// DeliverAlert доставляет уведомление об алерте
func (s *AlertDeliveryService) DeliverAlert(
	ctx context.Context,
	alert *model.Alert,
	channel *model.AlertChannel,
) error {
	ctx, span := s.tracer.Start(ctx, "AlertDeliveryService.DeliverAlert")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("alert_id", alert.ID.String()),
			attribute.String("user_id", alert.UserID.String()),
			attribute.String("monitor_id", alert.MonitorID.String()),
			attribute.String("channel_id", channel.ID.String()),
			attribute.String("channel_type", string(channel.Type)),
		}...,
	)

	if !channel.Enabled {
		err := errors.New("channel is disabled")
		span.RecordError(err)
		return err
	}

	if !channel.Verified {
		err := errors.New("channel is not verified")
		span.RecordError(err)
		return err
	}

	if s.maintenanceRepo != nil {
		underMaintenance, err := s.maintenanceRepo.IsUnderMaintenance(ctx, alert.MonitorID.String())
		if err != nil {
			slog.Default().Warn("failed to check maintenance status, proceeding with delivery",
				"error", err,
				"monitor_id", alert.MonitorID.String(),
			)
		} else if underMaintenance {
			span.AddEvent("delivery_suppressed_maintenance", trace.WithAttributes(
				attribute.String("monitor_id", alert.MonitorID.String()),
			))
			suppressedAttempt := &model.DeliveryAttempt{
				ID:             uuid.New(),
				AlertID:        alert.ID,
				AlertChannelID: channel.ID,
				Status:         model.DeliveryAttemptStatusSuppressed,
				ErrorMessage:   "suppressed: monitor is under maintenance",
				RetryCount:     0,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := s.deliveryRepo.Create(ctx, suppressedAttempt); err != nil {
				slog.Default().Warn("failed to create suppressed delivery attempt", "error", err)
			}
			return nil
		}
	}

	if s.isRateLimited(ctx, alert) {
		err := errors.New("rate limited: too many deliveries for this monitor")
		span.RecordError(err)
		return err
	}

	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        alert.ID,
		AlertChannelID: channel.ID,
		Status:         model.DeliveryAttemptStatusPending,
		RetryCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.deliveryRepo.Create(ctx, attempt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create delivery attempt: %v", err)
	}

	span.AddEvent("delivery_attempt_created", trace.WithAttributes(
		attribute.String("attempt_id", attempt.ID.String()),
	))

	message := s.formatAlertMessage(alert, false)

	var deliveryErr error
	switch channel.Type {
	case model.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			deliveryErr = s.emailClient.Send(ctx, channel.EmailConfig.Email, "Monitor Alert: "+alert.MonitorID.String(), message)
		} else {
			deliveryErr = errors.New("email config is missing")
		}

	case model.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			deliveryErr = s.telegramClient.Send(ctx, channel.TelegramConfig.ChatID, message)
		} else {
			deliveryErr = errors.New("telegram config is missing")
		}

	case model.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			payload := map[string]any{
				"alert_id":   alert.ID.String(),
				"user_id":    alert.UserID.String(),
				"monitor_id": alert.MonitorID.String(),
				"status":     string(alert.Status),
				"message":    message,
				"created_at": time.Now().Format(time.RFC3339),
			}
			if len(channel.WebhookConfig.Headers) > 0 {
				deliveryErr = s.webhookClient.SendWithHeaders(ctx, channel.WebhookConfig.URL, payload, channel.WebhookConfig.Headers)
			} else {
				deliveryErr = s.webhookClient.Send(ctx, channel.WebhookConfig.URL, payload)
			}
		} else {
			deliveryErr = errors.New("webhook config is missing")
		}

	default:
		deliveryErr = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if deliveryErr != nil {
		span.RecordError(deliveryErr)

		classifiedErr := s.classifyError(deliveryErr)
		attempt.Status = model.DeliveryAttemptStatusFailed
		attempt.ErrorMessage = fmt.Sprintf("[%s] %s", classifiedErr.Category, classifiedErr.Message)

		span.AddEvent("delivery_failed", trace.WithAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("error_category", string(classifiedErr.Category)),
			attribute.String("error", maskErrorSecrets(deliveryErr.Error())),
		))

		if s.metrics != nil {
			s.metrics.RecordDeliveryFailed(ctx, string(channel.Type), string(classifiedErr.Category))
		}

		s.handleChannelFailure(ctx, channel, classifiedErr)
	} else {
		attempt.Status = model.DeliveryAttemptStatusSuccess
		span.AddEvent("delivery_success", trace.WithAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
		))

		if s.metrics != nil {
			s.metrics.RecordDeliverySuccess(ctx, string(channel.Type))
		}
	}

	attempt.UpdatedAt = time.Now()

	if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update delivery attempt: %v", err)
	}

	return nil
}

// formatAlertMessage форматирует сообщение об алерте
func (s *AlertDeliveryService) formatAlertMessage(alert *model.Alert, isRecovery bool) string {
	if isRecovery {
		return fmt.Sprintf("🟢 Monitor %s is back to normal", alert.MonitorID.String())
	}

	var message string
	switch alert.Status {
	case model.AlertStatusTriggered:
		message = fmt.Sprintf("🔴 Monitor %s is DOWN", alert.MonitorID.String())
	case model.AlertStatusResolved:
		message = fmt.Sprintf("🟢 Monitor %s is back to normal", alert.MonitorID.String())
	default:
		message = fmt.Sprintf("⚠️ Monitor %s status changed to %s", alert.MonitorID.String(), alert.Status)
	}

	return message
}

// DeliverRecovery доставляет уведомление о восстановлении
func (s *AlertDeliveryService) DeliverRecovery(
	ctx context.Context,
	alert *model.Alert,
	channel *model.AlertChannel,
) error {
	ctx, span := s.tracer.Start(ctx, "AlertDeliveryService.DeliverRecovery")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("alert_id", alert.ID.String()),
			attribute.String("user_id", alert.UserID.String()),
			attribute.String("monitor_id", alert.MonitorID.String()),
			attribute.String("channel_id", channel.ID.String()),
			attribute.String("channel_type", string(channel.Type)),
		}...,
	)

	if !channel.Enabled {
		err := errors.New("channel is disabled")
		span.RecordError(err)
		return err
	}

	if !channel.Verified {
		err := errors.New("channel is not verified")
		span.RecordError(err)
		return err
	}

	attempt := &model.DeliveryAttempt{
		ID:             uuid.New(),
		AlertID:        alert.ID,
		AlertChannelID: channel.ID,
		Status:         model.DeliveryAttemptStatusPending,
		RetryCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.deliveryRepo.Create(ctx, attempt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create delivery attempt: %v", err)
	}

	span.AddEvent("recovery_attempt_created", trace.WithAttributes(
		attribute.String("attempt_id", attempt.ID.String()),
	))

	message := s.formatAlertMessage(alert, true)

	var deliveryErr error
	switch channel.Type {
	case model.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			deliveryErr = s.emailClient.Send(ctx, channel.EmailConfig.Email, "Monitor Recovered: "+alert.MonitorID.String(), message)
		} else {
			deliveryErr = errors.New("email config is missing")
		}

	case model.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			deliveryErr = s.telegramClient.Send(ctx, channel.TelegramConfig.ChatID, message)
		} else {
			deliveryErr = errors.New("telegram config is missing")
		}

	case model.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			payload := map[string]any{
				"alert_id":    alert.ID.String(),
				"user_id":     alert.UserID.String(),
				"monitor_id":  alert.MonitorID.String(),
				"status":      "resolved",
				"message":     message,
				"recovery":    true,
				"resolved_at": time.Now().Format(time.RFC3339),
			}
			if len(channel.WebhookConfig.Headers) > 0 {
				deliveryErr = s.webhookClient.SendWithHeaders(ctx, channel.WebhookConfig.URL, payload, channel.WebhookConfig.Headers)
			} else {
				deliveryErr = s.webhookClient.Send(ctx, channel.WebhookConfig.URL, payload)
			}
		} else {
			deliveryErr = errors.New("webhook config is missing")
		}

	default:
		deliveryErr = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if deliveryErr != nil {
		span.RecordError(deliveryErr)

		classifiedErr := s.classifyError(deliveryErr)
		attempt.Status = model.DeliveryAttemptStatusFailed
		attempt.ErrorMessage = fmt.Sprintf("[%s] %s", classifiedErr.Category, classifiedErr.Message)

		span.AddEvent("recovery_delivery_failed", trace.WithAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("error_category", string(classifiedErr.Category)),
			attribute.String("error", maskErrorSecrets(deliveryErr.Error())),
		))

		if s.metrics != nil {
			s.metrics.RecordDeliveryFailed(ctx, string(channel.Type), string(classifiedErr.Category))
		}

		s.handleChannelFailure(ctx, channel, classifiedErr)
	} else {
		attempt.Status = model.DeliveryAttemptStatusSuccess
		span.AddEvent("recovery_delivery_success", trace.WithAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
		))

		if s.metrics != nil {
			s.metrics.RecordDeliverySuccess(ctx, string(channel.Type))
		}
	}

	attempt.UpdatedAt = time.Now()

	if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update delivery attempt: %v", err)
	}

	return nil
}

// ProcessPendingAttempts обрабатывает pending попытки доставки
func (s *AlertDeliveryService) ProcessPendingAttempts(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "AlertDeliveryService.ProcessPendingAttempts")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.Int("max_attempts", 100),
		}...,
	)

	attempts, err := s.deliveryRepo.ListPending(ctx, 100)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to list pending attempts: %v", err)
	}

	span.AddEvent("pending_attempts_found", trace.WithAttributes(
		attribute.Int("count", len(attempts)),
	))

	for _, attempt := range attempts {
		if err := s.processAttempt(ctx, attempt); err != nil {
			span.AddEvent("attempt_failed", trace.WithAttributes(
				attribute.String("attempt_id", attempt.ID.String()),
				attribute.String("error", err.Error()),
			))
			continue
		}
	}

	return nil
}

// processAttempt обрабатывает одну попытку доставки
func (s *AlertDeliveryService) processAttempt(
	ctx context.Context,
	attempt *model.DeliveryAttempt,
) error {
	ctx, span := s.tracer.Start(ctx, "AlertDeliveryService.processAttempt")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("alert_id", attempt.AlertID.String()),
			attribute.String("alert_channel_id", attempt.AlertChannelID.String()),
			attribute.String("status", string(attempt.Status)),
			attribute.Int("retry_count", attempt.RetryCount),
		}...,
	)

	alert, err := s.alertRepo.GetByID(ctx, attempt.AlertID.String())
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get alert: %v", err)
	}

	channel, err := s.channelRepo.GetByID(ctx, attempt.AlertChannelID.String())
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get channel: %v", err)
	}

	message := s.formatAlertMessage(alert, false)

	var deliveryErr error
	switch channel.Type {
	case model.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			deliveryErr = s.emailClient.Send(ctx, channel.EmailConfig.Email, "Monitor Alert: "+alert.MonitorID.String(), message)
		} else {
			deliveryErr = errors.New("email config is missing")
		}

	case model.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			deliveryErr = s.telegramClient.Send(ctx, channel.TelegramConfig.ChatID, message)
		} else {
			deliveryErr = errors.New("telegram config is missing")
		}

	case model.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			payload := map[string]any{
				"alert_id":   alert.ID.String(),
				"user_id":    alert.UserID.String(),
				"monitor_id": alert.MonitorID.String(),
				"status":     string(alert.Status),
				"message":    message,
				"created_at": time.Now().Format(time.RFC3339),
			}
			if len(channel.WebhookConfig.Headers) > 0 {
				deliveryErr = s.webhookClient.SendWithHeaders(ctx, channel.WebhookConfig.URL, payload, channel.WebhookConfig.Headers)
			} else {
				deliveryErr = s.webhookClient.Send(ctx, channel.WebhookConfig.URL, payload)
			}
		} else {
			deliveryErr = errors.New("webhook config is missing")
		}

	default:
		deliveryErr = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if deliveryErr != nil {
		span.RecordError(deliveryErr)

		classifiedErr := s.classifyError(deliveryErr)
		attempt.Status = model.DeliveryAttemptStatusFailed
		attempt.ErrorMessage = fmt.Sprintf("[%s] %s", classifiedErr.Category, classifiedErr.Message)

		span.AddEvent("delivery_attempt_failed", trace.WithAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.String("error_category", string(classifiedErr.Category)),
			attribute.String("error", maskErrorSecrets(deliveryErr.Error())),
		))

		if s.metrics != nil {
			s.metrics.RecordDeliveryFailed(ctx, string(channel.Type), string(classifiedErr.Category))
		}

		switch {
		case !classifiedErr.Retryable:
			span.AddEvent("error_not_retryable", trace.WithAttributes(
				attribute.String("attempt_id", attempt.ID.String()),
				attribute.String("category", string(classifiedErr.Category)),
			))
		case attempt.RetryCount >= s.maxRetries || attempt.RetryCount >= classifiedErr.MaxRetries:
			span.AddEvent("max_retries_reached", trace.WithAttributes(
				attribute.String("attempt_id", attempt.ID.String()),
				attribute.Int("retry_count", attempt.RetryCount),
			))
		default:
			if err := s.ScheduleRetry(ctx, attempt, classifiedErr); err != nil {
				span.RecordError(err)
				return fmt.Errorf("failed to schedule retry: %v", err)
			}
		}

		s.handleChannelFailure(ctx, channel, classifiedErr)
	} else {
		attempt.Status = model.DeliveryAttemptStatusSuccess
		span.AddEvent("delivery_attempt_success", trace.WithAttributes(
			attribute.String("attempt_id", attempt.ID.String()),
		))

		if s.metrics != nil {
			s.metrics.RecordDeliverySuccess(ctx, string(channel.Type))
		}
	}

	attempt.UpdatedAt = time.Now()

	if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update delivery attempt: %v", err)
	}

	return nil
}

// ScheduleRetry планирует retry для неудачной доставки с учётом классификации ошибки
func (s *AlertDeliveryService) ScheduleRetry(
	ctx context.Context,
	attempt *model.DeliveryAttempt,
	classifiedErr *model.DeliveryError,
) error {
	ctx, span := s.tracer.Start(ctx, "AlertDeliveryService.ScheduleRetry")
	defer span.End()

	span.SetAttributes(
		[]attribute.KeyValue{
			attribute.String("attempt_id", attempt.ID.String()),
			attribute.Int("current_retry_count", attempt.RetryCount),
			attribute.Int("max_retries", s.maxRetries),
			attribute.String("error_category", string(classifiedErr.Category)),
		}...,
	)

	attempt.RetryCount++
	if attempt.RetryCount >= s.maxRetries || attempt.RetryCount >= classifiedErr.MaxRetries {
		attempt.Status = model.DeliveryAttemptStatusFailed
		span.AddEvent("attempt_failed_max_retries_reached")
	} else {
		var delay time.Duration
		switch classifiedErr.Category {
		case model.DeliveryErrorRateLimit:
			// фиксированная задержка для rate limit (без экспоненциального роста)
			delay = s.rateLimitRetryWait
			if delay <= 0 {
				delay = 5 * time.Minute
			}
		case model.DeliveryErrorTransient, model.DeliveryErrorTimeout:
			base := classifiedErr.BaseDelay
			if s.retryBackoffBase > 0 {
				base = s.retryBackoffBase
			}
			delay = s.calculateRetryDelay(attempt.RetryCount, base)
		default:
			delay = s.retryInterval
		}

		nextRetry := time.Now().Add(delay)
		attempt.NextRetryAt = &nextRetry
		span.AddEvent("retry_scheduled", trace.WithAttributes(
			attribute.Int("retry_count", attempt.RetryCount),
			attribute.String("next_retry_at", nextRetry.Format(time.RFC3339)),
			attribute.String("delay", delay.String()),
		))
	}

	if err := s.deliveryRepo.Update(ctx, attempt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update delivery attempt: %v", err)
	}

	return nil
}

// calculateRetryDelay вычисляет задержку для retry с exponential backoff и jitter
func (s *AlertDeliveryService) calculateRetryDelay(attempt int, baseDelay time.Duration) time.Duration {
	maxDelay := s.retryMaxDelay
	if maxDelay <= 0 {
		maxDelay = 10 * time.Minute
	}
	factor := float64(int(1) << uint(attempt)) // #nosec G115 -- attempt bounded by maxRetries (≤3)
	delay := time.Duration(float64(baseDelay) * factor)
	if delay > maxDelay {
		delay = maxDelay
	}
	// Генерация криптографически безопасного jitter
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return delay
	}
	jitterFraction := float64(n.Int64())/1000.0*0.4 - 0.2
	jitter := time.Duration(float64(delay) * jitterFraction)
	return delay + jitter
}

// classifyError классифицирует ошибку доставки
func (s *AlertDeliveryService) classifyError(err error) *model.DeliveryError {
	if statusCode, body, ok := parseWebhookHTTPError(err); ok {
		return model.ClassifyHTTPErr(statusCode, body)
	}
	return model.ClassifyNetworkErr(err)
}

// handleChannelFailure обрабатывает неудачную доставку: увеличивает счётчик, при необходимости отключает канал
func (s *AlertDeliveryService) handleChannelFailure(ctx context.Context, channel *model.AlertChannel, classifiedErr *model.DeliveryError) {
	// Ошибки авторизации (401/403) отключают канал немедленно без счётчика неудач
	if classifiedErr.Category == model.DeliveryErrorAuth {
		if err := s.channelRepo.DisableChannel(ctx, channel.ID.String(), fmt.Sprintf("auto-disabled: auth error (status %d)", classifiedErr.StatusCode)); err != nil {
			slog.Default().Warn("failed to disable channel", "channel_id", channel.ID.String(), "error", err)
		}
		return
	}

	failureCount, err := s.channelRepo.IncrementFailureCount(ctx, channel.ID.String())
	if err != nil {
		return
	}

	if failureCount >= maxChannelFailures {
		if err := s.channelRepo.DisableChannel(ctx, channel.ID.String(), fmt.Sprintf("auto-disabled after %d consecutive failures: %s", failureCount, classifiedErr.Category)); err != nil {
			slog.Default().Warn("failed to disable channel", "channel_id", channel.ID.String(), "error", err)
		}
	}
}

// isRateLimited проверяет, не превышен ли лимит доставки для монитора
func (s *AlertDeliveryService) isRateLimited(ctx context.Context, alert *model.Alert) bool {
	window := s.rateLimitWindow
	if window <= 0 {
		window = RateLimitPerMonitorPerStatus
	}
	count, err := s.deliveryRepo.CountRecentByMonitorAndChannel(
		ctx,
		alert.MonitorID.String(),
		"",
		window,
	)
	if err != nil {
		return false
	}
	return count >= maxDeliveriesPerRateLimitWindow
}

// maskErrorSecrets маскирует секреты в сообщении об ошибке
func maskErrorSecrets(msg string) string {
	secretPatterns := []*regexp.Regexp{reSecretPattern}
	for _, pattern := range secretPatterns {
		msg = pattern.ReplaceAllStringFunc(msg, func(match string) string {
			parts := strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=" + model.MaskSecret(parts[1])
			}
			parts = strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ":" + model.MaskSecret(parts[1])
			}
			return match
		})
	}
	return msg
}

// parseWebhookHTTPError извлекает HTTP статус код и тело ответа из ошибки webhook
func parseWebhookHTTPError(err error) (statusCode int, body string, ok bool) {
	if err == nil {
		return 0, "", false
	}

	matches := reWebhookError.FindStringSubmatch(err.Error())
	if len(matches) == 3 {
		code := 0
		_, err := fmt.Sscanf(matches[1], "%d", &code)
		if err == nil && code > 0 {
			return code, matches[2], true
		}
	}

	return 0, "", false
}

// ScheduleRetryWithoutClassification планирует retry без классификации (для обратной совместимости)
func (s *AlertDeliveryService) ScheduleRetryWithoutClassification(
	ctx context.Context,
	attempt *model.DeliveryAttempt,
) error {
	classifiedErr := &model.DeliveryError{
		Category:   model.DeliveryErrorUnknown,
		Retryable:  true,
		MaxRetries: s.maxRetries,
		BaseDelay:  s.retryInterval,
	}
	return s.ScheduleRetry(ctx, attempt, classifiedErr)
}
