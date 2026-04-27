package model

import (
	"time"

	"github.com/google/uuid"
)

// Role представляет роль участника организации в модели RBAC.
// Значения соответствуют enum membership_role из миграции 000005 и
// определяют доступные операции над ресурсами организации.
type Role string

// Перечисление ролей соответствует enum membership_role в миграции 000005.
const (
	RoleOwner  Role = "OWNER"
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
	RoleViewer Role = "VIEWER"
)

// IsValid проверяет, что роль входит в допустимый набор OWNER/ADMIN/MEMBER/VIEWER.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	}
	return false
}

// CanManageMembers возвращает true для ролей, которым разрешено управлять
// составом организации (приглашать, удалять, менять роли) — это OWNER и ADMIN.
func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

// InviteStatus представляет жизненный цикл приглашения от создания до
// финального состояния (ACCEPTED/DECLINED/EXPIRED/REVOKED).
type InviteStatus string

// Перечисление статусов соответствует enum invite_status в миграции 000005.
const (
	InvitePending  InviteStatus = "PENDING"
	InviteAccepted InviteStatus = "ACCEPTED"
	InviteDeclined InviteStatus = "DECLINED"
	InviteExpired  InviteStatus = "EXPIRED"
	InviteRevoked  InviteStatus = "REVOKED"
)

// Organization представляет организацию — владельца ресурсов и биллинговую
// единицу. Владелец фиксируется в поле OwnerID и может быть изменён через
// transfer ownership; Tier задаёт уровень подписки (Free/Starter/Pro/Enterprise).
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Tier      string    `json:"tier"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Membership связывает пользователя с организацией и ролью внутри неё.
// Один пользователь может состоять только в одной организации (на текущий
// момент — без поддержки multi-org); уникальность гарантируется индексом
// idx_memberships_user_id.
type Membership struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	UserID         uuid.UUID  `json:"user_id"`
	Role           Role       `json:"role"`
	JoinedAt       time.Time  `json:"joined_at"`
	LastActiveAt   *time.Time `json:"last_active_at,omitempty"`
}

// Invite представляет приглашение нового участника в организацию.
// Содержит одноразовый Token для акцепта, срок действия ExpiresAt и
// текущий Status из перечисления InviteStatus. Таймстемпы AcceptedAt и
// RevokedAt заполняются при переходе в соответствующий финальный статус.
type Invite struct {
	ID             uuid.UUID    `json:"id"`
	OrganizationID uuid.UUID    `json:"organization_id"`
	InviterID      uuid.UUID    `json:"inviter_id"`
	Email          string       `json:"email"`
	Role           Role         `json:"role"`
	Token          uuid.UUID    `json:"token"`
	Status         InviteStatus `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
	ExpiresAt      time.Time    `json:"expires_at"`
	AcceptedAt     *time.Time   `json:"accepted_at,omitempty"`
	RevokedAt      *time.Time   `json:"revoked_at,omitempty"`
}

// IsExpired возвращает true, если срок действия приглашения истёк
// относительно переданного момента времени now (сравнение по ExpiresAt).
func (i *Invite) IsExpired(now time.Time) bool {
	return !now.Before(i.ExpiresAt)
}
