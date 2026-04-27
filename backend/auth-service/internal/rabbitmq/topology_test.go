package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"

	"github.com/raul/monitor/backend/auth-service/internal/rabbitmq/mocks"
)

func testConfig() Config {
	return Config{
		Exchange:   "auth.events",
		Queue:      "auth.events",
		RoutingKey: "auth.events",
	}
}

func TestStart_channel_error(t *testing.T) {
	t.Parallel()

	dialer := mocks.NewChannelOpener(t)
	dialer.EXPECT().Channel().Return(nil, assert.AnError)

	topo := New(testConfig(), dialer)

	err := topo.Start(context.Background())

	assert.ErrorContains(t, err, "open channel")
}

func TestTopology_declare(t *testing.T) {
	t.Parallel()

	cfg := testConfig()

	tests := []struct {
		name       string
		setup      func(t *testing.T) *mocks.AmqpChannel
		wantErrStr string
	}{
		{
			name: "success",
			setup: func(t *testing.T) *mocks.AmqpChannel {
				ch := mocks.NewAmqpChannel(t)
				ch.EXPECT().ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, amqp.Table(nil)).Return(nil)
				ch.EXPECT().QueueDeclare(cfg.Queue, true, false, false, false, amqp.Table(nil)).Return(amqp.Queue{}, nil)
				ch.EXPECT().QueueBind(cfg.Queue, cfg.RoutingKey, cfg.Exchange, false, amqp.Table(nil)).Return(nil)
				return ch
			},
		},
		{
			name: "exchange_declare_error",
			setup: func(t *testing.T) *mocks.AmqpChannel {
				ch := mocks.NewAmqpChannel(t)
				ch.EXPECT().ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, amqp.Table(nil)).Return(assert.AnError)
				return ch
			},
			wantErrStr: `declare exchange "auth.events"`,
		},
		{
			name: "queue_declare_error",
			setup: func(t *testing.T) *mocks.AmqpChannel {
				ch := mocks.NewAmqpChannel(t)
				ch.EXPECT().ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, amqp.Table(nil)).Return(nil)
				ch.EXPECT().QueueDeclare(cfg.Queue, true, false, false, false, amqp.Table(nil)).Return(amqp.Queue{}, assert.AnError)
				return ch
			},
			wantErrStr: `declare queue "auth.events"`,
		},
		{
			name: "queue_bind_error",
			setup: func(t *testing.T) *mocks.AmqpChannel {
				ch := mocks.NewAmqpChannel(t)
				ch.EXPECT().ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, amqp.Table(nil)).Return(nil)
				ch.EXPECT().QueueDeclare(cfg.Queue, true, false, false, false, amqp.Table(nil)).Return(amqp.Queue{}, nil)
				ch.EXPECT().QueueBind(cfg.Queue, cfg.RoutingKey, cfg.Exchange, false, amqp.Table(nil)).Return(assert.AnError)
				return ch
			},
			wantErrStr: `bind queue "auth.events"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ch := tc.setup(t)
			topo := New(cfg, mocks.NewChannelOpener(t))

			err := topo.declare(context.Background(), ch)

			if tc.wantErrStr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErrStr)
			}
		})
	}
}
