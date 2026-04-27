package ws

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/raul/monitor/backend/dashboard-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WSHandler struct {
	hub           *realtime.Hub
	authenticator *auth.Authenticator
	logger        *applogger.Logger
	writeTimeout  time.Duration
	pongTimeout   time.Duration
	pingInterval  time.Duration
	maxMsgSize    int64
}

func NewWSHandler(
	hub *realtime.Hub,
	authenticator *auth.Authenticator,
	logger *applogger.Logger,
	writeTimeout, pongTimeout, pingInterval time.Duration,
	maxMsgSize int64,
) *WSHandler {
	return &WSHandler{
		hub:           hub,
		authenticator: authenticator,
		logger:        logger,
		writeTimeout:  writeTimeout,
		pongTimeout:   pongTimeout,
		pingInterval:  pingInterval,
		maxMsgSize:    maxMsgSize,
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Sec-WebSocket-Protocol")
	}
	if token == "" {
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return
	}

	claims, err := h.authenticator.ValidateToken(r.Context(), token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", "error", err)
		return
	}

	userID := claims.UserID.String()
	if userID == "" || claims.UserID == uuid.Nil {
		http.Error(w, "invalid user", http.StatusUnauthorized)
		if err := conn.Close(); err != nil {
			h.logger.Warn("failed to close websocket connection", "error", err)
		}
		return
	}

	client := realtime.NewClient(userID, conn, h.hub)
	h.hub.Register(client)

	go client.WritePump(h.writeTimeout, h.pingInterval)
	go client.ReadPump(h.pongTimeout, h.maxMsgSize)

	h.logger.Info("websocket client connected", "client_id", client.ID, "user_id", userID)
}
