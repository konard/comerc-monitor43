package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	serviceName = "notification-template-service"
)

// Init инициализирует OpenTelemetry tracing.
func Init(jaegerEndpoint string, environment string) error {
	if jaegerEndpoint == "" {
		return fmt.Errorf("jaeger endpoint is empty")
	}

	// Создаём exporter
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(jaegerEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Создаём resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentKey.String(environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Создаём trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Регистрируем глобальный trace provider
	otel.SetTracerProvider(tp)

	return nil
}

// UnaryServerInterceptor возвращает gRPC interceptor для трассировки.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(serviceName)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		spanName := info.FullMethod

		ctx, span := tracer.Start(
			ctx,
			spanName,
			trace.WithAttributes(
				attribute.String("rpc.method", info.FullMethod),
				attribute.String("rpc.system", "grpc"),
			),
		)
		defer span.End()

		// Вызываем handler
		resp, err := handler(ctx, req)

		// Добавляем атрибуты ответа
		if err != nil {
			span.SetAttributes(attribute.String("rpc.status", "error"))
		} else {
			span.SetAttributes(attribute.String("rpc.status", "ok"))
		}

		return resp, err
	}
}

// StartSpan начинает новый span.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	return tracer.Start(ctx, name)
}
