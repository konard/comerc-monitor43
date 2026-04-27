package handler

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
)

// SessionHandler реализует gRPC-методы для управления сессиями.
type SessionHandler struct {
	authv1.UnimplementedAuthServiceServer
	sessionService sessionServicer
	tokenService   jwtValidator
}

// NewSessionHandler создаёт новый SessionHandler.
func NewSessionHandler(sessionService sessionServicer, tokenService jwtValidator) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		tokenService:   tokenService,
	}
}

// ListSessions реализует получение списка сессий (uc_05_01_08).
func (h *SessionHandler) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	// Fallback на JWT, если user_id не передан (короткий путь /api/v1/sessions)
	var (
		userID uuid.UUID
		err    error
	)
	if req.UserId == "" {
		userID, err = h.extractUserID(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
	} else {
		userID, err = uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user ID")
		}
	}

	// Извлекаем ID текущей сессии из заголовка для пометки в ответе
	currentSessionID := extractSessionID(ctx)

	dtoReq := &dto.ListSessionsRequest{
		UserID:           userID,
		Limit:            int(req.PageSize),
		Offset:           int(req.Page-1) * int(req.PageSize),
		CurrentSessionID: currentSessionID,
	}

	resp, err := h.sessionService.ListSessions(ctx, dtoReq)
	if err != nil {
		return nil, errorToStatus(err)
	}

	sessions := make([]*authv1.Session, len(resp.Sessions))
	for i, s := range resp.Sessions {
		sessions[i] = &authv1.Session{
			Id:        s.ID,
			UserId:    s.UserID.String(),
			AuthType:  s.DeviceInfo,
			IpAddress: s.IPAddress,
			UserAgent: "",
			CreatedAt: timestamppb.New(s.CreatedAt),
			ExpiresAt: timestamppb.New(s.LastActiveAt.Add(24 * time.Hour)),
		}
	}

	return &authv1.ListSessionsResponse{
		Sessions: sessions,
		Total:    safeInt32(resp.Total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func safeInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}

	return int32(v)
}

// RevokeSession реализует отзыв сессии (uc_05_01_09).
func (h *SessionHandler) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.Empty, error) {
	// Извлекаем ID пользователя из JWT-токена для проверки владения сессией
	userID, err := h.extractUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	dtoReq := &dto.RevokeSessionRequest{
		SessionID: req.SessionId,
		UserID:    userID,
	}

	if err := h.sessionService.RevokeSession(ctx, dtoReq); err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.Empty{}, nil
}

// extractUserID извлекает ID пользователя из JWT-токена в gRPC метаданных.
func (h *SessionHandler) extractUserID(ctx context.Context) (uuid.UUID, error) {
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

// extractSessionID извлекает ID сессии из gRPC метаданных.
func extractSessionID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	sessionIDs := md.Get("x-session-id")
	if len(sessionIDs) > 0 {
		return sessionIDs[0]
	}
	return ""
}
