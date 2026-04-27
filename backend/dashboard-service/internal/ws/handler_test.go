package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

func newTestHandler() *WSHandler {
	hub := realtime.NewHub()
	go hub.Run()
	authenticator := auth.NewAuthenticator("test-secret-key-16ch")
	logger := applogger.New("error")

	return NewWSHandler(
		hub,
		authenticator,
		logger,
		10*time.Second,
		60*time.Second,
		54*time.Second,
		1024,
	)
}

func TestNewWSHandler_returns_instance(t *testing.T) {
	t.Parallel()

	h := newTestHandler()

	require.NotNil(t, h)
	assert.NotNil(t, h.hub)
	assert.NotNil(t, h.authenticator)
	assert.NotNil(t, h.logger)
	assert.Equal(t, 10*time.Second, h.writeTimeout)
	assert.Equal(t, 60*time.Second, h.pongTimeout)
	assert.Equal(t, 54*time.Second, h.pingInterval)
	assert.Equal(t, int64(1024), h.maxMsgSize)
}

func TestWSHandler_ServeHTTP_missing_token_returns_401(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWSHandler_ServeHTTP_invalid_token_returns_401(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/ws?token=invalid-token", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWSHandler_ServeHTTP_token_in_header_invalid_returns_401(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "bad-token")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
