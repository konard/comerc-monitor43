package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	client := NewClient(5 * time.Second)
	require.NotNil(t, client)
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		url       string
		wantErr   bool
		errTarget error
	}{
		{
			name:    "valid https url",
			url:     "https://example.com/webhook",
			wantErr: false,
		},
		{
			name:    "valid http url",
			url:     "http://example.com/webhook",
			wantErr: false,
		},
		{
			name:      "ftp scheme rejected",
			url:       "ftp://example.com",
			wantErr:   true,
			errTarget: model.ErrWebhookInvalidURL,
		},
		{
			name:      "no scheme rejected",
			url:       "example.com/webhook",
			wantErr:   true,
			errTarget: model.ErrWebhookInvalidURL,
		},
		{
			name:      "javascript injection",
			url:       "https://example.com/javascript:alert(1)",
			wantErr:   true,
			errTarget: model.ErrWebhookInjection,
		},
		{
			name:      "script tag injection",
			url:       "https://example.com/<script>alert(1)</script>",
			wantErr:   true,
			errTarget: model.ErrWebhookInjection,
		},
		{
			name:      "path traversal injection",
			url:       "https://example.com/../etc/passwd",
			wantErr:   true,
			errTarget: model.ErrWebhookInjection,
		},
		{
			name:      "file scheme injection",
			url:       "file:///etc/passwd",
			wantErr:   true,
			errTarget: model.ErrWebhookInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.ErrorIs(t, err, tt.errTarget)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContainsInjection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "clean url",
			url:      "https://example.com/webhook",
			expected: false,
		},
		{
			name:     "script tag",
			url:      "https://example.com/<script>",
			expected: true,
		},
		{
			name:     "javascript scheme",
			url:      "https://example.com/javascript:alert(1)",
			expected: true,
		},
		{
			name:     "onerror event",
			url:      "https://example.com/onerror=alert",
			expected: true,
		},
		{
			name:     "onload event",
			url:      "https://example.com/onload=test",
			expected: true,
		},
		{
			name:     "path traversal forward slash",
			url:      "https://example.com/../etc",
			expected: true,
		},
		{
			name:     "path traversal backslash",
			url:      "https://example.com/..\\etc",
			expected: true,
		},
		{
			name:     "data scheme",
			url:      "https://example.com/data:text/html",
			expected: true,
		},
		{
			name:     "closing script tag",
			url:      "https://example.com/</script>",
			expected: true,
		},
		{
			name:     "uppercase injection is also detected",
			url:      "https://example.com/JAVASCRIPT:alert(1)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, containsInjection(tt.url))
		})
	}
}

func TestTruncatePayload(t *testing.T) {
	t.Parallel()

	t.Run("payload within limit not truncated", func(t *testing.T) {
		t.Parallel()
		payload := []byte("hello world")
		result := truncatePayload(payload, 100)
		assert.Equal(t, payload, result)
	})

	t.Run("payload exceeds limit is truncated", func(t *testing.T) {
		t.Parallel()
		payload := []byte("hello world this is a long string")
		result := truncatePayload(payload, 5)
		assert.Equal(t, []byte("hello"), result)
	})

	t.Run("payload exactly at limit not truncated", func(t *testing.T) {
		t.Parallel()
		payload := []byte("hello")
		result := truncatePayload(payload, 5)
		assert.Equal(t, payload, result)
	})
}

func TestGenerateHMACSignature(t *testing.T) {
	t.Parallel()

	t.Run("generates sha256 prefix", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig := generateHMACSignature(payload, "mysecret", 1700000000)
		assert.True(t, len(sig) > 7)
		assert.Equal(t, "sha256=", sig[:7])
	})

	t.Run("same input produces same signature", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig1 := generateHMACSignature(payload, "secret", 1700000000)
		sig2 := generateHMACSignature(payload, "secret", 1700000000)
		assert.Equal(t, sig1, sig2)
	})

	t.Run("different secret produces different signature", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig1 := generateHMACSignature(payload, "secret1", 1700000000)
		sig2 := generateHMACSignature(payload, "secret2", 1700000000)
		assert.NotEqual(t, sig1, sig2)
	})
}

func TestHandleHTTPError(t *testing.T) {
	t.Parallel()

	c := NewClient(5 * time.Second)

	tests := []struct {
		name       string
		statusCode int
		errTarget  error
	}{
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			errTarget:  model.ErrWebhookAuthFailed,
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			errTarget:  model.ErrWebhookInvalidURL,
		},
		{
			name:       "408 request timeout",
			statusCode: http.StatusRequestTimeout,
			errTarget:  model.ErrWebhookTimeout,
		},
		{
			name:       "400 bad request permanent error",
			statusCode: http.StatusBadRequest,
			errTarget:  model.ErrDeliveryPermanentError,
		},
		{
			name:       "500 internal server error",
			statusCode: http.StatusInternalServerError,
			errTarget:  model.ErrWebhook5xxError,
		},
		{
			name:       "503 service unavailable",
			statusCode: http.StatusServiceUnavailable,
			errTarget:  model.ErrWebhook5xxError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := c.handleHTTPError(tt.statusCode, "error body")
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.errTarget)
		})
	}
}

func TestSendWebhookSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)

	resp, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:     server.URL + "/webhook",
		Method:  "POST",
		Payload: map[string]string{"key": "value"},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Body, "ok")
}

func TestSendWebhookWithSecret(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Signature"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Timestamp"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)
	secret := "my-webhook-secret" //nolint:gosec // G101: тестовые данные

	resp, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:       server.URL + "/webhook",
		Method:    "POST",
		Payload:   map[string]string{"key": "value"},
		SecretKey: &secret,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestSendWebhookPayloadTooLarge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)

	// Payload that is over the limit, with reject strategy
	largePayload := make([]byte, 1000)
	for i := range largePayload {
		largePayload[i] = 'a'
	}

	_, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Payload:             string(largePayload),
		MaxPayloadSizeBytes: 100,
		TruncateOnOverflow:  false,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrWebhookPayloadTooLarge)
}

func TestSendWebhookPayloadTruncated(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)

	_, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Payload:             map[string]string{"key": "value_that_is_long"},
		MaxPayloadSizeBytes: 5, // very small
		TruncateOnOverflow:  true,
	})

	// Truncated - request should succeed (server returns 200)
	require.NoError(t, err)
}

func TestSendWebhookInvalidURL(t *testing.T) {
	t.Parallel()

	c := NewClient(5 * time.Second)

	_, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:     "not-a-url",
		Method:  "POST",
		Payload: map[string]string{"key": "value"},
	})

	require.Error(t, err)
}

func TestSendWebhookHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)

	_, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:     server.URL + "/webhook",
		Method:  "POST",
		Payload: map[string]string{"key": "value"},
	})

	require.Error(t, err)
}

func TestTestWebhookSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)
	err := c.TestWebhook(context.Background(), server.URL+"/webhook", 5)
	assert.NoError(t, err)
}

func TestTestWebhookInvalidURL(t *testing.T) {
	t.Parallel()

	c := NewClient(5 * time.Second)
	err := c.TestWebhook(context.Background(), "not-a-url", 5)
	require.Error(t, err)
}

func TestTestWebhookHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)
	err := c.TestWebhook(context.Background(), server.URL+"/webhook", 5)
	require.Error(t, err)
}

func TestHandleRequestError(t *testing.T) {
	t.Parallel()

	c := NewClient(5 * time.Second)

	t.Run("context deadline exceeded returns timeout error", func(t *testing.T) {
		t.Parallel()
		err := c.handleRequestError(fmt.Errorf("context deadline exceeded"))
		assert.ErrorIs(t, err, model.ErrWebhookTimeout)
	})

	t.Run("other error wraps with message", func(t *testing.T) {
		t.Parallel()
		err := c.handleRequestError(fmt.Errorf("connection refused"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "webhook request failed")
	})
}

func TestSendWebhookWithCustomHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)

	_, err := c.SendWebhook(context.Background(), &WebhookRequest{
		URL:     server.URL + "/webhook",
		Method:  "POST",
		Headers: map[string]string{"Authorization": "Bearer token123"},
		Payload: map[string]string{"key": "value"},
	})

	require.NoError(t, err)
}

func TestTestWebhookTimeoutStatusCode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)
	err := c.TestWebhook(context.Background(), server.URL+"/webhook", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrWebhookTimeout)
}

func TestTestWebhookGatewayTimeoutStatusCode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	t.Cleanup(server.Close)

	c := NewClient(5 * time.Second)
	err := c.TestWebhook(context.Background(), server.URL+"/webhook", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrWebhookTimeout)
}

func TestTestWebhookConnectionRefused(t *testing.T) {
	t.Parallel()

	c := NewClient(1 * time.Second)
	// Port that is not listening
	err := c.TestWebhook(context.Background(), "http://127.0.0.1:19999/webhook", 1)
	require.Error(t, err)
}

func TestHandleRequestErrorURLTimeout(t *testing.T) {
	t.Parallel()

	c := NewClient(5 * time.Second)

	// Simulate url.Error with Timeout()
	urlErr := &url.Error{
		Op:  "Post",
		URL: "http://example.com",
		Err: context.DeadlineExceeded,
	}
	err := c.handleRequestError(urlErr)
	assert.ErrorIs(t, err, model.ErrWebhookTimeout)
}

func TestValidateURLInjection(t *testing.T) {
	t.Parallel()

	err := validateURL("https://example.com/path?q=<script>alert(1)</script>")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrWebhookInjection)
}
