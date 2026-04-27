package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
)

// ChannelPriorityService управляет приоритетами каналов уведомлений для правил алертов
type ChannelPriorityService struct {
	channelRepo repository.AlertChannelRepository
	tracer      trace.Tracer
}

// NewChannelPriorityService создаёт новый сервис приоритетов каналов
func NewChannelPriorityService(channelRepo repository.AlertChannelRepository) *ChannelPriorityService {
	return &ChannelPriorityService{
		channelRepo: channelRepo,
	}
}

// WithTracer устанавливает tracer для сервиса
func WithChannelPriorityTracer(tracer trace.Tracer) func(*ChannelPriorityService) {
	return func(s *ChannelPriorityService) {
		s.tracer = tracer
	}
}

// SetPriorities устанавливает приоритеты каналов для правила алерта
// Приоритет определяет порядок доставки (меньшее число = более высокий приоритет)
// Эскалация: если нет ACK в течение timeout, уведомление отправляется на следующий по приоритету канал
func (s *ChannelPriorityService) SetPriorities(
	ctx context.Context,
	ruleID uuid.UUID,
	channelIDs []uuid.UUID,
	priorities []int,
) error {
	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "ChannelPriorityService.SetPriorities")
		defer span.End()

		span.SetAttributes(
			attribute.String("rule_id", ruleID.String()),
			attribute.Int("channel_count", len(channelIDs)),
		)
	}

	if len(channelIDs) != len(priorities) {
		return errors.New("channel_ids and priorities must have the same length")
	}

	priorityModels := make([]model.AlertChannelPriority, len(channelIDs))
	for i := range channelIDs {
		priorityModels[i] = model.AlertChannelPriority{
			ID:             uuid.New(),
			AlertRuleID:    ruleID,
			AlertChannelID: channelIDs[i],
			Priority:       priorities[i],
			CreatedAt:      time.Now(),
		}
	}

	if err := s.channelRepo.SetChannelPriorities(ctx, ruleID.String(), priorityModels); err != nil {
		return fmt.Errorf("failed to set channel priorities: %v", err)
	}

	return nil
}

// GetPrioritizedChannels возвращает каналы для правила, отсортированные по приоритету (по возрастанию)
func (s *ChannelPriorityService) GetPrioritizedChannels(
	ctx context.Context,
	ruleID uuid.UUID,
) ([]*model.AlertChannelPriority, error) {
	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "ChannelPriorityService.GetPrioritizedChannels")
		defer span.End()

		span.SetAttributes(attribute.String("rule_id", ruleID.String()))
	}

	priorities, err := s.channelRepo.GetChannelPriorities(ctx, ruleID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get prioritized channels: %v", err)
	}

	return priorities, nil
}
