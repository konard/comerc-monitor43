package handler

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
)

// TracingInterceptor возвращает gRPC interceptor для OpenTelemetry трассировки
func TracingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Начинаем span
		tracer := otel.GetTracerProvider().Tracer("grpc-server")
		ctx, span := tracer.Start(ctx, info.FullMethod)

		// Добавляем атрибуты
		span.SetAttributes(
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.system", "grpc"),
		)

		// Вызываем handler
		resp, err := handler(ctx, req)

		// Записываем ошибку если есть
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		span.End()

		return resp, err
	}
}
