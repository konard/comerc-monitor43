package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// Локальные частично применяемые интерфейсы (см. x-unit-test-partial-interface).

// teamServiceUsers — методы UserRepository, нужные TeamService.
type teamServiceUsers interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

// orgRepo — методы OrganizationRepository, нужные TeamService.
type orgRepo interface {
	Create(ctx context.Context, org *model.Organization) (*model.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	UpdateOwner(ctx context.Context, orgID, newOwnerID uuid.UUID) error
}

// membershipRepo — методы MembershipRepository, нужные TeamService.
type membershipRepo interface {
	Create(ctx context.Context, m *model.Membership) (*model.Membership, error)
	GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*model.Membership, int, error)
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role model.Role) error
	Delete(ctx context.Context, orgID, userID uuid.UUID) error
	FindByUserEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Membership, error)
}

// sessionRevoker отзывает все активные сессии пользователя (uc_05_02_28).
type sessionRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// teamPublisher — методы Publisher, нужные TeamService.
type teamPublisher interface {
	PublishMemberRoleChanged(ctx context.Context, orgID, userID uuid.UUID, oldRole, newRole model.Role) error
	PublishMemberRemoved(ctx context.Context, orgID, userID uuid.UUID) error
}

// noopTeamPublisher — заглушка teamPublisher для тестов и dev-режима.
type noopTeamPublisher struct{}

func (noopTeamPublisher) PublishMemberRoleChanged(_ context.Context, _, _ uuid.UUID, _, _ model.Role) error {
	return nil
}

func (noopTeamPublisher) PublishMemberRemoved(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// noopSessionRevoker — заглушка sessionRevoker.
type noopSessionRevoker struct{}

func (noopSessionRevoker) RevokeAllForUser(_ context.Context, _ uuid.UUID) error { return nil }

// Operation описывает операцию для матрицы доступа CheckPermission (uc_05_02_25).
type Operation string

// Поддерживаемые операции RBAC матрицы.
const (
	OpInviteMember      Operation = "invite_member"
	OpRemoveMember      Operation = "remove_member"
	OpChangeRole        Operation = "change_role"
	OpTransferOwnership Operation = "transfer_ownership"
	OpCreateMonitor     Operation = "create_monitor"
	OpListMembers       Operation = "list_members"
)

// TeamService инкапсулирует бизнес-логику управления организацией и
// её участниками. Конструктор не делает I/O.
type TeamService struct {
	users       teamServiceUsers
	orgs        orgRepo
	memberships membershipRepo
	revoker     sessionRevoker
	publisher   teamPublisher
	logger      *slog.Logger
}

// TeamServiceOption настраивает опциональные зависимости TeamService.
type TeamServiceOption func(*TeamService)

// WithTeamPublisher подключает publisher событий team.*.
func WithTeamPublisher(p teamPublisher) TeamServiceOption {
	return func(s *TeamService) { s.publisher = p }
}

// WithSessionRevoker подключает компонент отзыва сессий.
func WithSessionRevoker(r sessionRevoker) TeamServiceOption {
	return func(s *TeamService) { s.revoker = r }
}

// WithTeamLogger подключает logger в TeamService.
func WithTeamLogger(l *slog.Logger) TeamServiceOption {
	return func(s *TeamService) { s.logger = l }
}

// NewTeamService создаёт TeamService без побочных эффектов.
func NewTeamService(
	users teamServiceUsers,
	orgs orgRepo,
	memberships membershipRepo,
	opts ...TeamServiceOption,
) *TeamService {
	s := &TeamService{
		users:       users,
		orgs:        orgs,
		memberships: memberships,
		revoker:     noopSessionRevoker{},
		publisher:   noopTeamPublisher{},
		logger:      slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// EnsureOrganization идемпотентно создаёт организацию для пользователя
// при первом OAuth-логине. Если у пользователя уже есть OWNER-membership,
// возвращает существующую организацию (uc_05_02_01). Второе значение —
// флаг created: true, если организация создана текущим вызовом, false
// если уже существовала; используется вызывающим кодом для записи
// audit-события organization_created только один раз.
func (s *TeamService) EnsureOrganization(ctx context.Context, ownerUserID uuid.UUID) (*model.Organization, bool, error) {
	user, err := s.users.GetByID(ctx, ownerUserID)
	if err != nil {
		return nil, false, fmt.Errorf("get user: %w", err)
	}

	name := user.FullName
	if name == "" {
		name = user.Email
	}
	org := &model.Organization{
		ID:      uuid.New(),
		Name:    name,
		OwnerID: ownerUserID,
		Tier:    user.Tier,
	}
	created, err := s.orgs.Create(ctx, org)
	if err != nil {
		return nil, false, fmt.Errorf("create organization: %w", err)
	}

	m := &model.Membership{
		ID:             uuid.New(),
		OrganizationID: created.ID,
		UserID:         ownerUserID,
		Role:           model.RoleOwner,
	}
	if _, err := s.memberships.Create(ctx, m); err != nil {
		return nil, false, fmt.Errorf("create owner membership: %w", err)
	}
	return created, true, nil
}

// GetMembership возвращает membership пользователя в указанной организации.
// Используется при выпуске JWT после EnsureOrganization, чтобы прокинуть
// роль в claims. Возвращает nil без ошибки, если membership не найден.
func (s *TeamService) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error) {
	m, err := s.memberships.GetByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("get membership: %w", err)
	}
	return m, nil
}

// ListMembers возвращает список участников с пагинацией (uc_05_02_16).
// Право чтения списка участников предоставлено всем ролям, поэтому
// дополнительный CheckPermission здесь не нужен — фильтрация по orgID
// должна выполняться вызывающим (handler) на основании JWT.
func (s *TeamService) ListMembers(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*model.Membership, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return s.memberships.ListByOrg(ctx, orgID, page, pageSize)
}

// resolveActorAndTarget возвращает membership actor'а и target'а в той же
// организации. Возвращает ErrOrganizationAccessDenied, если actor или target
// не состоят в одной организации.
func (s *TeamService) resolveActorAndTargetByOrg(
	ctx context.Context,
	orgID, actorID, targetID uuid.UUID,
) (*model.Membership, *model.Membership, error) {
	actor, err := s.memberships.GetByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return nil, nil, fmt.Errorf("get actor membership: %w", err)
	}
	if actor == nil {
		return nil, nil, ErrOrganizationAccessDenied
	}
	target, err := s.memberships.GetByOrgAndUser(ctx, orgID, targetID)
	if err != nil {
		return nil, nil, fmt.Errorf("get target membership: %w", err)
	}
	if target == nil {
		return nil, nil, ErrOrganizationAccessDenied
	}
	return actor, target, nil
}

// ChangeMemberRole меняет роль участника (uc_05_02_17, 18, 19).
// Требует, чтобы actor и target принадлежали одной организации.
func (s *TeamService) ChangeMemberRole(
	ctx context.Context,
	orgID, actorUserID, targetUserID uuid.UUID,
	newRole model.Role,
) error {
	if !newRole.IsValid() {
		return ErrInsufficientPermissions
	}
	if newRole == model.RoleOwner {
		return ErrCannotAssignOwnerRole
	}

	actor, target, err := s.resolveActorAndTargetByOrg(ctx, orgID, actorUserID, targetUserID)
	if err != nil {
		return err
	}
	if !actor.Role.CanManageMembers() {
		return ErrInsufficientPermissions
	}
	// ADMIN не может modify OWNER (uc_05_02_18).
	if target.Role == model.RoleOwner {
		return ErrCannotModifyOwner
	}
	oldRole := target.Role
	if oldRole == newRole {
		return nil
	}
	if err := s.memberships.UpdateRole(ctx, orgID, targetUserID, newRole); err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if err := s.publisher.PublishMemberRoleChanged(ctx, orgID, targetUserID, oldRole, newRole); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish member.role_changed", "error", err)
	}
	return nil
}

// RemoveMember удаляет участника из организации (uc_05_02_20, 21, 28).
// После удаления отзывает все активные сессии пользователя.
func (s *TeamService) RemoveMember(
	ctx context.Context,
	orgID, actorUserID, targetUserID uuid.UUID,
) error {
	actor, target, err := s.resolveActorAndTargetByOrg(ctx, orgID, actorUserID, targetUserID)
	if err != nil {
		return err
	}
	if !actor.Role.CanManageMembers() {
		return ErrInsufficientPermissions
	}
	// OWNER не может быть удалён (uc_05_02_21).
	if target.Role == model.RoleOwner {
		return ErrCannotRemoveOwner
	}
	if err := s.memberships.Delete(ctx, orgID, targetUserID); err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	if err := s.revoker.RevokeAllForUser(ctx, targetUserID); err != nil {
		s.logger.WarnContext(ctx, "Failed to revoke sessions for removed member",
			"user_id", targetUserID, "error", err)
	}
	if err := s.publisher.PublishMemberRemoved(ctx, orgID, targetUserID); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish member.removed", "error", err)
	}
	return nil
}

// TransferOwnership передаёт роль OWNER другому участнику (uc_05_02_22).
// Атомарно: actor становится ADMIN, target становится OWNER, owner_id орг
// обновляется. Атомарность обеспечивается порядком вызовов repo методов;
// при необходимости полной транзакции — обернуть в Tx через repo.
func (s *TeamService) TransferOwnership(
	ctx context.Context,
	orgID, actorUserID, targetUserID uuid.UUID,
) error {
	actor, target, err := s.resolveActorAndTargetByOrg(ctx, orgID, actorUserID, targetUserID)
	if err != nil {
		return err
	}
	if actor.Role != model.RoleOwner {
		return ErrInsufficientPermissions
	}
	if actorUserID == targetUserID {
		return nil
	}
	// Понижаем actor сначала, затем повышаем target — иначе нарушится
	// уникальный partial index (один OWNER на организацию).
	if err := s.memberships.UpdateRole(ctx, orgID, actorUserID, model.RoleAdmin); err != nil {
		return fmt.Errorf("demote owner: %w", err)
	}
	if err := s.memberships.UpdateRole(ctx, orgID, targetUserID, model.RoleOwner); err != nil {
		return fmt.Errorf("promote new owner: %w", err)
	}
	if err := s.orgs.UpdateOwner(ctx, orgID, targetUserID); err != nil {
		return fmt.Errorf("update organization owner: %w", err)
	}
	_ = target
	return nil
}

// LeaveOrganization удаляет actor'а из его организации (uc_05_02_23, 24).
// OWNER не может leave если в орге больше одного участника.
func (s *TeamService) LeaveOrganization(ctx context.Context, orgID, actorUserID uuid.UUID) error {
	actor, err := s.memberships.GetByOrgAndUser(ctx, orgID, actorUserID)
	if err != nil {
		return fmt.Errorf("get actor membership: %w", err)
	}
	if actor == nil {
		return ErrOrganizationAccessDenied
	}
	if actor.Role == model.RoleOwner {
		count, err := s.memberships.CountByOrg(ctx, orgID)
		if err != nil {
			return fmt.Errorf("count members: %w", err)
		}
		if count > 1 {
			return ErrOwnerCannotLeave
		}
	}
	if err := s.memberships.Delete(ctx, orgID, actorUserID); err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	if err := s.revoker.RevokeAllForUser(ctx, actorUserID); err != nil {
		s.logger.WarnContext(ctx, "Failed to revoke sessions on leave", "error", err)
	}
	return nil
}

// CheckPermission реализует RBAC матрицу uc_05_02_25.
// Возвращает (allowed, role, error). role возвращается даже если allowed=false,
// чтобы вызывающий мог записать actual_role в audit log.
func (s *TeamService) CheckPermission(
	ctx context.Context,
	orgID, userID uuid.UUID,
	op Operation,
) (bool, model.Role, error) {
	m, err := s.memberships.GetByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		return false, "", fmt.Errorf("get membership: %w", err)
	}
	if m == nil {
		return false, "", ErrOrganizationAccessDenied
	}
	return permissionFor(m.Role, op), m.Role, nil
}

// permissionFor — чистая функция матрицы доступа.
func permissionFor(role model.Role, op Operation) bool {
	switch op {
	case OpInviteMember, OpRemoveMember, OpChangeRole:
		return role == model.RoleOwner || role == model.RoleAdmin
	case OpTransferOwnership:
		return role == model.RoleOwner
	case OpCreateMonitor:
		return role == model.RoleOwner || role == model.RoleAdmin || role == model.RoleMember
	case OpListMembers:
		return role.IsValid()
	default:
		return false
	}
}
