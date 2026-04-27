package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// Pool represents an HTTP connection pool wrapper
type Pool struct {
	client *http.Client
	cfg    *Config
}

// New creates a new HTTP connection pool
func New(cfg *Config) *Pool {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.Timeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       time.Duration(cfg.IdleConnTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.ResponseHeaderTimeout) * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DisableCompression:    cfg.DisableCompression,
	}
	if cfg.InsecureSkipVerify {
		//nolint:gosec // Нужен переключатель для локальных/self-signed окружений.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	return &Pool{
		client: client,
		cfg:    cfg,
	}
}

// NewWithClient creates a pool with an existing http.Client
func NewWithClient(client *http.Client) *Pool {
	return &Pool{
		client: client,
		cfg:    DefaultConfig(),
	}
}

// Client returns the underlying http.Client
func (p *Pool) Client() *http.Client {
	return p.client
}

// Do executes an HTTP request
func (p *Pool) Do(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("request URL is required")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported request scheme: %s", req.URL.Scheme)
	}
	if req.URL.Host == "" {
		return nil, fmt.Errorf("request host is required")
	}

	//nolint:gosec // Цель запроса проходит базовую валидацию и задаётся конфигурацией сервиса.
	return p.client.Do(req)
}

// Get executes a GET request
func (p *Pool) Get(url string) (*http.Response, error) {
	return p.client.Get(url)
}

// Post executes a POST request
func (p *Pool) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return p.client.Post(url, contentType, body)
}

// HealthCheck checks if the pool is healthy by making a simple request
func (p *Pool) HealthCheck(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 500 {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

// Close closes idle connections
func (p *Pool) Close(ctx context.Context) error {
	p.client.CloseIdleConnections()
	return nil
}

// Stats returns pool statistics
func (p *Pool) Stats() PoolStats {
	t, ok := p.client.Transport.(*http.Transport)
	if !ok {
		return PoolStats{}
	}

	return PoolStats{
		MaxIdleConns:        t.MaxIdleConns,
		MaxIdleConnsPerHost: t.MaxIdleConnsPerHost,
		MaxConnsPerHost:     t.MaxConnsPerHost,
		IdleConnTimeout:     int(t.IdleConnTimeout.Seconds()),
	}
}

// PoolStats represents pool statistics
type PoolStats struct {
	MaxIdleConns        int `json:"max_idle_conns"`
	MaxIdleConnsPerHost int `json:"max_idle_conns_per_host"`
	MaxConnsPerHost     int `json:"max_conns_per_host"`
	IdleConnTimeout     int `json:"idle_conn_timeout_seconds"`
}

// EnableH2 enables HTTP/2 support
func (p *Pool) EnableH2() {
	t, ok := p.client.Transport.(*http.Transport)
	if ok {
		t.ForceAttemptHTTP2 = true
		if err := http2.ConfigureTransport(t); err != nil {
			return
		}
	}
}

// SetTimeout sets the request timeout
func (p *Pool) SetTimeout(timeout time.Duration) {
	p.client.Timeout = timeout
}

// NewCustom creates a pool with custom transport options
func NewCustom(opts ...PoolOption) *Pool {
	cfg := DefaultConfig()

	for _, opt := range opts {
		opt(cfg)
	}

	return New(cfg)
}

// PoolOption configures a pool
type PoolOption func(*Config)

// WithMaxIdleConns sets the maximum number of idle connections
func WithMaxIdleConns(n int) PoolOption {
	return func(c *Config) {
		c.MaxIdleConns = n
	}
}

// WithMaxIdleConnsPerHost sets the maximum number of idle connections per host
func WithMaxIdleConnsPerHost(n int) PoolOption {
	return func(c *Config) {
		c.MaxIdleConnsPerHost = n
	}
}

// WithTimeout sets the request timeout
func WithTimeout(seconds int) PoolOption {
	return func(c *Config) {
		c.Timeout = seconds
	}
}

// WithInsecureTLS sets TLS insecure skip verify
func WithInsecureTLS(insecure bool) PoolOption {
	return func(c *Config) {
		c.InsecureSkipVerify = insecure
	}
}
