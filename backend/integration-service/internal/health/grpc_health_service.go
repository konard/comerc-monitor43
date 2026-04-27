package health

import (
	"context"

	"google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCHealthServer реализует grpc_health_v1.HealthServer
type GRPCHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	checker *HealthChecker
}

// NewGRPCHealthServer создаёт новый gRPC health server
func NewGRPCHealthServer(checker *HealthChecker) *GRPCHealthServer {
	return &GRPCHealthServer{
		checker: checker,
	}
}

// Check реализует проверку здоровья по протоколу gRPC Health v1
func (s *GRPCHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Проверяем здоровье сервиса
	if err := s.checker.Check(ctx); err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch реализует потоковую проверку здоровья
func (s *GRPCHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	ctx := stream.Context()

	// Отправляем первоначальный статус
	if err := s.checker.Check(ctx); err != nil {
		return stream.Send(&grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		})
	}

	// Отправляем SERVING статус
	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}
