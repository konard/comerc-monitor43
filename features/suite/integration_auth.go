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

// integrationJWTSecret — общий с integration-service секрет в BDD-окружении.
const integrationJWTSecret = "test-secret-for-bdd-tests-minimum32chars"

// integrationEncryptionKey — 32-байтовый ключ шифрования для integration-service.
const integrationEncryptionKey = "bdd-integration-encryption-key32"

// integrationAuthTokenFor возвращает HS256 JWT, совместимый с integration JWT interceptor.
// Interceptor ожидает claims user_id, email, tier.
func integrationAuthTokenFor(userID uuid.UUID, tier string) string {
	if tier == "" {
		tier = "Free"
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().Unix()
	claims := map[string]any{
		"user_id": userID.String(),
		"email":   "user@bdd.test",
		"tier":    tier,
		"iat":     now,
		"exp":     now + 3600,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(integrationJWTSecret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

// integrationAuthCtx добавляет Authorization: Bearer <token> в исходящий gRPC контекст.
func integrationAuthCtx(ctx context.Context, userID uuid.UUID, tier string) context.Context {
	if userID == uuid.Nil {
		return ctx
	}
	md := metadata.Pairs(
		"authorization", "Bearer "+integrationAuthTokenFor(userID, tier),
	)
	return metadata.NewOutgoingContext(ctx, md)
}
