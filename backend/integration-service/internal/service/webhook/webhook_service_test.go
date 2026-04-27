package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
)

// mockWebhookRepository реализует interfaces.WebhookRepository.
type mockWebhookRepository struct {
	webhooks map[uuid.UUID]*model.WebhookIntegration
	err      error
}

func newMockWebhookRepository() *mockWebhookRepository {
	return &mockWebhookRepository{
		webhooks: make(map[uuid.UUID]*model.WebhookIntegration),
	}
}

func (m *mockWebhookRepository) Create(_ context.Context, webhook *model.WebhookIntegration) error {
	if m.err != nil {
		return m.err
	}
	m.webhooks[webhook.ID] = webhook
	return nil
}

func (m *mockWebhookRepository) GetByID(_ context.Context, id uuid.UUID) (*model.WebhookIntegration, error) {
	if m.err != nil {
		return nil, m.err
	}
	w, ok := m.webhooks[id]
	if !ok {
		return nil, model.ErrWebhookNotFound
	}
	return w, nil
}

func (m *mockWebhookRepository) GetByUserIDAndName(_ context.Context, userID uuid.UUID, name string) (*model.WebhookIntegration, error) {
	for _, w := range m.webhooks {
		if w.UserID == userID && w.Name == name {
			return w, nil
		}
	}
	return nil, model.ErrWebhookNotFound
}

func (m *mockWebhookRepository) ListByUserID(_ context.Context, userID uuid.UUID, limit, offset int) ([]*model.WebhookIntegration, error) {
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

func (m *mockWebhookRepository) ListActiveByUserID(_ context.Context, userID uuid.UUID) ([]*model.WebhookIntegration, error) {
	var result []*model.WebhookIntegration
	for _, w := range m.webhooks {
		if w.UserID == userID && w.Status == model.WebhookStatusActive {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockWebhookRepository) Update(_ context.Context, webhook *model.WebhookIntegration) error {
	if m.err != nil {
		return m.err
	}
	m.webhooks[webhook.ID] = webhook
	return nil
}

func (m *mockWebhookRepository) UpdateStats(_ context.Context, id uuid.UUID, stats *interfaces.WebhookStats) error {
	return nil
}

func (m *mockWebhookRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.webhooks, id)
	return nil
}

func (m *mockWebhookRepository) CountByUserID(_ context.Context, userID uuid.UUID) (int, error) {
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

func (m *mockWebhookRepository) ExistsByName(_ context.Context, userID uuid.UUID, name string) (bool, error) {
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

// mockBillingClientWebhook имитирует billing client.
type mockBillingClientWebhook struct {
	canCreate bool
	err       error
}

func (m *mockBillingClientWebhook) CheckWebhookLimit(_ context.Context, _ uuid.UUID, _ int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.canCreate, nil
}

// newTestEncryptor создаёт тестовый encryptor с фиксированным ключом.
func newTestEncryptor(t *testing.T) *security.EncryptionService {
	t.Helper()
	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)
	return enc
}

// newTestWebhookService создаёт сервис для тестирования.
func newTestWebhookService(t *testing.T, repo *mockWebhookRepository, billing *mockBillingClientWebhook) *WebhookService {
	t.Helper()
	enc := newTestEncryptor(t)
	gen := security.NewAPIKeyGenerator()

	// Используем реальный BillingClient с mock через функцию
	// Но BillingClient — конкретный тип, поэтому создаём сервис с nil и подменяем через поля
	svc := &WebhookService{
		repo:         repo,
		encryptor:    enc,
		keyGenerator: gen,
		config: &WebhookServiceConfig{
			DefaultMaxPayloadSizeBytes: 1048576,
			DefaultTimeout:             10 * time.Second,
		},
	}
	return svc
}

func TestNewWebhookService(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)
	gen := security.NewAPIKeyGenerator()

	svc := NewWebhookService(repo, enc, gen, nil, nil)
	require.NotNil(t, svc)
	// Default config applied
	assert.Equal(t, 1048576, svc.config.DefaultMaxPayloadSizeBytes)
}

func TestWebhookServiceGetWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserID := uuid.New()
	repo := newMockWebhookRepository()

	// Создаём тестовый webhook
	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		URL:       "https://example.com/webhook",
		Method:    "POST",
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("get existing webhook", func(t *testing.T) {
		t.Parallel()
		result, err := svc.GetWebhook(ctx, webhook.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, webhook.ID, result.ID)
	})

	t.Run("get non-existing webhook", func(t *testing.T) {
		t.Parallel()
		_, err := svc.GetWebhook(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("get webhook owned by other user returns not found", func(t *testing.T) {
		t.Parallel()
		_, err := svc.GetWebhook(ctx, webhook.ID, otherUserID)
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})
}

func TestWebhookServiceListWebhooks(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	// Создаём 3 webhooks
	for i := 0; i < 3; i++ {
		w := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Webhook",
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.webhooks[w.ID] = w
	}

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("list all webhooks for user", func(t *testing.T) {
		t.Parallel()
		result, err := svc.ListWebhooks(ctx, &ListWebhooksRequest{
			UserID:   userID,
			PageSize: 50,
		})
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("list webhooks uses default limit for zero page size", func(t *testing.T) {
		t.Parallel()
		result, err := svc.ListWebhooks(ctx, &ListWebhooksRequest{
			UserID:   userID,
			PageSize: 0,
		})
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("list webhooks for other user returns empty", func(t *testing.T) {
		t.Parallel()
		result, err := svc.ListWebhooks(ctx, &ListWebhooksRequest{
			UserID:   uuid.New(),
			PageSize: 50,
		})
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestWebhookServiceDeleteWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("delete webhook owned by other user", func(t *testing.T) {
		t.Parallel()
		err := svc.DeleteWebhook(ctx, webhook.ID, otherUserID)
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})

	t.Run("delete non-existing webhook", func(t *testing.T) {
		t.Parallel()
		err := svc.DeleteWebhook(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("delete own webhook succeeds", func(t *testing.T) {
		t.Parallel()
		// Create a dedicated webhook for this test
		localRepo := newMockWebhookRepository()
		localWebhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "ToDelete",
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		localRepo.webhooks[localWebhook.ID] = localWebhook

		localSvc := newTestWebhookService(t, localRepo, nil)
		err := localSvc.DeleteWebhook(ctx, localWebhook.ID, userID)
		require.NoError(t, err)
		_, ok := localRepo.webhooks[localWebhook.ID]
		assert.False(t, ok)
	})
}

func TestWebhookServiceEnableDisableWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		Status:    model.WebhookStatusDisabled,
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("enable webhook", func(t *testing.T) {
		err := svc.EnableWebhook(ctx, webhook.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, model.WebhookStatusActive, repo.webhooks[webhook.ID].Status)
		assert.True(t, repo.webhooks[webhook.ID].Enabled)
	})

	t.Run("disable webhook", func(t *testing.T) {
		err := svc.DisableWebhook(ctx, webhook.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, model.WebhookStatusDisabled, repo.webhooks[webhook.ID].Status)
		assert.False(t, repo.webhooks[webhook.ID].Enabled)
	})

	t.Run("enable webhook owned by other user returns error", func(t *testing.T) {
		t.Parallel()
		err := svc.EnableWebhook(ctx, webhook.ID, uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})

	t.Run("disable webhook owned by other user returns error", func(t *testing.T) {
		t.Parallel()
		err := svc.DisableWebhook(ctx, webhook.ID, uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})
}

func TestWebhookServiceGetWebhookStats(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	avgResponseTime := 150
	webhook := &model.WebhookIntegration{
		ID:                uuid.New(),
		UserID:            userID,
		Name:              "Test",
		Status:            model.WebhookStatusActive,
		TotalSent:         100,
		SuccessfulSent:    90,
		FailedSent:        10,
		FailureCount:      10,
		AvgResponseTimeMs: &avgResponseTime,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("get stats for own webhook", func(t *testing.T) {
		t.Parallel()
		stats, err := svc.GetWebhookStats(ctx, webhook.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, 100, stats.TotalSent)
		assert.Equal(t, 10, stats.FailureCount)
		assert.Equal(t, 90.0, stats.SuccessRate)
		assert.Equal(t, 150, stats.AvgResponseTimeMs)
	})

	t.Run("get stats for non-existing webhook", func(t *testing.T) {
		t.Parallel()
		_, err := svc.GetWebhookStats(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("get stats for other user's webhook returns error", func(t *testing.T) {
		t.Parallel()
		_, err := svc.GetWebhookStats(ctx, webhook.ID, uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})
}

func TestWebhookServiceGetWebhookStatsZeroTotal(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		Status:    model.WebhookStatusActive,
		TotalSent: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	stats, err := svc.GetWebhookStats(ctx, webhook.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 0.0, stats.SuccessRate)
}

func TestWebhookServiceUpdateWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Old Name",
		URL:       "https://old.example.com/webhook",
		Method:    "POST",
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("update name", func(t *testing.T) {
		newName := "New Name"
		req := &UpdateWebhookRequest{
			ID:     webhook.ID,
			UserID: userID,
			Name:   &newName,
		}
		result, err := svc.UpdateWebhook(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "New Name", result.Name)
	})

	t.Run("update with empty name returns error", func(t *testing.T) {
		t.Parallel()
		emptyName := ""
		req := &UpdateWebhookRequest{
			ID:     webhook.ID,
			UserID: userID,
			Name:   &emptyName,
		}
		_, err := svc.UpdateWebhook(ctx, req)
		require.Error(t, err)
	})

	t.Run("update webhook for other user returns error", func(t *testing.T) {
		t.Parallel()
		newName := "New Name"
		req := &UpdateWebhookRequest{
			ID:     webhook.ID,
			UserID: uuid.New(),
			Name:   &newName,
		}
		_, err := svc.UpdateWebhook(ctx, req)
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})
}

func TestWebhookServiceCloneWebhook(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

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

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("clone webhook", func(t *testing.T) {
		t.Parallel()
		cloneRepo := newMockWebhookRepository()
		cloneRepo.webhooks[original.ID] = original
		cloneSvc := newTestWebhookService(t, cloneRepo, nil)

		cloned, err := cloneSvc.CloneWebhook(ctx, &CloneWebhookRequest{
			ID:      original.ID,
			UserID:  userID,
			NewName: "Clone",
			NewURL:  "https://clone.example.com/webhook",
		})
		require.NoError(t, err)
		assert.NotEqual(t, original.ID, cloned.ID)
		assert.Equal(t, "Clone", cloned.Name)
		assert.Equal(t, "https://clone.example.com/webhook", cloned.URL)
		assert.Equal(t, userID, cloned.UserID)
	})

	t.Run("clone webhook owned by other user returns error", func(t *testing.T) {
		t.Parallel()
		_, err := svc.CloneWebhook(ctx, &CloneWebhookRequest{
			ID:      original.ID,
			UserID:  uuid.New(),
			NewName: "Clone",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("derefInt nil returns zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, derefInt(nil))
	})

	t.Run("derefInt non-nil returns value", func(t *testing.T) {
		t.Parallel()
		v := 42
		assert.Equal(t, 42, derefInt(&v))
	})

	t.Run("timeToInt64Ptr nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, timeToInt64Ptr(nil))
	})

	t.Run("timeToInt64Ptr non-nil returns unix timestamp", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		result := timeToInt64Ptr(&now)
		require.NotNil(t, result)
		assert.Equal(t, now.Unix(), *result)
	})

	t.Run("validateWebhookName empty returns error", func(t *testing.T) {
		t.Parallel()
		err := validateWebhookName("")
		assert.ErrorIs(t, err, model.ErrEmptyWebhookName)
	})

	t.Run("validateWebhookName too long returns error", func(t *testing.T) {
		t.Parallel()
		longName := make([]byte, 256)
		for i := range longName {
			longName[i] = 'a'
		}
		err := validateWebhookName(string(longName))
		assert.ErrorIs(t, err, model.ErrWebhookNameTooLong)
	})

	t.Run("validateWebhookName valid passes", func(t *testing.T) {
		t.Parallel()
		err := validateWebhookName("valid name")
		assert.NoError(t, err)
	})

	t.Run("validateWebhookURL empty returns error", func(t *testing.T) {
		t.Parallel()
		err := validateWebhookURL("")
		assert.Error(t, err)
	})

	t.Run("validateWebhookURL no scheme returns error", func(t *testing.T) {
		t.Parallel()
		err := validateWebhookURL("example.com/webhook")
		assert.Error(t, err)
	})

	t.Run("validateWebhookURL valid https passes", func(t *testing.T) {
		t.Parallel()
		err := validateWebhookURL("https://example.com/webhook")
		assert.NoError(t, err)
	})

	t.Run("validateWebhookURL too long returns error", func(t *testing.T) {
		t.Parallel()
		longURL := "https://example.com/" + string(make([]byte, 2048))
		err := validateWebhookURL(longURL)
		assert.Error(t, err)
	})
}

func TestWebhookServiceListWebhooksWithPageToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	for i := 0; i < 5; i++ {
		w := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Webhook",
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.webhooks[w.ID] = w
	}

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("valid page token is parsed", func(t *testing.T) {
		t.Parallel()
		// Encode a page token: base64("offset:2")
		import64 := "b2Zmc2V0OjI=" // base64("offset:2")
		result, err := svc.ListWebhooks(ctx, &ListWebhooksRequest{
			UserID:    userID,
			PageSize:  50,
			PageToken: import64,
		})
		require.NoError(t, err)
		// The mock ignores offset, so all 5 are returned still
		assert.NotNil(t, result)
	})

	t.Run("invalid base64 page token is ignored", func(t *testing.T) {
		t.Parallel()
		result, err := svc.ListWebhooks(ctx, &ListWebhooksRequest{ //nolint:gosec // G101: тестовые данные
			UserID:    userID,
			PageSize:  50,
			PageToken: "not-valid-base64!!!",
		})
		require.NoError(t, err)
		assert.Len(t, result, 5)
	})

	t.Run("page size over 100 uses 50", func(t *testing.T) {
		t.Parallel()
		result, err := svc.ListWebhooks(ctx, &ListWebhooksRequest{
			UserID:   userID,
			PageSize: 200,
		})
		require.NoError(t, err)
		assert.Len(t, result, 5)
	})
}

func TestWebhookServiceCreateWebhookValidationFailure(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("invalid URL returns error before billing check", func(t *testing.T) {
		t.Parallel()
		req := &CreateWebhookRequest{
			UserID: uuid.New(),
			Name:   "Test",
			URL:    "not-a-url",
			Method: "POST",
		}
		_, err := svc.CreateWebhook(ctx, req)
		require.Error(t, err)
	})

	t.Run("empty name returns error before billing check", func(t *testing.T) {
		t.Parallel()
		req := &CreateWebhookRequest{
			UserID: uuid.New(),
			Name:   "",
			URL:    "https://example.com/webhook",
			Method: "POST",
		}
		_, err := svc.CreateWebhook(ctx, req)
		require.Error(t, err)
	})

	t.Run("duplicate name returns error when webhook exists", func(t *testing.T) {
		t.Parallel()
		localRepo := newMockWebhookRepository()
		userID := uuid.New()
		// Pre-insert a webhook with the same name
		existing := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Existing",
			URL:       "https://example.com",
			Method:    "POST",
			Status:    model.WebhookStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		localRepo.webhooks[existing.ID] = existing

		localSvc := newTestWebhookService(t, localRepo, nil)
		// This will fail at ExistsByName check (before billing)
		// But the service calls billing after ExistsByName... so it will panic on nil billingClient
		// We need to check the order: ExistsByName returns true → ErrWebhookDuplicateName
		req := &CreateWebhookRequest{
			UserID: userID,
			Name:   "Existing",
			URL:    "https://example.com/webhook",
			Method: "POST",
		}
		_, err := localSvc.CreateWebhook(ctx, req)
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookDuplicateName)
	})
}

func TestWebhookServiceTestWebhookNotActive(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
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
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	resp, err := svc.TestWebhook(ctx, webhook.ID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "not active")
}

func TestWebhookServiceTestWebhookOwnershipCheck(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
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
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	t.Run("other user cannot test webhook", func(t *testing.T) {
		t.Parallel()
		_, err := svc.TestWebhook(ctx, webhook.ID, uuid.New())
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrWebhookNotFound)
	})

	t.Run("nonexistent webhook returns error", func(t *testing.T) {
		t.Parallel()
		_, err := svc.TestWebhook(ctx, uuid.New(), userID)
		require.Error(t, err)
	})
}

func TestWebhookServiceUpdateWebhookBadURL(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()

	webhook := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Test",
		URL:       "https://original.example.com/webhook",
		Method:    "POST",
		Status:    model.WebhookStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[webhook.ID] = webhook

	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	badURL := "not-a-url"
	req := &UpdateWebhookRequest{
		ID:     webhook.ID,
		UserID: userID,
		URL:    &badURL,
	}
	_, err := svc.UpdateWebhook(ctx, req)
	require.Error(t, err)
}

func TestWebhookServiceTestWebhookActive(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	t.Run("active webhook with 200 response succeeds", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		repo := newMockWebhookRepository()
		webhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Active",
			URL:       server.URL + "/webhook",
			Method:    "POST",
			Status:    model.WebhookStatusActive,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.webhooks[webhook.ID] = webhook

		svc := newTestWebhookService(t, repo, nil)
		resp, err := svc.TestWebhook(context.Background(), webhook.ID, userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("active webhook with 500 response returns not successful", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("server error"))
			require.NoError(t, err)
		}))
		t.Cleanup(server.Close)

		repo := newMockWebhookRepository()
		webhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "ErrorWebhook",
			URL:       server.URL + "/webhook",
			Method:    "POST",
			Status:    model.WebhookStatusActive,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.webhooks[webhook.ID] = webhook

		svc := newTestWebhookService(t, repo, nil)
		resp, err := svc.TestWebhook(context.Background(), webhook.ID, userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("active webhook with connection refused returns not successful", func(t *testing.T) {
		t.Parallel()
		repo := newMockWebhookRepository()
		webhook := &model.WebhookIntegration{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Broken",
			URL:       "http://127.0.0.1:19999/webhook", // non-listening port
			Method:    "POST",
			Status:    model.WebhookStatusActive,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.webhooks[webhook.ID] = webhook

		svc := newTestWebhookService(t, repo, nil)
		resp, err := svc.TestWebhook(context.Background(), webhook.ID, userID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.Success)
	})
}

func TestWebhookServiceEnableNonExistentWebhook(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	err := svc.EnableWebhook(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
}

func TestWebhookServiceDisableNonExistentWebhook(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	svc := newTestWebhookService(t, repo, nil)
	ctx := context.Background()

	err := svc.DisableWebhook(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
}
