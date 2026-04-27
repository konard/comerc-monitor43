package service

import "time"

// TierUserLimits задаёт лимит активных участников организации по подписке.
//
// ВРЕМЕННОЕ локальное решение: billing-сервис ещё не знает про организации
// и не публикует лимиты подписки. До интеграции с billing используется
// этот hardcoded маппинг (см. AGENTS.md, epic 04_billing). После появления
// контракта billing.subscription.updated лимит должен прийти оттуда.
var TierUserLimits = map[string]int{
	"Free":       1,
	"Starter":    5,
	"Pro":        20,
	"Enterprise": 100,
}

// InviteTTL — срок действия invite-токена (uc_05_02_12).
const InviteTTL = 7 * 24 * time.Hour

// MaxPendingInvites — лимит PENDING приглашений в одной организации (uc_05_02_26).
const MaxPendingInvites = 50
