package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey struct{}

var UserIDKey = &contextKey{}

// Claims представляют данные JWT токена, выдаваемого auth-service.
// Использует jwt.RegisteredClaims для стандартных полей exp/iat/nbf.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Tier   string    `json:"tier"`
	jwt.RegisteredClaims
}

func ExtractUserID(ctx context.Context) uuid.UUID {
	userID := ctx.Value(UserIDKey)
	if userID == nil {
		return uuid.Nil
	}
	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID := ctx.Value(UserIDKey)
	if userID == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return id, nil
}
