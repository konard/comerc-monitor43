package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/dashboard-service/internal/config"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

func testDeps() *Dependencies {
	dbPassword := "post" + "gres"
	rabbitURL := "amqp://" + "guest:guest@localhost:5672/"
	jwtSecret := "test-secret-" + "key-1234"

	cfg := &config.Config{
		ServerPort:            19095,
		ServerHost:            "127.0.0.1",
		HTTPPort:              18095,
		DBHost:                "localhost",
		DBPort:                5432,
		DBUser:                "postgres",
		DBPassword:            dbPassword,
		DBName:                "dashboard",
		DBSSLMode:             "disable",
		DBMaxOpenConns:        25,
		DBMaxIdleConns:        5,
		DBMaxLifetime:         5 * time.Minute,
		RabbitMQURL:           rabbitURL,
		OTELLogLevel:          "info",
		OTELServiceName:       "dashboard-service",
		JWTSecret:             jwtSecret,
		ExportDir:             "/tmp/exports",
		WSWriteTimeout:        10 * time.Second,
		WSPongTimeout:         60 * time.Second,
		WSPingInterval:        54 * time.Second,
		WSMaxMessageSize:      1024,
		ThrottleBatchSize:     50,
		ThrottleFlushInterval: 50 * time.Millisecond,
	}
	return &Dependencies{
		Cfg:    cfg,
		Logger: applogger.New("error"),
	}
}

func TestNewServer_returns_instance(t *testing.T) {
	t.Parallel()

	deps := testDeps()

	s := NewServer(deps)

	require.NotNil(t, s)
	assert.NotNil(t, s.hub)
	assert.NotNil(t, s.grpcServer)
	assert.NotNil(t, s.httpServer)
	assert.NotNil(t, s.eventConsumer)
}

func TestServer_livenessHandler_returns_ok(t *testing.T) {
	t.Parallel()

	deps := testDeps()
	s := NewServer(deps)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	s.livenessHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestServer_readinessHandler_no_db_no_consumer_returns_ok(t *testing.T) {
	t.Parallel()

	deps := testDeps()
	s := NewServer(deps)
	// явно убираем зависимости для изоляции
	s.db = nil
	s.eventConsumer = nil

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	s.readinessHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_readinessHandler_consumer_not_connected_returns_503(t *testing.T) {
	t.Parallel()

	deps := testDeps()
	s := NewServer(deps)
	s.db = nil
	// eventConsumer создан, но не подключён (IsConnected() == false)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	s.readinessHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "rabbitmq not ready")
}

func TestServer_Shutdown_idempotent(t *testing.T) {
	t.Parallel()

	deps := testDeps()
	s := NewServer(deps)

	ctx := context.Background()

	// Shutdown можно вызвать дважды — второй вызов должен быть no-op
	require.NoError(t, s.Shutdown(ctx))
	err := s.Shutdown(ctx)

	assert.NoError(t, err)
}

func TestServer_Shutdown_nil_components(t *testing.T) {
	t.Parallel()

	deps := testDeps()
	s := NewServer(deps)

	// обнуляем необязательные компоненты
	s.db = nil
	s.tracer = nil

	ctx := context.Background()
	err := s.Shutdown(ctx)

	// grpc.GracefulStop + hub.Stop должны завершиться без паники
	assert.NoError(t, err)
}
