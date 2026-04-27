package handler

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
)

// TracingInterceptor возвращает gRPC unary interceptor для трассировки OpenTelemetry.
func TracingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		tracer := otel.GetTracerProvider().Tracer("grpc-server")
		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		span.SetAttributes(
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.system", "grpc"),
		)

		resp, err := handler(ctx, req)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}
