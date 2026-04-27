package logging

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

// mockServerInfo создаёт тестовый grpc.UnaryServerInfo.
func mockServerInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

// TestNew проверяет создание logger.
func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		level       string
		environment string
	}{
		{
			name:        "info level production",
			level:       "info",
			environment: "production",
		},
		{
			name:        "debug level development",
			level:       "debug",
			environment: "development",
		},
		{
			name:        "invalid level defaults to info",
			level:       "invalid_level",
			environment: "production",
		},
		{
			name:        "warn level",
			level:       "warn",
			environment: "test",
		},
		{
			name:        "error level",
			level:       "error",
			environment: "staging",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			l := New(tc.level, tc.environment)

			// Assert
			if l == nil {
				t.Fatal("New() returned nil")
			}
			if l.ZLogger() == nil {
				t.Error("New() ZLogger() returned nil")
			}
		})
	}
}

// TestLoggerMethods проверяет все методы логгера.
func TestLoggerMethods(t *testing.T) {
	t.Parallel()

	// Arrange
	l := New("debug", "test")

	// Act & Assert — не должны паниковать
	_ = l.Info()
	_ = l.Warn()
	_ = l.Error()
	_ = l.Debug()

	l.Infof("test %s", "info")
	l.Warnf("test %s", "warn")
	l.Errorf("test %s", "error")
}

// TestWithFields проверяет добавление полей.
func TestWithFields(t *testing.T) {
	t.Parallel()

	// Arrange
	l := New("info", "test")

	// Act
	withFields := l.WithFields(map[string]any{
		"service": "test",
		"version": "1.0",
	})

	// Assert
	if withFields == nil {
		t.Fatal("WithFields() returned nil")
	}
}

// TestContextWithLogger проверяет сохранение logger в context.
func TestContextWithLogger(t *testing.T) {
	t.Parallel()

	// Arrange
	l := New("info", "test")
	ctx := context.Background()

	// Act
	ctxWithLogger := ContextWithLogger(ctx, l)

	// Assert
	retrieved, ok := FromContext(ctxWithLogger)
	if !ok {
		t.Fatal("FromContext() ok = false, want true")
	}
	if retrieved == nil {
		t.Fatal("FromContext() returned nil logger")
	}
}

// TestFromContext_Missing проверяет извлечение logger из context без logger.
func TestFromContext_Missing(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.Background()

	// Act
	retrieved, ok := FromContext(ctx)

	// Assert
	if ok {
		t.Error("FromContext() ok = true for empty context, want false")
	}
	if retrieved != nil {
		t.Error("FromContext() should return nil for empty context")
	}
}

// TestUnaryServerInterceptor проверяет создание interceptor.
func TestUnaryServerInterceptor(t *testing.T) {
	t.Parallel()

	// Arrange
	l := New("info", "test")

	// Act
	interceptor := UnaryServerInterceptor(l)

	// Assert
	if interceptor == nil {
		t.Fatal("UnaryServerInterceptor() returned nil")
	}
}

// TestUnaryServerInterceptor_Invoke проверяет выполнение interceptor.
func TestUnaryServerInterceptor_Invoke(t *testing.T) {
	t.Parallel()

	// Arrange
	l := New("info", "test")
	interceptor := UnaryServerInterceptor(l)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	}

	ctx := context.Background()

	// Act
	resp, err := interceptor(ctx, "request", mockServerInfo("/test.Service/Method"), handler)

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
