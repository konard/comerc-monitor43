package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// AlertChannelRepository реализует интерфейс AlertChannelRepository для PostgreSQL
type AlertChannelRepository struct {
	db *DB
}

// NewAlertChannelRepository создаёт новый экземпляр AlertChannelRepository
func NewAlertChannelRepository(db *DB) *AlertChannelRepository {
	return &AlertChannelRepository{db: db}
}

// Create создаёт новый канал уведомлений
func (r *AlertChannelRepository) Create(ctx context.Context, channel *model.AlertChannel) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.Create")
		defer span.End()

		span.SetAttributes(
			attribute.String("channel_id", channel.ID.String()),
			attribute.String("user_id", channel.UserID.String()),
			attribute.String("type", string(channel.Type)),
		)
	}

	startTime := time.Now()

	query := `
		INSERT INTO alert_channels (
			id, user_id, type, status, enabled, verified, failure_count, last_failure_at,
			telegram_config, email_config, webhook_config, created_at, updated_at
		) VALUES (
			:id, :user_id, :type, :status, :enabled, :verified, :failure_count, :last_failure_at,
			:telegram_config, :email_config, :webhook_config, :created_at, :updated_at
		)
	`

	// Сериализуем конфигурации в JSON
	telegramConfigJSON, err := json.Marshal(channel.TelegramConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram config: %v", err)
	}
	emailConfigJSON, err := json.Marshal(channel.EmailConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal email config: %v", err)
	}
	webhookConfigJSON, err := json.Marshal(channel.WebhookConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook config: %v", err)
	}

	params := map[string]any{
		"id":              channel.ID,
		"user_id":         channel.UserID,
		"type":            channel.Type,
		"status":          channel.Status,
		"enabled":         channel.Enabled,
		"verified":        channel.Verified,
		"failure_count":   channel.FailureCount,
		"last_failure_at": channel.LastFailureAt,
		"telegram_config": telegramConfigJSON,
		"email_config":    emailConfigJSON,
		"webhook_config":  webhookConfigJSON,
		"created_at":      channel.CreatedAt,
		"updated_at":      channel.UpdatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		if isUniqueViolationError(err) {
			return model.ErrDuplicateChannel
		}
		return fmt.Errorf("failed to create alert channel: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "create", "alert_channels", float64(duration))
	}

	return nil
}

// GetByID возвращает канал уведомлений по ID
func (r *AlertChannelRepository) GetByID(ctx context.Context, id string) (*model.AlertChannel, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.GetByID")
		defer span.End()

		span.SetAttributes(attribute.String("channel_id", id))
	}

	startTime := time.Now()

	query := `
		SELECT id, user_id, type, status, enabled, verified, failure_count, last_failure_at,
			telegram_config, email_config, webhook_config, created_at, updated_at
		FROM alert_channels
		WHERE id = $1
	`

	var channel struct {
		ID             uuid.UUID                `db:"id"`
		UserID         uuid.UUID                `db:"user_id"`
		Type           model.AlertChannelType   `db:"type"`
		Status         model.AlertChannelStatus `db:"status"`
		Enabled        bool                     `db:"enabled"`
		Verified       bool                     `db:"verified"`
		FailureCount   int                      `db:"failure_count"`
		LastFailureAt  *time.Time               `db:"last_failure_at"`
		TelegramConfig []byte                   `db:"telegram_config"`
		EmailConfig    []byte                   `db:"email_config"`
		WebhookConfig  []byte                   `db:"webhook_config"`
		CreatedAt      time.Time                `db:"created_at"`
		UpdatedAt      time.Time                `db:"updated_at"`
	}

	err := r.db.GetContext(ctx, &channel, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get alert channel by id: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getbyid", "alert_channels", float64(duration))
	}

	// Создаем AlertChannel и заполняем базовые поля
	result := &model.AlertChannel{
		ID:            channel.ID,
		UserID:        channel.UserID,
		Type:          channel.Type,
		Status:        channel.Status,
		Enabled:       channel.Enabled,
		Verified:      channel.Verified,
		FailureCount:  channel.FailureCount,
		LastFailureAt: channel.LastFailureAt,
		CreatedAt:     channel.CreatedAt,
		UpdatedAt:     channel.UpdatedAt,
	}

	// Десериализуем TelegramConfig
	if len(channel.TelegramConfig) > 0 {
		var config model.TelegramChannelConfig
		if err := json.Unmarshal(channel.TelegramConfig, &config); err == nil {
			result.TelegramConfig = &config
		}
	}

	// Десериализуем EmailConfig
	if len(channel.EmailConfig) > 0 {
		var config model.EmailChannelConfig
		if err := json.Unmarshal(channel.EmailConfig, &config); err == nil {
			result.EmailConfig = &config
		}
	}

	// Десериализуем WebhookConfig
	if len(channel.WebhookConfig) > 0 {
		var config model.WebhookChannelConfig
		if err := json.Unmarshal(channel.WebhookConfig, &config); err == nil {
			result.WebhookConfig = &config
		}
	}

	return result, nil
}

// GetByUserIDAndType возвращает канал уведомлений по ID пользователя и типу
func (r *AlertChannelRepository) GetByUserIDAndType(ctx context.Context, userID string, channelType model.AlertChannelType) (*model.AlertChannel, error) {
	query := `
		SELECT id, user_id, type, status, enabled, verified, failure_count, last_failure_at,
			telegram_config, email_config, webhook_config, created_at, updated_at
		FROM alert_channels
		WHERE user_id = $1 AND type = $2
		LIMIT 1
	`

	var channel struct {
		ID             uuid.UUID                `db:"id"`
		UserID         uuid.UUID                `db:"user_id"`
		Type           model.AlertChannelType   `db:"type"`
		Status         model.AlertChannelStatus `db:"status"`
		Enabled        bool                     `db:"enabled"`
		Verified       bool                     `db:"verified"`
		FailureCount   int                      `db:"failure_count"`
		LastFailureAt  *time.Time               `db:"last_failure_at"`
		TelegramConfig []byte                   `db:"telegram_config"`
		EmailConfig    []byte                   `db:"email_config"`
		WebhookConfig  []byte                   `db:"webhook_config"`
		CreatedAt      time.Time                `db:"created_at"`
		UpdatedAt      time.Time                `db:"updated_at"`
	}

	err := r.db.GetContext(ctx, &channel, query, userID, channelType)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert channel by user and type: %v", err)
	}

	// Создаем AlertChannel и заполняем базовые поля
	result := &model.AlertChannel{
		ID:            channel.ID,
		UserID:        channel.UserID,
		Type:          channel.Type,
		Status:        channel.Status,
		Enabled:       channel.Enabled,
		Verified:      channel.Verified,
		FailureCount:  channel.FailureCount,
		LastFailureAt: channel.LastFailureAt,
		CreatedAt:     channel.CreatedAt,
		UpdatedAt:     channel.UpdatedAt,
	}

	// Десериализуем TelegramConfig
	if len(channel.TelegramConfig) > 0 {
		var config model.TelegramChannelConfig
		if err := json.Unmarshal(channel.TelegramConfig, &config); err == nil {
			result.TelegramConfig = &config
		}
	}

	// Десериализуем EmailConfig
	if len(channel.EmailConfig) > 0 {
		var config model.EmailChannelConfig
		if err := json.Unmarshal(channel.EmailConfig, &config); err == nil {
			result.EmailConfig = &config
		}
	}

	// Десериализуем WebhookConfig
	if len(channel.WebhookConfig) > 0 {
		var config model.WebhookChannelConfig
		if err := json.Unmarshal(channel.WebhookConfig, &config); err == nil {
			result.WebhookConfig = &config
		}
	}

	return result, nil
}

// ListByUserID возвращает все каналы уведомлений пользователя
func (r *AlertChannelRepository) ListByUserID(ctx context.Context, userID string) ([]*model.AlertChannel, error) {
	query := `
		SELECT id, user_id, type, status, enabled, verified, failure_count, last_failure_at,
			telegram_config, email_config, webhook_config, created_at, updated_at
		FROM alert_channels
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var rows []struct {
		ID             uuid.UUID                `db:"id"`
		UserID         uuid.UUID                `db:"user_id"`
		Type           model.AlertChannelType   `db:"type"`
		Status         model.AlertChannelStatus `db:"status"`
		Enabled        bool                     `db:"enabled"`
		Verified       bool                     `db:"verified"`
		FailureCount   int                      `db:"failure_count"`
		LastFailureAt  *time.Time               `db:"last_failure_at"`
		TelegramConfig []byte                   `db:"telegram_config"`
		EmailConfig    []byte                   `db:"email_config"`
		WebhookConfig  []byte                   `db:"webhook_config"`
		CreatedAt      time.Time                `db:"created_at"`
		UpdatedAt      time.Time                `db:"updated_at"`
	}

	err := r.db.SelectContext(ctx, &rows, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert channels: %v", err)
	}

	// Преобразуем в []*model.AlertChannel
	channels := make([]*model.AlertChannel, len(rows))
	for i, row := range rows {
		channel := &model.AlertChannel{
			ID:            row.ID,
			UserID:        row.UserID,
			Type:          row.Type,
			Status:        row.Status,
			Enabled:       row.Enabled,
			Verified:      row.Verified,
			FailureCount:  row.FailureCount,
			LastFailureAt: row.LastFailureAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}

		// Десериализуем TelegramConfig
		if len(row.TelegramConfig) > 0 {
			var config model.TelegramChannelConfig
			if err := json.Unmarshal(row.TelegramConfig, &config); err == nil {
				channel.TelegramConfig = &config
			}
		}

		// Десериализуем EmailConfig
		if len(row.EmailConfig) > 0 {
			var config model.EmailChannelConfig
			if err := json.Unmarshal(row.EmailConfig, &config); err == nil {
				channel.EmailConfig = &config
			}
		}

		// Десериализуем WebhookConfig
		if len(row.WebhookConfig) > 0 {
			var config model.WebhookChannelConfig
			if err := json.Unmarshal(row.WebhookConfig, &config); err == nil {
				channel.WebhookConfig = &config
			}
		}

		channels[i] = channel
	}

	return channels, nil
}

// Update обновляет канал уведомлений
func (r *AlertChannelRepository) Update(ctx context.Context, channel *model.AlertChannel) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.Update")
		defer span.End()

		span.SetAttributes(
			attribute.String("channel_id", channel.ID.String()),
			attribute.String("status", string(channel.Status)),
		)
	}

	startTime := time.Now()

	query := `
		UPDATE alert_channels SET
			status = :status,
			enabled = :enabled,
			verified = :verified,
			failure_count = :failure_count,
			last_failure_at = :last_failure_at,
			telegram_config = :telegram_config,
			email_config = :email_config,
			webhook_config = :webhook_config,
			updated_at = :updated_at
		WHERE id = :id
	`

	// Сериализуем конфигурации в JSON
	telegramConfigJSON, err := json.Marshal(channel.TelegramConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram config: %v", err)
	}
	emailConfigJSON, err := json.Marshal(channel.EmailConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal email config: %v", err)
	}
	webhookConfigJSON, err := json.Marshal(channel.WebhookConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook config: %v", err)
	}

	params := map[string]any{
		"id":              channel.ID,
		"status":          channel.Status,
		"enabled":         channel.Enabled,
		"verified":        channel.Verified,
		"failure_count":   channel.FailureCount,
		"last_failure_at": channel.LastFailureAt,
		"telegram_config": telegramConfigJSON,
		"email_config":    emailConfigJSON,
		"webhook_config":  webhookConfigJSON,
		"updated_at":      channel.UpdatedAt,
	}

	result, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to update alert channel: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rows == 0 {
		if span != nil {
			span.SetStatus(codes.Error, "alert channel not found")
		}
		return model.ErrAlertChannelNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "update", "alert_channels", float64(duration))
	}

	return nil
}

// Delete удаляет канал уведомлений
func (r *AlertChannelRepository) Delete(ctx context.Context, id string) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.Delete")
		defer span.End()

		span.SetAttributes(attribute.String("channel_id", id))
	}

	startTime := time.Now()

	query := `DELETE FROM alert_channels WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete alert channel: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rows == 0 {
		if span != nil {
			span.SetStatus(codes.Error, "alert channel not found")
		}
		return model.ErrAlertChannelNotFound
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "delete", "alert_channels", float64(duration))
	}

	return nil
}

// ExistsDuplicate проверяет существование дубликата канала
func (r *AlertChannelRepository) ExistsDuplicate(ctx context.Context, userID string, channelType model.AlertChannelType, address string) (bool, error) {
	query := `
		SELECT telegram_config, email_config, webhook_config
		FROM alert_channels
		WHERE user_id = $1 AND type = $2
	`

	type configRow struct {
		TelegramConfig []byte `db:"telegram_config"`
		EmailConfig    []byte `db:"email_config"`
		WebhookConfig  []byte `db:"webhook_config"`
	}

	var rows []configRow
	err := r.db.SelectContext(ctx, &rows, query, userID, channelType)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate channel: %v", err)
	}

	for _, row := range rows {
		var cfgValue string
		switch channelType {
		case model.AlertChannelTypeTelegram:
			var cfg model.TelegramChannelConfig
			if err := json.Unmarshal(row.TelegramConfig, &cfg); err == nil {
				cfgValue = cfg.ChatID
			}
		case model.AlertChannelTypeEmail:
			var cfg model.EmailChannelConfig
			if err := json.Unmarshal(row.EmailConfig, &cfg); err == nil {
				cfgValue = cfg.Email
			}
		case model.AlertChannelTypeWebhook:
			var cfg model.WebhookChannelConfig
			if err := json.Unmarshal(row.WebhookConfig, &cfg); err == nil {
				cfgValue = cfg.URL
			}
		default:
			return false, nil
		}
		if cfgValue == address {
			return true, nil
		}
	}

	return false, nil
}

// MarkAsFailed помечает канал как неудачный
func (r *AlertChannelRepository) MarkAsFailed(ctx context.Context, id string, failureCount int) error {
	query := `
		UPDATE alert_channels SET
			failure_count = $1,
			last_failure_at = NOW(),
			status = 'failed'
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, failureCount, id)
	if err != nil {
		return fmt.Errorf("failed to mark channel as failed: %v", err)
	}

	return nil
}

// ListPendingForChannel возвращает pending попытки доставки для канала
func (r *AlertChannelRepository) ListPendingForChannel(ctx context.Context, channelID string, limit int) ([]*model.DeliveryAttempt, error) {
	return nil, errors.New("use DeliveryAttemptRepository.ListPending instead")
}

// CountByUserID возвращает количество каналов уведомлений пользователя
func (r *AlertChannelRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.CountByUserID")
		defer span.End()

		span.SetAttributes(attribute.String("user_id", userID))
	}

	startTime := time.Now()

	query := `SELECT COUNT(*) FROM alert_channels WHERE user_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return 0, fmt.Errorf("failed to count alert channels by user: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "countbyuser", "alert_channels", float64(duration))
	}

	return count, nil
}

// IncrementFailureCount увеличивает счётчик неудачных попыток канала и возвращает новое значение
func (r *AlertChannelRepository) IncrementFailureCount(ctx context.Context, id string) (int, error) {
	query := `
		UPDATE alert_channels
		SET failure_count = failure_count + 1, last_failure_at = NOW()
		WHERE id = $1
		RETURNING failure_count
	`

	var failureCount int
	err := r.db.GetContext(ctx, &failureCount, query, id)
	if err != nil {
		return 0, fmt.Errorf("failed to increment channel failure count: %v", err)
	}

	return failureCount, nil
}

// DisableChannel отключает канал уведомлений
func (r *AlertChannelRepository) DisableChannel(ctx context.Context, id, reason string) error {
	query := `
		UPDATE alert_channels
		SET status = 'failed', enabled = false, last_failure_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to disable channel: %v", err)
	}

	return nil
}

// SetChannelPriorities устанавливает приоритеты каналов для правила алерта
func (r *AlertChannelRepository) SetChannelPriorities(ctx context.Context, ruleID string, priorities []model.AlertChannelPriority) error {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.SetChannelPriorities")
		defer span.End()

		span.SetAttributes(attribute.String("rule_id", ruleID))
	}

	startTime := time.Now()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			if rbErr := r.db.Rollback(tx); rbErr != nil {
				if span != nil {
					span.RecordError(rbErr)
				}
			}
		}
	}()

	sqlxTx, ok := tx.(*sqlx.Tx)
	if !ok {
		return errors.New("invalid transaction type")
	}

	deleteQuery := `DELETE FROM alert_channel_priorities WHERE alert_rule_id = $1`
	if _, err := sqlxTx.ExecContext(ctx, deleteQuery, ruleID); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to delete existing channel priorities: %v", err)
	}

	if len(priorities) > 0 {
		insertQuery := `
			INSERT INTO alert_channel_priorities (id, alert_rule_id, alert_channel_id, priority, created_at)
			VALUES (:id, :alert_rule_id, :alert_channel_id, :priority, :created_at)
		`
		for i := range priorities {
			if priorities[i].ID == uuid.Nil {
				priorities[i].ID = uuid.New()
			}
			if priorities[i].CreatedAt.IsZero() {
				priorities[i].CreatedAt = time.Now()
			}
			if _, err := sqlxTx.NamedExecContext(ctx, insertQuery, priorities[i]); err != nil {
				if span != nil {
					span.RecordError(err)
				}
				return fmt.Errorf("failed to insert channel priority: %v", err)
			}
		}
	}

	if err := r.db.Commit(tx); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "setpriorities", "alert_channel_priorities", float64(duration))
	}

	return nil
}

// GetChannelPriorities возвращает приоритеты каналов для правила алерта, отсортированные по приоритету
func (r *AlertChannelRepository) GetChannelPriorities(ctx context.Context, ruleID string) ([]*model.AlertChannelPriority, error) {
	var span trace.Span
	if r.db.tracer != nil {
		ctx, span = r.db.tracer.Start(ctx, "AlertChannelRepository.GetChannelPriorities")
		defer span.End()

		span.SetAttributes(attribute.String("rule_id", ruleID))
	}

	startTime := time.Now()

	query := `
		SELECT id, alert_rule_id, alert_channel_id, priority, created_at
		FROM alert_channel_priorities
		WHERE alert_rule_id = $1
		ORDER BY priority ASC
	`

	var priorities []*model.AlertChannelPriority
	err := r.db.SelectContext(ctx, &priorities, query, ruleID)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get channel priorities: %v", err)
	}

	if r.db.metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		r.db.metrics.RecordDBQuery(ctx, "getpriorities", "alert_channel_priorities", float64(duration))
	}

	return priorities, nil
}

// isUniqueViolationError проверяет, является ли ошибка нарушением уникального ограничения PostgreSQL
func isUniqueViolationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") || strings.Contains(msg, "23505")
}
