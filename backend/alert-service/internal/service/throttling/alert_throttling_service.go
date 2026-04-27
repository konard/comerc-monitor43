package throttling

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/config"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

type ThrottleAction string

const (
	ThrottleActionDeliver  ThrottleAction = "deliver"
	ThrottleActionQueue    ThrottleAction = "queue"
	ThrottleActionSuppress ThrottleAction = "suppress"
	ThrottleActionStorm    ThrottleAction = "storm_group"
)

type ThrottleResult struct {
	Action     ThrottleAction
	Reason     string
	DeliverAt  *time.Time
	StormGroup bool
}

type AlertThrottlingService struct {
	config       config.AlertThrottlingConfig
	alertRepo    repository.AlertRepository
	deliveryRepo repository.DeliveryAttemptRepository
	tracer       trace.Tracer
	metrics      *apptelemetry.Metrics
}

func NewAlertThrottlingService(
	cfg config.AlertThrottlingConfig,
	alertRepo repository.AlertRepository,
	deliveryRepo repository.DeliveryAttemptRepository,
	tracer trace.Tracer,
	metrics *apptelemetry.Metrics,
) *AlertThrottlingService {
	return &AlertThrottlingService{
		config:       cfg,
		alertRepo:    alertRepo,
		deliveryRepo: deliveryRepo,
		tracer:       tracer,
		metrics:      metrics,
	}
}

func (s *AlertThrottlingService) CheckThrottle(
	ctx context.Context,
	userID, monitorID uuid.UUID,
	status string,
) (*ThrottleResult, error) {
	ctx, span := s.tracer.Start(ctx, "AlertThrottlingService.CheckThrottle")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("monitor_id", monitorID.String()),
		attribute.String("status", status),
	)

	result, err := s.checkStorm(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check alert storm: %v", err)
	}
	if result != nil {
		return result, nil
	}

	result, err = s.checkRateLimit(ctx, monitorID, status)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check rate limit: %v", err)
	}
	if result != nil {
		return result, nil
	}

	result, err = s.checkCooldown(ctx, monitorID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check cooldown: %v", err)
	}
	if result != nil {
		return result, nil
	}

	span.AddEvent("throttle_pass")
	return &ThrottleResult{Action: ThrottleActionDeliver}, nil
}

func (s *AlertThrottlingService) checkStorm(ctx context.Context) (*ThrottleResult, error) {
	stormWindow := time.Now().Add(-s.config.StormWindow)
	count, err := s.alertRepo.CountUniqueMonitorsWithAlertsSince(ctx, stormWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to count unique monitors with alerts: %v", err)
	}

	if count >= s.config.StormThreshold {
		duration := time.Now().Add(s.config.StormDuration)
		return &ThrottleResult{
			Action:     ThrottleActionStorm,
			Reason:     "alert storm detected",
			DeliverAt:  &duration,
			StormGroup: true,
		}, nil
	}

	return nil, nil
}

func (s *AlertThrottlingService) checkRateLimit(
	ctx context.Context,
	monitorID uuid.UUID,
	status string,
) (*ThrottleResult, error) {
	lastDelivery, err := s.deliveryRepo.GetLastDeliveryTimeForMonitorAndStatus(
		ctx, monitorID.String(), status,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get last delivery time: %v", err)
	}

	if lastDelivery != nil && time.Since(*lastDelivery) < s.config.RateLimitPeriod {
		deliverAt := lastDelivery.Add(s.config.RateLimitPeriod)
		return &ThrottleResult{
			Action:    ThrottleActionQueue,
			Reason:    "rate limit: too frequent for same monitor+status",
			DeliverAt: &deliverAt,
		}, nil
	}

	return nil, nil
}

func (s *AlertThrottlingService) checkCooldown(
	ctx context.Context,
	monitorID uuid.UUID,
) (*ThrottleResult, error) {
	lastAlert, err := s.alertRepo.GetLastAlertTimeAnyStatus(ctx, monitorID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get last alert time: %v", err)
	}

	if lastAlert != nil && time.Since(lastAlert.CreatedAt) < s.config.CooldownPeriod {
		return &ThrottleResult{
			Action: ThrottleActionSuppress,
			Reason: "cooldown: too soon after last alert for this monitor",
		}, nil
	}

	return nil, nil
}
