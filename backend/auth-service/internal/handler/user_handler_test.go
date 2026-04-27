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

func TestGetUser(t *testing.T) {
	t.Parallel()

	userSvc := mocks.NewUserServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewUserHandler(userSvc, jwtVal)

	userID := uuid.New()
	now := time.Now()

	userSvc.On("GetUser", mock.Anything, mock.MatchedBy(func(req *dto.GetUserRequest) bool {
		return req.UserID == userID
	})).Return(&dto.GetUserResponse{
		ID:        userID,
		Email:     "user@example.com",
		FullName:  "Test User",
		Tier:      "pro",
		CreatedAt: now,
	}, nil)

	resp, err := h.GetUser(context.Background(), &authv1.GetUserRequest{
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, userID.String(), resp.Id)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.Equal(t, "pro", resp.Tier)
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	userSvc := mocks.NewUserServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewUserHandler(userSvc, jwtVal)

	userID := uuid.New()
	now := time.Now()

	md := metadata.Pairs("authorization", "Bearer test-jwt-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	jwtVal.On("ValidateAccessToken", mock.Anything, "test-jwt-token").
		Return(&jwt.Claims{UserID: userID}, nil)

	userSvc.On("UpdateUser", mock.Anything, mock.MatchedBy(func(req *dto.UpdateUserRequest) bool {
		return req.UserID == userID && req.FullName == "Updated Name"
	})).Return(&dto.GetUserResponse{
		ID:        userID,
		Email:     "user@example.com",
		FullName:  "Updated Name",
		Tier:      "pro",
		CreatedAt: now,
	}, nil)

	resp, err := h.UpdateUser(ctx, &authv1.UpdateUserRequest{
		FullName: "Updated Name",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp.FullName)
	assert.Equal(t, userID.String(), resp.Id)
}

func TestLockAccount(t *testing.T) {
	t.Parallel()

	userSvc := mocks.NewUserServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewUserHandler(userSvc, jwtVal)

	userID := uuid.New()

	userSvc.On("LockAccount", mock.Anything, mock.MatchedBy(func(req *dto.LockAccountRequest) bool {
		return req.UserID == userID && req.DurationHours == 24
	})).Return(nil)

	resp, err := h.LockAccount(context.Background(), &authv1.LockAccountRequest{
		UserId:            userID.String(),
		LockDurationHours: 24,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestUnlockAccount(t *testing.T) {
	t.Parallel()

	userSvc := mocks.NewUserServicer(t)
	jwtVal := mocks.NewJwtValidator(t)
	h := handler.NewUserHandler(userSvc, jwtVal)

	userID := uuid.New()

	userSvc.On("UnlockAccount", mock.Anything, mock.MatchedBy(func(req *dto.UnlockAccountRequest) bool {
		return req.UserID == userID
	})).Return(nil)

	resp, err := h.UnlockAccount(context.Background(), &authv1.UnlockAccountRequest{
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
