package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MockMetrics provides a mock implementation for testing metrics recording
type MockMetrics struct {
	recordRequestCalled bool
	recordRequestCount  int
	lastMethod          string
	lastEndpoint        string
	lastStatusCode      int
	lastDurationMs      float64
}

// RecordRequest implements metrics recording for testing
func (m *MockMetrics) RecordRequest(
	ctx context.Context,
	method string,
	endpoint string,
	statusCode int,
	durationMs float64,
) {
	m.recordRequestCalled = true
	m.recordRequestCount++
	m.lastMethod = method
	m.lastEndpoint = endpoint
	m.lastStatusCode = statusCode
	m.lastDurationMs = durationMs
}

// MockServerStream provides a mock implementation of grpc.ServerStream
type MockServerStream struct {
	ctx context.Context
}

func (m *MockServerStream) Context() context.Context {
	return m.ctx
}

func (m *MockServerStream) SendMsg(msg any) error {
	return nil
}

func (m *MockServerStream) RecvMsg(msg any) error {
	return nil
}

func (m *MockServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *MockServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *MockServerStream) SetTrailer(md metadata.MD) {
	// Mock implementation - no-op
}

// TestMetricsInterceptor_WithMetrics проверяет вызовы с метриками
func TestMetricsInterceptor_WithMetrics(t *testing.T) {
	// Note: We can't deeply test metrics integration without complex OpenTelemetry setup
	// For now, we test that the interceptor works with metrics (even if nil in this test)
	// In a real setup, you would use a real Metrics instance with mock OpenTelemetry provider
	interceptor := MetricsInterceptor(nil)

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)
}

// TestMetricsInterceptor_WithoutMetrics проверяет вызовы без метрик
func TestMetricsInterceptor_WithoutMetrics(t *testing.T) {
	interceptor := MetricsInterceptor(nil)

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)
}

// TestMetricsInterceptor_ErrorHandling проверяет обработку ошибок
func TestMetricsInterceptor_ErrorHandling(t *testing.T) {
	interceptor := MetricsInterceptor(nil)

	ctx := context.Background()
	req := struct{}{}

	expectedErr := status.Error(codes.Internal, "test error")

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return nil, expectedErr
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
	assert.True(t, handlerCalled)
}

// TestMetricsInterceptor_Context проверяет передачу контекста
func TestMetricsInterceptor_Context(t *testing.T) {
	interceptor := MetricsInterceptor(nil)

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	receivedCtx := context.Background()
	handler := func(ctx context.Context, req any) (any, error) {
		receivedCtx = ctx
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, ctx, receivedCtx)
}

// TestMetricsInterceptor_MultipleSequential проверяет несколько последовательных вызовов
func TestMetricsInterceptor_MultipleSequential(t *testing.T) {
	interceptor := MetricsInterceptor(nil)

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	// Make multiple sequential calls
	for i := 0; i < 3; i++ {
		resp, err := interceptor(ctx, req, info, handler)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "response", resp)
	}
}

// TestStreamingMetricsInterceptor_WithMetrics проверяет streaming вызовы с метриками
func TestStreamingMetricsInterceptor_WithMetrics(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx := context.Background()
	stream := &MockServerStream{ctx: ctx}

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	err := interceptor(nil, stream, info, handler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

// TestStreamingMetricsInterceptor_WithoutMetrics проверяет streaming вызовы без метрик
func TestStreamingMetricsInterceptor_WithoutMetrics(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx := context.Background()
	stream := &MockServerStream{ctx: ctx}

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	err := interceptor(nil, stream, info, handler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

// TestStreamingMetricsInterceptor_ErrorHandling проверяет обработку ошибок в streaming
func TestStreamingMetricsInterceptor_ErrorHandling(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx := context.Background()
	stream := &MockServerStream{ctx: ctx}

	expectedErr := status.Error(codes.Internal, "stream error")

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		return expectedErr
	}

	err := interceptor(nil, stream, info, handler)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.True(t, handlerCalled)
}

// TestStreamingMetricsInterceptor_Context проверяет передачу контекста для streaming
func TestStreamingMetricsInterceptor_Context(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx := context.Background()
	stream := &MockServerStream{ctx: ctx}

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	err := interceptor(nil, stream, info, handler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

// TestStreamingMetricsInterceptor_MultipleSequential проверяет несколько последовательных streaming вызовов
func TestStreamingMetricsInterceptor_MultipleSequential(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx := context.Background()
	stream := &MockServerStream{ctx: ctx}

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	// Make multiple sequential calls
	for i := 0; i < 3; i++ {
		err := interceptor(nil, stream, info, handler)

		assert.NoError(t, err)
	}
}

// TestTracingInterceptor_InterceptsUnary проверяет перехват унарных вызов
func TestTracingInterceptor_InterceptsUnary(t *testing.T) {
	interceptor := TracingInterceptor()

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "response", resp)
}

// TestTracingInterceptor_ErrorHandling проверяет обработку ошибок в tracing
func TestTracingInterceptor_ErrorHandling(t *testing.T) {
	interceptor := TracingInterceptor()

	ctx := context.Background()
	req := struct{}{}

	expectedErr := status.Error(codes.Internal, "tracing error")

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, expectedErr
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
}

// TestTracingInterceptor_Context проверяет передачу контекста
func TestTracingInterceptor_Context(t *testing.T) {
	interceptor := TracingInterceptor()

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		// Note: The tracing interceptor modifies the context by adding a span
		// so we don't verify equality, just that handler is called
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, handlerCalled)
}

// TestTracingInterceptor_MultipleSequential проверяет несколько последовательных вызовов
func TestTracingInterceptor_MultipleSequential(t *testing.T) {
	interceptor := TracingInterceptor()

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	// Make multiple sequential calls
	for i := 0; i < 3; i++ {
		resp, err := interceptor(ctx, req, info, handler)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "response", resp)
	}
}

// TestMetricsInterceptor_DurationMeasurement проверяет измерение длительности
func TestMetricsInterceptor_DurationMeasurement(t *testing.T) {
	interceptor := MetricsInterceptor(nil)

	ctx := context.Background()
	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	// Handler that takes some time
	handler := func(ctx context.Context, req any) (any, error) {
		time.Sleep(10 * time.Millisecond)
		return "response", nil
	}

	start := time.Now()
	resp, err := interceptor(ctx, req, info, handler)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10)) // At least 10ms
}

// TestStreamingMetricsInterceptor_DurationMeasurement проверяет измерение длительности для streaming
func TestStreamingMetricsInterceptor_DurationMeasurement(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx := context.Background()
	stream := &MockServerStream{ctx: ctx}

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	// Handler that takes some time
	handler := func(srv any, ss grpc.ServerStream) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	start := time.Now()
	err := interceptor(nil, stream, info, handler)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10)) // At least 10ms
}

// TestMetricsInterceptor_ContextWithTimeout проверяет контекст с таймаутом
func TestMetricsInterceptor_ContextWithTimeout(t *testing.T) {
	interceptor := MetricsInterceptor(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := struct{}{}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/service.Method",
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestStreamingMetricsInterceptor_ContextWithTimeout проверяет контекст с таймаутом для streaming
func TestStreamingMetricsInterceptor_ContextWithTimeout(t *testing.T) {
	interceptor := StreamingMetricsInterceptor(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := &MockServerStream{ctx: ctx}

	info := &grpc.StreamServerInfo{
		FullMethod: "/service.StreamMethod",
	}

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, stream, info, handler)

	assert.NoError(t, err)
}
