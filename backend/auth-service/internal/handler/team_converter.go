package handler

import (
	"errors"
	"time"

	authv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service"
)

// errorDomain — домен для google.rpc.ErrorInfo в team management ошибках.
const errorDomain = "auth-service"

// roleToProto конвертирует доменную роль в proto enum.
func roleToProto(r model.Role) authv1.Role {
	switch r {
	case model.RoleOwner:
		return authv1.Role_ROLE_OWNER
	case model.RoleAdmin:
		return authv1.Role_ROLE_ADMIN
	case model.RoleMember:
		return authv1.Role_ROLE_MEMBER
	case model.RoleViewer:
		return authv1.Role_ROLE_VIEWER
	default:
		return authv1.Role_ROLE_UNSPECIFIED
	}
}

// roleFromProto конвертирует proto enum в доменную роль.
func roleFromProto(r authv1.Role) model.Role {
	switch r {
	case authv1.Role_ROLE_OWNER:
		return model.RoleOwner
	case authv1.Role_ROLE_ADMIN:
		return model.RoleAdmin
	case authv1.Role_ROLE_MEMBER:
		return model.RoleMember
	case authv1.Role_ROLE_VIEWER:
		return model.RoleViewer
	default:
		return ""
	}
}

// inviteStatusToProto конвертирует доменный статус приглашения в proto enum.
func inviteStatusToProto(s model.InviteStatus) authv1.InviteStatus {
	switch s {
	case model.InvitePending:
		return authv1.InviteStatus_INVITE_STATUS_PENDING
	case model.InviteAccepted:
		return authv1.InviteStatus_INVITE_STATUS_ACCEPTED
	case model.InviteDeclined:
		return authv1.InviteStatus_INVITE_STATUS_DECLINED
	case model.InviteExpired:
		return authv1.InviteStatus_INVITE_STATUS_EXPIRED
	case model.InviteRevoked:
		return authv1.InviteStatus_INVITE_STATUS_REVOKED
	default:
		return authv1.InviteStatus_INVITE_STATUS_UNSPECIFIED
	}
}

// membershipToProto конвертирует доменный membership в proto.
// email и fullName подгружаются handler'ом из user repo.
func membershipToProto(m *model.Membership, email, fullName string) *authv1.Membership {
	if m == nil {
		return nil
	}
	out := &authv1.Membership{
		Id:             m.ID.String(),
		OrganizationId: m.OrganizationID.String(),
		UserId:         m.UserID.String(),
		Email:          email,
		FullName:       fullName,
		Role:           roleToProto(m.Role),
		JoinedAt:       m.JoinedAt.UTC().Format(time.RFC3339),
	}
	if m.LastActiveAt != nil {
		out.LastActiveAt = m.LastActiveAt.UTC().Format(time.RFC3339)
	}
	return out
}

// inviteToProto конвертирует доменный invite в proto.
func inviteToProto(inv *model.Invite) *authv1.Invite {
	if inv == nil {
		return nil
	}
	return &authv1.Invite{
		Id:             inv.ID.String(),
		OrganizationId: inv.OrganizationID.String(),
		InviterId:      inv.InviterID.String(),
		Email:          inv.Email,
		Role:           roleToProto(inv.Role),
		Token:          inv.Token.String(),
		Status:         inviteStatusToProto(inv.Status),
		CreatedAt:      inv.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:      inv.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

// teamErrorMapping связывает sentinel error с gRPC code и Reason.
type teamErrorMapping struct {
	code   codes.Code
	reason string
}

// teamErrorMap — таблица соответствия доменных ошибок team-management
// gRPC статусам и Reason'ам в google.rpc.ErrorInfo.
var teamErrorMap = []struct {
	err     error
	mapping teamErrorMapping
}{
	{service.ErrInsufficientPermissions, teamErrorMapping{codes.PermissionDenied, "INSUFFICIENT_PERMISSIONS"}},
	{service.ErrCannotModifyOwner, teamErrorMapping{codes.PermissionDenied, "CANNOT_MODIFY_OWNER"}},
	{service.ErrCannotRemoveOwner, teamErrorMapping{codes.PermissionDenied, "CANNOT_REMOVE_OWNER"}},
	{service.ErrOwnerCannotLeave, teamErrorMapping{codes.FailedPrecondition, "OWNER_CANNOT_LEAVE"}},
	{service.ErrUserAlreadyMember, teamErrorMapping{codes.AlreadyExists, "USER_ALREADY_MEMBER"}},
	{service.ErrCannotAssignOwnerRole, teamErrorMapping{codes.InvalidArgument, "CANNOT_ASSIGN_OWNER_ROLE"}},
	{service.ErrUserLimitExceeded, teamErrorMapping{codes.ResourceExhausted, "USER_LIMIT_EXCEEDED"}},
	{service.ErrOrganizationAccessDenied, teamErrorMapping{codes.PermissionDenied, "ORGANIZATION_ACCESS_DENIED"}},
	{service.ErrInvalidEmail, teamErrorMapping{codes.InvalidArgument, "INVALID_EMAIL"}},
	{service.ErrInviteNotFound, teamErrorMapping{codes.NotFound, "INVITE_NOT_FOUND"}},
	{service.ErrInviteExpired, teamErrorMapping{codes.FailedPrecondition, "INVITE_EXPIRED"}},
	{service.ErrInviteAlreadyUsed, teamErrorMapping{codes.FailedPrecondition, "INVITE_ALREADY_USED"}},
	{service.ErrInviteEmailMismatch, teamErrorMapping{codes.PermissionDenied, "INVITE_EMAIL_MISMATCH"}},
	{service.ErrInviteLimitReached, teamErrorMapping{codes.ResourceExhausted, "INVITE_LIMIT_REACHED"}},
}

// teamErrorToStatus маппит доменные ошибки team management в gRPC status.
// Возвращает status с прикреплённым ErrorInfo (Reason + Domain).
// Для неизвестных ошибок возвращает codes.Internal с текстом исходной ошибки.
func teamErrorToStatus(err error) error {
	if err == nil {
		return nil
	}
	for _, m := range teamErrorMap {
		if errors.Is(err, m.err) {
			st := status.New(m.mapping.code, err.Error())
			info := &errdetails.ErrorInfo{
				Reason: m.mapping.reason,
				Domain: errorDomain,
			}
			detailed, derr := st.WithDetails(info)
			if derr != nil {
				return st.Err()
			}
			return detailed.Err()
		}
	}
	return status.Error(codes.Internal, "internal error: "+err.Error())
}
