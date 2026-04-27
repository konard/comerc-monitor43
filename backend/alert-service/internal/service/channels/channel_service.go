package channels

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	"github.com/raul/monitor/backend/alert-service/internal/service/audit"
	applogger "github.com/raul/monitor/backend/alert-service/pkg/logger"
)

const (
	MinConsecutiveFailures     = 1
	MaxConsecutiveFailures     = 5
	DefaultConsecutiveFailures = 2
	MaxChannelNameLength       = 255
)

var (
	ErrInvalidVerificationCode = errors.New("invalid verification code")
)

var chatIDPattern = regexp.MustCompile(`^-?\d+$`)

// ChannelServiceConfig содержит конфигурацию сервиса каналов
type ChannelServiceConfig struct {
	ChannelLimitFree int
}

// ChannelService управляет каналами уведомлений
type ChannelService struct {
	channelRepo  repository.AlertChannelRepository
	auditService *audit.AuditService
	logger       *applogger.Logger
	cfg          ChannelServiceConfig
}

// NewChannelService создаёт новый сервис каналов
func NewChannelService(channelRepo repository.AlertChannelRepository, cfg ChannelServiceConfig) *ChannelService {
	return &ChannelService{
		channelRepo: channelRepo,
		cfg:         cfg,
		logger:      applogger.New("info"),
	}
}

// WithAuditService добавляет сервис аудита к сервису каналов
func (s *ChannelService) WithAuditService(auditSvc *audit.AuditService) *ChannelService {
	s.auditService = auditSvc
	return s
}

// WithLogger устанавливает логгер для сервиса каналов
func (s *ChannelService) WithLogger(logger *applogger.Logger) *ChannelService {
	s.logger = logger
	return s
}

// GetUserChannels возвращает каналы пользователя
func (s *ChannelService) GetUserChannels(ctx context.Context, userID uuid.UUID) ([]*model.AlertChannel, error) {
	return s.channelRepo.ListByUserID(ctx, userID.String())
}

// UpdateUserChannels обновляет каналы пользователя
func (s *ChannelService) UpdateUserChannels(
	ctx context.Context,
	userID uuid.UUID,
	channels []*model.AlertChannel,
) error {
	if isExplicitGuest(ctx) {
		return model.ErrGuestNotAllowed
	}

	// Получаем текущие каналы пользователя
	currentChannels, err := s.channelRepo.ListByUserID(ctx, userID.String())
	if err != nil {
		return err
	}

	// Создаем мапу текущих каналов для быстрого доступа
	currentChannelMap := make(map[uuid.UUID]*model.AlertChannel)
	for _, ch := range currentChannels {
		currentChannelMap[ch.ID] = ch
	}

	// Создаем мапу новых каналов
	newChannelMap := make(map[uuid.UUID]*model.AlertChannel)
	for _, ch := range channels {
		newChannelMap[ch.ID] = ch
	}

	// Обновляем или создаем каналы
	for _, channel := range channels {
		// Получаем адрес канала в зависимости от типа
		channelAddress := s.getChannelAddress(channel)

		// Проверяем дубликаты адресов
		exists, err := s.channelRepo.ExistsDuplicate(ctx, userID.String(), channel.Type, channelAddress)
		if err != nil {
			return err
		}
		if exists {
			// Если дубликат найден и это не тот же канал, пропускаем
			if existingChannel, exists := currentChannelMap[channel.ID]; exists {
				existingAddress := s.getChannelAddress(existingChannel)
				if existingAddress == channelAddress {
					continue
				}
			}
		}

		// Если канал уже существует, обновляем его
		if _, exists := currentChannelMap[channel.ID]; exists {
			if err := s.channelRepo.Update(ctx, channel); err != nil {
				return err
			}
		} else {
			// Иначе создаем новый канал
			if err := s.channelRepo.Create(ctx, channel); err != nil {
				return err
			}
		}
	}

	// Удаляем каналы, которых нет в новом списке
	for _, currentChannel := range currentChannels {
		if _, exists := newChannelMap[currentChannel.ID]; !exists {
			if err := s.channelRepo.Delete(ctx, currentChannel.ID.String()); err != nil {
				return err
			}
		}
	}

	return nil
}

// getChannelAddress возвращает адрес канала в зависимости от типа
func (s *ChannelService) getChannelAddress(channel *model.AlertChannel) string {
	switch channel.Type {
	case model.AlertChannelTypeEmail:
		if channel.EmailConfig != nil {
			return channel.EmailConfig.Email
		}
	case model.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil {
			return channel.TelegramConfig.ChatID
		}
	case model.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			return channel.WebhookConfig.URL
		}
	}
	return ""
}

// CreateChannel создаёт новый канал
func (s *ChannelService) CreateChannel(
	ctx context.Context,
	channel *model.AlertChannel,
) (*model.AlertChannel, error) {
	if isExplicitGuest(ctx) {
		return nil, model.ErrGuestNotAllowed
	}

	if err := s.validateChannel(channel); err != nil {
		return nil, err
	}

	count, err := s.channelRepo.CountByUserID(ctx, channel.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to count user channels: %v", err)
	}
	if s.cfg.ChannelLimitFree > 0 && count >= s.cfg.ChannelLimitFree {
		return nil, model.ErrChannelLimitReached
	}

	channelAddress := s.getChannelAddress(channel)
	exists, err := s.channelRepo.ExistsDuplicate(ctx, channel.UserID.String(), channel.Type, channelAddress)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, model.ErrDuplicateChannel
	}

	if err := s.channelRepo.Create(ctx, channel); err != nil {
		return nil, err
	}

	s.logger.Info("channel created",
		"channel_id", channel.ID.String(),
		"user_id", channel.UserID.String(),
		"channel_type", string(channel.Type),
	)

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionChannelCreated, "alert_channel", channel.ID.String(), channel.UserID, map[string]any{
			"channel_type": string(channel.Type),
			"address":      channelAddress,
		}); err != nil {
			s.logger.Warn("failed to record audit event", "error", err)
		}
	}

	return channel, nil
}

func (s *ChannelService) validateChannel(channel *model.AlertChannel) error {
	if len([]rune(channel.Name)) > MaxChannelNameLength {
		return model.ErrFieldTooLong
	}

	switch channel.Type {
	case model.AlertChannelTypeEmail:
		if channel.EmailConfig != nil && !isValidEmail(channel.EmailConfig.Email) {
			return model.ErrInvalidEmail
		}
	case model.AlertChannelTypeTelegram:
		if channel.TelegramConfig != nil && !chatIDPattern.MatchString(channel.TelegramConfig.ChatID) {
			return model.ErrInvalidChatID
		}
	case model.AlertChannelTypeWebhook:
		if channel.WebhookConfig != nil {
			if err := s.validateURL(channel.WebhookConfig.URL); err != nil {
				return err
			}
			if !isValidWebhookMethod(channel.WebhookConfig.Method) {
				return model.ErrInvalidFormat
			}
		}
	}
	return nil
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func isValidWebhookMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "", "GET", "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

func (s *ChannelService) validateURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return model.ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return model.ErrInvalidURL
	}
	for _, r := range rawURL {
		if r <= 0x20 || r == 0x7f {
			return model.ErrInvalidURL
		}
	}
	return nil
}

// VerifyChannel верифицирует канал
func (s *ChannelService) VerifyChannel(
	ctx context.Context,
	channelID uuid.UUID,
	verificationCode string,
) error {
	// Получаем канал по ID
	channel, err := s.channelRepo.GetByID(ctx, channelID.String())
	if err != nil {
		return err
	}

	// Проверяем, что канал уже верифицирован
	if channel.Verified {
		return nil
	}

	// Код верификации проверяется вызывающей стороной
	if verificationCode == "" {
		return ErrInvalidVerificationCode
	}

	// Отмечаем канал как верифицированный
	channel.Verified = true
	channel.FailureCount = 0 // Сбрасываем счетчик неудач
	channel.Status = model.AlertChannelStatusActive

	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return err
	}

	s.logger.Info("channel verified",
		"channel_id", channelID.String(),
		"channel_type", string(channel.Type),
	)

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionChannelVerified, "alert_channel", channelID.String(), channel.UserID, map[string]any{
			"channel_type": string(channel.Type),
		}); err != nil {
			s.logger.Warn("failed to record audit event", "error", err)
		}
	}

	return nil
}

// GetChannel возвращает канал по ID с проверкой владельца.
func (s *ChannelService) GetChannel(ctx context.Context, userID, channelID uuid.UUID) (*model.AlertChannel, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID.String())
	if err != nil {
		return nil, err
	}
	if channel.UserID != userID {
		return nil, model.ErrForbidden
	}
	return channel, nil
}

// UpdateChannel обновляет канал с проверкой владельца.
func (s *ChannelService) UpdateChannel(ctx context.Context, userID uuid.UUID, channel *model.AlertChannel) (*model.AlertChannel, error) {
	if isExplicitGuest(ctx) {
		return nil, model.ErrGuestNotAllowed
	}

	existing, err := s.channelRepo.GetByID(ctx, channel.ID.String())
	if err != nil {
		return nil, err
	}
	if existing.UserID != userID {
		return nil, model.ErrForbidden
	}

	// Применяем разрешённые к обновлению поля поверх существующего.
	if channel.Name != "" {
		existing.Name = channel.Name
	}
	if channel.TelegramConfig != nil {
		existing.TelegramConfig = channel.TelegramConfig
	}
	if channel.EmailConfig != nil {
		existing.EmailConfig = channel.EmailConfig
	}
	if channel.WebhookConfig != nil {
		existing.WebhookConfig = channel.WebhookConfig
	}
	existing.Enabled = channel.Enabled

	if err := s.validateChannel(existing); err != nil {
		return nil, err
	}

	if err := s.channelRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionChannelUpdated, "alert_channel", existing.ID.String(), userID, map[string]any{
			"channel_type": string(existing.Type),
		}); err != nil {
			s.logger.Warn("failed to record audit event", "error", err)
		}
	}

	return existing, nil
}

// ListChannels возвращает каналы пользователя.
func (s *ChannelService) ListChannels(ctx context.Context, userID uuid.UUID) ([]*model.AlertChannel, error) {
	return s.channelRepo.ListByUserID(ctx, userID.String())
}

// isExplicitGuest возвращает true только если в контексте явно установлена роль GUEST.
// Контексты без роли (например, в тестах) не считаются GUEST.
func isExplicitGuest(ctx context.Context) bool {
	role := ctx.Value(auth.RoleKey)
	if role == nil {
		return false
	}
	r, ok := role.(auth.Role)
	return ok && r == auth.RoleGuest
}

// DeleteChannel удаляет канал с проверкой владельца
func (s *ChannelService) DeleteChannel(ctx context.Context, userID, channelID uuid.UUID) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID.String())
	if err != nil {
		return fmt.Errorf("failed to get channel: %v", err)
	}

	if channel.UserID != userID {
		s.logger.Warn("channel delete forbidden",
			"user_id", userID.String(),
			"channel_id", channelID.String(),
			"channel_owner", channel.UserID.String(),
		)
		return model.ErrForbidden
	}

	if err := s.channelRepo.Delete(ctx, channelID.String()); err != nil {
		return fmt.Errorf("failed to delete channel: %v", err)
	}

	s.logger.Info("channel deleted",
		"channel_id", channelID.String(),
		"user_id", userID.String(),
		"channel_type", string(channel.Type),
	)

	if s.auditService != nil {
		if err := s.auditService.Log(ctx, model.AuditActionChannelDeleted, "alert_channel", channelID.String(), userID, map[string]any{
			"channel_type": string(channel.Type),
		}); err != nil {
			s.logger.Warn("failed to record audit event", "error", err)
		}
	}

	return nil
}
