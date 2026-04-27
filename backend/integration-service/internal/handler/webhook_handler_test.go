package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	integrationv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
	webhooksvc "github.com/raul/monitor/backend/integration-service/internal/service/webhook"
)

// TestTimestampProto проверяет вспомогательную функцию timestampProto.
func TestTimestampProto(t *testing.T) {
	t.Parallel()

	t.Run("nil timestamp", func(t *testing.T) {
		t.Parallel()
		result := timestampProto(nil)
		assert.Nil(t, result)
	})

	t.Run("valid timestamp", func(t *testing.T) {
		t.Parallel()
		ts := int64(1700000000)
		result := timestampProto(&ts)
		require.NotNil(t, result)
		assert.Equal(t, int64(1700000000), result.AsTime().Unix())
	})
}

// TestModelToProtoWebhook проверяет конвертацию модели webhook в proto.
func TestModelToProtoWebhook(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()
	webhookID := uuid.New()

	webhook := &model.WebhookIntegration{
		ID:                      webhookID,
		UserID:                  userID,
		Name:                    "Test Webhook",
		URL:                     "https://example.com/webhook",
		Method:                  "POST",
		Headers:                 map[string]string{"X-Custom": "header"},
		Enabled:                 true,
		Status:                  model.WebhookStatusActive,
		Priority:                model.WebhookPriorityHigh,
		SeverityFilter:          []string{"critical", "warning"},
		MaxPayloadSizeBytes:     1048576,
		PayloadHandlingStrategy: model.PayloadHandlingStrategyTruncate,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	result := modelToProtoWebhook(webhook)

	require.NotNil(t, result)
	assert.Equal(t, webhookID.String(), result.Id)
	assert.Equal(t, userID.String(), result.UserId)
	assert.Equal(t, "Test Webhook", result.Name)
	assert.Equal(t, "https://example.com/webhook", result.Url)
	assert.Equal(t, "POST", result.Method)
	assert.Equal(t, map[string]string{"X-Custom": "header"}, result.Headers)
	assert.True(t, result.Enabled)
	assert.Equal(t, string(model.WebhookStatusActive), result.Status)
	assert.Equal(t, string(model.WebhookPriorityHigh), result.Priority)
	assert.Equal(t, []string{"critical", "warning"}, result.SeverityFilter)
	assert.Equal(t, int32(1048576), result.MaxPayloadSizeBytes)
	assert.Equal(t, string(model.PayloadHandlingStrategyTruncate), result.PayloadHandlingStrategy)
	require.NotNil(t, result.CreatedAt)
	require.NotNil(t, result.UpdatedAt)
}

// TestWebhookHandlerInvalidUUID проверяет обработку невалидных UUID.
func TestWebhookHandlerInvalidUUID(t *testing.T) {
	t.Parallel()

	// Создаём handler без реального сервиса, проверяем только UUID валидацию
	h := &WebhookHandler{}
	ctx := context.Background()

	t.Run("CreateWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.CreateWebhook(ctx, &integrationv1.CreateWebhookRequest{
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("GetWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetWebhook(ctx, &integrationv1.GetWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("GetWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetWebhook(ctx, &integrationv1.GetWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("ListWebhooks invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.ListWebhooks(ctx, &integrationv1.ListWebhooksRequest{
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("UpdateWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.UpdateWebhook(ctx, &integrationv1.UpdateWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("UpdateWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.UpdateWebhook(ctx, &integrationv1.UpdateWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("DeleteWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.DeleteWebhook(ctx, &integrationv1.DeleteWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("DeleteWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.DeleteWebhook(ctx, &integrationv1.DeleteWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("EnableWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.EnableWebhook(ctx, &integrationv1.EnableWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("EnableWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.EnableWebhook(ctx, &integrationv1.EnableWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("DisableWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.DisableWebhook(ctx, &integrationv1.DisableWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("DisableWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.DisableWebhook(ctx, &integrationv1.DisableWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("TestWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.TestWebhook(ctx, &integrationv1.TestWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("TestWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.TestWebhook(ctx, &integrationv1.TestWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("CloneWebhook invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.CloneWebhook(ctx, &integrationv1.CloneWebhookRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("CloneWebhook invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.CloneWebhook(ctx, &integrationv1.CloneWebhookRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("GetWebhookStats invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetWebhookStats(ctx, &integrationv1.GetWebhookStatsRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("GetWebhookStats invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetWebhookStats(ctx, &integrationv1.GetWebhookStatsRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})
}

// TestNewWebhookHandler проверяет создание нового обработчика.
func TestNewWebhookHandler(t *testing.T) {
	t.Parallel()

	h := NewWebhookHandler(nil)
	require.NotNil(t, h)
}

// mockWebhookRepo реализует interfaces.WebhookRepository для тестов хэндлеров.
type mockWebhookRepo struct {
	webhooks map[uuid.UUID]*model.WebhookIntegration
	err      error
}

func newMockWebhookRepo() *mockWebhookRepo {
	return &mockWebhookRepo{
		webhooks: make(map[uuid.UUID]*model.WebhookIntegration),
	}
}

func (m *mockWebhookRepo) Create(_ context.Context, w *model.WebhookIntegration) error {
	if m.err != nil {
		return m.err
	}
	m.webhooks[w.ID] = w
	return nil
}

func (m *mockWebhookRepo) GetByID(_ context.Context, id uuid.UUID) (*model.WebhookIntegration, error) {
	if m.err != nil {
		return nil, m.err
	}
	w, ok := m.webhooks[id]
	if !ok {
		return nil, model.ErrWebhookNotFound
	}
	return w, nil
}

func (m *mockWebhookRepo) GetByUserIDAndName(_ context.Context, userID uuid.UUID, name string) (*model.WebhookIntegration, error) {
	for _, w := range m.webhooks {
		if w.UserID == userID && w.Name == name {
			return w, nil
		}
	}
	return nil, model.ErrWebhookNotFound
}

func (m *mockWebhookRepo) ListByUserID(_ context.Context, userID uuid.UUID, _, _ int) ([]*model.WebhookIntegration, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*model.WebhookIntegration
	for _, w := range m.webhooks {
		if w.UserID == userID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWebhookRepo) ListActiveByUserID(_ context.Context, userID uuid.UUID) ([]*model.WebhookIntegration, error) {
	var result []*model.WebhookIntegration
	for _, w := range m.webhooks {
		if w.UserID == userID && w.Status == model.WebhookStatusActive {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWebhookRepo) Update(_ context.Context, w *model.WebhookIntegration) error {
	if m.err != nil {
		return m.err
	}
	m.webhooks[w.ID] = w
	return nil
}

func (m *mockWebhookRepo) UpdateStats(_ context.Context, _ uuid.UUID, _ *interfaces.WebhookStats) error {
	return nil
}

func (m *mockWebhookRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.webhooks, id)
	return nil
}

func (m *mockWebhookRepo) CountByUserID(_ context.Context, userID uuid.UUID) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	count := 0
	for _, w := range m.webhooks {
		if w.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockWebhookRepo) ExistsByName(_ context.Context, userID uuid.UUID, name string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	for _, w := range m.webhooks {
		if w.UserID == userID && w.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// newTestWebhookServiceForHandler создаёт реальный WebhookService с mock-репозиторием.
func newTestWebhookServiceForHandler(t *testing.T, repo *mockWebhookRepo) *webhooksvc.WebhookService {
	t.Helper()
	_, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)
	_ = security.NewAPIKeyGenerator()
	_ = repo
	return &webhooksvc.WebhookService{}
}

// newTestWebhookHandlerWithService создаёт WebhookHandler с реальным сервисом и mock-репозиторием.
func newTestWebhookHandlerWithService(t *testing.T, repo *mockWebhookRepo) *WebhookHandler {
	t.Helper()
	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)
	gen := security.NewAPIKeyGenerator()
	svc := webhooksvc.NewWebhookService(repo, enc, gen, nil, nil)
	return NewWebhookHandler(svc)
}

func TestWebhookHandlerGetWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	wh := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		URL:       "https://example.com/webhook",
		Method:    "POST",
		Status:    model.WebhookStatusActive,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("get existing webhook", func(t *testing.T) {
		t.Parallel()
		result, err := h.GetWebhook(ctx, &integrationv1.GetWebhookRequest{
			Id:     wh.ID.String(),
			UserId: userID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, wh.ID.String(), result.Id)
		assert.Equal(t, "Test", result.Name)
	})

	t.Run("get webhook not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetWebhook(ctx, &integrationv1.GetWebhookRequest{
			Id:     uuid.New().String(),
			UserId: userID.String(),
		})
		require.Error(t, err)
	})

	t.Run("get webhook belonging to other user returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetWebhook(ctx, &integrationv1.GetWebhookRequest{
			Id:     wh.ID.String(),
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

func TestWebhookHandlerListWebhooks(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	for i := 0; i < 3; i++ {
		w := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.webhooks[w.ID] = w
	}

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.ListWebhooks(ctx, &integrationv1.ListWebhooksRequest{
		UserId:   userID.String(),
		PageSize: 50,
	})
	require.NoError(t, err)
	assert.Len(t, result.Webhooks, 3)
}

func TestWebhookHandlerDeleteWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	wh := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "ToDelete",
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("delete own webhook succeeds", func(t *testing.T) {
		localRepo := newMockWebhookRepo()
		localWh := &model.WebhookIntegration{
			ID: uuid.New(), UserID: userID, Name: "X",
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		localRepo.webhooks[localWh.ID] = localWh
		localH := newTestWebhookHandlerWithService(t, localRepo)
		_, err := localH.DeleteWebhook(ctx, &integrationv1.DeleteWebhookRequest{
			Id:     localWh.ID.String(),
			UserId: userID.String(),
		})
		require.NoError(t, err)
	})

	t.Run("delete webhook owned by other user returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.DeleteWebhook(ctx, &integrationv1.DeleteWebhookRequest{
			Id:     wh.ID.String(),
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

func TestWebhookHandlerEnableDisable(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	wh := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		Status:    model.WebhookStatusDisabled,
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	_, err := h.EnableWebhook(ctx, &integrationv1.EnableWebhookRequest{
		Id:     wh.ID.String(),
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, model.WebhookStatusActive, repo.webhooks[wh.ID].Status)

	_, err = h.DisableWebhook(ctx, &integrationv1.DisableWebhookRequest{
		Id:     wh.ID.String(),
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, model.WebhookStatusDisabled, repo.webhooks[wh.ID].Status)
}

func TestWebhookHandlerGetWebhookStats(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	avgTime := 150
	wh := &model.WebhookIntegration{
		ID:                uuid.New(),
		UserID:            userID,
		Name:              "Test",
		Status:            model.WebhookStatusActive,
		TotalSent:         100,
		SuccessfulSent:    90,
		FailureCount:      10,
		AvgResponseTimeMs: &avgTime,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.GetWebhookStats(ctx, &integrationv1.GetWebhookStatsRequest{
		Id:     wh.ID.String(),
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(100), result.TotalSent)
	assert.Equal(t, int32(10), result.FailureCount)
	assert.Equal(t, int32(150), result.AvgResponseTimeMs)
}

func TestWebhookHandlerCloneWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	original := &model.WebhookIntegration{
		ID:                      uuid.New(),
		UserID:                  userID,
		Name:                    "Original",
		URL:                     "https://original.example.com/webhook",
		Method:                  "POST",
		Status:                  model.WebhookStatusActive,
		Priority:                model.WebhookPriorityNormal,
		MaxPayloadSizeBytes:     1048576,
		PayloadHandlingStrategy: model.PayloadHandlingStrategyTruncate,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	repo.webhooks[original.ID] = original

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.CloneWebhook(ctx, &integrationv1.CloneWebhookRequest{
		Id:      original.ID.String(),
		UserId:  userID.String(),
		NewName: "Clone",
		NewUrl:  "https://clone.example.com/webhook",
	})
	require.NoError(t, err)
	assert.NotEqual(t, original.ID.String(), result.Id)
	assert.Equal(t, "Clone", result.Name)
}

func TestWebhookHandlerUpdateWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	wh := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Old Name",
		URL:       "https://old.example.com/webhook",
		Method:    "POST",
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.UpdateWebhook(ctx, &integrationv1.UpdateWebhookRequest{
		Id:       wh.ID.String(),
		UserId:   userID.String(),
		Name:     "New Name",
		Url:      "https://new.example.com/webhook",
		Method:   "PUT",
		Priority: "high",
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestWebhookHandlerUpdateWebhookAllFields(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	wh := &model.WebhookIntegration{
		ID:                      uuid.New(),
		UserID:                  userID,
		Name:                    "Original",
		URL:                     "https://original.example.com/webhook",
		Method:                  "POST",
		Status:                  model.WebhookStatusActive,
		Priority:                model.WebhookPriorityNormal,
		MaxPayloadSizeBytes:     1048576,
		PayloadHandlingStrategy: model.PayloadHandlingStrategyTruncate,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.UpdateWebhook(ctx, &integrationv1.UpdateWebhookRequest{
		Id:                      wh.ID.String(),
		UserId:                  userID.String(),
		Name:                    "Updated",
		Url:                     "https://updated.example.com/webhook",
		Method:                  "PUT",
		Headers:                 map[string]string{"X-Custom": "value"},
		Priority:                "high",
		SeverityFilter:          []string{"critical"},
		MaxPayloadSizeBytes:     512,
		PayloadHandlingStrategy: "reject",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", result.Name)
	assert.Equal(t, "https://updated.example.com/webhook", result.Url)
	assert.Equal(t, "PUT", result.Method)
}

func TestWebhookHandlerTestWebhookNotActive(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()

	wh := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Inactive",
		URL:       "https://example.com/webhook",
		Method:    "POST",
		Status:    model.WebhookStatusDisabled,
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	result, err := h.TestWebhook(ctx, &integrationv1.TestWebhookRequest{
		Id:     wh.ID.String(),
		UserId: userID.String(),
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
}

// TestWebhookHandlerEnableDisableErrors проверяет обработку ошибок Enable/Disable.
func TestWebhookHandlerEnableDisableErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepo()
	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("enable not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.EnableWebhook(ctx, &integrationv1.EnableWebhookRequest{
			Id:     uuid.New().String(),
			UserId: userID.String(),
		})
		require.Error(t, err)
	})

	t.Run("disable not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.DisableWebhook(ctx, &integrationv1.DisableWebhookRequest{
			Id:     uuid.New().String(),
			UserId: userID.String(),
		})
		require.Error(t, err)
	})
}

// TestWebhookHandlerGetWebhookStatsError проверяет обработку ошибок GetWebhookStats.
func TestWebhookHandlerGetWebhookStatsError(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepo()
	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	_, err := h.GetWebhookStats(ctx, &integrationv1.GetWebhookStatsRequest{
		Id:     uuid.New().String(),
		UserId: uuid.New().String(),
	})
	require.Error(t, err)
}

// TestWebhookHandlerCloneWebhookError проверяет обработку ошибок CloneWebhook.
func TestWebhookHandlerCloneWebhookError(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepo()
	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	_, err := h.CloneWebhook(ctx, &integrationv1.CloneWebhookRequest{
		Id:      uuid.New().String(),
		UserId:  uuid.New().String(),
		NewName: "Clone",
	})
	require.Error(t, err)
}

// TestWebhookHandlerTestWebhookError проверяет обработку ошибок TestWebhook.
func TestWebhookHandlerTestWebhookError(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepo()
	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	_, err := h.TestWebhook(ctx, &integrationv1.TestWebhookRequest{
		Id:     uuid.New().String(),
		UserId: uuid.New().String(),
	})
	require.Error(t, err)
}

// TestWebhookHandlerCreateWebhookValidationError проверяет обработку ошибок валидации при создании webhook.
func TestWebhookHandlerCreateWebhookValidationError(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepo()
	h := newTestWebhookHandlerWithService(t, repo)
	ctx := context.Background()

	t.Run("empty name returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.CreateWebhook(ctx, &integrationv1.CreateWebhookRequest{
			UserId: uuid.New().String(),
			Name:   "",
			Url:    "https://example.com/webhook",
			Method: "POST",
		})
		require.Error(t, err)
	})

	t.Run("invalid url returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.CreateWebhook(ctx, &integrationv1.CreateWebhookRequest{
			UserId: uuid.New().String(),
			Name:   "My Webhook",
			Url:    "not-a-url",
			Method: "POST",
		})
		require.Error(t, err)
	})

	t.Run("invalid method returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.CreateWebhook(ctx, &integrationv1.CreateWebhookRequest{
			UserId: uuid.New().String(),
			Name:   "My Webhook",
			Url:    "https://example.com/webhook",
			Method: "INVALID",
		})
		require.Error(t, err)
	})
}
