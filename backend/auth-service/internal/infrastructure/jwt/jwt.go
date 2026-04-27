package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenService handles JWT token generation and validation
type TokenService struct {
	secret          []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
}

// NewTokenService creates a new TokenService
func NewTokenService(secret string, accessDuration, refreshDuration time.Duration) (*TokenService, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters")
	}
	return &TokenService{
		secret:          []byte(secret),
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
	}, nil
}

// GenerateTokenPair generates an access token and refresh token for a user
// without team context.
func (ts *TokenService) GenerateTokenPair(_ context.Context, userID uuid.UUID, email, tier string) (string, string, error) {
	return ts.generate(userID, email, tier, uuid.Nil, "")
}

// GenerateTokenPairWithTeam generates an access token and refresh token
// embedding org_id и role в claims (us=02_team_management).
func (ts *TokenService) GenerateTokenPairWithTeam(_ context.Context, userID uuid.UUID, email, tier string, orgID uuid.UUID, role string) (string, string, error) {
	return ts.generate(userID, email, tier, orgID, role)
}

func (ts *TokenService) generate(userID uuid.UUID, email, tier string, orgID uuid.UUID, role string) (string, string, error) {
	accessClaims := NewClaims(userID, email, tier, ts.accessDuration)
	if orgID != uuid.Nil {
		accessClaims = accessClaims.WithTeamContext(orgID, role)
	}
	accessToken, err := ts.signClaims(accessClaims)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %v", err)
	}

	refreshClaims := NewClaims(userID, email, tier, ts.refreshDuration)
	if orgID != uuid.Nil {
		refreshClaims = refreshClaims.WithTeamContext(orgID, role)
	}
	refreshToken, err := ts.signClaims(refreshClaims)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %v", err)
	}

	return accessToken, refreshToken, nil
}

// ValidateAccessToken validates an access token and returns parsed claims.
func (ts *TokenService) ValidateAccessToken(_ context.Context, token string) (*Claims, error) {
	claims := &Claims{}
	tokenObj, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ts.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	if !tokenObj.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

// signClaims signs claims into a JWT token
func (ts *TokenService) signClaims(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(ts.secret)
}
