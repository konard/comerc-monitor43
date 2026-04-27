// Package http_checker предоставляет HTTP клиент для выполнения проверок.
//
// Пакет поддерживает:
//   - Настройку таймаутов
//   - Следование редиректам
//   - Измерение response time
//   - Обработку различных ошибок сети
package http_checker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// HTTPChecker выполняет HTTP проверки.
type HTTPChecker struct {
	client *http.Client
}

// Config конфигурация HTTPChecker.
type Config struct {
	Timeout         time.Duration
	MaxRedirects    int
	UserAgent       string
	KeepAlive       time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

// NewHTTPChecker создаёт новый HTTPChecker.
func NewHTTPChecker(config Config) *HTTPChecker {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRedirects == 0 {
		config.MaxRedirects = 10
	}
	if config.KeepAlive == 0 {
		config.KeepAlive = 30 * time.Second
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.IdleConnTimeout == 0 {
		config.IdleConnTimeout = 90 * time.Second
	}

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:      config.MaxIdleConns,
			IdleConnTimeout:   config.IdleConnTimeout,
			DisableKeepAlives: false,
			MaxConnsPerHost:   10,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &HTTPChecker{
		client: client,
	}
}

// CheckResult результат HTTP проверки.
type CheckResult struct {
	StatusCode    int
	ResponseTime  time.Duration
	Error         error
	Success       bool
	ContentLength int64
}

// Check выполняет HTTP проверку URL.
func (c *HTTPChecker) Check(ctx context.Context, url string) *CheckResult {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &CheckResult{
			Error: errors.Wrap(err, "failed to create request"),
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return &CheckResult{
			Error: errors.Wrap(err, "failed to execute request"),
		}
	}

	responseTime := time.Since(startTime)

	result := &CheckResult{
		StatusCode:    resp.StatusCode,
		ResponseTime:  responseTime,
		Success:       resp.StatusCode >= 200 && resp.StatusCode < 400,
		ContentLength: resp.ContentLength,
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		result.Error = errors.Wrap(closeErr, "failed to close response body")
		result.Success = false
	}

	return result
}
