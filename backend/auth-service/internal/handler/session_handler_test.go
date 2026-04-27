package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/raul/monitor/backend/auth-service/internal/handler"
	"github.com/raul/monitor/backend/auth-service/internal/handler/mocks"
	"github.com/raul/monitor/backend/auth-service/internal/infrastructure/jwt"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
)

func TestListSessions_WithUserID(t *testing.T) {
	t.Parallel()

	sessionSvc := mocks.NewSessionServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewSessionHandler(sessionSvc, jwtVal)

	userID := uuid.New()
	now := time.Now()

	sessionSvc.On("ListSessions", mock.Anything, mock.MatchedBy(func(req *dto.ListSessionsRequest) bool {
		return req.UserID == userID
	})).Return(&dto.ListSessionsResponse{
		Sessions: []*dto.SessionDTO{
			{
				ID:           "session-1",
				UserID:       userID,
				DeviceInfo:   "Chrome/Linux",
				IPAddress:    "127.0.0.1",
				CreatedAt:    now,
				LastActiveAt: now,
			},
		},
		Total: 1,
	}, nil)

	resp, err := h.ListSessions(context.Background(), &authv1.ListSessionsRequest{
		UserId:   userID.String(),
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Total)
	assert.Len(t, resp.Sessions, 1)
	assert.Equal(t, "session-1", resp.Sessions[0].Id)
}

func TestListSessions_WithJWT(t *testing.T) {
	t.Parallel()

	sessionSvc := mocks.NewSessionServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewSessionHandler(sessionSvc, jwtVal)

	userID := uuid.New()
	now := time.Now()

	// Контекст с JWT-заголовком
	md := metadata.Pairs("authorization", "Bearer test-jwt-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	jwtVal.On("ValidateAccessToken", mock.Anything, "test-jwt-token").
		Return(&jwt.Claims{UserID: userID}, nil)

	sessionSvc.On("ListSessions", mock.Anything, mock.MatchedBy(func(req *dto.ListSessionsRequest) bool {
		return req.UserID == userID
	})).Return(&dto.ListSessionsResponse{
		Sessions: []*dto.SessionDTO{
			{
				ID:           "session-2",
				UserID:       userID,
				DeviceInfo:   "Safari/macOS",
				IPAddress:    "10.0.0.1",
				CreatedAt:    now,
				LastActiveAt: now,
			},
		},
		Total: 1,
	}, nil)

	resp, err := h.ListSessions(ctx, &authv1.ListSessionsRequest{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, "session-2", resp.Sessions[0].Id)
}

func TestRevokeSession(t *testing.T) {
	t.Parallel()

	sessionSvc := mocks.NewSessionServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewSessionHandler(sessionSvc, jwtVal)

	userID := uuid.New()

	md := metadata.Pairs("authorization", "Bearer test-jwt-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	jwtVal.On("ValidateAccessToken", mock.Anything, "test-jwt-token").
		Return(&jwt.Claims{UserID: userID}, nil)

	sessionSvc.On("RevokeSession", mock.Anything, mock.MatchedBy(func(req *dto.RevokeSessionRequest) bool {
		return req.SessionID == "session-to-revoke" && req.UserID == userID
	})).Return(nil)

	resp, err := h.RevokeSession(ctx, &authv1.RevokeSessionRequest{
		SessionId: "session-to-revoke",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
