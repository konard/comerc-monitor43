package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEventConsumer_returns_instance(t *testing.T) {
	t.Parallel()

	c := NewEventConsumer("amqp://localhost:5672", nil, nil, nil, nil, nil)

	assert.NotNil(t, c)
	assert.Equal(t, "amqp://localhost:5672", c.amqpURL)
	assert.False(t, c.IsConnected())
}

func TestClose_nil_connection(t *testing.T) {
	t.Parallel()

	c := &EventConsumer{}

	err := c.Close()

	assert.NoError(t, err)
	assert.False(t, c.IsConnected())
}

func TestIsConnected_initial_state(t *testing.T) {
	t.Parallel()

	c := &EventConsumer{}

	assert.False(t, c.IsConnected())
}
