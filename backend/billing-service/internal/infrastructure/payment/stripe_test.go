package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStripeProvider создаёт провайдер Stripe с тестовым HTTP-сервером.
func newTestStripeProvider(server *httptest.Server) *stripeProvider {
	p := NewStripeProvider("test-api-key", "test-webhook-secret").(*stripeProvider)
	p.baseURL = server.URL
	p.client = server.Client()
	return p
}

func TestStripeProvider_CreatePayment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "payment_intents")

			resp := map[string]any{
				"id":     "pi_test_123",
				"status": "requires_payment_method",
				"amount": int64(29900),
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		req := &CreatePaymentRequest{
			AmountKopeks: 29900,
			Currency:     "RUB",
			Description:  "Test payment",
			Metadata:     map[string]string{"user_id": "user-1"},
			ReturnURL:    "https://example.com/return",
		}

		// Act
		resp, err := p.CreatePayment(context.Background(), req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "pi_test_123", resp.PaymentID)
		assert.Equal(t, "requires_payment_method", resp.Status)
		assert.Equal(t, int64(29900), resp.AmountKopeks)
	})

	t.Run("api_error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"error": "invalid request"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		req := &CreatePaymentRequest{
			AmountKopeks: 29900,
			Currency:     "RUB",
		}

		_, err := p.CreatePayment(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stripe api error")
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`not valid json`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		req := &CreatePaymentRequest{
			AmountKopeks: 29900,
			Currency:     "RUB",
		}

		_, err := p.CreatePayment(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestStripeProvider_GetPayment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "pi_test_456")

			resp := map[string]any{
				"id":       "pi_test_456",
				"status":   "succeeded",
				"amount":   int64(29900),
				"currency": "rub",
				"metadata": map[string]string{"user_id": "user-1"},
				"created":  int64(1700000000),
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		details, err := p.GetPayment(context.Background(), "pi_test_456")

		require.NoError(t, err)
		assert.Equal(t, "pi_test_456", details.PaymentID)
		assert.Equal(t, "succeeded", details.Status)
		assert.Equal(t, int64(29900), details.AmountKopeks)
		assert.Equal(t, "rub", details.Currency)
	})

	t.Run("api_error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{"error": "not found"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		_, err := p.GetPayment(context.Background(), "pi_nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stripe api error")
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`not valid json`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		_, err := p.GetPayment(context.Background(), "pi_test")

		assert.Error(t, err)
	})
}

func TestStripeProvider_RefundPayment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "refunds")

			resp := map[string]any{
				"id":     "re_test_123",
				"status": "succeeded",
				"amount": int64(29900),
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		refund, err := p.RefundPayment(context.Background(), "pi_test_123")

		require.NoError(t, err)
		assert.Equal(t, "re_test_123", refund.RefundID)
		assert.Equal(t, "succeeded", refund.Status)
		assert.Equal(t, int64(29900), refund.AmountKopeks)
	})

	t.Run("api_error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"error": "payment already refunded"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		_, err := p.RefundPayment(context.Background(), "pi_test_123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stripe api error")
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`not valid json`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestStripeProvider(server)

		_, err := p.RefundPayment(context.Background(), "pi_test_123")

		assert.Error(t, err)
	})
}

func TestStripeProvider_VerifyWebhookSignature(t *testing.T) {
	t.Parallel()

	webhookSecret := "test-webhook-" + "secret"
	p := &stripeProvider{webhookSecret: webhookSecret}

	t.Run("valid_signature", func(t *testing.T) {
		t.Parallel()

		// Arrange — вычисляем правильную подпись
		payload := []byte(`{"type":"payment_intent.succeeded"}`)
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(payload)
		validSig := "v1=" + hex.EncodeToString(mac.Sum(nil))

		// Act
		err := p.VerifyWebhookSignature(context.Background(), payload, validSig)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("invalid_signature", func(t *testing.T) {
		t.Parallel()

		payload := []byte(`{"type":"payment_intent.succeeded"}`)

		err := p.VerifyWebhookSignature(context.Background(), payload, "v1=invalid_signature")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook signature")
	})
}

func TestStripeProvider_ParseWebhookEvent(t *testing.T) {
	t.Parallel()

	p := &stripeProvider{}

	t.Run("payment_intent_succeeded", func(t *testing.T) {
		t.Parallel()

		payload := []byte(`{
			"type": "payment_intent.succeeded",
			"data": {
				"object": {
					"id": "pi_test_123",
					"status": "succeeded",
					"metadata": {"user_id": "user-1"}
				}
			}
		}`)

		event, err := p.ParseWebhookEvent(context.Background(), payload)

		require.NoError(t, err)
		assert.Equal(t, "payment_intent.succeeded", event.EventType)
		assert.Equal(t, "pi_test_123", event.PaymentID)
		assert.Equal(t, "succeeded", event.Status)
		assert.Equal(t, "user-1", event.Metadata["user_id"])
	})

	t.Run("invalid_json", func(t *testing.T) {
		t.Parallel()

		_, err := p.ParseWebhookEvent(context.Background(), []byte(`not valid json`))

		assert.Error(t, err)
	})
}

func TestNewStripeProvider(t *testing.T) {
	t.Parallel()

	p := NewStripeProvider("api-key", "webhook-secret")

	assert.NotNil(t, p)

	// Проверяем, что это stripeProvider
	sp, ok := p.(*stripeProvider)
	require.True(t, ok)
	assert.Equal(t, "api-key", sp.apiKey)
	assert.Equal(t, "webhook-secret", sp.webhookSecret)
	assert.Equal(t, "https://api.stripe.com", sp.baseURL)
	assert.NotNil(t, sp.client)
}

func TestStripeProvider_constructSignature(t *testing.T) {
	t.Parallel()

	webhookSecret := "my-secret"
	p := &stripeProvider{webhookSecret: webhookSecret}
	payload := []byte("test payload")

	sig := p.constructSignature(payload)

	// Верифицируем, что подпись начинается с "v1="
	assert.Contains(t, sig, "v1=")

	// Верифицируем детерминированность
	sig2 := p.constructSignature(payload)
	assert.Equal(t, sig, sig2)
}

func TestStripeProvider_CreatePayment_NetworkError(t *testing.T) {
	t.Parallel()

	// Arrange — закрытый сервер вызовет ошибку соединения
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // закрываем сразу

	p := newTestStripeProvider(server)

	req := &CreatePaymentRequest{
		AmountKopeks: 29900,
		Currency:     "RUB",
	}

	// Act
	_, err := p.CreatePayment(context.Background(), req)

	// Assert
	assert.Error(t, err)
}

func TestStripeProvider_GetPayment_NetworkError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	p := newTestStripeProvider(server)

	_, err := p.GetPayment(context.Background(), "pi_test")

	assert.Error(t, err)
}

func TestStripeProvider_RefundPayment_NetworkError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	p := newTestStripeProvider(server)

	_, err := p.RefundPayment(context.Background(), "pi_test")

	assert.Error(t, err)
}
