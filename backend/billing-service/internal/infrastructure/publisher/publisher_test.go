package publisher

import (
	"context"
	"log/slog"
	"testing"

	"github.com/pure-golang/adapters/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPublisher реализует queue.Publisher для тестирования.
type mockPublisher struct {
	published   []queue.Message
	shouldError bool
}

func (m *mockPublisher) Publish(ctx context.Context, msgs ...queue.Message) error {
	if m.shouldError {
		return assert.AnError
	}
	m.published = append(m.published, msgs...)
	return nil
}

func newTestPublisher(t *testing.T, mock *mockPublisher) *EventPublisher {
	t.Helper()
	return NewEventProducer(mock, slog.Default())
}

func TestEventPublisher_PublishSubscriptionCreated(t *testing.T) {
	t.Parallel()

	mock := &mockPublisher{}
	pub := newTestPublisher(t, mock)

	event := &SubscriptionCreatedEvent{
		SubscriptionID:   "sub_123",
		UserID:           "user_456",
		PlanID:           "TIER_STARTER",
		MaxMonitors:      25,
		MinCheckInterval: 60,
		MaxAlerts:        100,
	}

	err := pub.PublishSubscriptionCreated(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, mock.published, 1)
	assert.Equal(t, "subscription.created", mock.published[0].Topic)
}

func TestEventPublisher_PublishSubscriptionUpgraded(t *testing.T) {
	t.Parallel()

	mock := &mockPublisher{}
	pub := newTestPublisher(t, mock)

	event := &SubscriptionUpgradedEvent{
		SubscriptionID:   "sub_123",
		UserID:           "user_456",
		OldPlanID:        "TIER_FREE",
		NewPlanID:        "TIER_STARTER",
		MaxMonitors:      25,
		MinCheckInterval: 60,
		MaxAlerts:        100,
	}

	err := pub.PublishSubscriptionUpgraded(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, mock.published, 1)
	assert.Equal(t, "subscription.upgraded", mock.published[0].Topic)
}

func TestEventPublisher_PublishSubscriptionCanceled(t *testing.T) {
	t.Parallel()

	mock := &mockPublisher{}
	pub := newTestPublisher(t, mock)

	event := &SubscriptionCanceledEvent{
		SubscriptionID: "sub_123",
		UserID:         "user_456",
		PlanID:         "TIER_STARTER",
		Reason:         "user requested",
	}

	err := pub.PublishSubscriptionCanceled(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, mock.published, 1)
	assert.Equal(t, "subscription.canceled", mock.published[0].Topic)
}

func TestEventPublisher_PublishSubscriptionExpired(t *testing.T) {
	t.Parallel()

	mock := &mockPublisher{}
	pub := newTestPublisher(t, mock)

	event := &SubscriptionExpiredEvent{
		SubscriptionID: "sub_123",
		UserID:         "user_456",
		PlanID:         "TIER_STARTER",
	}

	err := pub.PublishSubscriptionExpired(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, mock.published, 1)
	assert.Equal(t, "subscription.expired", mock.published[0].Topic)
}

func TestEventPublisher_PublishPaymentFailed(t *testing.T) {
	t.Parallel()

	mock := &mockPublisher{}
	pub := newTestPublisher(t, mock)

	event := &PaymentFailedEvent{
		SubscriptionID: "sub_123",
		UserID:         "user_456",
		AmountKopeks:   29900,
		Currency:       "RUB",
		Reason:         "card declined",
	}

	err := pub.PublishPaymentFailed(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, mock.published, 1)
	assert.Equal(t, "subscription.payment.failed", mock.published[0].Topic)
}

func TestEventPublisher_PublishError(t *testing.T) {
	t.Parallel()

	mock := &mockPublisher{shouldError: true}
	pub := newTestPublisher(t, mock)

	event := &SubscriptionCreatedEvent{
		SubscriptionID: "sub_123",
		UserID:         "user_456",
		PlanID:         "TIER_STARTER",
	}

	err := pub.PublishSubscriptionCreated(context.Background(), event)
	require.Error(t, err)
}

func TestEventProducerStructures(t *testing.T) {
	t.Parallel()

	t.Run("SubscriptionCreatedEvent", func(t *testing.T) {
		t.Parallel()
		event := &SubscriptionCreatedEvent{
			SubscriptionID:   "test_id",
			UserID:           "test_user",
			PlanID:           "TIER_STARTER",
			MaxMonitors:      25,
			MinCheckInterval: 60,
			MaxAlerts:        100,
		}
		assert.Equal(t, "test_id", event.SubscriptionID)
		assert.Equal(t, "test_user", event.UserID)
		assert.Equal(t, "TIER_STARTER", event.PlanID)
		assert.Equal(t, int32(25), event.MaxMonitors)
		assert.Equal(t, int32(60), event.MinCheckInterval)
		assert.Equal(t, int32(100), event.MaxAlerts)
	})

	t.Run("SubscriptionUpgradedEvent", func(t *testing.T) {
		t.Parallel()
		event := &SubscriptionUpgradedEvent{
			SubscriptionID: "test_id",
			UserID:         "test_user",
			OldPlanID:      "TIER_FREE",
			NewPlanID:      "TIER_STARTER",
			MaxMonitors:    25,
		}
		assert.Equal(t, "test_id", event.SubscriptionID)
		assert.Equal(t, "TIER_FREE", event.OldPlanID)
		assert.Equal(t, "TIER_STARTER", event.NewPlanID)
	})

	t.Run("SubscriptionCanceledEvent", func(t *testing.T) {
		t.Parallel()
		event := &SubscriptionCanceledEvent{
			SubscriptionID: "test_id",
			UserID:         "test_user",
			PlanID:         "TIER_STARTER",
			Reason:         "test reason",
		}
		assert.Equal(t, "test_id", event.SubscriptionID)
		assert.Equal(t, "test reason", event.Reason)
	})

	t.Run("PaymentFailedEvent", func(t *testing.T) {
		t.Parallel()
		event := &PaymentFailedEvent{
			SubscriptionID: "test_id",
			UserID:         "test_user",
			AmountKopeks:   29900,
			Currency:       "RUB",
			Reason:         "insufficient funds",
		}
		assert.Equal(t, "test_id", event.SubscriptionID)
		assert.Equal(t, int64(29900), event.AmountKopeks)
		assert.Equal(t, "RUB", event.Currency)
		assert.Equal(t, "insufficient funds", event.Reason)
	})
}
