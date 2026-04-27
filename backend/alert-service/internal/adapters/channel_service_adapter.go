package adapters

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

// ChannelServiceAdapter адаптирует репозиторий к интерфейсу ChannelService
type ChannelServiceAdapter struct {
	channelRepo repository.AlertChannelRepository
}

// NewChannelServiceAdapter создаёт новый адаптер
func NewChannelServiceAdapter(channelRepo repository.AlertChannelRepository) *ChannelServiceAdapter {
	return &ChannelServiceAdapter{
		channelRepo: channelRepo,
	}
}

// GetUserChannels возвращает каналы уведомлений пользователя
func (c *ChannelServiceAdapter) GetUserChannels(ctx context.Context, userID uuid.UUID) ([]*model.AlertChannel, error) {
	channels, err := c.channelRepo.ListByUserID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user channels: %v", err)
	}
	return channels, nil
}

// UpdateUserChannels обновляет каналы уведомлений пользователя
func (c *ChannelServiceAdapter) UpdateUserChannels(ctx context.Context, userID uuid.UUID, channels []*model.AlertChannel) error {
	// Получаем текущие каналы пользователя
	currentChannels, err := c.channelRepo.ListByUserID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("failed to get current channels: %v", err)
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
		channelAddress := c.getChannelAddress(channel)

		// Проверяем дубликаты адресов
		exists, err := c.channelRepo.ExistsDuplicate(ctx, userID.String(), channel.Type, channelAddress)
		if err != nil {
			return fmt.Errorf("failed to check for duplicate channels: %v", err)
		}
		if exists {
			// Если дубликат найден и это не тот же канал, пропускаем
			if existingChannel, exists := currentChannelMap[channel.ID]; exists {
				existingAddress := c.getChannelAddress(existingChannel)
				if existingAddress == channelAddress {
					continue
				}
			}
		}

		// Если канал уже существует, обновляем его
		if _, exists := currentChannelMap[channel.ID]; exists {
			if err := c.channelRepo.Update(ctx, channel); err != nil {
				return fmt.Errorf("failed to update channel: %v", err)
			}
		} else {
			// Иначе создаем новый канал
			if err := c.channelRepo.Create(ctx, channel); err != nil {
				return fmt.Errorf("failed to create channel: %v", err)
			}
		}
	}

	// Удаляем каналы, которых нет в новом списке
	for _, currentChannel := range currentChannels {
		if _, exists := newChannelMap[currentChannel.ID]; !exists {
			if err := c.channelRepo.Delete(ctx, currentChannel.ID.String()); err != nil {
				return fmt.Errorf("failed to delete channel: %v", err)
			}
		}
	}

	return nil
}

// getChannelAddress возвращает адрес канала в зависимости от типа
func (c *ChannelServiceAdapter) getChannelAddress(channel *model.AlertChannel) string {
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
