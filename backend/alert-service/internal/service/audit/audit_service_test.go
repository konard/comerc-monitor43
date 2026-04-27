package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

type mockAuditLogRepository struct {
	mock.Mock
}

func (m *mockAuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	args := m.Called(ctx, log)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *mockAuditLogRepository) List(ctx context.Context, resourceType string, resourceID string, limit int) ([]*model.AuditLog, error) {
	args := m.Called(ctx, resourceType, resourceID, limit)
	if args.Get(0) == nil {
		return []*model.AuditLog{}, args.Error(1)
	}
	return args.Get(0).([]*model.AuditLog), args.Error(1)
}

func TestLog_Success(t *testing.T) {
	t.Parallel()

	repo := &mockAuditLogRepository{}
	service := NewAuditService(repo, nil, nil)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.Log(ctx, model.AuditActionAlertTriggered, "alert", "res-123", userID, map[string]any{
		"monitor_id": "mon-456",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestLog_RepositoryError(t *testing.T) {
	t.Parallel()

	repo := &mockAuditLogRepository{}
	service := NewAuditService(repo, nil, nil)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(assert.AnError)

	err := service.Log(ctx, model.AuditActionAlertTriggered, "alert", "res-123", userID, nil)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestLog_WithTracer(t *testing.T) {
	t.Parallel()

	repo := &mockAuditLogRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAuditService(repo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.Log(ctx, model.AuditActionAlertTriggered, "alert", "res-123", userID, nil)

	assert.NoError(t, err)
}

func TestLog_WithTracerError(t *testing.T) {
	t.Parallel()

	repo := &mockAuditLogRepository{}
	tracer := otel.GetTracerProvider().Tracer("test")
	service := NewAuditService(repo, tracer, nil)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(assert.AnError)

	err := service.Log(ctx, model.AuditActionAlertTriggered, "alert", "res-123", userID, nil)

	assert.Error(t, err)
}

func TestLog_WithIP(t *testing.T) {
	t.Parallel()

	repo := &mockAuditLogRepository{}
	service := NewAuditService(repo, nil, nil)

	ctx := ContextWithIP(context.Background(), "192.168.1.1")
	userID := uuid.New()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.Log(ctx, model.AuditActionAlertTriggered, "alert", "res-123", userID, nil)

	assert.NoError(t, err)
}

func TestContextWithIP(t *testing.T) {
	t.Parallel()

	ctx := ContextWithIP(context.Background(), "10.0.0.1")
	ip, ok := ctx.Value(contextKeyIP{}).(string)

	assert.True(t, ok)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestExtractIPFromContext_NoIP(t *testing.T) {
	t.Parallel()

	ip := extractIPFromContext(context.Background())
	assert.Nil(t, ip)
}

func TestExtractIPFromContext_WithIP(t *testing.T) {
	t.Parallel()

	ctx := ContextWithIP(context.Background(), "172.16.0.1")
	ip := extractIPFromContext(ctx)

	assert.NotNil(t, ip)
	assert.Equal(t, "172.16.0.1", *ip)
}
