package dto

// CreateSubscriptionRequest запрос на создание подписки.
type CreateSubscriptionRequest struct {
	UserID string
	PlanID string
	// Duration в днях
	Duration int32
}

// SubscriptionResponse ответ с информацией о подписке.
type SubscriptionResponse struct {
	ID                string
	UserID            string
	PlanID            string
	Status            string
	StartedAt         string
	ExpiresAt         string
	CanceledAt        *string
	AutoRenew         bool
	ProviderPaymentID *string
	CreatedAt         string
}

// CancelSubscriptionRequest запрос на отмену подписки.
type CancelSubscriptionRequest struct {
	UserID string
	Reason string
}

// SubscriptionHistoryResponse ответ с историей подписок.
type SubscriptionHistoryResponse struct {
	Subscriptions []*SubscriptionResponse
	Total         int32
	Page          int32
	PageSize      int32
}
