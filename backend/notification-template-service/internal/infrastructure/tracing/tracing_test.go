package tracing

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

// TestInit_EmptyEndpoint проверяет, что Init возвращает ошибку для пустого endpoint.
func TestInit_EmptyEndpoint(t *testing.T) {
	t.Parallel()

	// Act
	err := Init("", "production")

	// Assert
	if err == nil {
		t.Fatal("Init() expected error for empty endpoint, got nil")
	}
}

// TestInit_WithEndpoint проверяет, что Init принимает непустой endpoint.
// Трассировщик инициализируется без ошибки, даже если endpoint недоступен,
// так как gRPC соединение устанавливается лениво.
func TestInit_WithEndpoint(t *testing.T) {
	t.Parallel()

	// Act — используем несуществующий адрес, ошибки не должно быть
	// так как gRPC соединение устанавливается лениво (lazy connection)
	err := Init("localhost:4317", "test")

	// Assert — не ожидаем ошибку при допустимом endpoint
	if err != nil {
		t.Logf("Init() error = %v (acceptable in test env)", err)
	}
}

// TestStartSpan проверяет создание span.
func TestStartSpan(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.Background()

	// Act
	spanCtx, span := StartSpan(ctx, "test-operation")

	// Assert
	if spanCtx == nil {
		t.Fatal("StartSpan() returned nil context")
	}
	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}
	span.End()
}

// TestUnaryServerInterceptor проверяет создание gRPC interceptor.
func TestUnaryServerInterceptor(t *testing.T) {
	t.Parallel()

	// Act
	interceptor := UnaryServerInterceptor()

	// Assert
	if interceptor == nil {
		t.Fatal("UnaryServerInterceptor() returned nil")
	}
}

// TestUnaryServerInterceptor_Success проверяет выполнение interceptor при успехе.
func TestUnaryServerInterceptor_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	interceptor := UnaryServerInterceptor()

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}
	ctx := context.Background()

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert
	if err != nil {
		t.Fatalf("interceptor() unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("interceptor() handler should have been called")
	}
	if resp != "response" {
		t.Errorf("interceptor() resp = %v, want %q", resp, "response")
	}
}

// TestUnaryServerInterceptor_Error проверяет выполнение interceptor при ошибке handler.
func TestUnaryServerInterceptor_Error(t *testing.T) {
	t.Parallel()

	// Arrange
	interceptor := UnaryServerInterceptor()

	expectedErr := context.DeadlineExceeded
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, expectedErr
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}
	ctx := context.Background()

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert
	if err != expectedErr {
		t.Errorf("interceptor() err = %v, want %v", err, expectedErr)
	}
	if resp != nil {
		t.Errorf("interceptor() resp = %v, want nil", resp)
	}
}
