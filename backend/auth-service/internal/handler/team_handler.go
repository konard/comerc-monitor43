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

// teamServicer описывает методы TeamService, нужные TeamHandler.
type teamServicer interface {
	ListMembers(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*model.Membership, int, error)
	ChangeMemberRole(ctx context.Context, orgID, actorUserID, targetUserID uuid.UUID, newRole model.Role) error
	RemoveMember(ctx context.Context, orgID, actorUserID, targetUserID uuid.UUID) error
	TransferOwnership(ctx context.Context, orgID, actorUserID, targetUserID uuid.UUID) error
	LeaveOrganization(ctx context.Context, orgID, actorUserID uuid.UUID) error
}

// teamUserLookup описывает методы UserRepository, нужные TeamHandler для
// обогащения membership данными пользователя (email, full_name).
type teamUserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// auditWriter — частично применяемый интерфейс AuditRepository: handler'у
// нужна только запись успешной аудит-записи после RPC.
type auditWriter interface {
	Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error)
}

// TeamHandler реализует gRPC-методы управления участниками организации.
type TeamHandler struct {
	authv1.UnimplementedAuthServiceServer
	teamService  teamServicer
	users        teamUserLookup
	tokenService jwtValidator
	auditRepo    auditWriter
	logger       *slog.Logger
}

// TeamHandlerOption настраивает опциональные зависимости TeamHandler.
type TeamHandlerOption func(*TeamHandler)

// WithTeamAuditRepo подключает audit repository для записи событий
// member_invited / member_role_changed / member_removed и т.п.
func WithTeamAuditRepo(repo auditWriter) TeamHandlerOption {
	return func(h *TeamHandler) { h.auditRepo = repo }
}

// NewTeamHandler создаёт новый TeamHandler.
func NewTeamHandler(teamService teamServicer, users teamUserLookup, tokenService jwtValidator, logger *slog.Logger, opts ...TeamHandlerOption) *TeamHandler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &TeamHandler{
		teamService:  teamService,
		users:        users,
		tokenService: tokenService,
		logger:       logger.With("module", "team_handler"),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// writeAudit пишет audit-запись и логирует ошибку, не прерывая RPC.
func (h *TeamHandler) writeAudit(ctx context.Context, userID uuid.UUID, eventType string) {
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

// ListMembers возвращает список участников активной организации.
func (h *TeamHandler) ListMembers(ctx context.Context, req *authv1.ListMembersRequest) (*authv1.ListMembersResponse, error) {
	ctx, _, orgID, err := h.authedContext(ctx)
	if err != nil {
		return nil, err
	}
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	members, total, err := h.teamService.ListMembers(ctx, orgID, page, pageSize)
	if err != nil {
		return nil, teamErrorToStatus(err)
	}
	out := make([]*authv1.Membership, 0, len(members))
	for _, m := range members {
		email, fullName := h.lookupUser(ctx, m.UserID)
		out = append(out, membershipToProto(m, email, fullName))
	}
	return &authv1.ListMembersResponse{
		Members:  out,
		Total:    safeInt32(total),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

// ChangeMemberRole меняет роль участника.
func (h *TeamHandler) ChangeMemberRole(ctx context.Context, req *authv1.ChangeMemberRoleRequest) (*authv1.Membership, error) {
	ctx, actorID, orgID, err := h.authedContext(ctx)
	if err != nil {
		return nil, err
	}
	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	role := roleFromProto(req.GetRole())
	if role == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}
	if err := h.teamService.ChangeMemberRole(ctx, orgID, actorID, targetID, role); err != nil {
		return nil, teamErrorToStatus(err)
	}
	h.writeAudit(ctx, actorID, model.EventTypeMemberRoleChanged)
	updated := &model.Membership{
		ID:             uuid.Nil,
		OrganizationID: orgID,
		UserID:         targetID,
		Role:           role,
	}
	email, fullName := h.lookupUser(ctx, targetID)
	return membershipToProto(updated, email, fullName), nil
}

// RemoveMember удаляет участника из организации.
func (h *TeamHandler) RemoveMember(ctx context.Context, req *authv1.RemoveMemberRequest) (*authv1.Empty, error) {
	ctx, actorID, orgID, err := h.authedContext(ctx)
	if err != nil {
		return nil, err
	}
	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	if err := h.teamService.RemoveMember(ctx, orgID, actorID, targetID); err != nil {
		return nil, teamErrorToStatus(err)
	}
	h.writeAudit(ctx, actorID, model.EventTypeMemberRemoved)
	return &authv1.Empty{}, nil
}

// TransferOwnership передаёт OWNER другому участнику.
func (h *TeamHandler) TransferOwnership(ctx context.Context, req *authv1.TransferOwnershipRequest) (*authv1.Empty, error) {
	ctx, actorID, orgID, err := h.authedContext(ctx)
	if err != nil {
		return nil, err
	}
	targetID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	if err := h.teamService.TransferOwnership(ctx, orgID, actorID, targetID); err != nil {
		return nil, teamErrorToStatus(err)
	}
	h.writeAudit(ctx, actorID, model.EventTypeOwnershipTransferred)
	return &authv1.Empty{}, nil
}

// LeaveOrganization удаляет actor'а из его организации.
func (h *TeamHandler) LeaveOrganization(ctx context.Context, _ *authv1.LeaveOrganizationRequest) (*authv1.Empty, error) {
	ctx, actorID, orgID, err := h.authedContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.teamService.LeaveOrganization(ctx, orgID, actorID); err != nil {
		return nil, teamErrorToStatus(err)
	}
	h.writeAudit(ctx, actorID, model.EventTypeMemberLeft)
	return &authv1.Empty{}, nil
}

// authedContext извлекает userID/orgID/role из bearer-токена и возвращает
// обогащённый контекст. Возвращает gRPC error со статусом Unauthenticated,
// если токен отсутствует или невалиден; PermissionDenied, если в токене
// нет org context (пользователь не принадлежит ни одной организации).
func (h *TeamHandler) authedContext(ctx context.Context) (context.Context, uuid.UUID, uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, uuid.Nil, uuid.Nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ctx, uuid.Nil, uuid.Nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}
	token := strings.TrimPrefix(authHeaders[0], "Bearer ")
	claims, err := h.tokenService.ValidateAccessToken(ctx, token)
	if err != nil {
		return ctx, uuid.Nil, uuid.Nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	if claims.OrgID == uuid.Nil {
		return ctx, uuid.Nil, uuid.Nil, status.Error(codes.PermissionDenied, "no active organization")
	}
	ctx = WithUserID(ctx, claims.UserID)
	ctx = WithOrgID(ctx, claims.OrgID)
	ctx = WithRole(ctx, claims.Role)
	return ctx, claims.UserID, claims.OrgID, nil
}

// lookupUser возвращает email и full_name пользователя; при ошибке логирует
// и возвращает пустые строки, чтобы не прерывать обогащение списка.
func (h *TeamHandler) lookupUser(ctx context.Context, userID uuid.UUID) (string, string) {
	user, err := h.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		h.logger.WarnContext(ctx, "Failed to lookup user for membership",
			"user_id", userID, "error", err)
		return "", ""
	}
	return user.Email, user.FullName
}
