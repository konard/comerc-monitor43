package service

import "errors"

// Sentinel errors для team management operations.
// Сообщения на английском в нижнем регистре согласно соглашениям проекта.
var (
	// ErrInsufficientPermissions возвращается когда у actor нет необходимой роли.
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	// ErrCannotModifyOwner возвращается когда не-OWNER пытается изменить OWNER'а.
	ErrCannotModifyOwner = errors.New("cannot modify owner")
	// ErrCannotRemoveOwner возвращается при попытке удалить OWNER из организации.
	ErrCannotRemoveOwner = errors.New("cannot remove owner")
	// ErrOwnerCannotLeave возвращается когда OWNER пытается покинуть организацию,
	// в которой больше одного участника.
	ErrOwnerCannotLeave = errors.New("owner cannot leave organization without transferring ownership")
	// ErrUserAlreadyMember возвращается при приглашении уже активного участника.
	ErrUserAlreadyMember = errors.New("user already member")
	// ErrCannotAssignOwnerRole возвращается при попытке выдать роль OWNER через invite.
	ErrCannotAssignOwnerRole = errors.New("cannot assign owner role")
	// ErrUserLimitExceeded возвращается при превышении лимита пользователей по тарифу.
	ErrUserLimitExceeded = errors.New("user limit exceeded")
	// ErrOrganizationAccessDenied возвращается при попытке доступа к чужой организации.
	ErrOrganizationAccessDenied = errors.New("organization access denied")

	// ErrInvalidEmail возвращается при невалидном email в invite.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrInviteNotFound возвращается когда invite по token не найден.
	ErrInviteNotFound = errors.New("invite not found")
	// ErrInviteExpired возвращается когда срок действия invite истёк.
	ErrInviteExpired = errors.New("invite expired")
	// ErrInviteAlreadyUsed возвращается при повторном использовании invite.
	ErrInviteAlreadyUsed = errors.New("invite already used")
	// ErrInviteEmailMismatch возвращается когда email авторизованного пользователя
	// не совпадает с email приглашения.
	ErrInviteEmailMismatch = errors.New("invite email mismatch")
	// ErrInviteLimitReached возвращается при превышении лимита PENDING приглашений.
	ErrInviteLimitReached = errors.New("invite limit reached")
)
