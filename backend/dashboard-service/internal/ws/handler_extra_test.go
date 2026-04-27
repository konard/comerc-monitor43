package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/infrastructure/auth"
)

const testSecret = "test-secret-key-16ch"

// makeToken создаёт подписанный JWT токен для тестов.
func makeToken(secret string, userID uuid.UUID, expiry time.Duration) string {
	claims := &auth.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		panic("failed to sign test token: " + err.Error())
	}
	return s
}

// TestWSHandler_ServeHTTP_valid_token_upgrades проверяет успешный WebSocket upgrade.
func TestWSHandler_ServeHTTP_valid_token_upgrades(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	userID := uuid.New()
	token := makeToken(testSecret, userID, time.Hour)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token=" + token
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})
	}
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	require.NoError(t, conn.Close())
}

// TestWSHandler_ServeHTTP_valid_token_in_header_upgrades проверяет upgrade с токеном в заголовке.
func TestWSHandler_ServeHTTP_valid_token_in_header_upgrades(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	userID := uuid.New()
	token := makeToken(testSecret, userID, time.Hour)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	header := http.Header{}
	header.Set("Sec-WebSocket-Protocol", token)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})
	}
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	require.NoError(t, conn.Close())
}

// TestWSHandler_ServeHTTP_expired_token_returns_401 проверяет, что истёкший токен отклоняется.
func TestWSHandler_ServeHTTP_expired_token_returns_401(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/ws?token="+makeToken(testSecret, uuid.New(), -time.Hour), nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestWSHandler_ServeHTTP_nil_uuid_closes проверяет, что uuid.Nil в токене отклоняет соединение.
// Этот тест покрывает ветку userID == "" || claims.UserID == uuid.Nil в ServeHTTP.
func TestWSHandler_ServeHTTP_nil_uuid_closes(t *testing.T) {
	t.Parallel()

	h := newTestHandler()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	// uuid.Nil как UserID
	token := makeToken(testSecret, uuid.Nil, time.Hour)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token=" + token
	// gorilla диалер может получить ошибку т.к. сервер закроет соединение
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})
	}
	if err == nil {
		// соединение установлено, но сервер немедленно закроет его
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(500*time.Millisecond)))
		_, _, readErr := conn.ReadMessage()
		assert.Error(t, readErr, "expected connection to be closed by server")
		require.NoError(t, conn.Close())
	}
	// ошибка диала тоже допустима — значит сервер отклонил
}
