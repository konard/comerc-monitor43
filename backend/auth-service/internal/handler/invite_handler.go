package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	authv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// inviteServicer описывает методы InviteService, нужные InviteHandler.
type inviteServicer interface {
	Invite(ctx context.Context, orgID, inviterID uuid.UUID, email string, role model.Role) (*model.Invite, bool, error)
	Revoke(ctx context.Context, actorID, inviteID uuid.UUID) error
	Accept(ctx context.Context, token uuid.UUID, authedUserEmail string, authedUserID uuid.UUID) (*model.Membership, error)
}

// InviteHandler реализует gRPC-методы управления приглашениями.
type InviteHandler struct {
	authv1.UnimplementedAuthServiceServer
	inviteService inviteServicer
	users         teamUserLookup
	tokenService  jwtValidator
	auditRepo     auditWriter
	logger        *slog.Logger
}

// InviteHandlerOption настраивает опциональные зависимости InviteHandler.
type InviteHandlerOption func(*InviteHandler)

// WithInviteAuditRepo подключает audit repository для записи событий
// member_invited / invite_accepted / invite_revoked.
func WithInviteAuditRepo(repo auditWriter) InviteHandlerOption {
	return func(h *InviteHandler) { h.auditRepo = repo }
}

// NewInviteHandler создаёт новый InviteHandler.
func NewInviteHandler(inviteService inviteServicer, users teamUserLookup, tokenService jwtValidator, logger *slog.Logger, opts ...InviteHandlerOption) *InviteHandler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &InviteHandler{
		inviteService: inviteService,
		users:         users,
		tokenService:  tokenService,
		logger:        logger.With("module", "invite_handler"),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// writeAudit пишет audit-запись и логирует ошибку, не прерывая RPC.
func (h *InviteHandler) writeAudit(ctx context.Context, userID uuid.UUID, eventType string) {
	if h.auditRepo == nil {
		return
	}
	uid := userID
	entry := model.NewAuditLog(&uid, eventType, "", true, "", "", "")
	if _, err := h.auditRepo.Create(ctx, entry); err != nil {
		h.logger.WarnContext(ctx, "Failed to write audit log",
			"event_type", eventType, "user_id", userID, "error", err)
	}
}

// InviteMember создаёт новое приглашение в активной организации.
func (h *InviteHandler) InviteMember(ctx context.Context, req *authv1.InviteMemberRequest) (*authv1.Invite, error) {
	claims, err := h.parseClaims(ctx)
	if err != nil {
		return nil, err
	}
	if claims.OrgID == uuid.Nil {
		return nil, status.Error(codes.PermissionDenied, "no active organization")
	}
	role := roleFromProto(req.GetRole())
	if role == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}
	inv, resent, err := h.inviteService.Invite(ctx, claims.OrgID, claims.UserID, req.GetEmail(), role)
	if err != nil {
		return nil, teamErrorToStatus(err)
	}
	eventType := model.EventTypeMemberInvited
	if resent {
		eventType = model.EventTypeInviteResent
	}
	h.writeAudit(ctx, claims.UserID, eventType)
	return inviteToProto(inv), nil
}

// RevokeInvite отзывает PENDING приглашение по ID.
func (h *InviteHandler) RevokeInvite(ctx context.Context, req *authv1.RevokeInviteRequest) (*authv1.Empty, error) {
	claims, err := h.parseClaims(ctx)
	if err != nil {
		return nil, err
	}
	inviteID, err := uuid.Parse(req.GetInviteId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invite_id")
	}
	if err := h.inviteService.Revoke(ctx, claims.UserID, inviteID); err != nil {
		return nil, teamErrorToStatus(err)
	}
	h.writeAudit(ctx, claims.UserID, model.EventTypeInviteRevoked)
	return &authv1.Empty{}, nil
}

// AcceptInvite принимает приглашение по token и создаёт membership.
func (h *InviteHandler) AcceptInvite(ctx context.Context, req *authv1.AcceptInviteRequest) (*authv1.Membership, error) {
	claims, err := h.parseClaims(ctx)
	if err != nil {
		return nil, err
	}
	token, err := uuid.Parse(req.GetToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid token")
	}
	m, err := h.inviteService.Accept(ctx, token, claims.Email, claims.UserID)
	if err != nil {
		return nil, teamErrorToStatus(err)
	}
	h.writeAudit(ctx, claims.UserID, model.EventTypeInviteAccepted)
	email, fullName := claims.Email, ""
	if user, lookupErr := h.users.GetByID(ctx, claims.UserID); lookupErr == nil && user != nil {
		fullName = user.FullName
		if user.Email != "" {
			email = user.Email
		}
	}
	return membershipToProto(m, email, fullName), nil
}

// claimsBundle — минимальный набор JWT claims, нужный invite handler'у.
type claimsBundle struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Email  string
	Role   string
}

func (h *InviteHandler) parseClaims(ctx context.Context) (*claimsBundle, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}
	token := strings.TrimPrefix(authHeaders[0], "Bearer ")
	claims, err := h.tokenService.ValidateAccessToken(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	return &claimsBundle{
		UserID: claims.UserID,
		OrgID:  claims.OrgID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
