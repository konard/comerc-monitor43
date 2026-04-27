package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	alertproto "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	domain "github.com/raul/monitor/backend/alert-service/internal/model"
)

// Моки возвращающие ошибки для проверки error-branch'ей обработчиков.

type mockAlertServiceErrors struct {
	acknowledgeErr error
	getAlertErr    error
	updateAlertErr error
}

func (m *mockAlertServiceErrors) CreateAlert(_ context.Context, alert *domain.Alert) (*domain.Alert, error) {
	return alert, nil
}
func (m *mockAlertServiceErrors) GetAlert(_ context.Context, alertID uuid.UUID) (*domain.Alert, error) {
	if m.getAlertErr != nil {
		return nil, m.getAlertErr
	}
	return &domain.Alert{ID: alertID}, nil
}
func (m *mockAlertServiceErrors) DeleteAlert(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockAlertServiceErrors) DisableAlert(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockAlertServiceErrors) EnableAlert(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockAlertServiceErrors) ListAlerts(_ context.Context, _ uuid.UUID, _ domain.AlertFilter) ([]*domain.Alert, int, error) {
	return nil, 0, nil
}
func (m *mockAlertServiceErrors) UpdateAlert(_ context.Context, alert *domain.Alert) (*domain.Alert, error) {
	if m.updateAlertErr != nil {
		return nil, m.updateAlertErr
	}
	return alert, nil
}
func (m *mockAlertServiceErrors) AcknowledgeAlert(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return m.acknowledgeErr
}

type mockRuleServiceErrors struct {
	deleteErr error
}

func (m *mockRuleServiceErrors) GetRuleByMonitorID(_ context.Context, userID, monitorID uuid.UUID) (*domain.AlertRule, error) {
	return &domain.AlertRule{ID: uuid.New(), UserID: userID, MonitorID: monitorID}, nil
}
func (m *mockRuleServiceErrors) GetRuleByID(_ context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	return &domain.AlertRule{ID: id}, nil
}
func (m *mockRuleServiceErrors) CreateRule(_ context.Context, rule *domain.AlertRule) (*domain.AlertRule, error) {
	return rule, nil
}
func (m *mockRuleServiceErrors) DeleteRule(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return m.deleteErr
}

// --- DeleteAlertRule tests ---

func TestAlertServiceServer_DeleteAlertRule_Success(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	monitorID := uuid.New()

	_, err := server.DeleteAlertRule(ctx, &alertproto.DeleteAlertRuleRequest{MonitorId: monitorID.String()})
	require.NoError(t, err)
}

func TestAlertServiceServer_DeleteAlertRule_EmptyMonitorID(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.DeleteAlertRule(ctx, &alertproto.DeleteAlertRuleRequest{MonitorId: ""})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_DeleteAlertRule_InvalidMonitorID(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.DeleteAlertRule(ctx, &alertproto.DeleteAlertRuleRequest{MonitorId: "not-a-uuid"})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_DeleteAlertRule_Unauthenticated(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.DeleteAlertRule(context.Background(), &alertproto.DeleteAlertRuleRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAlertServiceServer_DeleteAlertRule_NotFound(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleServiceErrors{deleteErr: domain.ErrAlertRuleNotFound}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.DeleteAlertRule(ctx, &alertproto.DeleteAlertRuleRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestAlertServiceServer_DeleteAlertRule_Forbidden(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleServiceErrors{deleteErr: domain.ErrForbidden}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.DeleteAlertRule(ctx, &alertproto.DeleteAlertRuleRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

// --- AcknowledgeAlert error branches ---

func TestAlertServiceServer_AcknowledgeAlert_Forbidden(t *testing.T) {
	t.Parallel()
	svc := &mockAlertServiceErrors{acknowledgeErr: domain.ErrForbidden}
	server := NewAlertServiceServer(svc, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.AcknowledgeAlert(ctx, &alertproto.AcknowledgeAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestAlertServiceServer_AcknowledgeAlert_NotFound(t *testing.T) {
	t.Parallel()
	svc := &mockAlertServiceErrors{acknowledgeErr: domain.ErrAlertNotFound}
	server := NewAlertServiceServer(svc, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.AcknowledgeAlert(ctx, &alertproto.AcknowledgeAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestAlertServiceServer_AcknowledgeAlert_NotAcknowledgeable(t *testing.T) {
	t.Parallel()
	svc := &mockAlertServiceErrors{acknowledgeErr: domain.ErrAlertNotAcknowledgeable}
	server := NewAlertServiceServer(svc, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.AcknowledgeAlert(ctx, &alertproto.AcknowledgeAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestAlertServiceServer_AcknowledgeAlert_AlreadyAcknowledged(t *testing.T) {
	t.Parallel()
	svc := &mockAlertServiceErrors{acknowledgeErr: domain.ErrAlertAlreadyAcknowledged}
	server := NewAlertServiceServer(svc, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.AcknowledgeAlert(ctx, &alertproto.AcknowledgeAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

// --- UpdateAlert type-specific field branches ---

func TestAlertServiceServer_UpdateAlert_TypeSpecificFields(t *testing.T) {
	t.Parallel()
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.Background()
	alertID := uuid.New()

	tests := []struct {
		name string
		req  *alertproto.UpdateAlertRequest
	}{
		{
			name: "expected_status_code",
			req: &alertproto.UpdateAlertRequest{
				Id: alertID.String(),
				TypeSpecificFields: &alertproto.UpdateAlertRequest_ExpectedStatusCode{
					ExpectedStatusCode: "200",
				},
			},
		},
		{
			name: "max_response_time_ms",
			req: &alertproto.UpdateAlertRequest{
				Id: alertID.String(),
				TypeSpecificFields: &alertproto.UpdateAlertRequest_MaxResponseTimeMs{
					MaxResponseTimeMs: 5000,
				},
			},
		},
		{
			name: "body_pattern",
			req: &alertproto.UpdateAlertRequest{
				Id: alertID.String(),
				TypeSpecificFields: &alertproto.UpdateAlertRequest_BodyPattern{
					BodyPattern: "error",
				},
			},
		},
		{
			name: "days_before_expiry",
			req: &alertproto.UpdateAlertRequest{
				Id: alertID.String(),
				TypeSpecificFields: &alertproto.UpdateAlertRequest_DaysBeforeExpiry{
					DaysBeforeExpiry: 7,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.UpdateAlert(ctx, tt.req)
			require.NoError(t, err)
		})
	}
}

func TestAlertServiceServer_UpdateAlert_GetAlertNotFound(t *testing.T) {
	t.Parallel()
	svc := &mockAlertServiceErrors{getAlertErr: domain.ErrAlertNotFound}
	server := NewAlertServiceServer(svc, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.UpdateAlert(context.Background(), &alertproto.UpdateAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestAlertServiceServer_UpdateAlert_GetAlertInternalError(t *testing.T) {
	t.Parallel()
	svc := &mockAlertServiceErrors{getAlertErr: assert.AnError}
	server := NewAlertServiceServer(svc, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.UpdateAlert(context.Background(), &alertproto.UpdateAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- DomainAlertRuleToProto nil branch ---

func TestDomainAlertRuleToProto_Nil(t *testing.T) {
	t.Parallel()
	result, err := DomainAlertRuleToProto(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}
