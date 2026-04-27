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

// newTestYookassaProvider создаёт провайдер Yookassa с тестовым HTTP-сервером.
func newTestYookassaProvider(server *httptest.Server) *yookassaProvider {
	p := NewYookassaProvider("shop-id", "secret-key", "webhook-secret").(*yookassaProvider)
	p.baseURL = server.URL
	p.client = server.Client()
	return p
}

func TestYookassaProvider_CreatePayment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "payments")

			resp := map[string]any{
				"id":     "yp_test_123",
				"status": "pending",
				"confirmation": map[string]any{
					"type":             "redirect",
					"confirmation_url": "https://yookassa.ru/pay/123",
				},
				"amount": map[string]any{
					"value":    "299.00",
					"currency": "RUB",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		req := &CreatePaymentRequest{
			AmountKopeks: 29900,
			Currency:     "RUB",
			Description:  "Подписка Test",
			Metadata:     map[string]string{"user_id": "user-1"},
			ReturnURL:    "https://example.com/return",
		}

		// Act
		resp, err := p.CreatePayment(context.Background(), req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "yp_test_123", resp.PaymentID)
		assert.Equal(t, "https://yookassa.ru/pay/123", resp.CheckoutURL)
		assert.Equal(t, "pending", resp.Status)
		assert.Equal(t, int64(29900), resp.AmountKopeks)
	})

	t.Run("api_error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, err := w.Write([]byte(`{"type":"error","code":"invalid_request"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		req := &CreatePaymentRequest{
			AmountKopeks: 29900,
			Currency:     "RUB",
		}

		_, err := p.CreatePayment(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "yookassa api error")
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`not valid json`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		req := &CreatePaymentRequest{
			AmountKopeks: 29900,
			Currency:     "RUB",
		}

		_, err := p.CreatePayment(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestYookassaProvider_GetPayment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "yp_test_456")

			resp := map[string]any{
				"id":     "yp_test_456",
				"status": "succeeded",
				"amount": map[string]any{
					"value":    "299.00",
					"currency": "RUB",
				},
				"metadata":   map[string]string{"user_id": "user-1"},
				"created_at": "2026-01-01T00:00:00Z",
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		details, err := p.GetPayment(context.Background(), "yp_test_456")

		require.NoError(t, err)
		assert.Equal(t, "yp_test_456", details.PaymentID)
		assert.Equal(t, "succeeded", details.Status)
		assert.Equal(t, int64(29900), details.AmountKopeks)
		assert.Equal(t, "RUB", details.Currency)
		assert.Equal(t, "user-1", details.Metadata["user_id"])
	})

	t.Run("api_error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{"type":"error","code":"not_found"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		_, err := p.GetPayment(context.Background(), "yp_nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "yookassa api error")
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`not valid json`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		_, err := p.GetPayment(context.Background(), "yp_test")

		assert.Error(t, err)
	})
}

func TestYookassaProvider_RefundPayment(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "refunds")

			resp := map[string]any{
				"id":     "re_test_123",
				"status": "succeeded",
				"amount": map[string]any{
					"value":    "100.00",
					"currency": "RUB",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		refund, err := p.RefundPayment(context.Background(), "yp_test_123")

		require.NoError(t, err)
		assert.Equal(t, "re_test_123", refund.RefundID)
		assert.Equal(t, "succeeded", refund.Status)
	})

	t.Run("api_error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"type":"error","code":"invalid_request"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		_, err := p.RefundPayment(context.Background(), "yp_test_123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "yookassa api error")
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`not valid json`))
			require.NoError(t, err)
		}))
		defer server.Close()

		p := newTestYookassaProvider(server)

		_, err := p.RefundPayment(context.Background(), "yp_test_123")

		assert.Error(t, err)
	})
}

func TestYookassaProvider_VerifyWebhookSignature(t *testing.T) {
	t.Parallel()

	webhookSecret := "test-webhook-" + "secret"
	p := &yookassaProvider{webhookSecret: webhookSecret}

	t.Run("valid_signature", func(t *testing.T) {
		t.Parallel()

		// Arrange — вычисляем правильную подпись
		payload := []byte(`{"event":"payment.succeeded"}`)
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(payload)
		validSig := hex.EncodeToString(mac.Sum(nil))

		// Act
		err := p.VerifyWebhookSignature(context.Background(), payload, validSig)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("invalid_signature", func(t *testing.T) {
		t.Parallel()

		payload := []byte(`{"event":"payment.succeeded"}`)

		err := p.VerifyWebhookSignature(context.Background(), payload, "invalid_signature")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook signature")
	})
}

func TestYookassaProvider_ParseWebhookEvent(t *testing.T) {
	t.Parallel()

	p := &yookassaProvider{}

	t.Run("payment_succeeded", func(t *testing.T) {
		t.Parallel()

		payload := []byte(`{
			"event": "payment.succeeded",
			"object": {
				"id": "yp_test_123",
				"status": "succeeded",
				"metadata": {"user_id": "user-1"}
			}
		}`)

		event, err := p.ParseWebhookEvent(context.Background(), payload)

		require.NoError(t, err)
		assert.Equal(t, "payment.succeeded", event.EventType)
		assert.Equal(t, "yp_test_123", event.PaymentID)
		assert.Equal(t, "succeeded", event.Status)
		assert.Equal(t, "user-1", event.Metadata["user_id"])
	})

	t.Run("invalid_json", func(t *testing.T) {
		t.Parallel()

		_, err := p.ParseWebhookEvent(context.Background(), []byte(`not valid json`))

		assert.Error(t, err)
	})
}

func TestNewYookassaProvider(t *testing.T) {
	t.Parallel()

	p := NewYookassaProvider("shop-id", "secret-key", "webhook-secret")

	assert.NotNil(t, p)

	yp, ok := p.(*yookassaProvider)
	require.True(t, ok)
	assert.Equal(t, "shop-id", yp.shopID)
	assert.Equal(t, "secret-key", yp.secretKey)
	assert.Equal(t, "webhook-secret", yp.webhookSecret)
	assert.Equal(t, "https://api.yookassa.ru/v3", yp.baseURL)
	assert.NotNil(t, yp.client)
}

func TestParseKopeks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{"integer rubles", "299", 29900},
		{"fractional rubles", "299.00", 29900},
		{"kopeks", "0.50", 50},
		{"zero", "0.00", 0},
		{"large amount", "10000.00", 1000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseKopeks(tc.value)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestYookassaProvider_CreatePayment_NetworkError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	p := newTestYookassaProvider(server)

	req := &CreatePaymentRequest{
		AmountKopeks: 29900,
		Currency:     "RUB",
		ReturnURL:    "https://example.com/return",
	}

	_, err := p.CreatePayment(context.Background(), req)

	assert.Error(t, err)
}

func TestYookassaProvider_GetPayment_NetworkError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	p := newTestYookassaProvider(server)

	_, err := p.GetPayment(context.Background(), "yp_test")

	assert.Error(t, err)
}

func TestYookassaProvider_RefundPayment_NetworkError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	p := newTestYookassaProvider(server)

	_, err := p.RefundPayment(context.Background(), "yp_test")

	assert.Error(t, err)
}
