package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// NewYookassaProvider создаёт новый провайдер Yookassa.
func NewYookassaProvider(shopID, secretKey, webhookSecret string) PaymentProvider {
	return NewYookassaProviderWithBaseURL(shopID, secretKey, webhookSecret, "https://api.yookassa.ru/v3")
}

// NewYookassaProviderWithBaseURL создаёт новый провайдер Yookassa с заданным API URL.
func NewYookassaProviderWithBaseURL(shopID, secretKey, webhookSecret, baseURL string) PaymentProvider {
	if baseURL == "" {
		baseURL = "https://api.yookassa.ru/v3"
	}

	return &yookassaProvider{
		shopID:        shopID,
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		baseURL:       strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type yookassaProvider struct {
	shopID        string
	secretKey     string
	webhookSecret string
	baseURL       string
	client        *http.Client
}

// CreatePayment создаёт платёж в Yookassa.
func (y *yookassaProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error) {
	// Формируем запрос к Yookassa API
	yooReq := map[string]any{
		"amount": map[string]any{
			"value":    fmt.Sprintf("%.2f", float64(req.AmountKopeks)/100),
			"currency": req.Currency,
		},
		"confirmation": map[string]any{
			"type":       "redirect",
			"return_url": req.ReturnURL,
		},
		"capture":     true,
		"description": req.Description,
		"metadata":    req.Metadata,
	}

	body, err := json.Marshal(yooReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", y.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.SetBasicAuth(y.shopID, y.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", fmt.Sprintf("%d", time.Now().UnixNano()))

	resp, err := y.client.Do(httpReq)
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
		return nil, errors.Errorf("yookassa api error: %s", string(respBody))
	}

	var yooResp struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		Confirmation struct {
			Type       string `json:"type"`
			ConfirmURL string `json:"confirmation_url"`
		} `json:"confirmation"`
		Amount struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}

	if err := json.Unmarshal(respBody, &yooResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &PaymentResponse{
		PaymentID:    yooResp.ID,
		CheckoutURL:  yooResp.Confirmation.ConfirmURL,
		Status:       yooResp.Status,
		AmountKopeks: req.AmountKopeks,
		Currency:     req.Currency,
	}, nil
}

// GetPayment получает информацию о платеже.
func (y *yookassaProvider) GetPayment(ctx context.Context, paymentID string) (*PaymentDetails, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", y.baseURL+"/payments/"+paymentID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.SetBasicAuth(y.shopID, y.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := y.client.Do(httpReq)
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
		return nil, errors.Errorf("yookassa api error: %s", string(respBody))
	}

	var yooResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`
		Metadata  map[string]string `json:"metadata"`
		CreatedAt string            `json:"created_at"`
	}

	if err := json.Unmarshal(respBody, &yooResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &PaymentDetails{
		PaymentID:    yooResp.ID,
		Status:       yooResp.Status,
		AmountKopeks: parseKopeks(yooResp.Amount.Value),
		Currency:     yooResp.Amount.Currency,
		Metadata:     yooResp.Metadata,
		CreatedAt:    yooResp.CreatedAt,
	}, nil
}

// parseKopeks конвертирует строку с суммой в копейки (1 рубль = 100 копеек).
func parseKopeks(value string) int64 {
	var rubles float64
	if _, err := fmt.Sscanf(value, "%f", &rubles); err != nil {
		return 0
	}
	return int64(rubles * 100)
}

// RefundPayment выполняет возврат платежа.
func (y *yookassaProvider) RefundPayment(ctx context.Context, paymentID string) (*RefundResponse, error) {
	refundReq := map[string]any{
		"payment_id": paymentID,
		"amount": map[string]any{
			"value":    "100.00",
			"currency": "RUB",
		},
	}

	body, err := json.Marshal(refundReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", y.baseURL+"/refunds", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	httpReq.SetBasicAuth(y.shopID, y.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := y.client.Do(httpReq)
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
		return nil, errors.Errorf("yookassa api error: %s", string(respBody))
	}

	var refundResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value string `json:"value"`
		} `json:"amount"`
	}

	if err := json.Unmarshal(respBody, &refundResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &RefundResponse{
		RefundID: refundResp.ID,
		Status:   refundResp.Status,
	}, nil
}

// VerifyWebhookSignature верифицирует подпись вебхука Yookassa.
func (y *yookassaProvider) VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error {
	// Yookassa использует HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(y.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("invalid webhook signature")
	}

	return nil
}

// ParseWebhookEvent парсит событие из вебхука Yookassa.
func (y *yookassaProvider) ParseWebhookEvent(ctx context.Context, payload []byte) (*WebhookEvent, error) {
	var event struct {
		Type   string `json:"event"` // payment.succeeded, payment.canceled, etc.
		ID     string `json:"id"`
		Status string `json:"status"`
		Object struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
		} `json:"object"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal webhook event")
	}

	paymentID := event.Object.ID
	if paymentID == "" {
		paymentID = event.ID
	}
	status := event.Object.Status
	if status == "" {
		status = event.Status
	}

	return &WebhookEvent{
		EventType: event.Type,
		PaymentID: paymentID,
		Status:    status,
		Metadata:  event.Object.Metadata,
	}, nil
}
