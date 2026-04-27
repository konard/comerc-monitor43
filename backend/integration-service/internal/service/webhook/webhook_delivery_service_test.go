package webhook

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpclient "github.com/raul/monitor/backend/integration-service/internal/infrastructure/httpclient"
	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
	"github.com/raul/monitor/backend/integration-service/internal/service/webhook/dto"
)

func TestNewWebhookDeliveryService(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	svc := NewWebhookDeliveryService(repo, nil, nil, nil)
	assert.NotNil(t, svc)
	assert.Equal(t, 10, svc.config.MaxWorkers)
	assert.Equal(t, 3, svc.config.MaxRetries)
}

func TestNewWebhookDeliveryServiceWithCustomConfig(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	config := &WebhookDeliveryConfig{
		MaxWorkers:         5,
		WorkerQueueSize:    500,
		MaxRetries:         2,
		RetryCheckInterval: 10 * time.Second,
		HTTPTimeout:        5 * time.Second,
	}
	svc := NewWebhookDeliveryService(repo, nil, nil, config)
	assert.NotNil(t, svc)
	assert.Equal(t, 5, svc.config.MaxWorkers)
	assert.Equal(t, 2, svc.config.MaxRetries)
}

func TestShouldRetry(t *testing.T) {
	t.Parallel()

	svc := &WebhookDeliveryService{
		config: &WebhookDeliveryConfig{MaxRetries: 3},
	}

	t.Run("timeout error should retry", func(t *testing.T) {
		t.Parallel()
		assert.True(t, svc.shouldRetry(model.ErrWebhookTimeout))
	})

	t.Run("5xx error should retry", func(t *testing.T) {
		t.Parallel()
		assert.True(t, svc.shouldRetry(model.ErrWebhook5xxError))
	})

	t.Run("auth failed should not retry", func(t *testing.T) {
		t.Parallel()
		assert.False(t, svc.shouldRetry(model.ErrWebhookAuthFailed))
	})

	t.Run("not found should not retry", func(t *testing.T) {
		t.Parallel()
		assert.False(t, svc.shouldRetry(model.ErrWebhookInvalidURL))
	})

	t.Run("permanent error should not retry", func(t *testing.T) {
		t.Parallel()
		assert.False(t, svc.shouldRetry(model.ErrDeliveryPermanentError))
	})
}

func TestBuildWebhookPayload(t *testing.T) {
	t.Parallel()

	svc := &WebhookDeliveryService{}

	alertID := uuid.New()
	monitorID := uuid.New()
	now := time.Now()

	event := &dto.AlertEvent{
		AlertID:             alertID,
		MonitorID:           monitorID,
		MonitorName:         "Test Monitor",
		MonitorStatus:       "down",
		Severity:            "critical",
		ConsecutiveFailures: 5,
		IsFlapping:          false,
		TriggeredAt:         now,
	}

	payload := svc.buildWebhookPayload(event)

	assert.Equal(t, alertID.String(), payload["alert_id"])
	assert.Equal(t, monitorID.String(), payload["monitor_id"])
	assert.Equal(t, "Test Monitor", payload["monitor_name"])
	assert.Equal(t, "down", payload["monitor_status"])
	assert.Equal(t, "critical", payload["severity"])
	assert.Equal(t, 5, payload["consecutive_failures"])
	assert.Equal(t, false, payload["is_flapping"])
	// flap_count should NOT be present when not flapping
	_, hasFlap := payload["flap_count"]
	assert.False(t, hasFlap)
}

func TestBuildWebhookPayloadFlapping(t *testing.T) {
	t.Parallel()

	svc := &WebhookDeliveryService{}

	event := &dto.AlertEvent{
		AlertID:     uuid.New(),
		MonitorID:   uuid.New(),
		IsFlapping:  true,
		FlapCount:   3,
		TriggeredAt: time.Now(),
	}

	payload := svc.buildWebhookPayload(event)

	assert.True(t, payload["is_flapping"].(bool))
	assert.Equal(t, 3, payload["flap_count"])
}

func TestSortByPriority(t *testing.T) {
	t.Parallel()

	webhooks := []*model.WebhookIntegration{
		{ID: uuid.New(), Priority: model.WebhookPriorityLow},
		{ID: uuid.New(), Priority: model.WebhookPriorityNormal},
		{ID: uuid.New(), Priority: model.WebhookPriorityHigh},
		{ID: uuid.New(), Priority: model.WebhookPriorityNormal},
	}

	sortByPriority(webhooks)

	assert.Equal(t, model.WebhookPriorityHigh, webhooks[0].Priority)
	assert.Equal(t, model.WebhookPriorityNormal, webhooks[1].Priority)
	assert.Equal(t, model.WebhookPriorityNormal, webhooks[2].Priority)
	assert.Equal(t, model.WebhookPriorityLow, webhooks[3].Priority)
}

func TestSortByPriorityEmpty(t *testing.T) {
	t.Parallel()

	// Should not panic on empty slice
	var webhooks []*model.WebhookIntegration
	sortByPriority(webhooks)
	assert.Empty(t, webhooks)
}

func TestSortByPrioritySingle(t *testing.T) {
	t.Parallel()

	webhooks := []*model.WebhookIntegration{
		{ID: uuid.New(), Priority: model.WebhookPriorityHigh},
	}
	sortByPriority(webhooks)
	assert.Len(t, webhooks, 1)
}

// mockDeliveryRepository реализует interfaces.WebhookDeliveryRepository.
type mockDeliveryRepository struct {
	attempts map[uuid.UUID]*model.WebhookDeliveryAttempt
	err      error
}

func newMockDeliveryRepository() *mockDeliveryRepository {
	return &mockDeliveryRepository{
		attempts: make(map[uuid.UUID]*model.WebhookDeliveryAttempt),
	}
}

func (m *mockDeliveryRepository) Create(_ context.Context, attempt *model.WebhookDeliveryAttempt) error {
	if m.err != nil {
		return m.err
	}
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *mockDeliveryRepository) GetByID(_ context.Context, id uuid.UUID) (*model.WebhookDeliveryAttempt, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, ok := m.attempts[id]
	if !ok {
		return nil, model.ErrWebhookNotFound
	}
	return a, nil
}

func (m *mockDeliveryRepository) ListByWebhookID(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.WebhookDeliveryAttempt, error) {
	return nil, nil
}

func (m *mockDeliveryRepository) ListPendingForRetry(_ context.Context, _ int) ([]*model.WebhookDeliveryAttempt, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*model.WebhookDeliveryAttempt
	for _, a := range m.attempts {
		if a.Status == "retry_scheduled" {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockDeliveryRepository) Update(_ context.Context, attempt *model.WebhookDeliveryAttempt) error {
	if m.err != nil {
		return m.err
	}
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *mockDeliveryRepository) DeleteOldAttempts(_ context.Context, _ int) (int64, error) {
	return 0, nil
}

func TestWebhookDeliveryServiceStartStop(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	svc := NewWebhookDeliveryService(repo, nil, nil, &WebhookDeliveryConfig{
		MaxWorkers:         2,
		WorkerQueueSize:    10,
		MaxRetries:         3,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 100 * time.Millisecond,
		HTTPTimeout:        1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := svc.Start(ctx)
	require.NoError(t, err)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(stopCancel)
	err = svc.Stop(stopCtx)
	require.NoError(t, err)
}

func TestWebhookDeliveryServiceProcessAlertEventNoWebhooks(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	svc := NewWebhookDeliveryService(repo, deliveryRepo, nil, &WebhookDeliveryConfig{
		MaxWorkers:         2,
		WorkerQueueSize:    10,
		MaxRetries:         3,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 100 * time.Millisecond,
		HTTPTimeout:        1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	require.NoError(t, svc.Start(ctx))
	stopWebhookDeliveryService(t, svc)

	event := &dto.AlertEvent{
		AlertID:   uuid.New(),
		MonitorID: uuid.New(),
		UserID:    uuid.New(), // No webhooks for this user
		Severity:  "critical",
	}

	err := svc.ProcessAlertEvent(ctx, event)
	require.NoError(t, err)
}

func TestWebhookDeliveryServiceProcessAlertEventWithWebhooks(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()

	// Add an active webhook for the user
	wh := &model.WebhookIntegration{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           "Test",
		URL:            "https://example.com/webhook",
		Method:         "POST",
		Status:         model.WebhookStatusActive,
		Enabled:        true,
		SeverityFilter: []string{"critical"},
		Priority:       model.WebhookPriorityNormal,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	svc := NewWebhookDeliveryService(repo, deliveryRepo, nil, &WebhookDeliveryConfig{
		MaxWorkers:         1,
		WorkerQueueSize:    10,
		MaxRetries:         0,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 1 * time.Second,
		HTTPTimeout:        100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	require.NoError(t, svc.Start(ctx))
	stopWebhookDeliveryService(t, svc)

	event := &dto.AlertEvent{
		AlertID:   uuid.New(),
		MonitorID: uuid.New(),
		UserID:    userID,
		Severity:  "critical",
	}

	err := svc.ProcessAlertEvent(ctx, event)
	require.NoError(t, err)
	// Give workers time to process
	time.Sleep(200 * time.Millisecond)
}

func TestWebhookDeliveryServiceProcessAlertEventSeverityFiltered(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()

	// Add a webhook that only accepts "critical"
	wh := &model.WebhookIntegration{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           "Critical Only",
		URL:            "https://example.com/webhook",
		Method:         "POST",
		Status:         model.WebhookStatusActive,
		Enabled:        true,
		SeverityFilter: []string{"critical"},
		Priority:       model.WebhookPriorityNormal,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	svc := NewWebhookDeliveryService(repo, deliveryRepo, nil, &WebhookDeliveryConfig{
		MaxWorkers:         1,
		WorkerQueueSize:    10,
		MaxRetries:         0,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 1 * time.Second,
		HTTPTimeout:        100 * time.Millisecond,
	})

	ctx := context.Background()
	require.NoError(t, svc.Start(ctx))
	stopWebhookDeliveryService(t, svc)

	// Send "warning" severity - should be filtered out
	event := &dto.AlertEvent{
		AlertID:   uuid.New(),
		MonitorID: uuid.New(),
		UserID:    userID,
		Severity:  "warning",
	}

	err := svc.ProcessAlertEvent(ctx, event)
	require.NoError(t, err)
	// No delivery attempts created since severity didn't match
	assert.Len(t, deliveryRepo.attempts, 0)
}

func TestWebhookDeliveryServiceUpdateWebhookFailureStats(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()

	wh := &model.WebhookIntegration{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Test",
		Status:    model.WebhookStatusActive,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	svc := &WebhookDeliveryService{
		webhookRepo: repo,
		log:         slog.Default(),
	}

	svc.updateWebhookFailureStats(context.Background(), wh)

	// After calling MarkAsFailed, the webhook should be updated
	assert.Equal(t, 1, wh.FailureCount)
}

func TestWebhookDeliveryServiceScheduleRetry(t *testing.T) {
	t.Parallel()

	deliveryRepo := newMockDeliveryRepository()

	attempt := model.NewWebhookDeliveryAttempt(uuid.New(), uuid.New())
	deliveryRepo.attempts[attempt.ID] = attempt

	svc := &WebhookDeliveryService{
		deliveryRepo: deliveryRepo,
		config: &WebhookDeliveryConfig{
			MaxRetries:     3,
			RetryIntervals: []time.Duration{30 * time.Second, 60 * time.Second},
		},
		log: slog.Default(),
	}

	wh := &model.WebhookIntegration{ID: uuid.New()}
	err := svc.scheduleRetry(context.Background(), wh, attempt, "TIMEOUT", "Connection timed out")
	require.NoError(t, err)

	assert.Equal(t, "failed", attempt.Status)
	assert.NotNil(t, attempt.NextRetryAt)
}

func newTestDeliveryService(t *testing.T, webhookRepo *mockWebhookRepository, deliveryRepo *mockDeliveryRepository, httpClient *httpclient.Client) *WebhookDeliveryService {
	t.Helper()
	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)
	return &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		encryptor:    enc,
		httpClient:   httpClient,
		config: &WebhookDeliveryConfig{
			MaxRetries:     3,
			RetryIntervals: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
		},
		log: slog.Default(),
	}
}

func TestDeliverWebhookSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	httpClient := httpclient.NewClient(5 * time.Second)

	svc := newTestDeliveryService(t, webhookRepo, deliveryRepo, httpClient)

	wh := &model.WebhookIntegration{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		Name:                "Test",
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Status:              model.WebhookStatusActive,
		MaxPayloadSizeBytes: 1048576,
	}
	webhookRepo.webhooks[wh.ID] = wh

	attempt := model.NewWebhookDeliveryAttempt(wh.ID, uuid.New())
	deliveryRepo.attempts[attempt.ID] = attempt

	event := &dto.AlertEvent{
		AlertID:     uuid.New(),
		MonitorID:   uuid.New(),
		TriggeredAt: time.Now(),
	}

	job := &WebhookDeliveryJob{
		Webhook: wh,
		Attempt: attempt,
		Event:   event,
	}

	err := svc.DeliverWebhook(context.Background(), job)
	require.NoError(t, err)
	assert.Equal(t, "sent", attempt.Status)
}

func TestDeliverWebhookWithEncryptedSecret(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)

	encryptedSecret, err := enc.EncryptWebhookSecret("my-secret")
	require.NoError(t, err)

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	httpClient := httpclient.NewClient(5 * time.Second)

	svc := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		encryptor:    enc,
		httpClient:   httpClient,
		config: &WebhookDeliveryConfig{
			MaxRetries:     3,
			RetryIntervals: []time.Duration{100 * time.Millisecond},
		},
		log: slog.Default(),
	}

	wh := &model.WebhookIntegration{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		Name:                "Test",
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Status:              model.WebhookStatusActive,
		SecretKey:           &encryptedSecret,
		MaxPayloadSizeBytes: 1048576,
	}
	webhookRepo.webhooks[wh.ID] = wh

	attempt := model.NewWebhookDeliveryAttempt(wh.ID, uuid.New())
	deliveryRepo.attempts[attempt.ID] = attempt

	event := &dto.AlertEvent{
		AlertID:     uuid.New(),
		MonitorID:   uuid.New(),
		TriggeredAt: time.Now(),
	}

	job := &WebhookDeliveryJob{Webhook: wh, Attempt: attempt, Event: event}
	err = svc.DeliverWebhook(context.Background(), job)
	require.NoError(t, err)
}

func TestDeliverWebhookWithBadEncryptedSecret(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	httpClient := httpclient.NewClient(5 * time.Second)

	svc := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		encryptor:    enc,
		httpClient:   httpClient,
		config: &WebhookDeliveryConfig{
			MaxRetries:     0,
			RetryIntervals: []time.Duration{100 * time.Millisecond},
		},
		log: slog.Default(),
	}

	badSecret := "not-valid-encrypted-data" //nolint:gosec // G101: тестовые данные
	wh := &model.WebhookIntegration{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		Name:                "Test",
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Status:              model.WebhookStatusActive,
		SecretKey:           &badSecret,
		MaxPayloadSizeBytes: 1048576,
	}
	webhookRepo.webhooks[wh.ID] = wh

	attempt := model.NewWebhookDeliveryAttempt(wh.ID, uuid.New())
	deliveryRepo.attempts[attempt.ID] = attempt

	event := &dto.AlertEvent{
		AlertID:     uuid.New(),
		MonitorID:   uuid.New(),
		TriggeredAt: time.Now(),
	}

	// Continues without secret key (warns and continues)
	job := &WebhookDeliveryJob{Webhook: wh, Attempt: attempt, Event: event}
	err = svc.DeliverWebhook(context.Background(), job)
	require.NoError(t, err)
}

func TestDeliverWebhookPermanentError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	httpClient := httpclient.NewClient(5 * time.Second)

	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)

	svc := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		encryptor:    enc,
		httpClient:   httpClient,
		config: &WebhookDeliveryConfig{
			MaxRetries:     3,
			RetryIntervals: []time.Duration{100 * time.Millisecond},
		},
		log: slog.Default(),
	}

	wh := &model.WebhookIntegration{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		Name:                "Test",
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Status:              model.WebhookStatusActive,
		MaxPayloadSizeBytes: 1048576,
	}
	webhookRepo.webhooks[wh.ID] = wh

	attempt := model.NewWebhookDeliveryAttempt(wh.ID, uuid.New())
	deliveryRepo.attempts[attempt.ID] = attempt

	job := &WebhookDeliveryJob{Webhook: wh, Attempt: attempt, Event: &dto.AlertEvent{
		AlertID: uuid.New(), MonitorID: uuid.New(), TriggeredAt: time.Now(),
	}}

	err = svc.DeliverWebhook(context.Background(), job)
	require.Error(t, err)
	assert.Equal(t, "failed", attempt.Status)
}

func TestDeliverWebhookRetryableError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	httpClient := httpclient.NewClient(5 * time.Second)

	enc, err := security.NewEncryptionService("12345678901234567890123456789012")
	require.NoError(t, err)

	svc := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		encryptor:    enc,
		httpClient:   httpClient,
		config: &WebhookDeliveryConfig{
			MaxRetries:     3,
			RetryIntervals: []time.Duration{100 * time.Millisecond},
		},
		log: slog.Default(),
	}

	wh := &model.WebhookIntegration{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		Name:                "Test",
		URL:                 server.URL + "/webhook",
		Method:              "POST",
		Status:              model.WebhookStatusActive,
		MaxPayloadSizeBytes: 1048576,
	}
	webhookRepo.webhooks[wh.ID] = wh

	attempt := model.NewWebhookDeliveryAttempt(wh.ID, uuid.New())
	deliveryRepo.attempts[attempt.ID] = attempt

	job := &WebhookDeliveryJob{Webhook: wh, Attempt: attempt, Event: &dto.AlertEvent{
		AlertID: uuid.New(), MonitorID: uuid.New(), TriggeredAt: time.Now(),
	}}

	// Returns nil because the retry was scheduled
	err = svc.DeliverWebhook(context.Background(), job)
	require.NoError(t, err)
}

func TestProcessRetryQueueListError(t *testing.T) {
	t.Parallel()

	deliveryRepo := newMockDeliveryRepository()
	deliveryRepo.err = assert.AnError

	svc := &WebhookDeliveryService{
		deliveryRepo: deliveryRepo,
		config:       &WebhookDeliveryConfig{MaxRetries: 3},
		log:          slog.Default(),
	}

	// Should not panic on error
	svc.processRetryQueue(context.Background())
}

func TestProcessRetryQueueMaxRetriesExceeded(t *testing.T) {
	t.Parallel()

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()

	attempt := model.NewWebhookDeliveryAttempt(uuid.New(), uuid.New())
	attempt.Status = "retry_scheduled"
	attempt.RetryCount = 3 // Already at max
	deliveryRepo.attempts[attempt.ID] = attempt

	svc := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		config:       &WebhookDeliveryConfig{MaxRetries: 3},
		log:          slog.Default(),
	}

	svc.processRetryQueue(context.Background())

	// Attempt should be marked as failed
	assert.Equal(t, "failed", attempt.Status)
}

func TestProcessRetryQueueWebhookNotFound(t *testing.T) {
	t.Parallel()

	webhookRepo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()

	attempt := model.NewWebhookDeliveryAttempt(uuid.New(), uuid.New())
	attempt.Status = "retry_scheduled"
	attempt.RetryCount = 0
	now := time.Now().Add(-time.Minute) // Past time, ready for retry
	attempt.NextRetryAt = &now
	deliveryRepo.attempts[attempt.ID] = attempt
	// Webhook not in repo, so GetByID will return error

	svc := &WebhookDeliveryService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		config:       &WebhookDeliveryConfig{MaxRetries: 3},
		log:          slog.Default(),
	}

	// Should not panic when webhook not found
	svc.processRetryQueue(context.Background())
}

func TestWebhookDeliveryServiceProcessAlertEventDeliveryRepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repo := newMockWebhookRepository()
	deliveryRepo := newMockDeliveryRepository()
	deliveryRepo.err = assert.AnError // Create fails

	// Add an active webhook for the user
	wh := &model.WebhookIntegration{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           "Test",
		URL:            "https://example.com/webhook",
		Method:         "POST",
		Status:         model.WebhookStatusActive,
		Enabled:        true,
		SeverityFilter: []string{"critical"},
		Priority:       model.WebhookPriorityNormal,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.webhooks[wh.ID] = wh

	svc := NewWebhookDeliveryService(repo, deliveryRepo, nil, &WebhookDeliveryConfig{
		MaxWorkers:         1,
		WorkerQueueSize:    10,
		MaxRetries:         0,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 1 * time.Second,
		HTTPTimeout:        100 * time.Millisecond,
	})

	ctx := context.Background()
	require.NoError(t, svc.Start(ctx))
	stopWebhookDeliveryService(t, svc)

	event := &dto.AlertEvent{
		AlertID:   uuid.New(),
		MonitorID: uuid.New(),
		UserID:    userID,
		Severity:  "critical",
	}

	// Should succeed (errors are logged but not returned)
	err := svc.ProcessAlertEvent(ctx, event)
	require.NoError(t, err)
}

func TestUpdateWebhookFailureStatsUpdateError(t *testing.T) {
	t.Parallel()

	repo := newMockWebhookRepository()
	repo.err = assert.AnError // Make Update fail

	wh := &model.WebhookIntegration{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Name:   "Test",
	}
	// Don't add to repo - Update will fail

	svc := &WebhookDeliveryService{
		webhookRepo: repo,
		log:         slog.Default(),
	}

	// Should not panic even when update fails
	svc.updateWebhookFailureStats(context.Background(), wh)
}
