package payment

import (
	"context"
)

// PaymentProvider определяет интерфейс платежного провайдера.
type PaymentProvider interface {
	// CreatePayment создаёт платёж и возвращает URL для редиректа и ID платежа.
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error)

	// GetPayment получает информацию о платеже.
	GetPayment(ctx context.Context, paymentID string) (*PaymentDetails, error)

	// RefundPayment выполняет возврат платежа.
	RefundPayment(ctx context.Context, paymentID string) (*RefundResponse, error)

	// VerifyWebhookSignature верифицирует подпись вебхука.
	VerifyWebhookSignature(ctx context.Context, payload []byte, signature string) error

	// ParseWebhookEvent парсит событие из вебхука.
	ParseWebhookEvent(ctx context.Context, payload []byte) (*WebhookEvent, error)
}

// CreatePaymentRequest запрос на создание платежа.
type CreatePaymentRequest struct {
	AmountKopeks int64
	Currency     string
	Description  string
	Metadata     map[string]string
	ReturnURL    string
}

// PaymentResponse ответ на создание платежа.
type PaymentResponse struct {
	PaymentID    string
	CheckoutURL  string
	Status       string
	AmountKopeks int64
	Currency     string
}

// PaymentDetails детали платежа.
type PaymentDetails struct {
	PaymentID    string
	Status       string
	AmountKopeks int64
	Currency     string
	Metadata     map[string]string
	CreatedAt    string
}

// RefundResponse ответ на возврат платежа.
type RefundResponse struct {
	RefundID     string
	Status       string
	AmountKopeks int64
}

// WebhookEvent событие из вебхука.
type WebhookEvent struct {
	EventType string // payment.succeeded, payment.canceled, etc.
	PaymentID string
	Status    string
	Metadata  map[string]string
}
