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

// alertJWTSecret — общий с alert-service секрет, используемый в BDD-окружении.
const alertJWTSecret = "test-secret-for-bdd-tests-minimum32chars"

// authTokenFor возвращает HS256 JWT токен для тестового пользователя.
// Используется когда alert-service ожидает Authorization: Bearer <token>.
func authTokenFor(userID uuid.UUID, role string) string {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().Unix()
	claims := map[string]any{
		"user_id": userID.String(),
		"sub":     userID.String(),
		"role":    role,
		"iat":     now,
		"exp":     now + 3600,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(alertJWTSecret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

// alertAuthCtx добавляет метаданные пользователя в исходящий gRPC контекст.
func alertAuthCtx(ctx context.Context, userID uuid.UUID, role string) context.Context {
	if userID == uuid.Nil {
		return ctx
	}
	md := metadata.Pairs(
		"x-user-id", userID.String(),
		"x-user-role", role,
		"authorization", "Bearer "+authTokenFor(userID, role),
	)
	return metadata.NewOutgoingContext(ctx, md)
}
