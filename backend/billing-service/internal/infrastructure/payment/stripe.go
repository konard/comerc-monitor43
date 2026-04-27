package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// NewStripeProvider создаёт новый провайдер Stripe.
func NewStripeProvider(apiKey, webhookSecret string) PaymentProvider {
	return &stripeProvider{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		baseURL:       "https://api.stripe.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type stripeProvider struct {
	apiKey        string
	webhookSecret string
	baseURL       string
	client        *http.Client
}

// CreatePayment создаёт платёж в Stripe (Payment Intent).
func (s *stripeProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error) {
	// Формируем запрос к Stripe Payment Intents API
	stripeReq := map[string]any{
		"amount":   req.AmountKopeks,
		"currency": "rub",
		"metadata": req.Metadata,
	}

	body, err := json.Marshal(stripeReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/payment_intents", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("stripe api error: %s", string(respBody))
	}

	var stripeResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount int64  `json:"amount"`
	}

	if err := json.Unmarshal(respBody, &stripeResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &PaymentResponse{
		PaymentID:    stripeResp.ID,
		CheckoutURL:  req.ReturnURL, // Stripe использует client-side подтверждение
		Status:       stripeResp.Status,
		AmountKopeks: stripeResp.Amount,
		Currency:     req.Currency,
	}, nil
}

// GetPayment получает информацию о платеже.
func (s *stripeProvider) GetPayment(ctx context.Context, paymentID string) (*PaymentDetails, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/v1/payment_intents/"+paymentID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("stripe api error: %s", string(respBody))
	}

	var stripeResp struct {
		ID       string            `json:"id"`
		Status   string            `json:"status"`
		Amount   int64             `json:"amount"`
		Currency string            `json:"currency"`
		Metadata map[string]string `json:"metadata"`
		Created  int64             `json:"created"`
	}

	if err := json.Unmarshal(respBody, &stripeResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &PaymentDetails{
		PaymentID:    stripeResp.ID,
		Status:       stripeResp.Status,
		AmountKopeks: stripeResp.Amount,
		Currency:     stripeResp.Currency,
		Metadata:     stripeResp.Metadata,
		CreatedAt:    time.Unix(stripeResp.Created, 0).Format(time.RFC3339),
	}, nil
}

// RefundPayment выполняет возврат платежа.
func (s *stripeProvider) RefundPayment(ctx context.Context, paymentID string) (*RefundResponse, error) {
	refundReq := map[string]any{
		"payment_intent": paymentID,
	}

	body, err := json.Marshal(refundReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/refunds", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("stripe api error: %s", string(respBody))
	}

	var refundResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount int64  `json:"amount"`
	}

	if err := json.Unmarshal(respBody, &refundResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &RefundResponse{
		RefundID:     refundResp.ID,
		Status:       refundResp.Status,
		AmountKopeks: refundResp.Amount,
	}, nil
}

// VerifyWebhookSignature верифицирует подпись вебхука Stripe.
func (s *stripeProvider) VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error {
	// Stripe использует signature header в формате "t=<timestamp>,v1=<signature>"
	expectedPayload := s.constructSignature(payload)
	if !hmac.Equal([]byte(signature), []byte(expectedPayload)) {
		return errors.New("invalid webhook signature")
	}

	return nil
}

// ParseWebhookEvent парсит событие из вебхука Stripe.
func (s *stripeProvider) ParseWebhookEvent(ctx context.Context, payload []byte) (*WebhookEvent, error) {
	var event struct {
		Type string `json:"type"` // payment_intent.succeeded, etc.
		Data struct {
			Object struct {
				ID       string            `json:"id"`
				Status   string            `json:"status"`
				Metadata map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal webhook event")
	}

	return &WebhookEvent{
		EventType: event.Type,
		PaymentID: event.Data.Object.ID,
		Status:    event.Data.Object.Status,
		Metadata:  event.Data.Object.Metadata,
	}, nil
}

// constructSignature конструирует сигнатуру для Stripe.
func (s *stripeProvider) constructSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}
