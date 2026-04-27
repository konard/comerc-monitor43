package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- MetricsInterceptor ---

func TestMetricsInterceptor_nil_metrics_success(t *testing.T) {
	t.Parallel()

	interceptor := MetricsInterceptor(nil)

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	resp, err := interceptor(context.Background(), nil, &googlegrpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)
}

func TestMetricsInterceptor_nil_metrics_handler_error(t *testing.T) {
	t.Parallel()

	interceptor := MetricsInterceptor(nil)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}

	resp, err := interceptor(context.Background(), nil, &googlegrpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestMetricsInterceptor_nil_metrics_unknown_error(t *testing.T) {
	t.Parallel()

	interceptor := MetricsInterceptor(nil)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("raw error")
	}

	_, err := interceptor(context.Background(), nil, &googlegrpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	assert.Error(t, err)
}

// --- StreamingMetricsInterceptor ---

type mockServerStream struct {
	googlegrpc.ServerStream
}

func (m *mockServerStream) Context() context.Context { return context.Background() }

func TestStreamingMetricsInterceptor_nil_metrics_success(t *testing.T) {
	t.Parallel()

	interceptor := StreamingMetricsInterceptor(nil)

	handler := func(srv any, ss googlegrpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, &mockServerStream{}, &googlegrpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"}, handler)

	require.NoError(t, err)
}

func TestStreamingMetricsInterceptor_nil_metrics_handler_error(t *testing.T) {
	t.Parallel()

	interceptor := StreamingMetricsInterceptor(nil)

	handler := func(srv any, ss googlegrpc.ServerStream) error {
		return status.Error(codes.Internal, "internal error")
	}

	err := interceptor(nil, &mockServerStream{}, &googlegrpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"}, handler)

	assert.Error(t, err)
}

// --- TracingInterceptor ---

func TestTracingInterceptor_success(t *testing.T) {
	t.Parallel()

	interceptor := TracingInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, &googlegrpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestTracingInterceptor_handler_error(t *testing.T) {
	t.Parallel()

	interceptor := TracingInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("tracing test error")
	}

	_, err := interceptor(context.Background(), nil, &googlegrpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	assert.Error(t, err)
}
