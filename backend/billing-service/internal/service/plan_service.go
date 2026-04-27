package service

import (
	"context"

	"github.com/pkg/errors"

	"github.com/raul/monitor/backend/billing-service/internal/model"
	"github.com/raul/monitor/backend/billing-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/billing-service/internal/service/dto"
	billingerrors "github.com/raul/monitor/backend/billing-service/pkg/errors"
)

// NewPlanService создаёт новый PlanService.
func NewPlanService(planRepo interfaces.PlanRepository) PlanService {
	return &planService{
		planRepo: planRepo,
	}
}

type planService struct {
	planRepo interfaces.PlanRepository
}

// GetAllPlans возвращает все тарифные планы.
func (s *planService) GetAllPlans(ctx context.Context) (*dto.PlansResponse, error) {
	plans, err := s.planRepo.GetAll(ctx, true) // только активные
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plans")
	}

	response := &dto.PlansResponse{
		Plans: make([]*dto.PlanResponse, 0, len(plans)),
	}

	for _, plan := range plans {
		response.Plans = append(response.Plans, s.planToDTO(plan))
	}

	return response, nil
}

// GetPlan возвращает план по ID.
func (s *planService) GetPlan(ctx context.Context, planID string) (*dto.PlanResponse, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, billingerrors.NotFound("plan", planID)
	}

	return s.planToDTO(plan), nil
}

// ValidatePlanLimits проверяет лимиты плана.
func (s *planService) ValidatePlanLimits(ctx context.Context, planID string, monitorsCount, checkInterval int) error {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return billingerrors.NotFound("plan", planID)
	}

	if !plan.IsValidForMonitors(monitorsCount) {
		return billingerrors.InvalidArgument("monitors_count", "exceeds plan limit")
	}

	if !plan.IsValidForCheckInterval(checkInterval) {
		return billingerrors.InvalidArgument("check_interval", "less than minimum allowed by plan")
	}

	return nil
}

// planToDTO конвертирует модель в DTO.
func (s *planService) planToDTO(plan *model.Plan) *dto.PlanResponse {
	features := make([]*dto.FeatureResponse, 0, len(plan.Features))
	for _, f := range plan.Features {
		features = append(features, &dto.FeatureResponse{
			ID:          f.ID,
			Name:        f.Name,
			Description: f.Description,
		})
	}

	return &dto.PlanResponse{
		ID:                      plan.ID,
		Name:                    plan.Name,
		Description:             plan.Description,
		PriceKopeks:             plan.PriceKopeks,
		BillingPeriodDays:       plan.BillingPeriodDays,
		MaxMonitors:             plan.MaxMonitors,
		MinCheckIntervalSeconds: plan.MinCheckIntervalSeconds,
		MaxAlertsPerDay:         plan.MaxAlertsPerDay,
		Features:                features,
		IsActive:                plan.IsActive,
	}
}
