package realtime

import (
	"sync"
)

type BroadcastMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client
	userSubs   map[string]map[string]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		userSubs:   make(map[string]map[string]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if h.userSubs[client.UserID] == nil {
				h.userSubs[client.UserID] = make(map[string]struct{})
			}
			h.userSubs[client.UserID][client.ID] = struct{}{}
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if subs, ok := h.userSubs[client.UserID]; ok {
					delete(subs, client.ID)
					if len(subs) == 0 {
						delete(h.userSubs, client.UserID)
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- msg:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToUser(userID string, msg *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clientIDs, ok := h.userSubs[userID]; ok {
		for cid := range clientIDs {
			if client, ok := h.clients[cid]; ok {
				select {
				case client.Send <- msg:
				default:
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client)   { h.register <- client }
func (h *Hub) Unregister(client *Client) { h.unregister <- client }
func (h *Hub) Stop()                     { close(h.done) }
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
