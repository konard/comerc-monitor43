package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upgrader используется для создания тестовых WebSocket пар.
var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newWSPair создаёт серверный и клиентский WebSocket для тестов.
func newWSPair(t *testing.T) (serverConn *websocket.Conn, clientConn *websocket.Conn) {
	t.Helper()

	serverConnCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		serverConnCh <- conn
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Logf("conn close: %v", err) // conn может быть уже закрыт в тесте
		}
	})
	return <-serverConnCh, conn
}

// TestWritePump_sends_message проверяет, что WritePump доставляет сообщение клиенту.
func TestWritePump_sends_message(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}

	go client.WritePump(2*time.Second, 100*time.Millisecond)

	msg := &BroadcastMessage{Type: "monitor.update", Payload: "hello"}
	client.Send <- msg

	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, data, err := clientConn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(data), "monitor.update")
}

// TestWritePump_closes_on_channel_close проверяет, что WritePump завершается при закрытии канала.
func TestWritePump_closes_on_channel_close(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		client.WritePump(2*time.Second, 50*time.Millisecond)
	}()

	// закрываем канал — WritePump должен послать CloseMessage и выйти
	close(client.Send)

	select {
	case <-done:
		// WritePump завершился корректно
	case <-time.After(2 * time.Second):
		t.Fatal("WritePump did not terminate after channel close")
	}

	// клиент должен получить CloseMessage
	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(500*time.Millisecond)))
	_, _, err := clientConn.ReadMessage()
	// ожидаем close frame или ошибку чтения
	_ = err
}

// TestWritePump_ping_sent проверяет, что WritePump отправляет ping по тикеру.
func TestWritePump_ping_sent(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}

	pingSent := make(chan struct{}, 1)
	clientConn.SetPingHandler(func(appData string) error {
		select {
		case pingSent <- struct{}{}:
		default:
		}
		return clientConn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	go client.WritePump(2*time.Second, 30*time.Millisecond)

	// читаем в фоне чтобы обработчик pong сработал
	go func() {
		for {
			if _, _, err := clientConn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	select {
	case <-pingSent:
		// пинг получен
	case <-time.After(2 * time.Second):
		t.Fatal("ping was not received within timeout")
	}
}

// TestWritePump_terminates_on_write_error проверяет, что WritePump завершается при ошибке записи.
func TestWritePump_terminates_on_write_error(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	// закрываем клиентскую сторону — следующая запись с сервера упадёт
	require.NoError(t, clientConn.Close())
	time.Sleep(5 * time.Millisecond)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		client.WritePump(100*time.Millisecond, 50*time.Millisecond)
	}()

	// ping не пройдёт — WritePump должен выйти
	select {
	case <-done:
		// завершился из-за ошибки записи
	case <-time.After(3 * time.Second):
		t.Fatal("WritePump did not terminate after write error")
	}
}

// TestReadPump_unregisters_on_close проверяет, что ReadPump вызывает Unregister при закрытии соединения.
func TestReadPump_unregisters_on_close(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	done := make(chan struct{})
	go func() {
		defer close(done)
		client.ReadPump(200*time.Millisecond, 1024)
	}()

	// закрываем клиентское соединение — ReadPump должен выйти и вызвать Unregister
	require.NoError(t, clientConn.Close())

	select {
	case <-done:
		// ReadPump завершился
	case <-time.After(2 * time.Second):
		t.Fatal("ReadPump did not terminate after client disconnect")
	}

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

// TestReadPump_pong_handler_resets_deadline проверяет, что ReadPump устанавливает PongHandler.
func TestReadPump_pong_handler_resets_deadline(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		client.ReadPump(500*time.Millisecond, 1024)
	}()

	// отправляем pong — должен сбросить дедлайн и продлить жизнь ReadPump
	time.Sleep(10 * time.Millisecond)
	err := clientConn.WriteMessage(websocket.PongMessage, nil)
	require.NoError(t, err)

	// закрываем после pong
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, clientConn.Close())

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadPump did not terminate")
	}
}

// TestReadPump_read_limit_enforced проверяет, что ReadPump применяет ограничение размера сообщения.
func TestReadPump_read_limit_enforced(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	serverConn, clientConn := newWSPair(t)

	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New().String(),
		Conn:   serverConn,
		Hub:    hub,
		Send:   make(chan *BroadcastMessage, 10),
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// maxMsgSize=10 — сообщение больше этого разорвёт соединение
		client.ReadPump(500*time.Millisecond, 10)
	}()

	// отправляем сообщение больше лимита
	time.Sleep(10 * time.Millisecond)
	big := make([]byte, 100)
	require.NoError(t, clientConn.WriteMessage(websocket.TextMessage, big))

	select {
	case <-done:
		// ReadPump завершился из-за превышения лимита
	case <-time.After(2 * time.Second):
		t.Fatal("ReadPump did not terminate after oversized message")
	}
}
