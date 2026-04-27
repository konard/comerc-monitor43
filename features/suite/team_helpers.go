//go:build bdd

package suite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	authv1 "github.com/raul/monitor/api/proto"
)

// authJWTSecret — общий с auth-service секрет, используемый в BDD-окружении.
const authJWTSecret = "test-secret-for-bdd-tests-minimum32chars"

// teamClaims повторяет структуру auth-service jwt.Claims для прямой генерации
// токенов в BDD-сценариях, требующих ручного контроля над OrgID/Role.
type teamClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Tier   string    `json:"tier"`
	OrgID  uuid.UUID `json:"org_id,omitempty"`
	Role   string    `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// mintTeamToken создаёт HS256 JWT с заданными user/org/role claims.
func mintTeamToken(userID uuid.UUID, email, role string, orgID uuid.UUID) (string, error) {
	now := time.Now()
	claims := teamClaims{
		UserID: userID,
		Email:  email,
		Tier:   "Free",
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(authJWTSecret))
}

// parseTeamToken разбирает JWT и возвращает teamClaims без проверки подписи.
// Используется для извлечения OrgID/UserID из выданного auth-service токена.
func parseTeamToken(token string) (*teamClaims, error) {
	c := &teamClaims{}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	if _, _, err := parser.ParseUnverified(token, c); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return c, nil
}

// withAuth добавляет authorization Bearer header в outgoing gRPC контекст.
func withAuth(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

// uniqueEmail генерирует уникальный email для регистрации тестового юзера.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@team.bdd", prefix, uuid.New().String()[:8])
}

// registerUser регистрирует пользователя с уникальным email и паролем,
// затем выполняет PasswordLogin, чтобы получить access/refresh токены
// (Register сам по себе токены не выпускает — см. auth_service.Register).
func registerUser(ctx context.Context, client authv1.AuthServiceClient, email, fullName string) (*authv1.AuthResponse, error) {
	const password = "ValidPass123!"
	if _, err := client.Register(ctx, &authv1.RegisterRequest{
		Email:    email,
		Password: password,
		FullName: fullName,
	}); err != nil {
		return nil, fmt.Errorf("register %s: %w", email, err)
	}
	resp, err := client.PasswordLogin(ctx, &authv1.PasswordLoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("login %s: %w", email, err)
	}
	return resp, nil
}

// ownerContext регистрирует нового owner-пользователя и возвращает его
// AuthResponse + распарсенные claims (UserID + OrgID).
type ownerContext struct {
	Auth   *authv1.AuthResponse
	UserID uuid.UUID
	OrgID  uuid.UUID
	Email  string
}

// registerOwner регистрирует уникального OWNER пользователя.
// По умолчанию апгрейдит организацию до тарифа Pro, чтобы тестовые сценарии
// могли свободно создавать приглашения и членов (Free-лимит = 1 user).
func registerOwner(ctx context.Context, client authv1.AuthServiceClient) (*ownerContext, error) {
	email := uniqueEmail("owner")
	resp, err := registerUser(ctx, client, email, "BDD Owner")
	if err != nil {
		return nil, err
	}
	c, err := parseTeamToken(resp.AccessToken)
	if err != nil {
		return nil, err
	}
	return &ownerContext{Auth: resp, UserID: c.UserID, OrgID: c.OrgID, Email: email}, nil
}

// registerOwnerWithTier как registerOwner, но позволяет указать тариф организации.
// Для большинства BDD-сценариев следует использовать "Pro", чтобы не упираться
// в лимит Free=1 при первом invite.
func registerOwnerWithTier(ctx context.Context, client authv1.AuthServiceClient, stack *Stack, tier string) (*ownerContext, error) {
	o, err := registerOwner(ctx, client)
	if err != nil {
		return nil, err
	}
	if err := updateOrgTier(stack, o.OrgID, tier); err != nil {
		return nil, err
	}
	return o, nil
}

// forceInviteCreatedAt прямым SQL переписывает created_at и expires_at
// существующего invite. Используется для timing-сценариев (uc_05_02_12).
func forceInviteCreatedAt(stack *Stack, inviteID uuid.UUID, createdAt time.Time) error {
	expires := createdAt.Add(7 * 24 * time.Hour)
	_, err := stack.AuthDB().Exec(
		`UPDATE invites SET created_at=$1, expires_at=$2 WHERE id=$3`,
		createdAt, expires, inviteID,
	)
	if err != nil {
		return fmt.Errorf("force invite created_at: %w", err)
	}
	return nil
}

// insertMembership напрямую создаёт запись membership для тестов.
func insertMembership(stack *Stack, orgID, userID uuid.UUID, role string) error {
	res, err := stack.AuthDB().Exec(
		`INSERT INTO memberships (id, organization_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, $4::membership_role, NOW())
		 ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		uuid.New(), orgID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("insert membership (org=%s user=%s role=%s): %w", orgID, userID, role, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("insert membership: 0 rows affected (org=%s user=%s role=%s)", orgID, userID, role)
	}
	return nil
}

// updateOrgTier ставит подписку организации (Starter/Free/...) для лимитов.
func updateOrgTier(stack *Stack, orgID uuid.UUID, tier string) error {
	_, err := stack.AuthDB().Exec(
		`UPDATE organizations SET tier=$1 WHERE id=$2`,
		tier, orgID,
	)
	if err != nil {
		return fmt.Errorf("update tier: %w", err)
	}
	return nil
}

// seedDummyUser создаёт фиктивного пользователя в users + опционально membership
// в указанной организации с ролью.
func seedDummyUser(stack *Stack, email, role string, orgID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	hash := "$2a$10$bdd.placeholder.hash................................"
	if !strings.Contains(email, "@") {
		email = email + "@bdd.local"
	}
	_, err := stack.AuthDB().Exec(
		`INSERT INTO users (id, email, password_hash, full_name, tier, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Seed User', 'Free', NOW(), NOW())`,
		id, email, hash,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("seed user: %w", err)
	}
	if orgID != uuid.Nil {
		if err := insertMembership(stack, orgID, id, role); err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil
}

// fetchLastAuditLog читает последнюю запись auth_audit_log для (userID, eventType).
// Возвращает (nil, nil), если записи нет.
func fetchLastAuditLog(stack *Stack, userID uuid.UUID, eventType string) (*AuditLogRow, error) {
	row := &AuditLogRow{}
	var uid uuid.UUID
	err := stack.AuthDB().QueryRow(
		`SELECT id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid),
		        event_type, success,
		        COALESCE(ip_address::text, ''), COALESCE(user_agent, ''),
		        created_at
		   FROM auth_audit_log
		  WHERE event_type = $1 AND user_id = $2
		  ORDER BY created_at DESC LIMIT 1`,
		eventType, userID,
	).Scan(&row.ID, &uid, &row.EventType, &row.Success, &row.IPAddress, &row.UserAgent, &row.CreatedAt)
	if err != nil {
		// sql.ErrNoRows → возвращаем nil без ошибки.
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	row.UserID = uid
	return row, nil
}

// seedPendingInvite создаёт фиктивный PENDING invite без вызова API.
func seedPendingInvite(stack *Stack, orgID, inviterID uuid.UUID, email string) error {
	_, err := stack.AuthDB().Exec(
		`INSERT INTO invites (id, organization_id, inviter_id, email, role, token, status, expires_at)
		 VALUES ($1, $2, $3, $4, 'MEMBER', $5, 'PENDING', NOW() + INTERVAL '7 days')`,
		uuid.New(), orgID, inviterID, email, uuid.New(),
	)
	if err != nil {
		return fmt.Errorf("seed pending invite: %w", err)
	}
	return nil
}
