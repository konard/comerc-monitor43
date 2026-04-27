package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/codes"

	"github.com/raul/monitor/gateway/internal/repository/interfaces"
	"github.com/raul/monitor/gateway/internal/service/dto"
)

// GatewayService handles gateway-related business logic
type GatewayService struct {
	authRepo      interfaces.AuthRepository
	serviceRepo   interfaces.ServiceRepository
	healthChecker interfaces.ServiceHealthChecker
	services      []string
	startTime     time.Time
}

// NewGatewayService creates a new gateway service
func NewGatewayService(
	authRepo interfaces.AuthRepository,
	serviceRepo interfaces.ServiceRepository,
	healthChecker interfaces.ServiceHealthChecker,
	services []string,
) *GatewayService {
	return &GatewayService{
		authRepo:      authRepo,
		serviceRepo:   serviceRepo,
		healthChecker: healthChecker,
		services:      services,
		startTime:     time.Now(),
	}
}

// CheckHealth performs a comprehensive health check
func (s *GatewayService) CheckHealth(ctx context.Context) *dto.HealthResponse {
	ctx, span := tracer.Start(ctx, "GatewayService.CheckHealth")
	defer span.End()

	start := time.Now()

	// Check auth service
	authHealthy, authLatency, authErr := s.CheckAuthHealth(ctx)

	// Check all downstream services
	serviceHealthResults := s.healthChecker.CheckAllServices(ctx)

	// Determine overall status
	status := "healthy"
	if !authHealthy {
		status = "unhealthy"
	} else {
		for _, sh := range serviceHealthResults {
			if !sh.Healthy {
				status = "degraded"
				break
			}
		}
	}

	response := dto.NewHealthResponse(status, start.Format(time.RFC3339))
	response.Gateway.StartedAt = s.startTime.Format(time.RFC3339)

	// Add auth service health
	response.Services = append(response.Services, dto.ServiceHealthDTO{
		Name:      "auth-service",
		Healthy:   authHealthy,
		LatencyMS: authLatency.Milliseconds(),
		Message:   errorMessage(authErr),
	})

	// Add downstream services health
	for _, sh := range serviceHealthResults {
		response.Services = append(response.Services, dto.ServiceHealthDTO{
			Name:      sh.Name,
			Healthy:   sh.Healthy,
			LatencyMS: sh.LatencyMS,
			Message:   sh.Message,
		})
	}

	return response
}

// CheckAuthHealth checks the auth service health
func (s *GatewayService) CheckAuthHealth(ctx context.Context) (bool, time.Duration, error) {
	start := time.Now()
	healthy, err := s.authRepo.CheckHealth(ctx)
	return healthy, time.Since(start), err
}

// ProxyRequest proxies a request to a downstream service
func (s *GatewayService) ProxyRequest(ctx context.Context, serviceName, path string, body []byte) (*dto.ProxyResponse, error) {
	ctx, span := tracer.Start(ctx, "GatewayService.ProxyRequest")
	defer span.End()

	// заглушка — реализация после готовности нижестоящих сервисов
	_ = ctx
	_ = span
	_ = codes.Ok
	return &dto.ProxyResponse{
		StatusCode: 200,
		Message:    "Service proxy not yet implemented",
	}, nil
}

// GetUptime returns the gateway uptime
func (s *GatewayService) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
