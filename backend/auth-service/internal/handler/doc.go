// Package handler реализует gRPC-хэндлеры сервиса аутентификации.
//
// Включает AuthHandler (OAuth/password login, register, refresh, logout,
// validate), SessionHandler (list/revoke sessions), UserHandler (профиль и
// смена пароля), TeamHandler (управление участниками организации) и
// InviteHandler (жизненный цикл приглашений). Общий helper auth_context
// кладёт userID/orgID/role в context.Context, team_converter содержит
// mapping'и между доменной моделью и protobuf-типами team/invite.
package handler
