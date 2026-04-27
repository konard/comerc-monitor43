package grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Pool represents a gRPC connection pool
type Pool struct {
	cfg    *Config
	conns  chan *grpc.ClientConn
	mu     sync.RWMutex
	closed bool
}

// New creates a new gRPC connection pool
func New(cfg *Config) (*Pool, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	p := &Pool{
		cfg:   cfg,
		conns: make(chan *grpc.ClientConn, cfg.MaxConns),
	}

	// Pre-warm connections if MaxIdleConns > 0
	for i := 0; i < cfg.MaxIdleConns; i++ {
		conn, err := p.newConnection()
		if err != nil {
			if closeErr := p.Close(context.Background()); closeErr != nil {
				return nil, fmt.Errorf("failed to close pool after init error: %w", closeErr)
			}
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		p.conns <- conn
	}

	return p, nil
}

// newConnection creates a new gRPC connection
func (p *Pool) newConnection() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.cfg.ConnectTimeout)*time.Second)
	defer cancel()

	opts := make([]grpc.DialOption, 0, 3)
	opts = append(opts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(p.cfg.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(p.cfg.MaxSendMsgSize),
		),
	)

	// Add keepalive options
	opts = append(opts,
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             time.Second,
				PermitWithoutStream: true,
			},
		),
	)

	conn, err := grpc.NewClient(p.cfg.Address, opts...)
	if err != nil {
		return nil, err
	}

	conn.Connect()

	for state := conn.GetState(); ; state = conn.GetState() {
		switch state {
		case connectivity.Ready:
			return conn, nil
		case connectivity.TransientFailure, connectivity.Shutdown:
			if closeErr := conn.Close(); closeErr != nil {
				return nil, fmt.Errorf("failed to close failed connection: %w", closeErr)
			}
			return nil, errors.New("connection did not become ready")
		}

		if !conn.WaitForStateChange(ctx, state) {
			if closeErr := conn.Close(); closeErr != nil {
				return nil, fmt.Errorf("failed to close timed out connection: %w", closeErr)
			}
			return nil, fmt.Errorf("timeout waiting for connection to become ready: %w", ctx.Err())
		}
	}
}

// Get returns a connection from the pool
func (p *Pool) Get(ctx context.Context) (*grpc.ClientConn, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("pool is closed")
	}

	select {
	case conn := <-p.conns:
		// Check if connection is still healthy
		if conn.GetState() == connectivity.TransientFailure ||
			conn.GetState() == connectivity.Shutdown {
			if err := conn.Close(); err != nil {
				return nil, fmt.Errorf("failed to close unhealthy connection: %w", err)
			}
			return p.newConnection()
		}
		return conn, nil

	default:
		// No idle connections available, create a new one
		// But first check if we haven't exceeded max connections
		if len(p.conns) < p.cfg.MaxConns {
			return p.newConnection()
		}

		// Wait for an available connection with timeout
		select {
		case conn := <-p.conns:
			if conn.GetState() == connectivity.TransientFailure ||
				conn.GetState() == connectivity.Shutdown {
				if err := conn.Close(); err != nil {
					return nil, fmt.Errorf("failed to close unhealthy connection: %w", err)
				}
				return p.newConnection()
			}
			return conn, nil

		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for connection: %w", ctx.Err())
		}
	}
}

// Put returns a connection to the pool
func (p *Pool) Put(conn *grpc.ClientConn) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		if err := conn.Close(); err != nil {
			return
		}
		return
	}

	select {
	case p.conns <- conn:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		if err := conn.Close(); err != nil {
			return
		}
	}
}

// HealthCheck checks if the pool is healthy
func (p *Pool) HealthCheck(ctx context.Context) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer p.Put(conn)

	if conn.GetState() == connectivity.TransientFailure ||
		conn.GetState() == connectivity.Shutdown {
		return fmt.Errorf("connection is not healthy: %s", conn.GetState())
	}

	return nil
}

// Close closes all connections in the pool
func (p *Pool) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	close(p.conns)
	for conn := range p.conns {
		if err := conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	return nil
}

// Stats returns pool statistics
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		Total:   cap(p.conns),
		Idle:    len(p.conns),
		InUse:   cap(p.conns) - len(p.conns),
		Address: p.cfg.Address,
	}
}

// PoolStats represents pool statistics
type PoolStats struct {
	Total   int    `json:"total"`
	Idle    int    `json:"idle"`
	InUse   int    `json:"in_use"`
	Address string `json:"address"`
}

// NewConn creates a new connection (without pooling)
func NewConn(addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	defaultOpts := make([]grpc.DialOption, 0, len(opts)+1)
	defaultOpts = append(defaultOpts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	defaultOpts = append(defaultOpts, opts...)

	conn, err := grpc.NewClient(addr, defaultOpts...)
	if err != nil {
		return nil, err
	}

	conn.Connect()
	return conn, nil
}
