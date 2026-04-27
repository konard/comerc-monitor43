package grpc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	httppool "github.com/raul/monitor/gateway/internal/adapter/http"
	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/repository/interfaces"
)

// serviceRepository implements the ServiceRepository interface using HTTP
type serviceRepository struct {
	httpPool    *httppool.Pool
	serviceURLs map[string]string
}

// NewServiceRepository creates a new HTTP-based service repository with http.Client
func NewServiceRepository(timeout time.Duration, serviceURLs map[string]string) interfaces.ServiceRepository {
	return NewServiceRepositoryFromPool(httppool.New(&httppool.Config{
		Timeout:               int(timeout.Seconds()),
		ResponseHeaderTimeout: int(timeout.Seconds()),
	}), serviceURLs)
}

// NewServiceRepositoryFromPool creates a repository using an HTTP connection pool
func NewServiceRepositoryFromPool(pool *httppool.Pool, serviceURLs map[string]string) interfaces.ServiceRepository {
	return &serviceRepository{
		httpPool:    pool,
		serviceURLs: serviceURLs,
	}
}

// ProxyRequest proxies a request to a downstream service
func (r *serviceRepository) ProxyRequest(ctx context.Context, serviceName, path string, body io.Reader) (*model.ProxyResponse, error) {
	ctx, span := tracer.Start(ctx, "serviceRepository.ProxyRequest")
	defer span.End()

	span.SetAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("http.path", path),
	)

	baseURL, ok := r.serviceURLs[serviceName]
	if !ok {
		err := fmt.Errorf("service not found: %s", serviceName)
		span.SetStatus(codes.Error, err.Error())
		return model.NewProxyErrorResponse(err), nil
	}

	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, body)
	if err != nil {
		return model.NewProxyErrorResponse(err), nil
	}

	resp, err := r.httpPool.Do(req)
	if err != nil {
		return model.NewProxyErrorResponse(err), nil
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.NewProxyErrorResponse(err), nil
	}

	proxyResp := model.NewProxyResponse(resp.StatusCode, respBody)
	for k, v := range resp.Header {
		if len(v) > 0 {
			proxyResp.Headers[k] = v[0]
		}
	}

	return proxyResp, nil
}

// CheckHealth checks if a downstream service is healthy
func (r *serviceRepository) CheckHealth(ctx context.Context, serviceName string) (bool, error) {
	ctx, span := tracer.Start(ctx, "serviceRepository.CheckHealth")
	defer span.End()

	span.SetAttributes(attribute.String("service.name", serviceName))

	baseURL, ok := r.serviceURLs[serviceName]
	if !ok {
		err := fmt.Errorf("service not found: %s", serviceName)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	url := baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := r.httpPool.Do(req)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return
		}
	}()

	return resp.StatusCode == http.StatusOK, nil
}

// NewServiceHealthChecker creates a service health checker
func NewServiceHealthChecker(repo interfaces.ServiceRepository, services []string) interfaces.ServiceHealthChecker {
	return &serviceHealthChecker{
		repo:     repo,
		services: services,
	}
}

type serviceHealthChecker struct {
	repo     interfaces.ServiceRepository
	services []string
}

// CheckAllServices checks health of all registered services
func (c *serviceHealthChecker) CheckAllServices(ctx context.Context) []model.ServiceHealth {
	results := make([]model.ServiceHealth, 0, len(c.services))

	for _, serviceName := range c.services {
		start := time.Now()
		healthy, err := c.repo.CheckHealth(ctx, serviceName)
		latency := time.Since(start).Milliseconds()

		message := "OK"
		if err != nil {
			message = err.Error()
		}

		results = append(results, model.ServiceHealth{
			Name:      serviceName,
			Healthy:   healthy,
			LatencyMS: latency,
			Message:   message,
		})
	}

	return results
}
