package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/mocks"
)

// newInviteSvc собирает InviteService с фиксированными часами и стандартным набором
// моков. Helper существует, чтобы тесты различались только ожиданиями и не
// дублировали boilerplate конструктора с WithInviteClock.
func newInviteSvc(
	t *testing.T,
	invites *mocks.InviteRepo,
	mems *mocks.InviteMembershipRepo,
	orgs *mocks.InviteOrgRepo,
	now time.Time,
) *InviteService {
	t.Helper()
	return NewInviteService(
		invites,
		mocks.NewInviteUsers(t),
		mems,
		orgs,
		WithInviteClock(func() time.Time { return now }),
	)
}

// TestInvite_RevokesPreviousPending покрывает uc_05_02_06.
func TestInvite_RevokesPreviousPending(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	inviterID := uuid.New()
	prevInviteID := uuid.New()
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	mems := mocks.NewInviteMembershipRepo(t)
	mems.EXPECT().GetByOrgAndUser(mock.Anything, orgID, inviterID).
		Return(&model.Membership{Role: model.RoleOwner}, nil)
	mems.EXPECT().FindByUserEmail(mock.Anything, orgID, "pending@example.com").
		Return((*model.Membership)(nil), nil)
	mems.EXPECT().CountByOrg(mock.Anything, orgID).Return(2, nil)

	orgs := mocks.NewInviteOrgRepo(t)
	orgs.EXPECT().GetByID(mock.Anything, orgID).
		Return(&model.Organization{ID: orgID, Tier: "Pro"}, nil)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().CountPendingByOrg(mock.Anything, orgID).Return(1, nil)
	invites.EXPECT().FindPendingByOrgEmail(mock.Anything, orgID, "pending@example.com").
		Return(&model.Invite{ID: prevInviteID, Status: model.InvitePending}, nil)
	invites.EXPECT().UpdateStatus(mock.Anything, prevInviteID, model.InviteRevoked, now).Return(nil)
	invites.EXPECT().Create(mock.Anything, mock.AnythingOfType("*model.Invite")).
		RunAndReturn(func(_ context.Context, inv *model.Invite) (*model.Invite, error) {
			return inv, nil
		})

	svc := newInviteSvc(t, invites, mems, orgs, now)

	// Act
	inv, _, err := svc.Invite(context.Background(), orgID, inviterID, "pending@example.com", model.RoleMember)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.InvitePending, inv.Status)
}

// TestInvite_UserLimitExceeded покрывает uc_05_02_27.
func TestInvite_UserLimitExceeded(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	inviterID := uuid.New()
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	mems := mocks.NewInviteMembershipRepo(t)
	mems.EXPECT().GetByOrgAndUser(mock.Anything, orgID, inviterID).
		Return(&model.Membership{Role: model.RoleOwner}, nil)
	mems.EXPECT().FindByUserEmail(mock.Anything, orgID, "new@example.com").
		Return((*model.Membership)(nil), nil)
	mems.EXPECT().CountByOrg(mock.Anything, orgID).Return(5, nil)

	orgs := mocks.NewInviteOrgRepo(t)
	orgs.EXPECT().GetByID(mock.Anything, orgID).
		Return(&model.Organization{ID: orgID, Tier: "Starter"}, nil)

	invites := mocks.NewInviteRepo(t)

	svc := newInviteSvc(t, invites, mems, orgs, now)

	// Act
	_, _, err := svc.Invite(context.Background(), orgID, inviterID, "new@example.com", model.RoleMember)

	// Assert
	assert.ErrorIs(t, err, ErrUserLimitExceeded)
}

// TestInvite_InvalidEmail покрывает uc_05_02_04.
func TestInvite_InvalidEmail(t *testing.T) {
	t.Parallel()

	// Arrange
	svc := NewInviteService(
		mocks.NewInviteRepo(t),
		mocks.NewInviteUsers(t),
		mocks.NewInviteMembershipRepo(t),
		mocks.NewInviteOrgRepo(t),
	)

	// Act
	_, _, err := svc.Invite(context.Background(), uuid.New(), uuid.New(), "not-an-email", model.RoleMember)

	// Assert
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

// TestInvite_CannotAssignOwner покрывает uc_05_02_09.
func TestInvite_CannotAssignOwner(t *testing.T) {
	t.Parallel()

	// Arrange
	svc := NewInviteService(
		mocks.NewInviteRepo(t),
		mocks.NewInviteUsers(t),
		mocks.NewInviteMembershipRepo(t),
		mocks.NewInviteOrgRepo(t),
	)

	// Act
	_, _, err := svc.Invite(context.Background(), uuid.New(), uuid.New(), "x@example.com", model.RoleOwner)

	// Assert
	assert.ErrorIs(t, err, ErrCannotAssignOwnerRole)
}

// TestInvite_InsufficientPermissions покрывает uc_05_02_07.
func TestInvite_InsufficientPermissions(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	inviterID := uuid.New()

	mems := mocks.NewInviteMembershipRepo(t)
	mems.EXPECT().GetByOrgAndUser(mock.Anything, orgID, inviterID).
		Return(&model.Membership{Role: model.RoleMember}, nil)

	svc := NewInviteService(
		mocks.NewInviteRepo(t),
		mocks.NewInviteUsers(t),
		mems,
		mocks.NewInviteOrgRepo(t),
	)

	// Act
	_, _, err := svc.Invite(context.Background(), orgID, inviterID, "x@example.com", model.RoleMember)

	// Assert
	assert.ErrorIs(t, err, ErrInsufficientPermissions)
}

// TestAccept_Expired покрывает uc_05_02_12.
func TestAccept_Expired(t *testing.T) {
	t.Parallel()

	// Arrange
	token := uuid.New()
	createdAt := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().GetByToken(mock.Anything, token).Return(&model.Invite{
		ID:        uuid.New(),
		Token:     token,
		Status:    model.InvitePending,
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(InviteTTL),
		Email:     "x@example.com",
	}, nil)
	invites.EXPECT().UpdateStatus(mock.Anything, mock.AnythingOfType("uuid.UUID"), model.InviteExpired, now).
		Return(nil)

	svc := newInviteSvc(t, invites,
		mocks.NewInviteMembershipRepo(t), mocks.NewInviteOrgRepo(t), now)

	// Act
	_, err := svc.Accept(context.Background(), token, "x@example.com", uuid.New())

	// Assert
	assert.ErrorIs(t, err, ErrInviteExpired)
}

// TestAccept_EmailMismatch покрывает uc_05_02_11.
func TestAccept_EmailMismatch(t *testing.T) {
	t.Parallel()

	// Arrange
	token := uuid.New()
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().GetByToken(mock.Anything, token).Return(&model.Invite{
		ID:        uuid.New(),
		Token:     token,
		Status:    model.InvitePending,
		ExpiresAt: now.Add(time.Hour),
		Email:     "invited@example.com",
	}, nil)

	svc := newInviteSvc(t, invites,
		mocks.NewInviteMembershipRepo(t), mocks.NewInviteOrgRepo(t), now)

	// Act
	_, err := svc.Accept(context.Background(), token, "other@example.com", uuid.New())

	// Assert
	assert.ErrorIs(t, err, ErrInviteEmailMismatch)
}

// TestAccept_AlreadyUsed покрывает uc_05_02_13.
func TestAccept_AlreadyUsed(t *testing.T) {
	t.Parallel()

	// Arrange
	token := uuid.New()
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().GetByToken(mock.Anything, token).Return(&model.Invite{
		ID:        uuid.New(),
		Token:     token,
		Status:    model.InviteAccepted,
		ExpiresAt: now.Add(time.Hour),
		Email:     "x@example.com",
	}, nil)

	svc := newInviteSvc(t, invites,
		mocks.NewInviteMembershipRepo(t), mocks.NewInviteOrgRepo(t), now)

	// Act
	_, err := svc.Accept(context.Background(), token, "x@example.com", uuid.New())

	// Assert
	assert.ErrorIs(t, err, ErrInviteAlreadyUsed)
}

// TestAccept_NotFound покрывает uc_05_02_14.
func TestAccept_NotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	token := uuid.UUID{}
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().GetByToken(mock.Anything, token).Return((*model.Invite)(nil), nil)

	svc := newInviteSvc(t, invites,
		mocks.NewInviteMembershipRepo(t), mocks.NewInviteOrgRepo(t), now)

	// Act
	_, err := svc.Accept(context.Background(), token, "x@example.com", uuid.New())

	// Assert
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

// TestInvite_LimitReached покрывает uc_05_02_26.
func TestInvite_LimitReached(t *testing.T) {
	t.Parallel()

	// Arrange
	orgID := uuid.New()
	inviterID := uuid.New()
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	mems := mocks.NewInviteMembershipRepo(t)
	mems.EXPECT().GetByOrgAndUser(mock.Anything, orgID, inviterID).
		Return(&model.Membership{Role: model.RoleOwner}, nil)
	mems.EXPECT().FindByUserEmail(mock.Anything, orgID, "x@example.com").
		Return((*model.Membership)(nil), nil)
	mems.EXPECT().CountByOrg(mock.Anything, orgID).Return(2, nil)

	orgs := mocks.NewInviteOrgRepo(t)
	orgs.EXPECT().GetByID(mock.Anything, orgID).
		Return(&model.Organization{ID: orgID, Tier: "Pro"}, nil)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().CountPendingByOrg(mock.Anything, orgID).Return(MaxPendingInvites, nil)

	svc := newInviteSvc(t, invites, mems, orgs, now)

	// Act
	_, _, err := svc.Invite(context.Background(), orgID, inviterID, "x@example.com", model.RoleMember)

	// Assert
	assert.ErrorIs(t, err, ErrInviteLimitReached)
}

// TestInvite_EmailMatchIsCaseInsensitive проверяет что Accept(token, "X@Example.com")
// и Accept(token, "x@example.com") считаются совпадающими (uc_05_02_11 boundary).
func TestInvite_EmailMatchIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	// Arrange: invite с email "x@example.com" PENDING.
	token := uuid.New()
	inviteID := uuid.New()
	orgID := uuid.New()
	authedUserID := uuid.New()
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	invites := mocks.NewInviteRepo(t)
	invites.EXPECT().GetByToken(mock.Anything, token).Return(&model.Invite{
		ID:             inviteID,
		OrganizationID: orgID,
		Token:          token,
		Status:         model.InvitePending,
		ExpiresAt:      now.Add(time.Hour),
		Email:          "x@example.com",
		Role:           model.RoleMember,
	}, nil)
	invites.EXPECT().UpdateStatus(mock.Anything, inviteID, model.InviteAccepted, now).Return(nil)

	mems := mocks.NewInviteMembershipRepo(t)
	mems.EXPECT().Create(mock.Anything, mock.AnythingOfType("*model.Membership")).
		RunAndReturn(func(_ context.Context, m *model.Membership) (*model.Membership, error) {
			return m, nil
		})

	svc := newInviteSvc(t, invites, mems, mocks.NewInviteOrgRepo(t), now)

	// Act: Accept c authedUserEmail = "X@Example.com".
	membership, err := svc.Accept(context.Background(), token, "X@Example.com", authedUserID)

	// Assert: nil error, membership создан.
	require.NoError(t, err)
	require.NotNil(t, membership)
	assert.Equal(t, orgID, membership.OrganizationID)
	assert.Equal(t, authedUserID, membership.UserID)
	assert.Equal(t, model.RoleMember, membership.Role)
}
