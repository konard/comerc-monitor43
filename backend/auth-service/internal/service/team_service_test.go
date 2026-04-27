package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/mocks"
)

// newTeamSvc создаёт TeamService с общими заглушками и переданными мок-зависимостями.
// Вынесено в helper, чтобы тесты различались только конкретными ожиданиями и
// не дублировали boilerplate конструктора.
func newTeamSvc(
	t *testing.T,
	memberships *mocks.MembershipRepo,
	opts ...TeamServiceOption,
) *TeamService {
	t.Helper()
	return NewTeamService(
		mocks.NewTeamServiceUsers(t),
		mocks.NewOrgRepo(t),
		memberships,
		opts...,
	)
}

// TestPermissionMatrix покрывает RBAC матрицу uc_05_02_25.
func TestPermissionMatrix(t *testing.T) {
	t.Parallel()

	// Arrange
	cases := []struct {
		role    model.Role
		op      Operation
		allowed bool
	}{
		{model.RoleOwner, OpInviteMember, true},
		{model.RoleAdmin, OpInviteMember, true},
		{model.RoleMember, OpInviteMember, false},
		{model.RoleViewer, OpInviteMember, false},
		{model.RoleOwner, OpRemoveMember, true},
		{model.RoleAdmin, OpRemoveMember, true},
		{model.RoleMember, OpRemoveMember, false},
		{model.RoleViewer, OpRemoveMember, false},
		{model.RoleOwner, OpChangeRole, true},
		{model.RoleAdmin, OpChangeRole, true},
		{model.RoleMember, OpChangeRole, false},
		{model.RoleViewer, OpChangeRole, false},
		{model.RoleOwner, OpTransferOwnership, true},
		{model.RoleAdmin, OpTransferOwnership, false},
		{model.RoleMember, OpTransferOwnership, false},
		{model.RoleViewer, OpTransferOwnership, false},
		{model.RoleOwner, OpCreateMonitor, true},
		{model.RoleAdmin, OpCreateMonitor, true},
		{model.RoleMember, OpCreateMonitor, true},
		{model.RoleViewer, OpCreateMonitor, false},
		{model.RoleOwner, OpListMembers, true},
		{model.RoleAdmin, OpListMembers, true},
		{model.RoleMember, OpListMembers, true},
		{model.RoleViewer, OpListMembers, true},
	}

	// Act + Assert
	for _, tc := range cases {
		assert.Equal(t, tc.allowed, permissionFor(tc.role, tc.op),
			"role=%s op=%s", tc.role, tc.op)
	}
}

// TestRemoveMember_OwnerCannotBeRemoved покрывает uc_05_02_21.
func TestRemoveMember_OwnerCannotBeRemoved(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	actorID := uuid.New()
	ownerID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, actorID).
		Return(&model.Membership{OrganizationID: orgID, UserID: actorID, Role: model.RoleAdmin}, nil)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, ownerID).
		Return(&model.Membership{OrganizationID: orgID, UserID: ownerID, Role: model.RoleOwner}, nil)

	svc := newTeamSvc(t, memberships)

	// Act
	err := svc.RemoveMember(context.Background(), orgID, actorID, ownerID)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotRemoveOwner)
}

// TestRemoveMember_RevokesSessions покрывает uc_05_02_28.
func TestRemoveMember_RevokesSessions(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, actorID).
		Return(&model.Membership{OrganizationID: orgID, UserID: actorID, Role: model.RoleAdmin}, nil)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, targetID).
		Return(&model.Membership{OrganizationID: orgID, UserID: targetID, Role: model.RoleMember}, nil)
	memberships.EXPECT().Delete(mock.Anything, orgID, targetID).Return(nil)

	revoker := mocks.NewSessionRevoker(t)
	revoker.EXPECT().RevokeAllForUser(mock.Anything, targetID).Return(nil)

	pub := mocks.NewTeamPublisher(t)
	pub.EXPECT().PublishMemberRemoved(mock.Anything, orgID, targetID).Return(nil)

	svc := newTeamSvc(t, memberships,
		WithSessionRevoker(revoker),
		WithTeamPublisher(pub),
	)

	// Act
	err := svc.RemoveMember(context.Background(), orgID, actorID, targetID)

	// Assert
	require.NoError(t, err)
}

// TestRemoveMember_RevokerErrorIsLoggedNotPropagated покрывает uc_05_02_28:
// ошибка sessionRevoker НЕ должна ломать удаление, только логируется.
func TestRemoveMember_RevokerErrorIsLoggedNotPropagated(t *testing.T) {
	t.Parallel()

	// Arrange: actor=ADMIN, target=MEMBER, Delete OK, RevokeAllForUser возвращает ошибку.
	orgID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, actorID).
		Return(&model.Membership{OrganizationID: orgID, UserID: actorID, Role: model.RoleAdmin}, nil)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, targetID).
		Return(&model.Membership{OrganizationID: orgID, UserID: targetID, Role: model.RoleMember}, nil)
	memberships.EXPECT().Delete(mock.Anything, orgID, targetID).Return(nil)

	revoker := mocks.NewSessionRevoker(t)
	revoker.EXPECT().RevokeAllForUser(mock.Anything, targetID).
		Return(errors.New("revoker unavailable"))

	pub := mocks.NewTeamPublisher(t)
	pub.EXPECT().PublishMemberRemoved(mock.Anything, orgID, targetID).Return(nil)

	svc := newTeamSvc(t, memberships,
		WithSessionRevoker(revoker),
		WithTeamPublisher(pub),
	)

	// Act: вызов RemoveMember.
	err := svc.RemoveMember(context.Background(), orgID, actorID, targetID)

	// Assert: возвращён nil (не ошибка), Delete был вызван, PublishMemberRemoved тоже.
	require.NoError(t, err)
	memberships.AssertCalled(t, "Delete", mock.Anything, orgID, targetID)
	pub.AssertCalled(t, "PublishMemberRemoved", mock.Anything, orgID, targetID)
}

// TestChangeMemberRole_AdminCannotModifyOwner покрывает uc_05_02_18.
func TestChangeMemberRole_AdminCannotModifyOwner(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	actorID := uuid.New()
	ownerID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, actorID).
		Return(&model.Membership{OrganizationID: orgID, UserID: actorID, Role: model.RoleAdmin}, nil)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, ownerID).
		Return(&model.Membership{OrganizationID: orgID, UserID: ownerID, Role: model.RoleOwner}, nil)

	svc := newTeamSvc(t, memberships)

	// Act
	err := svc.ChangeMemberRole(context.Background(), orgID, actorID, ownerID, model.RoleMember)

	// Assert
	assert.ErrorIs(t, err, ErrCannotModifyOwner)
}

// TestChangeMemberRole_RejectsOwnerRole покрывает uc_05_02_22 (transfer-only path).
func TestChangeMemberRole_RejectsOwnerRole(t *testing.T) {
	t.Parallel()

	// Arrange
	svc := newTeamSvc(t, mocks.NewMembershipRepo(t))

	// Act
	err := svc.ChangeMemberRole(context.Background(), uuid.New(), uuid.New(), uuid.New(), model.RoleOwner)

	// Assert
	assert.ErrorIs(t, err, ErrCannotAssignOwnerRole)
}

// TestLeaveOrganization_OwnerCannotLeave покрывает uc_05_02_24.
func TestLeaveOrganization_OwnerCannotLeave(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	actorID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, actorID).
		Return(&model.Membership{OrganizationID: orgID, UserID: actorID, Role: model.RoleOwner}, nil)
	memberships.EXPECT().CountByOrg(mock.Anything, orgID).Return(3, nil)

	svc := newTeamSvc(t, memberships)

	// Act
	err := svc.LeaveOrganization(context.Background(), orgID, actorID)

	// Assert
	assert.ErrorIs(t, err, ErrOwnerCannotLeave)
}

// TestEnsureOrganization_CreatesOwnerMembership покрывает uc_05_02_01.
func TestEnsureOrganization_CreatesOwnerMembership(t *testing.T) {
	t.Parallel()

	// Arrange
	userID := uuid.New()
	user := &model.User{ID: userID, Email: "owner@example.com", FullName: "Owner", Tier: "Free"}

	users := mocks.NewTeamServiceUsers(t)
	users.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)

	orgs := mocks.NewOrgRepo(t)
	orgs.EXPECT().Create(mock.Anything, mock.AnythingOfType("*model.Organization")).
		RunAndReturn(func(_ context.Context, o *model.Organization) (*model.Organization, error) {
			return o, nil
		})

	memberships := mocks.NewMembershipRepo(t)
	var createdRole model.Role
	memberships.EXPECT().Create(mock.Anything, mock.AnythingOfType("*model.Membership")).
		RunAndReturn(func(_ context.Context, m *model.Membership) (*model.Membership, error) {
			createdRole = m.Role
			return m, nil
		})

	svc := NewTeamService(users, orgs, memberships)

	// Act
	org, created, err := svc.EnsureOrganization(context.Background(), userID)

	// Assert
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, userID, org.OwnerID)
	assert.Equal(t, model.RoleOwner, createdRole)
}

// TestRemoveMember_InsufficientPermissions покрывает uc_05_02_19.
func TestRemoveMember_InsufficientPermissions(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, actorID).
		Return(&model.Membership{Role: model.RoleMember}, nil)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, targetID).
		Return(&model.Membership{Role: model.RoleMember}, nil)

	svc := newTeamSvc(t, memberships)

	// Act
	err := svc.RemoveMember(context.Background(), orgID, actorID, targetID)

	// Assert
	assert.ErrorIs(t, err, ErrInsufficientPermissions)
}

// TestCheckPermission_OrganizationAccessDenied покрывает uc_05_02_30.
func TestCheckPermission_OrganizationAccessDenied(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	userID := uuid.New()

	memberships := mocks.NewMembershipRepo(t)
	memberships.EXPECT().GetByOrgAndUser(mock.Anything, orgID, userID).
		Return((*model.Membership)(nil), nil)

	svc := newTeamSvc(t, memberships)

	// Act
	allowed, role, err := svc.CheckPermission(context.Background(), orgID, userID, OpListMembers)

	// Assert
	assert.False(t, allowed)
	assert.Empty(t, role)
	assert.True(t, errors.Is(err, ErrOrganizationAccessDenied))
}
