package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// inviteRepo — методы InviteRepository, нужные InviteService.
type inviteRepo interface {
	Create(ctx context.Context, inv *model.Invite) (*model.Invite, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Invite, error)
	GetByToken(ctx context.Context, token uuid.UUID) (*model.Invite, error)
	FindPendingByOrgEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Invite, error)
	CountPendingByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.InviteStatus, at time.Time) error
}

// inviteUsers — методы UserRepository, нужные InviteService.
type inviteUsers interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// inviteMembershipRepo — методы MembershipRepository, нужные InviteService.
type inviteMembershipRepo interface {
	Create(ctx context.Context, m *model.Membership) (*model.Membership, error)
	GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error)
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	FindByUserEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Membership, error)
}

// inviteOrgRepo — методы OrganizationRepository, нужные InviteService.
type inviteOrgRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
}

// invitePublisher — методы Publisher, нужные InviteService.
type invitePublisher interface {
	PublishInviteCreated(ctx context.Context, inv *model.Invite) error
}

// noopInvitePublisher — заглушка invitePublisher для тестов и dev.
type noopInvitePublisher struct{}

func (noopInvitePublisher) PublishInviteCreated(_ context.Context, _ *model.Invite) error {
	return nil
}

// InviteService инкапсулирует жизненный цикл приглашений (uc_05_02_02..15, 26, 27).
type InviteService struct {
	invites     inviteRepo
	users       inviteUsers
	memberships inviteMembershipRepo
	orgs        inviteOrgRepo
	publisher   invitePublisher
	logger      *slog.Logger
	now         func() time.Time
}

// InviteServiceOption настраивает опциональные зависимости.
type InviteServiceOption func(*InviteService)

// WithInvitePublisher подключает publisher к InviteService.
func WithInvitePublisher(p invitePublisher) InviteServiceOption {
	return func(s *InviteService) { s.publisher = p }
}

// WithInviteLogger подключает logger в InviteService.
func WithInviteLogger(l *slog.Logger) InviteServiceOption {
	return func(s *InviteService) { s.logger = l }
}

// WithInviteClock подменяет источник текущего времени (для тестов).
func WithInviteClock(now func() time.Time) InviteServiceOption {
	return func(s *InviteService) { s.now = now }
}

// NewInviteService создаёт InviteService без побочных эффектов.
func NewInviteService(
	invites inviteRepo,
	users inviteUsers,
	memberships inviteMembershipRepo,
	orgs inviteOrgRepo,
	opts ...InviteServiceOption,
) *InviteService {
	s := &InviteService{
		invites:     invites,
		users:       users,
		memberships: memberships,
		orgs:        orgs,
		publisher:   noopInvitePublisher{},
		logger:      slog.Default(),
		now:         func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Invite создаёт новое приглашение в организации orgID.
//
// Бизнес-правила:
//   - inviter должен быть OWNER или ADMIN в orgID (uc_05_02_07, 08)
//   - role не может быть OWNER (uc_05_02_09)
//   - email валидируется через net/mail (uc_05_02_04)
//   - email не должен принадлежать активному участнику (uc_05_02_05)
//   - предыдущее PENDING-приглашение для того же email отзывается (uc_05_02_06)
//   - проверяется лимит PENDING (uc_05_02_26) и лимит участников по тарифу (uc_05_02_27)
//
// orgID извлекается handler'ом из JWT и передаётся явно.
//
// Возвращает (invite, resent, err). resent=true, если перед созданием нового
// приглашения было найдено и отозвано предыдущее PENDING для того же email
// (uc_05_02_06): handler использует флаг для выбора event_type аудита
// (invite_resent vs member_invited).
func (s *InviteService) Invite(
	ctx context.Context,
	orgID, inviterID uuid.UUID,
	email string,
	role model.Role,
) (*model.Invite, bool, error) {
	normEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, false, err
	}
	if !role.IsValid() {
		return nil, false, ErrInsufficientPermissions
	}
	if role == model.RoleOwner {
		return nil, false, ErrCannotAssignOwnerRole
	}

	// RBAC: inviter должен быть OWNER или ADMIN.
	inviterMembership, err := s.memberships.GetByOrgAndUser(ctx, orgID, inviterID)
	if err != nil {
		return nil, false, fmt.Errorf("get inviter membership: %w", err)
	}
	if inviterMembership == nil {
		return nil, false, ErrOrganizationAccessDenied
	}
	if !inviterMembership.Role.CanManageMembers() {
		return nil, false, ErrInsufficientPermissions
	}

	// USER_ALREADY_MEMBER (uc_05_02_05).
	existing, err := s.memberships.FindByUserEmail(ctx, orgID, normEmail)
	if err != nil {
		return nil, false, fmt.Errorf("find existing member: %w", err)
	}
	if existing != nil {
		return nil, false, ErrUserAlreadyMember
	}

	// USER_LIMIT_EXCEEDED (uc_05_02_27).
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, false, fmt.Errorf("get organization: %w", err)
	}
	limit, ok := TierUserLimits[org.Tier]
	if ok {
		current, err := s.memberships.CountByOrg(ctx, orgID)
		if err != nil {
			return nil, false, fmt.Errorf("count members: %w", err)
		}
		if current >= limit {
			return nil, false, ErrUserLimitExceeded
		}
	}

	// INVITE_LIMIT_REACHED (uc_05_02_26).
	pending, err := s.invites.CountPendingByOrg(ctx, orgID)
	if err != nil {
		return nil, false, fmt.Errorf("count pending invites: %w", err)
	}
	if pending >= MaxPendingInvites {
		return nil, false, ErrInviteLimitReached
	}

	// Если уже есть PENDING для этого email — отзываем (uc_05_02_06).
	prev, err := s.invites.FindPendingByOrgEmail(ctx, orgID, normEmail)
	if err != nil {
		return nil, false, fmt.Errorf("find pending invite: %w", err)
	}
	now := s.now()
	if prev != nil {
		if err := s.invites.UpdateStatus(ctx, prev.ID, model.InviteRevoked, now); err != nil {
			return nil, false, fmt.Errorf("revoke previous invite: %w", err)
		}
	}

	inv := &model.Invite{
		ID:             uuid.New(),
		OrganizationID: orgID,
		InviterID:      inviterID,
		Email:          normEmail,
		Role:           role,
		Token:          uuid.New(),
		Status:         model.InvitePending,
		CreatedAt:      now,
		ExpiresAt:      now.Add(InviteTTL),
	}
	created, err := s.invites.Create(ctx, inv)
	if err != nil {
		return nil, false, fmt.Errorf("create invite: %w", err)
	}
	if err := s.publisher.PublishInviteCreated(ctx, created); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish invite.created", "error", err)
	}
	return created, prev != nil, nil
}

// Revoke отзывает приглашение (uc_05_02_15).
// actor должен иметь право invite_member в этой организации.
func (s *InviteService) Revoke(ctx context.Context, actorID, inviteID uuid.UUID) error {
	inv, err := s.invites.GetByID(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("get invite: %w", err)
	}
	if inv == nil {
		return ErrInviteNotFound
	}
	actor, err := s.memberships.GetByOrgAndUser(ctx, inv.OrganizationID, actorID)
	if err != nil {
		return fmt.Errorf("get actor membership: %w", err)
	}
	if actor == nil {
		return ErrOrganizationAccessDenied
	}
	if !actor.Role.CanManageMembers() {
		return ErrInsufficientPermissions
	}
	if inv.Status != model.InvitePending {
		return ErrInviteAlreadyUsed
	}
	return s.invites.UpdateStatus(ctx, inv.ID, model.InviteRevoked, s.now())
}

// Accept принимает приглашение и создаёт membership (uc_05_02_10..14).
func (s *InviteService) Accept(
	ctx context.Context,
	token uuid.UUID,
	authedUserEmail string,
	authedUserID uuid.UUID,
) (*model.Membership, error) {
	inv, err := s.invites.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get invite by token: %w", err)
	}
	if inv == nil {
		return nil, ErrInviteNotFound
	}

	now := s.now()
	// INVITE_EXPIRED (uc_05_02_12).
	if inv.IsExpired(now) {
		// best-effort переход в EXPIRED, ошибку лога игнорируем.
		if inv.Status == model.InvitePending {
			if err := s.invites.UpdateStatus(ctx, inv.ID, model.InviteExpired, now); err != nil {
				s.logger.WarnContext(ctx, "Failed to mark invite expired",
					"invite_id", inv.ID, "error", err)
			}
		}
		return nil, ErrInviteExpired
	}
	// INVITE_ALREADY_USED (uc_05_02_13).
	if inv.Status != model.InvitePending {
		return nil, ErrInviteAlreadyUsed
	}
	// INVITE_EMAIL_MISMATCH (uc_05_02_11).
	if !strings.EqualFold(strings.TrimSpace(authedUserEmail), inv.Email) {
		return nil, ErrInviteEmailMismatch
	}

	if err := s.invites.UpdateStatus(ctx, inv.ID, model.InviteAccepted, now); err != nil {
		return nil, fmt.Errorf("accept invite: %w", err)
	}
	m := &model.Membership{
		ID:             uuid.New(),
		OrganizationID: inv.OrganizationID,
		UserID:         authedUserID,
		Role:           inv.Role,
	}
	created, err := s.memberships.Create(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}
	return created, nil
}

// normalizeEmail валидирует email и приводит к каноническому виду.
func normalizeEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", ErrInvalidEmail
	}
	// mail.ParseAddress допускает форматы вида "Name <a@b>"; для invite нужен
	// чистый адрес без display name.
	if addr.Name != "" {
		return "", ErrInvalidEmail
	}
	return strings.ToLower(addr.Address), nil
}
