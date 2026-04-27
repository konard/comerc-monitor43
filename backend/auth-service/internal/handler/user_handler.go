package handler

import (
	"context"
	"strings"

	"github.com/google/uuid"
	authv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
)

// UserHandler реализует gRPC-методы для управления пользователями.
type UserHandler struct {
	authv1.UnimplementedAuthServiceServer
	userService  userServicer
	tokenService jwtValidator
}

// NewUserHandler создаёт новый UserHandler.
func NewUserHandler(userService userServicer, tokenService jwtValidator) *UserHandler {
	return &UserHandler{
		userService:  userService,
		tokenService: tokenService,
	}
}

// GetUser implements get user (uc_05_01_10)
func (h *UserHandler) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	dtoReq := &dto.GetUserRequest{
		UserID: userID,
	}

	resp, err := h.userService.GetUser(ctx, dtoReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.User{
		Id:          resp.ID.String(),
		Email:       resp.Email,
		FullName:    resp.FullName,
		Tier:        resp.Tier,
		CreatedAt:   timestamppb.New(resp.CreatedAt),
		LastLoginAt: timestampProtoPtr(resp.LastLoginAt),
	}, nil
}

// UpdateUser обновляет профиль текущего пользователя (uc_05_01_13).
// user_id берём из JWT, так как PATCH /users/me не содержит его в теле запроса.
func (h *UserHandler) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.User, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	dtoReq := &dto.UpdateUserRequest{
		UserID:   userID,
		Email:    req.Email,
		FullName: req.FullName,
	}

	resp, err := h.userService.UpdateUser(ctx, dtoReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.User{
		Id:          resp.ID.String(),
		Email:       resp.Email,
		FullName:    resp.FullName,
		Tier:        resp.Tier,
		CreatedAt:   timestamppb.New(resp.CreatedAt),
		LastLoginAt: timestampProtoPtr(resp.LastLoginAt),
	}, nil
}

// extractUserID извлекает ID пользователя из JWT-токена в gRPC метаданных.
func (h *UserHandler) extractUserID(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := strings.TrimPrefix(authHeaders[0], "Bearer ")
	claims, err := h.tokenService.ValidateAccessToken(ctx, token)
	if err != nil {
		return uuid.Nil, err
	}

	return claims.UserID, nil
}

// LockAccount implements account locking (uc_05_01_11)
func (h *UserHandler) LockAccount(ctx context.Context, req *authv1.LockAccountRequest) (*authv1.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	dtoReq := &dto.LockAccountRequest{
		UserID:        userID,
		DurationHours: req.LockDurationHours,
	}

	if err := h.userService.LockAccount(ctx, dtoReq); err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.Empty{}, nil
}

// UnlockAccount implements account unlocking (uc_05_01_12)
func (h *UserHandler) UnlockAccount(ctx context.Context, req *authv1.UnlockAccountRequest) (*authv1.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	dtoReq := &dto.UnlockAccountRequest{
		UserID: userID,
	}

	if err := h.userService.UnlockAccount(ctx, dtoReq); err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.Empty{}, nil
}
