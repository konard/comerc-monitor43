// Package service содержит бизнес-логику сервиса аутентификации.
//
// Включает AuthService (OAuth/password/register flow), TokenService
// (issue/refresh/validate JWT), SessionService (управление сессиями
// пользователя), UserService (профиль), TeamService (участники организации,
// transfer ownership, leave) и InviteService (создание, отзыв, приём
// приглашений).
//
// # Конфигурация
//
// Лимиты team management заданы в team_limits.go:
//   - TierUserLimits — hardcoded маппинг tier → лимит активных участников
//     (TODO: интеграция с billing-service через billing.subscription.updated);
//   - InviteTTL — срок действия invite-токена (uc_05_02_12);
//   - MaxPendingInvites — лимит PENDING приглашений на организацию
//     (uc_05_02_26).
package service
