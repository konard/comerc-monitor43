package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	pool := New(nil)

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	if pool.client == nil {
		t.Fatal("expected non-nil http.Client")
	}
}

func TestNew_withConfig(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       60,
		ResponseHeaderTimeout: 15,
		Timeout:               15,
		DisableCompression:    true,
		InsecureSkipVerify:    false,
	}

	pool := New(cfg)

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestNewWithClient(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: 5 * time.Second}

	pool := NewWithClient(client)

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	if pool.Client() != client {
		t.Error("expected same client reference")
	}
}

func TestPool_Client(t *testing.T) {
	t.Parallel()

	pool := New(nil)

	client := pool.Client()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestPool_Do(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	pool := New(nil)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := pool.Do(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPool_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	pool := New(nil)

	resp, err := pool.Get(server.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPool_Post(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	pool := New(nil)

	resp, err := pool.Post(server.URL, "application/json", strings.NewReader(`{"key":"value"}`))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestPool_HealthCheck_healthy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	pool := New(nil)
	err := pool.HealthCheck(context.Background(), server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPool_HealthCheck_serverError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	pool := New(nil)
	err := pool.HealthCheck(context.Background(), server.URL)

	if err == nil {
		t.Error("expected error for 5xx status code")
	}
}

func TestPool_HealthCheck_4xxAccepted(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	pool := New(nil)
	err := pool.HealthCheck(context.Background(), server.URL)

	if err != nil {
		t.Errorf("unexpected error for 4xx: %v", err)
	}
}

func TestPool_HealthCheck_unreachable(t *testing.T) {
	t.Parallel()

	pool := New(&Config{Timeout: 1, ResponseHeaderTimeout: 1})

	err := pool.HealthCheck(context.Background(), "http://127.0.0.1:19999")

	if err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestPool_Close(t *testing.T) {
	t.Parallel()

	pool := New(nil)

	err := pool.Close(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPool_Stats(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 2,
		MaxConnsPerHost:     10,
		IdleConnTimeout:     90,
		Timeout:             30,
	}
	pool := New(cfg)

	stats := pool.Stats()

	if stats.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns 5, got %d", stats.MaxIdleConns)
	}
	if stats.MaxIdleConnsPerHost != 2 {
		t.Errorf("expected MaxIdleConnsPerHost 2, got %d", stats.MaxIdleConnsPerHost)
	}
	if stats.MaxConnsPerHost != 10 {
		t.Errorf("expected MaxConnsPerHost 10, got %d", stats.MaxConnsPerHost)
	}
}

func TestPool_Stats_withNonTransport(t *testing.T) {
	t.Parallel()

	// Пул с нестандартным транспортом — Stats должен вернуть пустую структуру
	pool := NewWithClient(&http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, io.EOF
		}),
	})

	stats := pool.Stats()

	if stats.MaxIdleConns != 0 {
		t.Errorf("expected empty stats, got %+v", stats)
	}
}

func TestPool_EnableH2(t *testing.T) {
	t.Parallel()

	pool := New(nil)

	// Не должно паниковать
	pool.EnableH2()
}

func TestPool_SetTimeout(t *testing.T) {
	t.Parallel()

	pool := New(nil)
	pool.SetTimeout(5 * time.Second)

	if pool.client.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", pool.client.Timeout)
	}
}

func TestNewCustom(t *testing.T) {
	t.Parallel()

	pool := NewCustom(
		WithMaxIdleConns(20),
		WithMaxIdleConnsPerHost(5),
		WithTimeout(15),
		WithInsecureTLS(false),
	)

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	stats := pool.Stats()
	if stats.MaxIdleConns != 20 {
		t.Errorf("expected MaxIdleConns 20, got %d", stats.MaxIdleConns)
	}
	if stats.MaxIdleConnsPerHost != 5 {
		t.Errorf("expected MaxIdleConnsPerHost 5, got %d", stats.MaxIdleConnsPerHost)
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns 10, got %d", cfg.MaxIdleConns)
	}
	if cfg.Timeout != 30 {
		t.Errorf("expected Timeout 30, got %d", cfg.Timeout)
	}
}

// roundTripperFunc — вспомогательный тип для тестов.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
