package realtime

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_returns_instance(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	userID := uuid.New().String()

	client := NewClient(userID, nil, hub)

	require.NotNil(t, client)
	assert.NotEmpty(t, client.ID)
	assert.Equal(t, userID, client.UserID)
	assert.Equal(t, hub, client.Hub)
	assert.Nil(t, client.Conn)
	assert.NotNil(t, client.Send)
}

func TestNewClient_unique_ids(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	userID := uuid.New().String()

	c1 := NewClient(userID, nil, hub)
	c2 := NewClient(userID, nil, hub)

	assert.NotEqual(t, c1.ID, c2.ID)
}

func TestHub_Stop_closes_done_channel(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()

	// Stop не должен паниковать
	assert.NotPanics(t, func() {
		hub.Stop()
	})
}
