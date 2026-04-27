//go:build bdd

package suite

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// monitorJWTSecret — общий с monitor-service секрет, используемый в BDD-окружении.
const monitorJWTSecret = alertJWTSecret

// monitorAuthTokenFor возвращает HS256 JWT токен для monitor-service.
// Monitor-service ожидает claims user_id и tier (в отличие от alert-service, где sub).
func monitorAuthTokenFor(userID uuid.UUID, tier string) string {
	if tier == "" {
		tier = "Free"
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().Unix()
	claims := map[string]any{
		"user_id": userID.String(),
		"tier":    tier,
		"role":    "USER",
		"iat":     now,
		"exp":     now + 3600,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(monitorJWTSecret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

// monitorAuthCtx добавляет Authorization metadata для вызовов monitor-service.
func monitorAuthCtx(ctx context.Context, userID uuid.UUID, tier string) context.Context {
	if userID == uuid.Nil {
		return ctx
	}
	md := metadata.Pairs(
		"authorization", "Bearer "+monitorAuthTokenFor(userID, tier),
	)
	return metadata.NewOutgoingContext(ctx, md)
}
