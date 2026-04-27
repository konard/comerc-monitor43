package handler

import (
	"context"

	authv1 "github.com/raul/monitor/api/proto"
)

// CombinedHandler объединяет auth, session, user, team и invite хэндлеры
// в единый AuthServiceServer.
type CombinedHandler struct {
	authv1.UnimplementedAuthServiceServer
	authHandler    *AuthHandler
	sessionHandler *SessionHandler
	userHandler    *UserHandler
	teamHandler    *TeamHandler
	inviteHandler  *InviteHandler
}

// NewCombinedHandler создаёт новый объединённый хэндлер.
func NewCombinedHandler(
	authHandler *AuthHandler,
	sessionHandler *SessionHandler,
	userHandler *UserHandler,
	teamHandler *TeamHandler,
	inviteHandler *InviteHandler,
) *CombinedHandler {
	return &CombinedHandler{
		authHandler:    authHandler,
		sessionHandler: sessionHandler,
		userHandler:    userHandler,
		teamHandler:    teamHandler,
		inviteHandler:  inviteHandler,
	}
}

// Auth methods

func (h *CombinedHandler) OAuthLogin(ctx context.Context, req *authv1.OAuthLoginRequest) (*authv1.OAuthLoginResponse, error) {
	return h.authHandler.GoogleOAuthLogin(ctx, req)
}

func (h *CombinedHandler) OAuthCallback(ctx context.Context, req *authv1.OAuthCallbackRequest) (*authv1.AuthResponse, error) {
	return h.authHandler.GoogleOAuthCallback(ctx, req)
}

func (h *CombinedHandler) PasswordLogin(ctx context.Context, req *authv1.PasswordLoginRequest) (*authv1.AuthResponse, error) {
	return h.authHandler.PasswordLogin(ctx, req)
}

func (h *CombinedHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	return h.authHandler.Register(ctx, req)
}

func (h *CombinedHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	return h.authHandler.RefreshToken(ctx, req)
}

func (h *CombinedHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.Empty, error) {
	return h.authHandler.Logout(ctx, req)
}

func (h *CombinedHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	return h.authHandler.ValidateToken(ctx, req)
}

// Session methods

func (h *CombinedHandler) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	return h.sessionHandler.ListSessions(ctx, req)
}

func (h *CombinedHandler) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.Empty, error) {
	return h.sessionHandler.RevokeSession(ctx, req)
}

// User methods

func (h *CombinedHandler) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	return h.userHandler.GetUser(ctx, req)
}

func (h *CombinedHandler) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.User, error) {
	return h.userHandler.UpdateUser(ctx, req)
}

func (h *CombinedHandler) LockAccount(ctx context.Context, req *authv1.LockAccountRequest) (*authv1.Empty, error) {
	return h.userHandler.LockAccount(ctx, req)
}

func (h *CombinedHandler) UnlockAccount(ctx context.Context, req *authv1.UnlockAccountRequest) (*authv1.Empty, error) {
	return h.userHandler.UnlockAccount(ctx, req)
}

// Team methods

func (h *CombinedHandler) ListMembers(ctx context.Context, req *authv1.ListMembersRequest) (*authv1.ListMembersResponse, error) {
	return h.teamHandler.ListMembers(ctx, req)
}

func (h *CombinedHandler) ChangeMemberRole(ctx context.Context, req *authv1.ChangeMemberRoleRequest) (*authv1.Membership, error) {
	return h.teamHandler.ChangeMemberRole(ctx, req)
}

func (h *CombinedHandler) RemoveMember(ctx context.Context, req *authv1.RemoveMemberRequest) (*authv1.Empty, error) {
	return h.teamHandler.RemoveMember(ctx, req)
}

func (h *CombinedHandler) TransferOwnership(ctx context.Context, req *authv1.TransferOwnershipRequest) (*authv1.Empty, error) {
	return h.teamHandler.TransferOwnership(ctx, req)
}

func (h *CombinedHandler) LeaveOrganization(ctx context.Context, req *authv1.LeaveOrganizationRequest) (*authv1.Empty, error) {
	return h.teamHandler.LeaveOrganization(ctx, req)
}

// Invite methods

func (h *CombinedHandler) InviteMember(ctx context.Context, req *authv1.InviteMemberRequest) (*authv1.Invite, error) {
	return h.inviteHandler.InviteMember(ctx, req)
}

func (h *CombinedHandler) RevokeInvite(ctx context.Context, req *authv1.RevokeInviteRequest) (*authv1.Empty, error) {
	return h.inviteHandler.RevokeInvite(ctx, req)
}

func (h *CombinedHandler) AcceptInvite(ctx context.Context, req *authv1.AcceptInviteRequest) (*authv1.Membership, error) {
	return h.inviteHandler.AcceptInvite(ctx, req)
}
