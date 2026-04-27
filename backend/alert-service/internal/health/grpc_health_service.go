package health

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
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

// Watch реализует потоковую проверку здоровья (упрощённая версия)
func (s *GRPCHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// Отправляем первоначальный статус
	ctx := stream.Context()

	if err := s.checker.Check(ctx); err != nil {
		if sendErr := stream.Send(&grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}); sendErr != nil {
			return status.Errorf(codes.Internal, "failed to send health status: %v", sendErr)
		}
		return status.Error(codes.Unavailable, "service not healthy")
	}

	// Для упрощения отправляем один статус и закрываем поток
	// В продакшене можно сделать периодические проверки
	if err := stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}); err != nil {
		return status.Errorf(codes.Internal, "failed to send health status: %v", err)
	}

	return nil
}

// NewGRPCHealthServerWithDB создаёт health server напрямую из *sqlx.DB
func NewGRPCHealthServerWithDB(db interface {
	PingContext(ctx context.Context) error
}) *GRPCHealthServer {
	checker := NewHealthCheckerFromInterface(db)
	return NewGRPCHealthServer(checker)
}
