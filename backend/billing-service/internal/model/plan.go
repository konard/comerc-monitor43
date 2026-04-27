package model

import (
	"time"
)

// Plan представляет тарифный план.
type Plan struct {
	ID                      string
	Name                    string
	Description             string
	PriceKopeks             int64
	BillingPeriodDays       int32
	MaxMonitors             int32
	MinCheckIntervalSeconds int32
	MaxAlertsPerDay         int32
	Features                []Feature
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Feature представляет функцию тарифного плана.
type Feature struct {
	ID          string
	Name        string
	Description string
}

// Plan tier ID.
const (
	PlanTierFree         = "TIER_FREE"
	PlanTierStarter      = "TIER_STARTER"
	PlanTierProfessional = "TIER_PROFESSIONAL"
	PlanTierBusiness     = "TIER_BUSINESS"
)

// IsFree проверяет, является ли план бесплатным.
func (p *Plan) IsFree() bool {
	return p.ID == PlanTierFree || p.PriceKopeks == 0
}

// CalculateExpiryDate рассчитывает дату окончания подписки.
func (p *Plan) CalculateExpiryDate() time.Time {
	return time.Now().AddDate(0, 0, int(p.BillingPeriodDays))
}

// IsValidForMonitors проверяет, допустимо ли количество мониторов для плана.
func (p *Plan) IsValidForMonitors(count int) bool {
	if p.MaxMonitors < 0 {
		return true // -1 означает безлимит
	}
	return count <= int(p.MaxMonitors)
}

// IsValidForCheckInterval проверяет, допустим ли интервал проверки для плана.
func (p *Plan) IsValidForCheckInterval(seconds int) bool {
	return seconds >= int(p.MinCheckIntervalSeconds)
}

// IsValidForAlerts проверяет, допустимо ли количество алертов для плана.
func (p *Plan) IsValidForAlerts(count int) bool {
	if p.MaxAlertsPerDay < 0 {
		return true // -1 означает безлимит
	}
	return count <= int(p.MaxAlertsPerDay)
}

// GetDefaultPlan возвращает дефолтный Free план.
func GetDefaultPlan() *Plan {
	return &Plan{
		ID:                      PlanTierFree,
		Name:                    "Бесплатный",
		Description:             "Бесплатный тариф для начинающих",
		PriceKopeks:             0,
		BillingPeriodDays:       30,
		MaxMonitors:             5,
		MinCheckIntervalSeconds: 300,
		MaxAlertsPerDay:         10,
		Features: []Feature{
			{ID: "basic_monitoring", Name: "Базовый мониторинг", Description: "Базовый HTTP-мониторинг"},
			{ID: "email_alerts", Name: "Email-уведомления", Description: "Уведомления по электронной почте"},
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ValidationError представляет ошибку валидации модели.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
