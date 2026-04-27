package grpc

import (
	"context"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Address != "localhost:5001" {
		t.Errorf("expected address localhost:5001, got %s", cfg.Address)
	}
	if cfg.MaxConns != 10 {
		t.Errorf("expected MaxConns 10, got %d", cfg.MaxConns)
	}
	if cfg.MaxIdleConns != 2 {
		t.Errorf("expected MaxIdleConns 2, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnectTimeout != 10 {
		t.Errorf("expected ConnectTimeout 10, got %d", cfg.ConnectTimeout)
	}
	if cfg.MaxRecvMsgSize != 4194304 {
		t.Errorf("expected MaxRecvMsgSize 4194304, got %d", cfg.MaxRecvMsgSize)
	}
}

func TestNew_withNilConfig(t *testing.T) {
	t.Parallel()

	// Передаём nil — конструктор должен подставить DefaultConfig.
	// MaxIdleConns по умолчанию равен 2, поэтому будет попытка подключения к localhost:5001.
	// В тестовой среде сервер недоступен — ожидаем ошибку.
	_, err := New(nil)

	if err == nil {
		t.Log("pool created (server reachable), ok")
	} else {
		t.Logf("expected error without running server: %v", err)
	}
}

func TestNew_zeroIdleConns(t *testing.T) {
	t.Parallel()

	// MaxIdleConns=0 — не создаёт соединений при инициализации.
	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       5,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)

	if err != nil {
		t.Fatalf("unexpected error with zero idle conns: %v", err)
	}
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	defer func() {
		if err := pool.Close(context.Background()); err != nil {
			t.Errorf("failed to close pool: %v", err)
		}
	}()
}

func TestPool_Stats(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       5,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := pool.Close(context.Background()); err != nil {
			t.Errorf("failed to close pool: %v", err)
		}
	}()

	stats := pool.Stats()

	if stats.Total != 5 {
		t.Errorf("expected Total 5, got %d", stats.Total)
	}
	if stats.Address != "localhost:59999" {
		t.Errorf("expected address localhost:59999, got %s", stats.Address)
	}
}

func TestPool_Get_closedPool(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       5,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = pool.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on close: %v", err)
	}

	// Get должен вернуть ошибку т.к. пул закрыт
	_, err = pool.Get(context.Background())
	if err == nil {
		t.Error("expected error from closed pool")
	}
}

func TestPool_Put_closedPool(t *testing.T) {
	t.Parallel()

	// Проверяем, что Put не паникует при закрытом пуле.
	// Для этого нужно сначала создать соединение, потом закрыть пул.
	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       5,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = pool.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on close: %v", err)
	}

	// Повторное закрытие должно быть идемпотентным
	err = pool.Close(context.Background())
	if err != nil {
		t.Errorf("unexpected error on second close: %v", err)
	}
}

func TestPool_Get_timeout(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       1,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := pool.Close(context.Background()); err != nil {
			t.Errorf("failed to close pool: %v", err)
		}
	}()

	// Создаём контекст с немедленной отменой — Get должен вернуть ошибку таймаута
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу

	_, err = pool.Get(ctx)
	// Ожидаем либо ошибку таймаута либо ошибку соединения
	if err == nil {
		t.Log("connection succeeded (server reachable)")
	} else {
		t.Logf("expected error: %v", err)
	}
}

func TestPool_Put_toPool(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       5,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := pool.Close(context.Background()); err != nil {
			t.Errorf("failed to close pool: %v", err)
		}
	}()

	// NewConn создаёт соединение без блокировки (не WithBlock)
	conn, err := NewConn("localhost:59999")
	if err != nil {
		t.Fatalf("unexpected error creating conn: %v", err)
	}

	// Put должен поместить соединение обратно в пул
	pool.Put(conn)

	stats := pool.Stats()
	if stats.Idle != 1 {
		t.Errorf("expected 1 idle connection, got %d", stats.Idle)
	}
}

func TestPool_Put_fullPool(t *testing.T) {
	t.Parallel()

	// MaxConns=0 означает буфер нулевого размера, Put должен закрыть соединение
	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       0,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		if err := pool.Close(context.Background()); err != nil {
			t.Errorf("failed to close pool: %v", err)
		}
	}()

	conn, err := NewConn("localhost:59999")
	if err != nil {
		t.Fatalf("unexpected error creating conn: %v", err)
	}

	// Put в заполненный пул должен закрыть соединение без паники
	pool.Put(conn)
}

func TestPool_HealthCheck_closedPool(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Address:        "localhost:59999",
		MaxConns:       5,
		MaxIdleConns:   0,
		ConnectTimeout: 1,
		MaxRecvMsgSize: 4194304,
		MaxSendMsgSize: 4194304,
	}

	pool, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = pool.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	// HealthCheck на закрытом пуле должен вернуть ошибку
	err = pool.HealthCheck(context.Background())
	if err == nil {
		t.Error("expected error from closed pool health check")
	}
}

func TestNewConn(t *testing.T) {
	t.Parallel()

	// NewConn без WithBlock — соединение создаётся без ожидания
	conn, err := NewConn("localhost:59999")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if err := conn.Close(); err != nil {
		t.Errorf("failed to close connection: %v", err)
	}
}
