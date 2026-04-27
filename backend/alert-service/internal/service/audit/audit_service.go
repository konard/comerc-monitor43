package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/alert-service/internal/model"
	"github.com/raul/monitor/backend/alert-service/internal/repository"
	apptelemetry "github.com/raul/monitor/backend/alert-service/pkg/telemetry"
)

type contextKeyIP struct{}

// AuditService управляет записью событий в журнал аудита
type AuditService struct {
	repo    repository.AuditLogRepository
	tracer  trace.Tracer
	metrics *apptelemetry.Metrics
}

// NewAuditService создаёт новый сервис аудита
func NewAuditService(repo repository.AuditLogRepository, tracer trace.Tracer, metrics *apptelemetry.Metrics) *AuditService {
	return &AuditService{
		repo:    repo,
		tracer:  tracer,
		metrics: metrics,
	}
}

// Log записывает событие в журнал аудита
func (s *AuditService) Log(ctx context.Context, action model.AuditAction, resourceType, resourceID string, userID uuid.UUID, fields map[string]any) error {
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, "AuditService.Log")
		defer span.End()

		span.SetAttributes(
			attribute.String("action", string(action)),
			attribute.String("resource_type", resourceType),
			attribute.String("resource_id", resourceID),
		)
	}

	log := &model.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Fields:       fields,
		IPAddress:    extractIPFromContext(ctx),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, log); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return err
	}

	return nil
}

// extractIPFromContext извлекает IP-адрес из контекста запроса
func extractIPFromContext(ctx context.Context) *string {
	ip, ok := ctx.Value(contextKeyIP{}).(string)
	if !ok || ip == "" {
		return nil
	}
	return &ip
}

// ContextWithIP создаёт новый контекст с IP-адресом
func ContextWithIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, contextKeyIP{}, ip)
}
