package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// OrganizationRepository управляет жизненным циклом организаций.
type OrganizationRepository interface {
	// Create создаёт новую организацию.
	Create(ctx context.Context, org *model.Organization) (*model.Organization, error)
	// GetByID возвращает организацию по идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	// UpdateOwner меняет owner_id организации (используется при transfer ownership).
	UpdateOwner(ctx context.Context, orgID, newOwnerID uuid.UUID) error
}

// MembershipRepository управляет связями пользователь-организация-роль.
type MembershipRepository interface {
	// Create добавляет участника в организацию.
	Create(ctx context.Context, m *model.Membership) (*model.Membership, error)
	// GetByOrgAndUser возвращает membership по паре (organization_id, user_id).
	GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error)
	// ListByOrg возвращает участников организации с пагинацией.
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*model.Membership, int, error)
	// CountByOrg возвращает число активных участников организации.
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	// UpdateRole меняет роль участника.
	UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role model.Role) error
	// Delete удаляет участника из организации.
	Delete(ctx context.Context, orgID, userID uuid.UUID) error
	// FindByUserEmail возвращает membership по (organization_id, email пользователя).
	// Используется для проверки USER_ALREADY_MEMBER.
	FindByUserEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Membership, error)
}

// InviteRepository управляет приглашениями.
type InviteRepository interface {
	// Create создаёт новое приглашение со статусом PENDING.
	Create(ctx context.Context, inv *model.Invite) (*model.Invite, error)
	// GetByID возвращает приглашение по идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Invite, error)
	// GetByToken возвращает приглашение по token.
	GetByToken(ctx context.Context, token uuid.UUID) (*model.Invite, error)
	// FindPendingByOrgEmail возвращает PENDING приглашение для пары (org, email),
	// если оно есть. Возвращает nil без ошибки, если не найдено.
	FindPendingByOrgEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Invite, error)
	// CountPendingByOrg возвращает число PENDING приглашений в организации.
	CountPendingByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	// UpdateStatus меняет статус приглашения и проставляет соответствующий timestamp:
	// ACCEPTED -> accepted_at, REVOKED/EXPIRED/DECLINED -> revoked_at.
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.InviteStatus, at time.Time) error
}
