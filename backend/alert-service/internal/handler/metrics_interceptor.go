package handler

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

// MetricsInterceptor возвращает gRPC interceptor для записи метрик
func MetricsInterceptor(metrics *apptelemetry.Metrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		// Вызываем handler
		resp, err := handler(ctx, req)

		// Вычисляем длительность запроса
		duration := time.Since(start)
		durationMs := float64(duration.Milliseconds())

		// Получаем статус код
		st, _ := status.FromError(err)
		statusCode := int(st.Code())

		// Записываем метрики
		if metrics != nil {
			metrics.RecordRequest(ctx, "POST", info.FullMethod, statusCode, durationMs)
		}

		return resp, err
	}
}

// StreamingMetricsInterceptor возвращает interceptor для streaming RPC
func StreamingMetricsInterceptor(metrics *apptelemetry.Metrics) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		durationMs := float64(duration.Milliseconds())

		st, _ := status.FromError(err)
		statusCode := int(st.Code())

		if metrics != nil {
			metrics.RecordRequest(context.Background(), "STREAM", info.FullMethod, statusCode, durationMs)
		}

		return err
	}
}
