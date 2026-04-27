package apikey

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// Mock implementations для тестирования

type mockAPIKeyRepository struct {
	keys map[uuid.UUID]*model.APIKey
}

func newMockAPIKeyRepository() *mockAPIKeyRepository {
	return &mockAPIKeyRepository{
		keys: make(map[uuid.UUID]*model.APIKey),
	}
}

func (m *mockAPIKeyRepository) Create(ctx context.Context, key *model.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.APIKey, error) {
	key, exists := m.keys[id]
	if !exists {
		return nil, model.ErrAPIKeyNotFound
	}
	return key, nil
}

func (m *mockAPIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	for _, key := range m.keys {
		if key.KeyHash == keyHash {
			return key, nil
		}
	}
	return nil, model.ErrAPIKeyNotFound
}

func (m *mockAPIKeyRepository) GetByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*model.APIKey, error) {
	for _, key := range m.keys {
		if key.UserID == userID && key.Name == name {
			return key, nil
		}
	}
	return nil, model.ErrAPIKeyNotFound
}

func (m *mockAPIKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.APIKey, error) {
	var result []*model.APIKey
	for _, key := range m.keys {
		if key.UserID == userID {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *mockAPIKeyRepository) Update(ctx context.Context, key *model.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockAPIKeyRepository) UpdateUsage(ctx context.Context, id uuid.UUID, stats any) error {
	return nil
}

func (m *mockAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.keys, id)
	return nil
}

func (m *mockAPIKeyRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, key := range m.keys {
		if key.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockAPIKeyRepository) ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	for _, key := range m.keys {
		if key.UserID == userID && key.Name == name {
			return true, nil
		}
	}
	return false, nil
}

type mockAPIKeyUsageRepository struct{}

func newMockAPIKeyUsageRepository() *mockAPIKeyUsageRepository {
	return &mockAPIKeyUsageRepository{}
}

func (m *mockAPIKeyUsageRepository) Create(ctx context.Context, log *model.APIKeyUsageLog) error {
	return nil
}

func TestCreateAPIKeyRequest_Validate(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name    string
		req     *CreateAPIKeyRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: &CreateAPIKeyRequest{
				UserID:  userID,
				Name:    "Test Key",
				KeyType: model.APIKeyTypeReadOnly,
				Scopes:  []string{"read_monitors"},
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			req: &CreateAPIKeyRequest{
				UserID:  userID,
				Name:    "",
				KeyType: model.APIKeyTypeReadOnly,
				Scopes:  []string{"read_monitors"},
			},
			wantErr: model.ErrEmptyAPIKeyName,
		},
		{
			name: "no scopes",
			req: &CreateAPIKeyRequest{
				UserID:  userID,
				Name:    "Test Key",
				KeyType: model.APIKeyTypeReadOnly,
				Scopes:  []string{},
			},
			wantErr: model.ErrInvalidScope,
		},
		{
			name: "invalid scope",
			req: &CreateAPIKeyRequest{
				UserID:  userID,
				Name:    "Test Key",
				KeyType: model.APIKeyTypeReadOnly,
				Scopes:  []string{"invalid_scope"},
			},
			wantErr: model.ErrInvalidScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIKeyStatsFromModel(t *testing.T) {
	userID := uuid.New()
	key, _, err := model.NewAPIKey(userID, "Test", model.APIKeyTypeReadOnly, []string{"read_monitors"})
	require.NoError(t, err)

	// Симулируем использование
	for i := 0; i < 10; i++ {
		key.MarkAsUsed("/api/v1/monitors", "192.168.1.1", true)
	}

	stats := APIKeyStatsFromModel(key, 100)

	assert.Equal(t, int64(10), stats.TotalRequests)
	assert.Equal(t, int64(10), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.Equal(t, float64(100), stats.SuccessRate)
	assert.Equal(t, int32(100), stats.AvgLatencyMs)
	assert.NotNil(t, stats.LastUsedAt)
	assert.NotNil(t, stats.LastUsedIP)
}
