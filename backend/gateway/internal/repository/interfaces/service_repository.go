package interfaces

import (
	"context"
	"io"

	"github.com/raul/monitor/gateway/internal/model"
)

// ServiceRepository defines the interface for downstream service operations
type ServiceRepository interface {
	// ProxyRequest proxies a request to a downstream service
	ProxyRequest(ctx context.Context, serviceName, path string, body io.Reader) (*model.ProxyResponse, error)

	// CheckHealth checks if a downstream service is healthy
	CheckHealth(ctx context.Context, serviceName string) (bool, error)
}

// ServiceHealthChecker defines the interface for checking service health
type ServiceHealthChecker interface {
	// CheckAllServices checks health of all registered services
	CheckAllServices(ctx context.Context) []model.ServiceHealth
}
