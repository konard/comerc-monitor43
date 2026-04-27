package publisher

import (
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func closeAMQPConnection(tb testing.TB, conn *amqp.Connection) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			tb.Errorf("close amqp connection: %v", err)
		}
	})
}

func closeAMQPChannel(tb testing.TB, channel *amqp.Channel) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			tb.Errorf("close amqp channel: %v", err)
		}
	})
}
