package grpc

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
)

func TracingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		tracer := otel.GetTracerProvider().Tracer("grpc-server-dashboard")
		ctx, span := tracer.Start(ctx, info.FullMethod)

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

		span.End()

		return resp, err
	}
}
