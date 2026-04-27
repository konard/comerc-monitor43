package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims for auth tokens
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Tier   string    `json:"tier"`
	// OrgID — идентификатор активной организации пользователя; пусто, пока
	// фича team management не активна. См. epic=05_security, us=02_team_management.
	OrgID uuid.UUID `json:"org_id,omitempty"`
	// Role — роль пользователя в активной организации; пусто, пока фича
	// team management не активна.
	Role string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// WithTeamContext возвращает копию claims с проставленным контекстом
// организации и роли. Используется на этапе green-реализации
// us=02_team_management — на этапе red вызывать не нужно.
func (c *Claims) WithTeamContext(orgID uuid.UUID, role string) *Claims {
	cp := *c
	cp.OrgID = orgID
	cp.Role = role
	return &cp
}

// NewClaims creates new JWT claims for a user
func NewClaims(userID uuid.UUID, email, tier string, expiry time.Duration) *Claims {
	now := time.Now()
	return &Claims{
		UserID: userID,
		Email:  email,
		Tier:   tier,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
}
