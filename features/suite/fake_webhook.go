//go:build bdd

package suite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// ReceivedWebhook описывает HTTP-запрос, принятый FakeWebhookServer.
type ReceivedWebhook struct {
	Method    string
	Path      string
	Headers   http.Header
	Body      json.RawMessage
	Timestamp time.Time
}

// FakeWebhookServer — HTTP-сервер, принимающий webhook-уведомления и сохраняющий их в памяти.
type FakeWebhookServer struct {
	server *http.Server
	url    string

	mu       sync.Mutex
	requests []ReceivedWebhook
}

// NewFakeWebhookServer поднимает HTTP-сервер на свободном порту 127.0.0.1.
func NewFakeWebhookServer() (*FakeWebhookServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen webhook: %w", err)
	}
	addr := ln.Addr().(*net.TCPAddr)

	s := &FakeWebhookServer{
		url: fmt.Sprintf("http://%s:%d/", addr.IP.String(), addr.Port),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)

	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		//nolint:errcheck // ошибки сервера обрабатываются через Stop
		_ = s.server.Serve(ln)
	}()

	return s, nil
}

// URL возвращает базовый URL сервера (с финальным "/").
func (s *FakeWebhookServer) URL() string { return s.url }

// Requests возвращает копию принятых запросов.
func (s *FakeWebhookServer) Requests() []ReceivedWebhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ReceivedWebhook, len(s.requests))
	copy(out, s.requests)
	return out
}

// Reset очищает накопленные запросы.
func (s *FakeWebhookServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}

// Stop останавливает HTTP-сервер с graceful shutdown.
func (s *FakeWebhookServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *FakeWebhookServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	//nolint:errcheck // best-effort close
	_ = r.Body.Close()

	rw := ReceivedWebhook{
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   r.Header.Clone(),
		Body:      json.RawMessage(append([]byte(nil), body...)),
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	s.requests = append(s.requests, rw)
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	//nolint:errcheck // best-effort write
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
