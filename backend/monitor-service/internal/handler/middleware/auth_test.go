package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthenticateJWT_EmptyToken тестирует ошибку при пустом токене.
func TestAuthenticateJWT_EmptyToken(t *testing.T) {
	t.Parallel()

	claims, err := AuthenticateJWT(context.Background(), "", nil)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "missing auth token")
}

// TestAuthenticateJWT_InvalidFormat тестирует ошибку при неверном формате токена.
func TestAuthenticateJWT_InvalidFormat(t *testing.T) {
	t.Parallel()

	claims, err := AuthenticateJWT(context.Background(), "InvalidToken", nil)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "invalid auth token format")
}

// TestAuthenticateJWT_FullFormat тестирует разбор токена формата "Bearer <id>:<role>:<tier>".
func TestAuthenticateJWT_FullFormat(t *testing.T) {
	t.Parallel()

	claims, err := AuthenticateJWT(context.Background(), "Bearer user-123:ADMIN:Pro", nil)

	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "ADMIN", claims.Role)
	assert.Equal(t, "Pro", claims.Tier)
}

// TestAuthenticateJWT_SimpleToken тестирует разбор простого токена (без разделителей).
func TestAuthenticateJWT_SimpleToken(t *testing.T) {
	t.Parallel()

	claims, err := AuthenticateJWT(context.Background(), "Bearer simple-token", nil)

	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "simple-token", claims.UserID)
	assert.Equal(t, "USER", claims.Role)
	assert.Equal(t, "Free", claims.Tier)
}

// TestAuthenticateJWT_TwoPartToken тестирует разбор токена с двумя частями (без tier) — используется простой режим.
func TestAuthenticateJWT_TwoPartToken(t *testing.T) {
	t.Parallel()

	claims, err := AuthenticateJWT(context.Background(), "Bearer user-id:ROLE", nil)

	require.NoError(t, err)
	require.NotNil(t, claims)
	// Только 2 части — меньше 3, поэтому используется простой режим: UserID = весь токен
	assert.Equal(t, "user-id:ROLE", claims.UserID)
	assert.Equal(t, "USER", claims.Role)
	assert.Equal(t, "Free", claims.Tier)
}

// TestExtractUserID_Success тестирует успешное извлечение user_id из контекста.
func TestExtractUserID_Success(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserIDKey, "user-abc-123")

	userID, err := ExtractUserID(ctx)

	require.NoError(t, err)
	assert.Equal(t, "user-abc-123", userID)
}

// TestExtractUserID_Missing тестирует ошибку, если user_id отсутствует в контексте.
func TestExtractUserID_Missing(t *testing.T) {
	t.Parallel()

	userID, err := ExtractUserID(context.Background())

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Contains(t, err.Error(), "user_id not found")
}

// TestExtractUserID_Empty тестирует ошибку при пустом user_id.
func TestExtractUserID_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserIDKey, "")

	userID, err := ExtractUserID(ctx)

	assert.Error(t, err)
	assert.Empty(t, userID)
}

// TestExtractUserRole_Success тестирует успешное извлечение роли из контекста.
func TestExtractUserRole_Success(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserRoleKey, "ADMIN")

	role, err := ExtractUserRole(ctx)

	require.NoError(t, err)
	assert.Equal(t, "ADMIN", role)
}

// TestExtractUserRole_Missing тестирует ошибку, если роль отсутствует.
func TestExtractUserRole_Missing(t *testing.T) {
	t.Parallel()

	role, err := ExtractUserRole(context.Background())

	assert.Error(t, err)
	assert.Empty(t, role)
	assert.Contains(t, err.Error(), "user_role not found")
}

// TestExtractUserTier_Success тестирует успешное извлечение тира из контекста.
func TestExtractUserTier_Success(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserTierKey, "Enterprise")

	tier, err := ExtractUserTier(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Enterprise", tier)
}

// TestExtractUserTier_Missing тестирует ошибку, если тир отсутствует.
func TestExtractUserTier_Missing(t *testing.T) {
	t.Parallel()

	tier, err := ExtractUserTier(context.Background())

	assert.Error(t, err)
	assert.Empty(t, tier)
	assert.Contains(t, err.Error(), "user_tier not found")
}

// TestContextWithAuth тестирует создание контекста с данными аутентификации.
func TestContextWithAuth(t *testing.T) {
	t.Parallel()

	claims := &AuthClaims{
		UserID: "user-456",
		Role:   "USER",
		Tier:   "Pro",
	}

	ctx := ContextWithAuth(context.Background(), claims)

	// Проверяем, что все значения корректно сохранены в контексте
	userID, err := ExtractUserID(ctx)
	require.NoError(t, err)
	assert.Equal(t, "user-456", userID)

	role, err := ExtractUserRole(ctx)
	require.NoError(t, err)
	assert.Equal(t, "USER", role)

	tier, err := ExtractUserTier(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Pro", tier)
}

// TestContextWithAuth_OverwritesExisting тестирует перезапись существующих значений.
func TestContextWithAuth_OverwritesExisting(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), UserIDKey, "old-user")

	claims := &AuthClaims{
		UserID: "new-user",
		Role:   "ADMIN",
		Tier:   "Enterprise",
	}

	ctx = ContextWithAuth(ctx, claims)

	userID, err := ExtractUserID(ctx)
	require.NoError(t, err)
	assert.Equal(t, "new-user", userID)
}
