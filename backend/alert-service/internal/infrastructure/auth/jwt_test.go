package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthenticator(t *testing.T) {
	secretKey := "test-secret-key"

	authenticator := NewAuthenticator(secretKey)

	assert.NotNil(t, authenticator)
	assert.NotNil(t, authenticator.secretKey)
}

func TestAuthenticator_ValidateToken_Success(t *testing.T) {
	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)

	ctx := context.Background()
	userID := uuid.New()

	// Create JWT token
	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))

	require.NoError(t, err)

	// Validate token
	validateClaims, err := authenticator.ValidateToken(ctx, signedToken)
	require.NoError(t, err)
	assert.Equal(t, userID, validateClaims.UserID)
	// Verify expiration is approximately 1 hour from now
	expectedExpiration := time.Now().Add(time.Hour)
	assert.WithinDuration(t, expectedExpiration, validateClaims.ExpiresAt.Time, time.Minute)
}

func TestAuthenticator_ValidateToken_Missing(t *testing.T) {
	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)

	ctx := context.Background()

	// Test with empty token
	_, err := authenticator.ValidateToken(ctx, "")
	assert.Error(t, err)
	assert.Equal(t, ErrTokenMissing, err)
}

func TestAuthenticator_ValidateToken_Expired(t *testing.T) {
	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)

	ctx := context.Background()
	userID := uuid.New()

	// Create expired token
	expirationTime := time.Now().Add(-time.Hour)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))

	require.NoError(t, err)

	// Validate expired token
	_, err = authenticator.ValidateToken(ctx, signedToken)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
}

func TestAuthenticator_ValidateToken_Invalid(t *testing.T) {
	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)

	ctx := context.Background()

	// Test with invalid token format
	invalidToken := "invalid.token.format"

	_, err := authenticator.ValidateToken(ctx, invalidToken)
	assert.Error(t, err)
}

func TestAuthenticator_ValidateToken_WrongSecret(t *testing.T) {
	secretKey := "test-secret-key"

	ctx := context.Background()
	userID := uuid.New()
	expirationTime := time.Now().Add(time.Hour)

	// Create token with one secret
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))

	require.NoError(t, err)

	// Try to validate with different secret
	wrongSecretAuthenticator := NewAuthenticator("wrong-secret")
	_, err = wrongSecretAuthenticator.ValidateToken(ctx, signedToken)
	assert.Error(t, err)
}

func TestClaims_RegisteredClaimsEmbedded(t *testing.T) {
	now := time.Now()
	inOneHour := now.Add(time.Hour)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(inOneHour),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: uuid.New(),
		Role:   RoleUser,
	}

	// Verify expiration via embedded RegisteredClaims
	expTime, err := claims.GetExpirationTime()
	require.NoError(t, err)
	require.NotNil(t, expTime)
	assert.WithinDuration(t, inOneHour, expTime.Time, time.Second)

	// Verify issued at via embedded RegisteredClaims
	iatTime, err := claims.GetIssuedAt()
	require.NoError(t, err)
	require.NotNil(t, iatTime)
	assert.WithinDuration(t, now, iatTime.Time, time.Second)
}
