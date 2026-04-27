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

// billingJWTSecret — общий с billing-service секрет, используемый в BDD-окружении.
// Должен совпадать с JWT_SECRET, который получает subprocess billing-service.
const billingJWTSecret = "test-secret-for-bdd-tests-minimum32chars"

// bddYookassaWebhookSecret — общий секрет подписи webhook для billing BDD.
const bddYookassaWebhookSecret = "bdd-yookassa-webhook-secret"

// billingAuthTokenFor возвращает HS256 JWT токен, совместимый с billing JWT interceptor.
// billing-service ожидает в claims поле `user_id` (в отличие от alert-service, где `sub`).
func billingAuthTokenFor(userID uuid.UUID) string {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().Unix()
	claims := map[string]any{
		"user_id": userID.String(),
		"email":   "user@bdd.test",
		"tier":    "TIER_FREE",
		"iat":     now,
		"exp":     now + 3600,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(billingJWTSecret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

// billingAuthCtx добавляет Authorization: Bearer <token> в исходящий gRPC контекст.
func billingAuthCtx(ctx context.Context, userID uuid.UUID) context.Context {
	if userID == uuid.Nil {
		return ctx
	}
	md := metadata.Pairs(
		"authorization", "Bearer "+billingAuthTokenFor(userID),
	)
	return metadata.NewOutgoingContext(ctx, md)
}
