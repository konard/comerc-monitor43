package handler

import (
	"context"

	"github.com/google/uuid"
)

// authContextKey — приватный тип ключа для значений auth-контекста.
// Используется, чтобы избежать коллизий ключей между пакетами.
type authContextKey int

const (
	ctxKeyUserID authContextKey = iota
	ctxKeyOrgID
	ctxKeyRole
)

// WithUserID возвращает контекст с проставленным идентификатором пользователя.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// UserIDFromContext извлекает идентификатор пользователя из контекста.
// Возвращает (uuid.Nil, false), если значение не установлено.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(uuid.UUID)
	return v, ok
}

// WithOrgID возвращает контекст с проставленным идентификатором организации.
func WithOrgID(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyOrgID, orgID)
}

// OrgIDFromContext извлекает идентификатор организации из контекста.
func OrgIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyOrgID).(uuid.UUID)
	return v, ok
}

// WithRole возвращает контекст с проставленной ролью пользователя.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxKeyRole, role)
}

// RoleFromContext извлекает роль пользователя из контекста.
func RoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyRole).(string)
	return v, ok
}
