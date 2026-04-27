package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	integrationv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	apikeysvc "github.com/raul/monitor/backend/integration-service/internal/service/apikey"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
)

// TestModelToProtoAPIKey проверяет конвертацию модели APIKey в proto.
func TestModelToProtoAPIKey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()
	keyID := uuid.New()
	desc := "My test key"
	expiresAt := now.Add(24 * time.Hour)
	rotatedAt := now.Add(-1 * time.Hour)

	key := &model.APIKey{
		ID:                 keyID,
		UserID:             userID,
		Name:               "Test API Key",
		Description:        &desc,
		KeyHash:            "abc123hash",
		KeyPrefix:          "baku_ro_abc...",
		Scopes:             []string{"read_monitors", "read_alerts"},
		Status:             model.APIKeyStatusActive,
		ExpiresAt:          &expiresAt,
		IPWhitelist:        []string{"192.168.1.1"},
		RateLimitPerMinute: 100,
		TotalRequests:      50,
		SuccessfulRequests: 45,
		FailedRequests:     5,
		CreatedAt:          now,
		UpdatedAt:          now,
		LastRotatedAt:      &rotatedAt,
	}

	result := modelToProtoAPIKey(key)

	require.NotNil(t, result)
	assert.Equal(t, keyID.String(), result.Id)
	assert.Equal(t, userID.String(), result.UserId)
	assert.Equal(t, "Test API Key", result.Name)
	assert.Equal(t, "My test key", result.Description)
	assert.Equal(t, "baku_ro_abc...", result.KeyPrefix)
	assert.Equal(t, []string{"read_monitors", "read_alerts"}, result.Scopes)
	assert.Equal(t, string(model.APIKeyStatusActive), result.Status)
	assert.Equal(t, int32(100), result.RateLimitPerMinute)
	assert.Equal(t, []string{"192.168.1.1"}, result.IpWhitelist)
	require.NotNil(t, result.ExpiresAt)
	require.NotNil(t, result.LastRotatedAt)
	require.NotNil(t, result.CreatedAt)
	require.NotNil(t, result.UpdatedAt)
}

// TestModelToProtoAPIKeyNoOptionalFields проверяет конвертацию с минимальными полями.
func TestModelToProtoAPIKeyNoOptionalFields(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	userID := uuid.New()
	keyID := uuid.New()

	key := &model.APIKey{
		ID:                 keyID,
		UserID:             userID,
		Name:               "Minimal Key",
		KeyHash:            "hash",
		KeyPrefix:          "baku_",
		Scopes:             []string{"read_monitors"},
		Status:             model.APIKeyStatusActive,
		RateLimitPerMinute: 60,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	result := modelToProtoAPIKey(key)

	require.NotNil(t, result)
	assert.Equal(t, "", result.Description)
	assert.Nil(t, result.ExpiresAt)
	assert.Nil(t, result.LastRotatedAt)
	assert.Nil(t, result.IpWhitelist)
}

// TestAPIKeyHandlerInvalidUUID проверяет обработку невалидных UUID.
func TestAPIKeyHandlerInvalidUUID(t *testing.T) {
	t.Parallel()

	h := &APIKeyHandler{}
	ctx := context.Background()

	t.Run("CreateAPIKey invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.CreateAPIKey(ctx, &integrationv1.CreateAPIKeyRequest{
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("GetAPIKey invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetAPIKey(ctx, &integrationv1.GetAPIKeyRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("GetAPIKey invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetAPIKey(ctx, &integrationv1.GetAPIKeyRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("ListAPIKeys invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.ListAPIKeys(ctx, &integrationv1.ListAPIKeysRequest{
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("UpdateAPIKey invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.UpdateAPIKey(ctx, &integrationv1.UpdateAPIKeyRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("UpdateAPIKey invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.UpdateAPIKey(ctx, &integrationv1.UpdateAPIKeyRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("DeleteAPIKey invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.DeleteAPIKey(ctx, &integrationv1.DeleteAPIKeyRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("DeleteAPIKey invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.DeleteAPIKey(ctx, &integrationv1.DeleteAPIKeyRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("RotateAPIKeySecret invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.RotateAPIKeySecret(ctx, &integrationv1.RotateAPIKeySecretRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("RotateAPIKeySecret invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.RotateAPIKeySecret(ctx, &integrationv1.RotateAPIKeySecretRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("GetAPIKeyStats invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetAPIKeyStats(ctx, &integrationv1.GetAPIKeyStatsRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("GetAPIKeyStats invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetAPIKeyStats(ctx, &integrationv1.GetAPIKeyStatsRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})
}

// TestNewAPIKeyHandler проверяет создание нового обработчика.
func TestNewAPIKeyHandler(t *testing.T) {
	t.Parallel()

	h := NewAPIKeyHandler(nil)
	require.NotNil(t, h)
}

// handlerMockAPIKeyRepo реализует interfaces.APIKeyRepository для тестов обработчиков.
type handlerMockAPIKeyRepo struct {
	keys map[uuid.UUID]*model.APIKey
}

func newHandlerMockAPIKeyRepo() *handlerMockAPIKeyRepo {
	return &handlerMockAPIKeyRepo{keys: make(map[uuid.UUID]*model.APIKey)}
}

func (m *handlerMockAPIKeyRepo) Create(_ context.Context, key *model.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *handlerMockAPIKeyRepo) GetByID(_ context.Context, id uuid.UUID) (*model.APIKey, error) {
	k, ok := m.keys[id]
	if !ok {
		return nil, model.ErrAPIKeyNotFound
	}
	return k, nil
}

func (m *handlerMockAPIKeyRepo) GetByKeyHash(_ context.Context, keyHash string) (*model.APIKey, error) {
	for _, k := range m.keys {
		if k.KeyHash == keyHash {
			return k, nil
		}
	}
	return nil, model.ErrAPIKeyNotFound
}

func (m *handlerMockAPIKeyRepo) GetByUserIDAndName(_ context.Context, userID uuid.UUID, name string) (*model.APIKey, error) {
	for _, k := range m.keys {
		if k.UserID == userID && k.Name == name {
			return k, nil
		}
	}
	return nil, model.ErrAPIKeyNotFound
}

func (m *handlerMockAPIKeyRepo) ListByUserID(_ context.Context, userID uuid.UUID, _, _ int) ([]*model.APIKey, error) {
	var result []*model.APIKey
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *handlerMockAPIKeyRepo) Update(_ context.Context, key *model.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *handlerMockAPIKeyRepo) UpdateUsage(_ context.Context, _ uuid.UUID, _ *interfaces.APIKeyUsageStats) error {
	return nil
}

func (m *handlerMockAPIKeyRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.keys, id)
	return nil
}

func (m *handlerMockAPIKeyRepo) CountByUserID(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, k := range m.keys {
		if k.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *handlerMockAPIKeyRepo) ExistsByName(_ context.Context, userID uuid.UUID, name string) (bool, error) {
	for _, k := range m.keys {
		if k.UserID == userID && k.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// handlerMockAPIKeyUsageRepo реализует interfaces.APIKeyUsageRepository.
type handlerMockAPIKeyUsageRepo struct{}

func (m *handlerMockAPIKeyUsageRepo) Create(_ context.Context, _ *model.APIKeyUsageLog) error {
	return nil
}
func (m *handlerMockAPIKeyUsageRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.APIKeyUsageLog, error) {
	return nil, nil
}
func (m *handlerMockAPIKeyUsageRepo) ListByAPIKeyID(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.APIKeyUsageLog, error) {
	return nil, nil
}
func (m *handlerMockAPIKeyUsageRepo) ListByAPIKeyIDAndPeriod(_ context.Context, _ uuid.UUID, _, _ int64, _, _ int) ([]*model.APIKeyUsageLog, error) {
	return nil, nil
}
func (m *handlerMockAPIKeyUsageRepo) DeleteOldLogs(_ context.Context, _ int) (int64, error) {
	return 0, nil
}

// newTestAPIKeyHandlerWithService создаёт APIKeyHandler с реальным сервисом.
func newTestAPIKeyHandlerWithService(t *testing.T, repo *handlerMockAPIKeyRepo) *APIKeyHandler {
	t.Helper()
	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)
	gen := security.NewAPIKeyGenerator()
	rl := apikeysvc.NewSlidingWindowRateLimiter()
	svc := apikeysvc.NewAPIKeyService(repo, &handlerMockAPIKeyUsageRepo{}, enc, gen, nil, rl)
	return NewAPIKeyHandler(svc)
}

func TestAPIKeyHandlerGetAPIKey(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	key := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test Key",
		KeyHash:   "hash",
		KeyPrefix: "baku_ro_abc",
		Scopes:    []string{"read:monitors"},
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.keys[key.ID] = key

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("get existing key", func(t *testing.T) {
		t.Parallel()
		result, err := h.GetAPIKey(ctx, &integrationv1.GetAPIKeyRequest{
			Id:     key.ID.String(),
			UserId: userID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, key.ID.String(), result.Id)
		assert.Equal(t, "Test Key", result.Name)
	})

	t.Run("get key not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetAPIKey(ctx, &integrationv1.GetAPIKeyRequest{
			Id:     uuid.New().String(),
			UserId: userID.String(),
		})
		require.Error(t, err)
	})

	t.Run("get key belonging to other user returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetAPIKey(ctx, &integrationv1.GetAPIKeyRequest{
			Id:     key.ID.String(),
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

func TestAPIKeyHandlerListAPIKeys(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	for i := 0; i < 3; i++ {
		k := &model.APIKey{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Key",
			KeyHash:   "hash",
			KeyPrefix: "baku_ro_",
			Status:    model.APIKeyStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.keys[k.ID] = k
	}

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.ListAPIKeys(ctx, &integrationv1.ListAPIKeysRequest{
		UserId:   userID.String(),
		PageSize: 50,
	})
	require.NoError(t, err)
	assert.Len(t, result.ApiKeys, 3)
	assert.Equal(t, int32(3), result.TotalCount)
}

func TestAPIKeyHandlerDeleteAPIKey(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	key := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "ToDelete",
		KeyHash:   "hash",
		KeyPrefix: "baku_ro_",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.keys[key.ID] = key

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("delete own key succeeds", func(t *testing.T) {
		localRepo := newHandlerMockAPIKeyRepo()
		localKey := &model.APIKey{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "X",
			KeyHash:   "h",
			KeyPrefix: "p",
			Status:    model.APIKeyStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		localRepo.keys[localKey.ID] = localKey
		localH := newTestAPIKeyHandlerWithService(t, localRepo)
		_, err := localH.DeleteAPIKey(ctx, &integrationv1.DeleteAPIKeyRequest{
			Id:     localKey.ID.String(),
			UserId: userID.String(),
		})
		require.NoError(t, err)
	})

	t.Run("delete key owned by other user returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.DeleteAPIKey(ctx, &integrationv1.DeleteAPIKeyRequest{
			Id:     key.ID.String(),
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

func TestAPIKeyHandlerUpdateAPIKey(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	key := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Old Name",
		KeyHash:   "hash",
		KeyPrefix: "baku_ro_",
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.keys[key.ID] = key

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.UpdateAPIKey(ctx, &integrationv1.UpdateAPIKeyRequest{
		Id:                 key.ID.String(),
		UserId:             userID.String(),
		Name:               "New Name",
		Description:        "Updated description",
		RateLimitPerMinute: 120,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestAPIKeyHandlerUpdateAPIKeyAllFields(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	key := &model.APIKey{
		ID:                 uuid.New(),
		UserID:             userID,
		Name:               "Old Name",
		KeyHash:            "hash",
		KeyPrefix:          "baku_ro_",
		Status:             model.APIKeyStatusActive,
		RateLimitPerMinute: 60,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	repo.keys[key.ID] = key

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	expiresAt := time.Now().Add(24 * time.Hour)
	expiresProto := timestamppb.New(expiresAt)

	result, err := h.UpdateAPIKey(ctx, &integrationv1.UpdateAPIKeyRequest{
		Id:                 key.ID.String(),
		UserId:             userID.String(),
		Name:               "Updated Name",
		Description:        "New description",
		RateLimitPerMinute: 200,
		IpWhitelist:        []string{"10.0.0.1"},
		ExpiresAt:          expiresProto,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", result.Name)
}

func TestAPIKeyHandlerGetAPIKeyStats(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	key := &model.APIKey{
		ID:                 uuid.New(),
		UserID:             userID,
		Name:               "Stats Key",
		KeyHash:            "hash",
		KeyPrefix:          "baku_ro_",
		Status:             model.APIKeyStatusActive,
		TotalRequests:      100,
		SuccessfulRequests: 90,
		FailedRequests:     10,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	repo.keys[key.ID] = key

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.GetAPIKeyStats(ctx, &integrationv1.GetAPIKeyStatsRequest{
		Id:     key.ID.String(),
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 100, result.TotalRequests)
}

// makeValidAPIKey создаёт валидный API ключ формата baku_ro_<64 hex chars>.
func makeValidAPIKey() string {
	return "baku_ro_" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
}

func TestAPIKeyHandlerValidateAPIKey(t *testing.T) {
	t.Parallel()

	repo := newHandlerMockAPIKeyRepo()
	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("invalid key format returns valid=false no error", func(t *testing.T) {
		t.Parallel()
		result, err := h.ValidateAPIKey(ctx, &integrationv1.ValidateAPIKeyRequest{
			ApiKey:    "invalid_key",
			Endpoint:  "/api/v1/monitors",
			Method:    "GET",
			IpAddress: "127.0.0.1",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "INVALID_API_KEY", result.ErrorCode)
	})

	t.Run("valid format but not found returns valid=false no error", func(t *testing.T) {
		t.Parallel()
		result, err := h.ValidateAPIKey(ctx, &integrationv1.ValidateAPIKeyRequest{
			ApiKey:    makeValidAPIKey(),
			Endpoint:  "/api/v1/monitors",
			Method:    "GET",
			IpAddress: "127.0.0.1",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "API_KEY_NOT_FOUND", result.ErrorCode)
	})

	t.Run("disabled key returns valid=false", func(t *testing.T) {
		t.Parallel()
		validKey := makeValidAPIKey()
		keyHash := model.HashAPIKey(validKey)

		localRepo := newHandlerMockAPIKeyRepo()
		disabledKey := &model.APIKey{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Name:               "Disabled",
			KeyHash:            keyHash,
			KeyPrefix:          "baku_ro_a1b",
			Scopes:             []string{"read:monitors"},
			Status:             model.APIKeyStatusDisabled,
			RateLimitPerMinute: 60,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		localRepo.keys[disabledKey.ID] = disabledKey
		localH := newTestAPIKeyHandlerWithService(t, localRepo)

		result, err := localH.ValidateAPIKey(ctx, &integrationv1.ValidateAPIKeyRequest{
			ApiKey:    validKey,
			Endpoint:  "/api/v1/monitors",
			Method:    "GET",
			IpAddress: "127.0.0.1",
		})
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "API_KEY_DISABLED", result.ErrorCode)
	})

	t.Run("active key with allowed IP returns valid=true", func(t *testing.T) {
		t.Parallel()
		validKey := makeValidAPIKey()
		keyHash := model.HashAPIKey(validKey)

		localRepo := newHandlerMockAPIKeyRepo()
		activeKey := &model.APIKey{
			ID:                 uuid.New(),
			UserID:             uuid.New(),
			Name:               "Active",
			KeyHash:            keyHash,
			KeyPrefix:          "baku_ro_a1b",
			Scopes:             []string{"read:monitors"},
			Status:             model.APIKeyStatusActive,
			RateLimitPerMinute: 60,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		localRepo.keys[activeKey.ID] = activeKey
		localH := newTestAPIKeyHandlerWithService(t, localRepo)

		result, err := localH.ValidateAPIKey(ctx, &integrationv1.ValidateAPIKeyRequest{
			ApiKey:    validKey,
			Endpoint:  "/api/v1/monitors",
			Method:    "GET",
			IpAddress: "127.0.0.1",
		})
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, activeKey.UserID.String(), result.UserId)
	})
}

// TestAPIKeyHandlerGetAPIKeyStatsError проверяет обработку ошибок GetAPIKeyStats.
func TestAPIKeyHandlerGetAPIKeyStatsError(t *testing.T) {
	t.Parallel()

	repo := newHandlerMockAPIKeyRepo()
	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	// Запрашиваем статистику для несуществующего ключа
	_, err := h.GetAPIKeyStats(ctx, &integrationv1.GetAPIKeyStatsRequest{
		Id:     uuid.New().String(),
		UserId: uuid.New().String(),
	})
	require.Error(t, err)
}

// TestAPIKeyHandlerListAPIKeysError проверяет обработку ошибок ListAPIKeys.
func TestAPIKeyHandlerListAPIKeysError(t *testing.T) {
	t.Parallel()

	repo := newHandlerMockAPIKeyRepo()
	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	// Запрашиваем список с невалидным user_id
	_, err := h.ListAPIKeys(ctx, &integrationv1.ListAPIKeysRequest{
		UserId: "not-a-uuid",
	})
	require.Error(t, err)
}

// TestAPIKeyHandlerValidateAPIKeyError проверяет обработку ошибок валидации.
func TestAPIKeyHandlerValidateAPIKeyError(t *testing.T) {
	t.Parallel()

	// Создаём репозиторий, который возвращает ошибку при GetByKeyHash
	repo := newHandlerMockAPIKeyRepo()
	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	// Ключ с правильным форматом baku_ro_ + 64 hex chars, но не найден в repo,
	// возвращает valid=false, без ошибки
	validKey := makeValidAPIKey()
	result, err := h.ValidateAPIKey(ctx, &integrationv1.ValidateAPIKeyRequest{
		ApiKey:    validKey,
		Endpoint:  "/api/v1/monitors",
		Method:    "GET",
		IpAddress: "127.0.0.1",
	})
	require.NoError(t, err)
	assert.False(t, result.Valid)
}

// TestAPIKeyHandlerCreateAPIKeyValidationError проверяет обработку ошибок валидации при создании API ключа.
func TestAPIKeyHandlerCreateAPIKeyValidationError(t *testing.T) {
	t.Parallel()

	repo := newHandlerMockAPIKeyRepo()
	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.CreateAPIKey(ctx, &integrationv1.CreateAPIKeyRequest{
			UserId:  uuid.New().String(),
			Name:    "",
			KeyType: "read_only",
			Scopes:  []string{"read:monitors"},
		})
		require.Error(t, err)
	})

	t.Run("with expires_at builds request correctly then fails validation", func(t *testing.T) {
		t.Parallel()
		import_time := time.Now().Add(24 * time.Hour)
		_, err := h.CreateAPIKey(ctx, &integrationv1.CreateAPIKeyRequest{
			UserId:    uuid.New().String(),
			Name:      "",
			KeyType:   "read_only",
			Scopes:    []string{"read:monitors"},
			ExpiresAt: timestamppb.New(import_time),
		})
		require.Error(t, err)
	})
}

func TestAPIKeyHandlerRotateAPIKeySecret(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newHandlerMockAPIKeyRepo()

	key := &model.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Rotatable",
		KeyHash:   "hash",
		KeyPrefix: "baku_ro_",
		Scopes:    []string{"read:monitors"},
		Status:    model.APIKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.keys[key.ID] = key

	h := newTestAPIKeyHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("rotate own key succeeds", func(t *testing.T) {
		result, err := h.RotateAPIKeySecret(ctx, &integrationv1.RotateAPIKeySecretRequest{
			Id:     key.ID.String(),
			UserId: userID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, key.ID.String(), result.Id)
	})

	t.Run("rotate key owned by other user returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.RotateAPIKeySecret(ctx, &integrationv1.RotateAPIKeySecretRequest{
			Id:     key.ID.String(),
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}
