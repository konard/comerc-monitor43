package realtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newTestClient(hub *Hub, userID string) *Client {
	return &Client{
		ID:     uuid.New().String(),
		UserID: userID,
		Conn:   nil,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}
}

func startHub(t *testing.T, hub *Hub) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
	}()
	go hub.Run()
	return cancel
}

func drainAndStop(t *testing.T, hub *Hub, cancel context.CancelFunc) {
	t.Helper()
	// allow hub goroutine to process pending register/unregister
	time.Sleep(10 * time.Millisecond)
	cancel()
}

func TestNewHub(t *testing.T) {
	t.Parallel()

	hub := NewHub()

	assert.NotNil(t, hub)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestRegisterAndClientCount(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	cancel := startHub(t, hub)
	t.Cleanup(func() { drainAndStop(t, hub, cancel) })

	client := newTestClient(hub, uuid.New().String())
	hub.Register(client)

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())
}

func TestUnregister(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	cancel := startHub(t, hub)
	t.Cleanup(func() { drainAndStop(t, hub, cancel) })

	client := newTestClient(hub, uuid.New().String())
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestBroadcastToUser(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	cancel := startHub(t, hub)
	t.Cleanup(func() { drainAndStop(t, hub, cancel) })

	userID := uuid.New().String()
	client := newTestClient(hub, userID)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	msg := &BroadcastMessage{Type: "monitor.update", Payload: "test"}
	hub.BroadcastToUser(userID, msg)

	select {
	case received := <-client.Send:
		assert.Equal(t, "monitor.update", received.Type)
		assert.Equal(t, "test", received.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}
}

func TestBroadcastToUser_no_subscribers(t *testing.T) {
	t.Parallel()

	hub := NewHub()

	msg := &BroadcastMessage{Type: "monitor.update", Payload: "test"}

	assert.NotPanics(t, func() {
		hub.BroadcastToUser(uuid.New().String(), msg)
	})
}

func TestBroadcast_all_clients(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	cancel := startHub(t, hub)
	t.Cleanup(func() { drainAndStop(t, hub, cancel) })

	client1 := newTestClient(hub, uuid.New().String())
	client2 := newTestClient(hub, uuid.New().String())
	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	msg := &BroadcastMessage{Type: "global.ping", Payload: nil}
	hub.broadcast <- msg

	for i, c := range []*Client{client1, client2} {
		select {
		case received := <-c.Send:
			assert.Equal(t, "global.ping", received.Type, "client %d should receive broadcast", i)
		case <-time.After(time.Second):
			t.Fatalf("client %d: timed out waiting for broadcast", i)
		}
	}
}

func TestHub_concurrent_access(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	cancel := startHub(t, hub)
	t.Cleanup(func() { drainAndStop(t, hub, cancel) })

	var wg sync.WaitGroup
	clients := make([]*Client, 50)

	for i := range clients {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c := newTestClient(hub, uuid.New().String())
			clients[idx] = c
			hub.Register(c)
			time.Sleep(time.Millisecond)
		}(i)
	}
	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 50, hub.ClientCount())

	for i := range clients {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			hub.Unregister(clients[idx])
		}(i)
	}
	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
}
