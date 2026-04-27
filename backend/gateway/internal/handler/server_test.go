package handler

import (
	"log"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	authService := &mockAuthService{}
	gatewayService := &mockGatewayService{}
	stdLogger := log.New(nil, "", 0)

	server, err := NewServer(
		":8080",
		authService,
		gatewayService,
		nil,
		"",
		nil,
		10*time.Second,
		10*time.Second,
		60*time.Second,
		stdLogger,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if server.httpServer == nil {
		t.Error("expected httpServer to be set")
	}

	if server.authHandler == nil {
		t.Error("expected authHandler to be set")
	}

	if server.gatewayHandler == nil {
		t.Error("expected gatewayHandler to be set")
	}

	if server.authMiddleware == nil {
		t.Error("expected authMiddleware to be set")
	}

	if server.loggingMiddleware == nil {
		t.Error("expected loggingMiddleware to be set")
	}
}

func TestServer_GetHTTPServer(t *testing.T) {
	authService := &mockAuthService{}
	gatewayService := &mockGatewayService{}
	stdLogger := log.New(nil, "", 0)

	server, err := NewServer(
		":8080",
		authService,
		gatewayService,
		nil,
		"",
		nil,
		10*time.Second,
		10*time.Second,
		60*time.Second,
		stdLogger,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	httpServer := server.GetHTTPServer()

	if httpServer == nil {
		t.Fatal("expected httpServer to be returned")
	}

	if httpServer.Addr != ":8080" {
		t.Errorf("expected address ':8080', got '%s'", httpServer.Addr)
	}
}
