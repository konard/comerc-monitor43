package dto

// PlanResponse ответ с информацией о тарифном плане.
type PlanResponse struct {
	ID                      string
	Name                    string
	Description             string
	PriceKopeks             int64
	BillingPeriodDays       int32
	MaxMonitors             int32
	MinCheckIntervalSeconds int32
	MaxAlertsPerDay         int32
	Features                []*FeatureResponse
	IsActive                bool
}

// FeatureResponse ответ с информацией о функции плана.
type FeatureResponse struct {
	ID          string
	Name        string
	Description string
}

// PlansResponse ответ со списком тарифных планов.
type PlansResponse struct {
	Plans []*PlanResponse
}
