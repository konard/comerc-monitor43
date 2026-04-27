package dto

// CreatePaymentRequest запрос на создание платежа.
type CreatePaymentRequest struct {
	UserID         string
	SubscriptionID string
	PlanID         string
	AmountKopeks   int64
	Currency       string
	Provider       string // YOOKASSA, STRIPE
	ReturnURL      string
}

// PaymentResponse ответ с информацией о платеже.
type PaymentResponse struct {
	ID                string
	UserID            string
	SubscriptionID    *string
	Provider          string
	ProviderPaymentID string
	Status            string
	AmountKopeks      int64
	Currency          string
	CreatedAt         string
	UpdatedAt         string
}

// GetPaymentHistoryRequest запрос на получение истории платежей.
type GetPaymentHistoryRequest struct {
	UserID   string
	Page     int32
	PageSize int32
}

// PaymentHistoryResponse ответ с историей платежей.
type PaymentHistoryResponse struct {
	Payments []*PaymentResponse
	Total    int64
	Page     int32
	PageSize int32
}

// WebhookRequest запрос на обработку вебхука.
type WebhookRequest struct {
	Provider  string            // YOOKASSA, STRIPE
	Payload   map[string]string // Raw payload
	Signature string            // HMAC signature
}

// CheckoutResponse ответ на создание checkout сессии.
type CheckoutResponse struct {
	CheckoutURL string
	PaymentID   string
}
