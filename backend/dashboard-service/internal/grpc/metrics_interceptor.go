package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apptelemetry "github.com/raul/monitor/backend/dashboard-service/pkg/telemetry"
)

func MetricsInterceptor(metrics *apptelemetry.Metrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		durationMs := float64(duration.Milliseconds())

		st, _ := status.FromError(err)
		if err != nil && st.Code() == codes.Unknown {
			st = status.New(codes.Unknown, err.Error())
		}
		statusCode := int(st.Code())

		if metrics != nil {
			metrics.RecordRequest(ctx, "POST", info.FullMethod, statusCode, durationMs)
		}

		return resp, err
	}
}

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
		if err != nil && st.Code() == codes.Unknown {
			st = status.New(codes.Unknown, err.Error())
		}
		statusCode := int(st.Code())

		if metrics != nil {
			metrics.RecordRequest(ss.Context(), "STREAM", info.FullMethod, statusCode, durationMs)
		}

		return err
	}
}
