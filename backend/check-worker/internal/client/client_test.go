package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCheckResultProto(t *testing.T) {
}

func TestNewSchedulerClient(t *testing.T) {
	t.Parallel()

	client := NewSchedulerClient("localhost:9094", nil, nil)
	assert.NotNil(t, client)
	assert.Equal(t, "localhost:9094", client.address)
	assert.False(t, client.IsConnected())
}

func TestNewMonitorClient(t *testing.T) {
	t.Parallel()

	client := NewMonitorClient("localhost:9091", nil, nil)
	assert.NotNil(t, client)
	assert.Equal(t, "localhost:9091", client.address)
	assert.False(t, client.IsConnected())
}

func TestSchedulerClient_NotConnected(t *testing.T) {
	t.Parallel()

	client := NewSchedulerClient("localhost:9094", nil, nil)

	err := client.Close()
	assert.NoError(t, err)

	err = client.UnregisterWorker(context.TODO())
	assert.NoError(t, err)
}

func TestMonitorClient_NotConnected(t *testing.T) {
	t.Parallel()

	client := NewMonitorClient("localhost:9091", nil, nil)

	err := client.Close()
	assert.NoError(t, err)
}

func TestBuildCheckResultProto_returns_api_type(t *testing.T) {
	t.Parallel()

	result := BuildCheckResultProto(true, 200, 1.0, "")
	assert.NotNil(t, result)
}
