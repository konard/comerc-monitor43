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
	t.Parallel()

	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)

	assert.NotNil(t, authenticator)
	assert.NotNil(t, authenticator.secretKey)
}

func TestAuthenticator_ValidateToken_Success(t *testing.T) {
	t.Parallel()

	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)

	ctx := context.Background()
	userID := uuid.New()

	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	require.NoError(t, err)

	validateClaims, err := authenticator.ValidateToken(ctx, signedToken)
	require.NoError(t, err)
	assert.Equal(t, userID, validateClaims.UserID)
	expectedExpiration := time.Now().Add(time.Hour)
	require.NotNil(t, validateClaims.ExpiresAt)
	assert.WithinDuration(t, expectedExpiration, validateClaims.ExpiresAt.Time, time.Minute)
}

func TestAuthenticator_ValidateToken_Missing(t *testing.T) {
	t.Parallel()

	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)
	ctx := context.Background()

	_, err := authenticator.ValidateToken(ctx, "")
	assert.Error(t, err)
	assert.Equal(t, ErrTokenMissing, err)
}

func TestAuthenticator_ValidateToken_Expired(t *testing.T) {
	t.Parallel()

	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)
	ctx := context.Background()
	userID := uuid.New()

	expirationTime := time.Now().Add(-time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	require.NoError(t, err)

	_, err = authenticator.ValidateToken(ctx, signedToken)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
}

func TestAuthenticator_ValidateToken_Invalid(t *testing.T) {
	t.Parallel()

	secretKey := "test-secret-key"
	authenticator := NewAuthenticator(secretKey)
	ctx := context.Background()

	_, err := authenticator.ValidateToken(ctx, "invalid.token.format")
	assert.Error(t, err)
}

func TestAuthenticator_ValidateToken_WrongSecret(t *testing.T) {
	t.Parallel()

	secretKey := "test-secret-key"
	ctx := context.Background()
	userID := uuid.New()

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	require.NoError(t, err)

	wrongAuthenticator := NewAuthenticator("wrong-secret")
	_, err = wrongAuthenticator.ValidateToken(ctx, signedToken)
	assert.Error(t, err)
}
