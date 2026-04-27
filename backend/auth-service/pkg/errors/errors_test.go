package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	err := New("CODE", "message")

	assert.Equal(t, "CODE", err.Code)
	assert.Equal(t, "message", err.Message)
	assert.Nil(t, err.Details)
}

func TestAppError_Error(t *testing.T) {
	t.Parallel()

	err := New("MY_CODE", "my message")
	assert.Equal(t, "MY_CODE: my message", err.Error())
}

func TestWithDetails(t *testing.T) {
	t.Parallel()

	err := New("CODE", "message").WithDetails(map[string]any{"key": "value"})

	assert.Equal(t, "value", err.Details["key"])
}

func TestInvalidCredentials(t *testing.T) {
	t.Parallel()

	err := InvalidCredentials()
	assert.Equal(t, ErrorCodeInvalidCredentials, err.Code)
	assert.NotEmpty(t, err.Message)
}

func TestInvalidToken(t *testing.T) {
	t.Parallel()

	err := InvalidToken()
	assert.Equal(t, ErrorCodeInvalidToken, err.Code)
}

func TestExpiredToken(t *testing.T) {
	t.Parallel()

	err := ExpiredToken()
	assert.Equal(t, ErrorCodeExpiredToken, err.Code)
}

func TestInvalidRefreshToken(t *testing.T) {
	t.Parallel()

	err := InvalidRefreshToken()
	assert.Equal(t, ErrorCodeInvalidRefreshToken, err.Code)
}

func TestAccountLocked(t *testing.T) {
	t.Parallel()

	err := AccountLocked()
	assert.Equal(t, ErrorCodeAccountLocked, err.Code)
}

func TestAccountNotFound(t *testing.T) {
	t.Parallel()

	err := AccountNotFound()
	assert.Equal(t, ErrorCodeAccountNotFound, err.Code)
}

func TestUserExists(t *testing.T) {
	t.Parallel()

	err := UserExists()
	assert.Equal(t, ErrorCodeUserExists, err.Code)
}

func TestUserNotFound(t *testing.T) {
	t.Parallel()

	err := UserNotFound()
	assert.Equal(t, ErrorCodeUserNotFound, err.Code)
}

func TestInvalidRequest(t *testing.T) {
	t.Parallel()

	err := InvalidRequest("bad input")
	assert.Equal(t, ErrorCodeInvalidRequest, err.Code)
	assert.Equal(t, "bad input", err.Message)
}

func TestOAuthFailed(t *testing.T) {
	t.Parallel()

	underlying := fmt.Errorf("exchange failed")
	err := OAuthFailed(underlying)
	assert.Equal(t, ErrorCodeOAuthFailed, err.Code)
	assert.Contains(t, err.Message, "exchange failed")
}

func TestOAuthStateMismatch(t *testing.T) {
	t.Parallel()

	err := OAuthStateMismatch()
	assert.Equal(t, ErrorCodeOAuthStateMismatch, err.Code)
}

func TestSessionNotFound(t *testing.T) {
	t.Parallel()

	err := SessionNotFound()
	assert.Equal(t, ErrorCodeSessionNotFound, err.Code)
}

func TestUnauthorized(t *testing.T) {
	t.Parallel()

	err := Unauthorized()
	assert.Equal(t, ErrorCodeUnauthorized, err.Code)
}

func TestInvalidEmail(t *testing.T) {
	t.Parallel()

	err := InvalidEmail()
	assert.Equal(t, ErrorCodeInvalidEmail, err.Code)
}

func TestInvalidPassword(t *testing.T) {
	t.Parallel()

	err := InvalidPassword()
	assert.Equal(t, ErrorCodeInvalidPassword, err.Code)
}

func TestInvalidOAuthCode(t *testing.T) {
	t.Parallel()

	err := InvalidOAuthCode()
	assert.Equal(t, ErrorCodeInvalidOAuthCode, err.Code)
}

func TestTooManyRequests(t *testing.T) {
	t.Parallel()

	err := TooManyRequests()
	assert.Equal(t, ErrorCodeTooManyRequests, err.Code)
}

func TestForbidden(t *testing.T) {
	t.Parallel()

	err := Forbidden()
	assert.Equal(t, ErrorCodeForbidden, err.Code)
}

func TestInternalError(t *testing.T) {
	t.Parallel()

	underlying := fmt.Errorf("db connection failed")
	err := InternalError(underlying)
	assert.Equal(t, ErrorCodeInternalError, err.Code)
	assert.Contains(t, err.Message, "db connection failed")
}
