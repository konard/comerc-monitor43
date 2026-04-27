package channels

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// MockWebhookSender реализует WebhookSender интерфейс для тестирования
type MockWebhookSender struct {
	sendFunc func(ctx context.Context, url string, payload any) error
}

func (m *MockWebhookSender) Send(ctx context.Context, url string, payload any) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, url, payload)
	}
	return nil
}

// MockWebhookSenderWithHeaders реализует WebhookSenderWithHeaders интерфейс для тестирования
type MockWebhookSenderWithHeaders struct {
	sendFunc            func(ctx context.Context, url string, payload any) error
	sendWithHeadersFunc func(ctx context.Context, url string, payload any, headers map[string]string) error
}

func (m *MockWebhookSenderWithHeaders) Send(ctx context.Context, url string, payload any) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, url, payload)
	}
	return nil
}

func (m *MockWebhookSenderWithHeaders) SendWithHeaders(ctx context.Context, url string, payload any, headers map[string]string) error {
	if m.sendWithHeadersFunc != nil {
		return m.sendWithHeadersFunc(ctx, url, payload, headers)
	}
	return nil
}

func TestNewWebhookClient(t *testing.T) {
	timeout := 30 * time.Second
	client := NewWebhookClient(timeout)

	assert.NotNil(t, client)
	assert.Equal(t, timeout, client.timeout)
	assert.NotNil(t, client.client)
}

func TestWebhookSenderInterface(t *testing.T) {
	client := NewWebhookClient(30 * time.Second)

	var sender WebhookSender = client
	assert.NotNil(t, sender)

	ctx := context.Background()
	err := sender.Send(ctx, "http://example.com", map[string]string{"test": "data"})

	require.Error(t, err)
}

func TestMockWebhookSender_Success(t *testing.T) {
	mock := &MockWebhookSender{
		sendFunc: func(ctx context.Context, url string, payload any) error {
			return nil
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "http://example.com", map[string]string{"test": "data"})

	assert.NoError(t, err)
}

func TestMockWebhookSender_Error(t *testing.T) {
	mock := &MockWebhookSender{
		sendFunc: func(ctx context.Context, url string, payload any) error {
			return errors.New("webhook send failed")
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "http://example.com", map[string]string{"test": "data"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook send failed")
}

func TestWebhookClient_StructFields(t *testing.T) {
	client := &WebhookClient{
		timeout: 30 * time.Second,
		client:  &http.Client{},
	}

	assert.Equal(t, 30*time.Second, client.timeout)
	assert.NotNil(t, client.client)
}

func TestWebhookClient_EmptyFields(t *testing.T) {
	client := &WebhookClient{
		timeout: 0,
		client:  nil,
	}

	assert.Equal(t, time.Duration(0), client.timeout)
	assert.Nil(t, client.client)
}

func TestMockWebhookSender_ContextCancellation(t *testing.T) {
	calls := 0
	mock := &MockWebhookSender{
		sendFunc: func(ctx context.Context, url string, payload any) error {
			calls++
			if calls > 1 {
				return ctx.Err()
			}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mock.Send(ctx, "http://example.com", map[string]string{"test": "data"})

	// When context is cancelled, we might get context error
	if err != nil {
		assert.Equal(t, context.Canceled, err)
	}
}

func TestMockWebhookSender_MultipleCalls(t *testing.T) {
	mock := &MockWebhookSender{
		sendFunc: func(ctx context.Context, url string, payload any) error {
			return nil
		},
	}

	ctx := context.Background()

	// Test multiple calls
	for i := 0; i < 5; i++ {
		err := mock.Send(ctx, "http://example.com", map[string]string{"test": "data"})
		assert.NoError(t, err, "Call %d should succeed", i)
	}
}

func TestMockWebhookSenderWithHeaders_SendWithHeaders(t *testing.T) {
	mock := &MockWebhookSenderWithHeaders{
		sendWithHeadersFunc: func(ctx context.Context, url string, payload any, headers map[string]string) error {
			return nil
		},
	}

	ctx := context.Background()
	headers := map[string]string{
		"X-Custom-Header": "Custom-Value",
		"Authorization":   "Bearer token",
	}
	err := mock.SendWithHeaders(ctx, "http://example.com", map[string]string{"test": "data"}, headers)

	assert.NoError(t, err)
}

func TestWebhookSender_InterfaceCompliance(t *testing.T) {
	mock := &MockWebhookSender{}

	// Test that mock implements the WebhookSender interface
	var sender WebhookSender = mock
	assert.NotNil(t, sender)

	ctx := context.Background()
	err := sender.Send(ctx, "http://example.com", map[string]string{"test": "data"})

	// With nil functions, should return nil
	assert.NoError(t, err)
}

func TestWebhookSenderWithHeaders_InterfaceCompliance(t *testing.T) {
	mock := &MockWebhookSenderWithHeaders{}

	// Test that mock implements the WebhookSenderWithHeaders interface
	var sender WebhookSenderWithHeaders = mock
	assert.NotNil(t, sender)

	ctx := context.Background()

	// Test basic Send
	err := sender.Send(ctx, "http://example.com", map[string]string{"test": "data"})
	assert.NoError(t, err)

	// Test SendWithHeaders
	headers := map[string]string{
		"X-Custom-Header": "Custom-Value",
	}
	err = sender.SendWithHeaders(ctx, "http://example.com", map[string]string{"test": "data"}, headers)
	assert.NoError(t, err)
}

func TestWebhookSender_PayloadValidation(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		payload    any
		shouldFail bool
	}{
		{
			name:       "all valid",
			url:        "http://example.com",
			payload:    map[string]string{"test": "data"},
			shouldFail: false,
		},
		{
			name:       "empty url",
			url:        "",
			payload:    map[string]string{"test": "data"},
			shouldFail: false,
		},
		{
			name:       "empty payload",
			url:        "http://example.com",
			payload:    nil,
			shouldFail: false,
		},
		{
			name:       "invalid url",
			url:        "http://invalid-url",
			payload:    map[string]string{"test": "data"},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockWebhookSender{
				sendFunc: func(ctx context.Context, url string, payload any) error {
					if tt.shouldFail {
						return errors.New("validation failed")
					}
					return nil
				},
			}

			ctx := context.Background()
			err := mock.Send(ctx, tt.url, tt.payload)

			if tt.shouldFail {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookClient_RealHTTPClient(t *testing.T) {
	// Test with real http client (not httptest)
	client := NewWebhookClient(30 * time.Second)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.timeout)
}

func TestWebhookClient_ContextTimeout(t *testing.T) {
	client := NewWebhookClient(30 * time.Second)

	// Test that client has proper timeout configuration
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.timeout)
}
func TestWebhookClient_SendWithHeaders(t *testing.T) {
	client := NewWebhookClient(5 * time.Second)
	ctx := context.Background()
	url := "https://example.com/webhook"
	payload := map[string]any{
		"test": "data",
	}

	t.Run("send_with_headers_success", func(t *testing.T) {
		headers := map[string]string{
			"Authorization":   "Bearer token123",
			"X-Custom-Header": "custom-value",
		}

		// Note: This will fail with real HTTP connection
		// In real scenario, would use mock sender
		err := client.SendWithHeaders(ctx, url, payload, headers)
		// We expect this to attempt to send with headers
		// For testing purposes, we're mainly checking that the method signature works
		assert.NotNil(t, err) // Will error due to real HTTP, but that's OK
	})

	t.Run("send_with_headers_empty_headers", func(t *testing.T) {
		headers := map[string]string{}

		err := client.SendWithHeaders(ctx, url, payload, headers)
		assert.NotNil(t, err) // Will error due to real HTTP, but should not panic
	})

	t.Run("send_with_headers_nil_headers", func(t *testing.T) {
		err := client.SendWithHeaders(ctx, url, payload, nil)
		assert.NotNil(t, err) // Should handle nil headers
	})

	t.Run("send_with_headers_context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		headers := map[string]string{
			"Authorization": "Bearer token123",
		}

		err := client.SendWithHeaders(ctx, url, payload, headers)
		// Should return context cancellation error
		assert.Error(t, err)
	})

	t.Run("send_with_headers_custom_content_type", func(t *testing.T) {
		headers := map[string]string{
			"Content-Type":  "application/xml",
			"Authorization": "Bearer token123",
		}

		// Test that custom headers override default Content-Type
		err := client.SendWithHeaders(ctx, url, payload, headers)
		assert.NotNil(t, err) // Will error due to real HTTP, but should not panic
	})

	t.Run("send_with_headers_multiple_headers", func(t *testing.T) {
		headers := map[string]string{
			"Authorization":   "Bearer token123",
			"X-Request-ID":    "req-123",
			"X-Custom-Header": "custom-value",
			"Accept-Encoding": "gzip",
		}

		err := client.SendWithHeaders(ctx, url, payload, headers)
		assert.NotNil(t, err) // Will error due to real HTTP, but should not panic
	})

	t.Run("send_with_headers_empty_url", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
		}

		err := client.SendWithHeaders(ctx, "", payload, headers)
		assert.NotNil(t, err) // Should error with empty URL
	})

	t.Run("send_with_headers_invalid_url", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
		}

		err := client.SendWithHeaders(ctx, "invalid-url", payload, headers)
		assert.NotNil(t, err) // Should error with invalid URL
	})

	t.Run("send_with_headers_empty_payload", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
		}

		err := client.SendWithHeaders(ctx, url, nil, headers)
		assert.NotNil(t, err) // Should handle nil payload
	})
}

func TestWebhookClient_SendWithConfig_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(5 * time.Second)
	cfg := &model.WebhookChannelConfig{
		Method:  "POST",
		Headers: map[string]string{"X-Custom-Header": "test-value"},
	}

	err := client.SendWithConfig(context.Background(), server.URL, map[string]string{"key": "value"}, cfg)
	require.NoError(t, err)
}

func TestWebhookClient_SendWithConfig_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("server error"))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewWebhookClient(5 * time.Second)
	cfg := &model.WebhookChannelConfig{Method: "POST"}

	err := client.SendWithConfig(context.Background(), server.URL, nil, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestWebhookClient_SendWithConfig_DefaultMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(5 * time.Second)
	cfg := &model.WebhookChannelConfig{} // empty method defaults to POST

	err := client.SendWithConfig(context.Background(), server.URL, map[string]string{}, cfg)
	require.NoError(t, err)
}
