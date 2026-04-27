package realtime

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	Hub    *Hub
	Send   chan *BroadcastMessage
}

func NewClient(userID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:     uuid.New().String(),
		UserID: userID,
		Conn:   conn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 256),
	}
}

func (c *Client) WritePump(writeTimeout, pingInterval time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		if err := c.Conn.Close(); err != nil {
			return
		}
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return
			}
			if !ok {
				if err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					return
				}
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(pongTimeout time.Duration, maxMessageSize int64) {
	defer func() {
		c.Hub.Unregister(c)
		if err := c.Conn.Close(); err != nil {
			return
		}
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongTimeout)); err != nil {
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
