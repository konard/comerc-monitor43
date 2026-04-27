package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenMalformed = errors.New("token malformed")
	ErrTokenMissing   = errors.New("token missing")
)

// Authenticator валидирует JWT токены
type Authenticator struct {
	secretKey []byte
}

// NewAuthenticator создаёт новый аутентификатор
func NewAuthenticator(secretKey string) *Authenticator {
	return &Authenticator{
		secretKey: []byte(secretKey),
	}
}

// ValidateToken проверяет валидность JWT токена и извлекает claims
func (a *Authenticator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrTokenMissing
	}

	// Парсим и валидируем токен
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return a.secretKey, nil
	})

	if err != nil {
		// Проверяем, истёк ли токен
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrTokenMalformed
	}

	return claims, nil
}
