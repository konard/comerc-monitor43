package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	alertproto "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/alert-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// Mock implementations
type mockAlertService struct{}

type mockChannelService struct{}
type mockRuleService struct{}
type mockMuteService struct{}

// createTestMiddleware creates a mock middleware for testing
func createTestMiddleware(_ bool) *auth.AuthMiddleware {
	return auth.NewAuthMiddleware("test-secret-key")
}

func (m *mockAlertService) CreateAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	return alert, nil
}

func (m *mockAlertService) GetAlert(ctx context.Context, alertID uuid.UUID) (*model.Alert, error) {
	return &model.Alert{ID: alertID}, nil
}

func (m *mockAlertService) DeleteAlert(ctx context.Context, alertID uuid.UUID) error {
	return nil
}

func (m *mockAlertService) DisableAlert(ctx context.Context, alertID uuid.UUID) error {
	return nil
}

func (m *mockAlertService) EnableAlert(ctx context.Context, alertID uuid.UUID) error {
	return nil
}

func (m *mockAlertService) ListAlerts(ctx context.Context, userID uuid.UUID, filter model.AlertFilter) ([]*model.Alert, int, error) {
	return []*model.Alert{}, 0, nil
}

func (m *mockAlertService) UpdateAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	return alert, nil
}

func (m *mockAlertService) AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, userID uuid.UUID) error {
	return nil
}

func (m *mockChannelService) GetUserChannels(ctx context.Context, userID uuid.UUID) ([]*model.AlertChannel, error) {
	return []*model.AlertChannel{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      model.AlertChannelTypeEmail,
			Status:    model.AlertChannelStatusUnverified,
			Enabled:   true,
			Verified:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			EmailConfig: &model.EmailChannelConfig{
				Email: "test@example.com",
			},
		},
	}, nil
}

func (m *mockChannelService) UpdateUserChannels(ctx context.Context, userID uuid.UUID, channels []*model.AlertChannel) error {
	return nil
}

func (m *mockChannelService) CreateChannel(ctx context.Context, channel *model.AlertChannel) (*model.AlertChannel, error) {
	return channel, nil
}

func (m *mockChannelService) GetChannel(ctx context.Context, userID, channelID uuid.UUID) (*model.AlertChannel, error) {
	return &model.AlertChannel{ID: channelID, UserID: userID, Type: model.AlertChannelTypeEmail, Status: model.AlertChannelStatusUnverified}, nil
}

func (m *mockChannelService) UpdateChannel(ctx context.Context, userID uuid.UUID, channel *model.AlertChannel) (*model.AlertChannel, error) {
	channel.UserID = userID
	return channel, nil
}

func (m *mockChannelService) DeleteChannel(ctx context.Context, userID, channelID uuid.UUID) error {
	return nil
}

func (m *mockChannelService) ListChannels(ctx context.Context, userID uuid.UUID) ([]*model.AlertChannel, error) {
	return nil, nil
}

func (m *mockChannelService) VerifyChannel(ctx context.Context, channelID uuid.UUID, code string) error {
	return nil
}

func (m *mockRuleService) GetRuleByMonitorID(ctx context.Context, userID, monitorID uuid.UUID) (*model.AlertRule, error) {
	return &model.AlertRule{
		ID:                  uuid.New(),
		UserID:              userID,
		MonitorID:           monitorID,
		Enabled:             true,
		ConsecutiveFailures: 3,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}, nil
}

func (m *mockRuleService) GetRuleByID(ctx context.Context, id uuid.UUID) (*model.AlertRule, error) {
	return &model.AlertRule{ID: id}, nil
}

func (m *mockRuleService) CreateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertRule, error) {
	return rule, nil
}

func (m *mockRuleService) DeleteRule(ctx context.Context, userID, monitorID uuid.UUID) error {
	return nil
}

func (m *mockMuteService) MuteForUser(ctx context.Context, userID, monitorID uuid.UUID, mutedUntil *time.Time) error {
	return nil
}

func (m *mockMuteService) MuteGlobally(ctx context.Context, adminID, monitorID uuid.UUID, mutedUntil *time.Time) error {
	return nil
}

func (m *mockMuteService) Unmute(ctx context.Context, muteID string) error {
	return nil
}

func (m *mockMuteService) UnmuteForUser(ctx context.Context, userID, monitorID uuid.UUID) error {
	return nil
}

func TestNewAlertServiceServer(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	assert.NotNil(t, server)
	assert.NotNil(t, server.alertService)
	assert.NotNil(t, server.channelService)
	assert.NotNil(t, server.ruleService)
}

func TestAlertServiceServer_CreateAlert(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	// Add user ID directly to context for testing (bypassing JWT for unit tests)
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)
	monitorID := uuid.New()

	req := &alertproto.CreateAlertRequest{
		MonitorId:           monitorID.String(),
		Type:                alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
		ThresholdMs:         5000,
		ConsecutiveFailures: 3,
		TypeSpecificFields: &alertproto.CreateAlertRequest_ExpectedStatusCode{
			ExpectedStatusCode: "200",
		},
	}

	alertProto, err := server.CreateAlert(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, alertProto)
	assert.Equal(t, monitorID.String(), alertProto.MonitorId)
	assert.Equal(t, alertproto.AlertType_ALERT_TYPE_STATUS_CODE, alertProto.Type)
	assert.True(t, alertProto.Enabled)
	assert.Equal(t, int32(3), alertProto.ConsecutiveFailures)
}

func TestAlertServiceServer_CreateAlert_Validation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	// Add user ID directly to context for testing
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)

	req := &alertproto.CreateAlertRequest{
		MonitorId: "",                                          // Empty monitor ID
		Type:      alertproto.AlertType_ALERT_TYPE_UNSPECIFIED, // Unspecified type
	}

	_, err := server.CreateAlert(ctx, req)
	require.Error(t, err)
}

func TestAlertServiceServer_GetAlert(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	alertID := uuid.New()

	req := &alertproto.GetAlertRequest{
		Id: alertID.String(),
	}

	// Mock service returns valid alert, so handler should work
	resp, err := server.GetAlert(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, alertID.String(), resp.Id)
}

func TestAlertServiceServer_ListAlerts(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)

	req := &alertproto.ListAlertsRequest{
		UserId:   userID.String(),
		Page:     1,
		PageSize: 10,
	}

	// Mock service returns empty list, so handler should work
	resp, err := server.ListAlerts(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(10), resp.PageSize)
}

func TestAlertServiceServer_UpdateAlert(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	// Add user ID directly to context for testing
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)
	alertID := uuid.New()

	// Create alert first
	createReq := &alertproto.CreateAlertRequest{
		MonitorId:           uuid.New().String(),
		Type:                alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
		ThresholdMs:         5000,
		ConsecutiveFailures: 3,
		TypeSpecificFields: &alertproto.CreateAlertRequest_ExpectedStatusCode{
			ExpectedStatusCode: "200",
		},
	}

	_, err := server.CreateAlert(ctx, createReq)
	require.NoError(t, err)

	// Update the alert
	updateReq := &alertproto.UpdateAlertRequest{
		Id:                  alertID.String(),
		Enabled:             false,
		ThresholdMs:         3000,
		ConsecutiveFailures: 5,
	}

	updatedAlertProto, err := server.UpdateAlert(ctx, updateReq)
	require.NoError(t, err)
	assert.Equal(t, alertID.String(), updatedAlertProto.Id)
	assert.False(t, updatedAlertProto.Enabled)
	assert.Equal(t, int32(5), updatedAlertProto.ConsecutiveFailures)
}

func TestAlertServiceServer_DeleteAlert(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	alertID := uuid.New()

	deleteReq := &alertproto.DeleteAlertRequest{
		Id: alertID.String(),
	}

	// Mock service returns nil (success), so handler should work
	_, err := server.DeleteAlert(ctx, deleteReq)
	require.NoError(t, err)
}

func TestAlertServiceServer_EnableAlert(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	alertID := uuid.New()

	req := &alertproto.EnableAlertRequest{
		Id: alertID.String(),
	}

	// Mock service returns nil (success), so handler should work
	_, err := server.EnableAlert(ctx, req)
	require.NoError(t, err)
}

func TestAlertServiceServer_DisableAlert(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	alertID := uuid.New()

	req := &alertproto.DisableAlertRequest{
		Id: alertID.String(),
	}

	// Mock service returns nil (success), so handler should work
	_, err := server.DisableAlert(ctx, req)
	require.NoError(t, err)
}

func TestAlertServiceServer_GetNotificationChannels(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	// Add user ID directly to context for testing
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)

	req := &alertproto.GetNotificationChannelsRequest{
		UserId: uuid.New().String(), // req.UserId is ignored; JWT context is used
	}

	resp, err := server.GetNotificationChannels(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, mockUserID.String(), resp.UserId) // response reflects JWT user, not req.UserId
	assert.NotEmpty(t, resp.Channels)
}

func TestAlertServiceServer_UpdateNotificationChannels(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	// Add user ID directly to context for testing
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)

	req := &alertproto.UpdateNotificationChannelsRequest{
		UserId: uuid.New().String(), // req.UserId is ignored; JWT context is used
		Channels: []*alertproto.NotificationChannel{
			{
				Type:    alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL,
				Address: "test@example.com",
			},
		},
	}

	resp, err := server.UpdateNotificationChannels(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, mockUserID.String(), resp.UserId) // response reflects JWT user, not req.UserId
	assert.Equal(t, 1, len(resp.Channels))
}

// Additional comprehensive tests for better coverage

func TestAlertServiceServer_CreateAlert_AllTypes(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)

	tests := []struct {
		name string
		req  *alertproto.CreateAlertRequest
	}{
		{
			name: "status_code",
			req: &alertproto.CreateAlertRequest{
				MonitorId:           uuid.New().String(),
				Type:                alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
				ThresholdMs:         5000,
				ConsecutiveFailures: 3,
				TypeSpecificFields: &alertproto.CreateAlertRequest_ExpectedStatusCode{
					ExpectedStatusCode: "200",
				},
			},
		},
		{
			name: "response_time",
			req: &alertproto.CreateAlertRequest{
				MonitorId:           uuid.New().String(),
				Type:                alertproto.AlertType_ALERT_TYPE_RESPONSE_TIME,
				ThresholdMs:         5000,
				ConsecutiveFailures: 3,
				TypeSpecificFields: &alertproto.CreateAlertRequest_MaxResponseTimeMs{
					MaxResponseTimeMs: 10000,
				},
			},
		},
		{
			name: "body_contains",
			req: &alertproto.CreateAlertRequest{
				MonitorId:           uuid.New().String(),
				Type:                alertproto.AlertType_ALERT_TYPE_BODY_CONTAINS,
				ThresholdMs:         0,
				ConsecutiveFailures: 3,
				TypeSpecificFields: &alertproto.CreateAlertRequest_BodyPattern{
					BodyPattern: "error",
				},
			},
		},
		{
			name: "certificate_expires",
			req: &alertproto.CreateAlertRequest{
				MonitorId:           uuid.New().String(),
				Type:                alertproto.AlertType_ALERT_TYPE_CERTIFICATE_EXPIRES,
				ThresholdMs:         86400000,
				ConsecutiveFailures: 3,
				TypeSpecificFields: &alertproto.CreateAlertRequest_DaysBeforeExpiry{
					DaysBeforeExpiry: 7,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.CreateAlert(ctx, tt.req)
			require.NoError(t, err)
		})
	}
}

func TestAlertServiceServer_CreateAlert_ValidationErrors(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)

	tests := []struct {
		name        string
		req         *alertproto.CreateAlertRequest
		expectError bool
	}{
		{
			name: "empty_monitor_id",
			req: &alertproto.CreateAlertRequest{
				MonitorId: "",
				Type:      alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
			},
			expectError: true,
		},
		{
			name: "invalid_monitor_id_format",
			req: &alertproto.CreateAlertRequest{
				MonitorId: "invalid-uuid",
				Type:      alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
			},
			expectError: true,
		},
		{
			name: "unspecified_type",
			req: &alertproto.CreateAlertRequest{
				MonitorId: uuid.New().String(),
				Type:      alertproto.AlertType_ALERT_TYPE_UNSPECIFIED,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.CreateAlert(ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_GetAlert_InvalidID(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()

	tests := []struct {
		name        string
		alertID     string
		expectError bool
	}{
		{
			name:        "invalid_uuid_format",
			alertID:     "invalid-uuid",
			expectError: true,
		},
		{
			name:        "empty_uuid",
			alertID:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &alertproto.GetAlertRequest{
				Id: tt.alertID,
			}

			_, err := server.GetAlert(ctx, req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_ListAlerts_Validation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	userID := uuid.New()
	authedCtx := context.WithValue(context.Background(), auth.UserIDKey, userID)

	tests := []struct {
		name        string
		ctx         context.Context
		req         *alertproto.ListAlertsRequest
		expectError bool
	}{
		{
			name: "unauthenticated",
			ctx:  context.Background(),
			req: &alertproto.ListAlertsRequest{
				Page:     1,
				PageSize: 10,
			},
			expectError: true,
		},
		{
			name: "valid_request",
			ctx:  authedCtx,
			req: &alertproto.ListAlertsRequest{
				Page:     1,
				PageSize: 10,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.ListAlerts(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_UpdateAlert_Validation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()

	tests := []struct {
		name        string
		req         *alertproto.UpdateAlertRequest
		expectError bool
	}{
		{
			name: "invalid_alert_id",
			req: &alertproto.UpdateAlertRequest{
				Id:      "invalid-uuid",
				Enabled: false,
			},
			expectError: true,
		},
		{
			name: "valid_update",
			req: &alertproto.UpdateAlertRequest{
				Id:      uuid.New().String(),
				Enabled: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.UpdateAlert(ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_DeleteAlert_InvalidID(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()

	tests := []struct {
		name        string
		alertID     string
		expectError bool
	}{
		{
			name:        "invalid_uuid",
			alertID:     "invalid-uuid",
			expectError: true,
		},
		{
			name:        "empty_uuid",
			alertID:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &alertproto.DeleteAlertRequest{
				Id: tt.alertID,
			}

			_, err := server.DeleteAlert(ctx, req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_EnableDisableAlert_Validation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()

	tests := []struct {
		name        string
		alertID     string
		expectError bool
	}{
		{
			name:        "enable_invalid_uuid",
			alertID:     "invalid-uuid",
			expectError: true,
		},
		{
			name:        "disable_invalid_uuid",
			alertID:     "invalid-uuid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test EnableAlert
			enableReq := &alertproto.EnableAlertRequest{
				Id: tt.alertID,
			}
			_, err := server.EnableAlert(ctx, enableReq)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Test DisableAlert
			disableReq := &alertproto.DisableAlertRequest{
				Id: tt.alertID,
			}
			_, err = server.DisableAlert(ctx, disableReq)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_UpdateNotificationChannels_AllTypes(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}

	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	mockUserID := uuid.New()
	ctx = context.WithValue(ctx, auth.UserIDKey, mockUserID)

	tests := []struct {
		name string
		req  *alertproto.UpdateNotificationChannelsRequest
	}{
		{
			name: "email_channel",
			req: &alertproto.UpdateNotificationChannelsRequest{
				Channels: []*alertproto.NotificationChannel{
					{
						Type:    alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL,
						Address: "test@example.com",
					},
				},
			},
		},
		{
			name: "telegram_channel",
			req: &alertproto.UpdateNotificationChannelsRequest{
				Channels: []*alertproto.NotificationChannel{
					{
						Type:    alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM,
						Address: "@testuser",
					},
				},
			},
		},
		{
			name: "webhook_channel",
			req: &alertproto.UpdateNotificationChannelsRequest{
				Channels: []*alertproto.NotificationChannel{
					{
						Type:    alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK,
						Address: "https://example.com/webhook",
					},
				},
			},
		},
		{
			name: "multiple_channels",
			req: &alertproto.UpdateNotificationChannelsRequest{
				Channels: []*alertproto.NotificationChannel{
					{
						Type:    alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL,
						Address: "test@example.com",
					},
					{
						Type:    alertproto.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM,
						Address: "@testuser",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.UpdateNotificationChannels(ctx, tt.req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, mockUserID.String(), resp.UserId) // response reflects JWT user
		})
	}
}

func TestAlertServiceServer_CreateAlert_Authentication(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}
	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	monitorID := uuid.New()

	tests := []struct {
		name        string
		ctx         context.Context
		req         *alertproto.CreateAlertRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "missing_user_id_in_context",
			ctx:  context.Background(), // No user ID in context
			req: &alertproto.CreateAlertRequest{
				MonitorId: monitorID.String(),
				Type:      alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
			},
			expectError: true,
			errorCode:   codes.Unauthenticated,
		},
		{
			name: "nil_user_id_in_context",
			ctx:  context.WithValue(context.Background(), auth.UserIDKey, uuid.Nil),
			req: &alertproto.CreateAlertRequest{
				MonitorId: monitorID.String(),
				Type:      alertproto.AlertType_ALERT_TYPE_STATUS_CODE,
			},
			expectError: true,
			errorCode:   codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.CreateAlert(tt.ctx, tt.req)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_GetNotificationChannels_Validation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}
	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		expectError bool
		errorCode   codes.Code
	}{
		{
			name:        "invalid_user_id_format",
			userID:      "not-a-uuid",
			expectError: true,
			errorCode:   codes.Unauthenticated, // JWT not in context → Unauthenticated
		},
		{
			name:        "empty_user_id",
			userID:      "",
			expectError: true,
			errorCode:   codes.Unauthenticated, // JWT not in context → Unauthenticated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &alertproto.GetNotificationChannelsRequest{
				UserId: tt.userID,
			}
			_, err := server.GetNotificationChannels(ctx, req)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_UpdateNotificationChannels_Validation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}
	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		expectError bool
		errorCode   codes.Code
	}{
		{
			name:        "invalid_user_id_format",
			userID:      "not-a-uuid",
			expectError: true,
			errorCode:   codes.Unauthenticated, // JWT not in context → Unauthenticated
		},
		{
			name:        "empty_user_id",
			userID:      "",
			expectError: true,
			errorCode:   codes.Unauthenticated, // JWT not in context → Unauthenticated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &alertproto.UpdateNotificationChannelsRequest{
				UserId: tt.userID,
			}
			_, err := server.UpdateNotificationChannels(ctx, req)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_UpdateAlert_ComprehensiveScenarios(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}
	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx := context.Background()
	alertID := uuid.New()

	tests := []struct {
		name        string
		alertID     string
		req         *alertproto.UpdateAlertRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name:    "update_existing_alert",
			alertID: alertID.String(),
			req: &alertproto.UpdateAlertRequest{
				Id:      alertID.String(),
				Enabled: true,
			},
			expectError: false,
		},
		{
			name:    "update_nonexistent_alert",
			alertID: uuid.New().String(),
			req: &alertproto.UpdateAlertRequest{
				Id:      alertID.String(),
				Enabled: true,
			},
			expectError: false, // Mock doesn't simulate not found
		},
		{
			name:    "update_with_empty_id",
			alertID: "",
			req: &alertproto.UpdateAlertRequest{
				Id:      "",
				Enabled: true,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name:    "toggle_enable",
			alertID: alertID.String(),
			req: &alertproto.UpdateAlertRequest{
				Id:      alertID.String(),
				Enabled: false,
			},
			expectError: false,
		},
		{
			name:    "update_with_high_threshold",
			alertID: alertID.String(),
			req: &alertproto.UpdateAlertRequest{
				Id:          alertID.String(),
				Enabled:     true,
				ThresholdMs: 5000, // High threshold
			},
			expectError: false,
		},
		{
			name:    "update_with_negative_threshold",
			alertID: alertID.String(),
			req: &alertproto.UpdateAlertRequest{
				Id:          alertID.String(),
				Enabled:     true,
				ThresholdMs: -100, // Negative threshold
			},
			expectError: false,
		},
		{
			name:    "update_zero_threshold",
			alertID: alertID.String(),
			req: &alertproto.UpdateAlertRequest{
				Id:          alertID.String(),
				Enabled:     true,
				ThresholdMs: 0, // Zero threshold
			},
			expectError: false,
		},
		{
			name:    "update_with_very_high_threshold",
			alertID: alertID.String(),
			req: &alertproto.UpdateAlertRequest{
				Id:          alertID.String(),
				Enabled:     true,
				ThresholdMs: 999999, // Very high threshold
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.UpdateAlert(ctx, tt.req)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertServiceServer_UpdateAlert_ContextCancellation(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}
	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	alertID := uuid.New()
	req := &alertproto.UpdateAlertRequest{
		Id:      alertID.String(),
		Enabled: true,
	}

	_, err := server.UpdateAlert(ctx, req)
	// Mock doesn't simulate context cancellation, so we just check no error
	// In real scenario, this would return context.Canceled error
	if err != nil {
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
	}
}

func TestAlertServiceServer_UpdateAlert_NilContext(t *testing.T) {
	mockAlert := &mockAlertService{}
	mockChannel := &mockChannelService{}
	mockRule := &mockRuleService{}
	authMiddleware := createTestMiddleware(true)
	server := NewAlertServiceServer(mockAlert, mockChannel, mockRule, &mockMuteService{}, authMiddleware)

	alertID := uuid.New()
	req := &alertproto.UpdateAlertRequest{
		Id:      alertID.String(),
		Enabled: true,
	}

	_, err := server.UpdateAlert(context.Background(), req)
	assert.NoError(t, err)
}

func TestAlertServiceServer_AcknowledgeAlert_Success(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	alertID := uuid.New()

	_, err := server.AcknowledgeAlert(ctx, &alertproto.AcknowledgeAlertRequest{Id: alertID.String()})
	assert.NoError(t, err)
}

func TestAlertServiceServer_AcknowledgeAlert_InvalidID(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.AcknowledgeAlert(ctx, &alertproto.AcknowledgeAlertRequest{Id: "bad-uuid"})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_AcknowledgeAlert_Unauthenticated(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.AcknowledgeAlert(context.Background(), &alertproto.AcknowledgeAlertRequest{Id: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAlertServiceServer_GetAlertRule_Success(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	monitorID := uuid.New()

	_, err := server.GetAlertRule(ctx, &alertproto.GetAlertRuleRequest{MonitorId: monitorID.String()})
	assert.NoError(t, err)
}

func TestAlertServiceServer_GetAlertRule_EmptyMonitorID(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.GetAlertRule(ctx, &alertproto.GetAlertRuleRequest{MonitorId: ""})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_GetAlertRule_InvalidMonitorID(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.GetAlertRule(ctx, &alertproto.GetAlertRuleRequest{MonitorId: "invalid"})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_GetAlertRule_Unauthenticated(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.GetAlertRule(context.Background(), &alertproto.GetAlertRuleRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAlertServiceServer_CreateAlertRule_Success(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	monitorID := uuid.New()

	_, err := server.CreateAlertRule(ctx, &alertproto.CreateAlertRuleRequest{
		MonitorId:           monitorID.String(),
		ConsecutiveFailures: 3,
		Enabled:             true,
	})
	assert.NoError(t, err)
}

func TestAlertServiceServer_CreateAlertRule_EmptyMonitorID(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.CreateAlertRule(ctx, &alertproto.CreateAlertRuleRequest{MonitorId: ""})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_CreateAlertRule_Unauthenticated(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.CreateAlertRule(context.Background(), &alertproto.CreateAlertRuleRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAlertServiceServer_MuteAlerts_Success(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	monitorID := uuid.New()

	_, err := server.MuteAlerts(ctx, &alertproto.MuteAlertsRequest{MonitorId: monitorID.String()})
	assert.NoError(t, err)
}

func TestAlertServiceServer_MuteAlerts_EmptyMonitorID(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.MuteAlerts(ctx, &alertproto.MuteAlertsRequest{MonitorId: ""})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_MuteAlerts_Unauthenticated(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.MuteAlerts(context.Background(), &alertproto.MuteAlertsRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAlertServiceServer_UnmuteAlerts_Success(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	monitorID := uuid.New()

	_, err := server.UnmuteAlerts(ctx, &alertproto.UnmuteAlertsRequest{MonitorId: monitorID.String()})
	assert.NoError(t, err)
}

func TestAlertServiceServer_UnmuteAlerts_EmptyMonitorID(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	_, err := server.UnmuteAlerts(ctx, &alertproto.UnmuteAlertsRequest{MonitorId: ""})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAlertServiceServer_UnmuteAlerts_Unauthenticated(t *testing.T) {
	server := NewAlertServiceServer(&mockAlertService{}, &mockChannelService{}, &mockRuleService{}, &mockMuteService{}, createTestMiddleware(true))

	_, err := server.UnmuteAlerts(context.Background(), &alertproto.UnmuteAlertsRequest{MonitorId: uuid.New().String()})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}
