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

func createTestToken(secret string, claims *Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		panic("failed to sign test token: " + err.Error())
	}
	return s
}

func TestValidateToken_valid_token(t *testing.T) {
	t.Parallel()

	auth := NewAuthenticator("test-secret")
	userID := uuid.New()

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenString := createTestToken("test-secret", claims)

	// Act
	result, err := auth.ValidateToken(context.Background(), tokenString)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
}

func TestValidateToken_expired_token(t *testing.T) {
	t.Parallel()

	auth := NewAuthenticator("test-secret")

	claims := &Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tokenString := createTestToken("test-secret", claims)

	// Act
	_, err := auth.ValidateToken(context.Background(), tokenString)

	// Assert
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestValidateToken_empty_string(t *testing.T) {
	t.Parallel()

	auth := NewAuthenticator("test-secret")

	// Act
	_, err := auth.ValidateToken(context.Background(), "")

	// Assert
	assert.ErrorIs(t, err, ErrTokenMissing)
}

func TestValidateToken_invalid_format(t *testing.T) {
	t.Parallel()

	auth := NewAuthenticator("test-secret")

	// Act
	_, err := auth.ValidateToken(context.Background(), "not-a-token")

	// Assert
	assert.ErrorIs(t, err, ErrTokenInvalid)
}

func TestValidateToken_wrong_secret(t *testing.T) {
	t.Parallel()

	auth := NewAuthenticator("secret-B")

	claims := &Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenString := createTestToken("secret-A", claims)

	// Act
	_, err := auth.ValidateToken(context.Background(), tokenString)

	// Assert
	assert.ErrorIs(t, err, ErrTokenInvalid)
}

func TestValidateToken_unsigned_token(t *testing.T) {
	t.Parallel()

	auth := NewAuthenticator("test-secret")

	claims := &Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	// Act
	_, err = auth.ValidateToken(context.Background(), tokenString)

	// Assert
	assert.ErrorIs(t, err, ErrTokenInvalid)
}
