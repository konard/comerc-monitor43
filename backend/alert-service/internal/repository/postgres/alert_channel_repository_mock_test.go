package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/alert-service/internal/model"
)

// ─── AlertChannelRepository.GetByID ──────────────────────────────────────────

func TestAlertChannelRepository_GetByID_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	telegramCfg, err := json.Marshal(model.TelegramChannelConfig{ChatID: "12345", BotToken: "tok"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).AddRow(
		id, userID, model.AlertChannelTypeTelegram, model.AlertChannelStatusActive,
		true, true, 0, nil, telegramCfg, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(id.String()).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), id.String())
	require.NoError(t, err)
	assert.Equal(t, id, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, model.AlertChannelTypeTelegram, result.Type)
	assert.Equal(t, model.AlertChannelStatusActive, result.Status)
	require.NotNil(t, result.TelegramConfig)
	assert.Equal(t, "12345", result.TelegramConfig.ChatID)
}

func TestAlertChannelRepository_GetByID_WithEmailConfig(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	emailCfg, err := json.Marshal(model.EmailChannelConfig{Email: "test@example.com"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).AddRow(
		id, userID, model.AlertChannelTypeEmail, model.AlertChannelStatusActive,
		true, true, 0, nil, nil, emailCfg, nil, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(id.String()).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), id.String())
	require.NoError(t, err)
	require.NotNil(t, result.EmailConfig)
	assert.Equal(t, "test@example.com", result.EmailConfig.Email)
}

func TestAlertChannelRepository_GetByID_WithWebhookConfig(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	webhookCfg, err := json.Marshal(model.WebhookChannelConfig{URL: "https://hook.example.com", Method: "POST"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).AddRow(
		id, userID, model.AlertChannelTypeWebhook, model.AlertChannelStatusActive,
		true, true, 0, nil, nil, nil, webhookCfg, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(id.String()).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), id.String())
	require.NoError(t, err)
	require.NotNil(t, result.WebhookConfig)
	assert.Equal(t, "https://hook.example.com", result.WebhookConfig.URL)
}

func TestAlertChannelRepository_GetByID_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("not found"))

	_, err := repo.GetByID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert channel by id")
}

// ─── AlertChannelRepository.GetByUserIDAndType ────────────────────────────────

func TestAlertChannelRepository_GetByUserIDAndType_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	telegramCfg, err := json.Marshal(model.TelegramChannelConfig{ChatID: "9999"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).AddRow(
		id, userID, model.AlertChannelTypeTelegram, model.AlertChannelStatusActive,
		true, true, 0, nil, telegramCfg, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT").
		WithArgs(userID.String(), model.AlertChannelTypeTelegram).
		WillReturnRows(rows)

	result, err := repo.GetByUserIDAndType(context.Background(), userID.String(), model.AlertChannelTypeTelegram)
	require.NoError(t, err)
	assert.Equal(t, id, result.ID)
	assert.Equal(t, model.AlertChannelTypeTelegram, result.Type)
}

func TestAlertChannelRepository_GetByUserIDAndType_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetByUserIDAndType(context.Background(), uuid.New().String(), model.AlertChannelTypeEmail)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alert channel by user and type")
}

// ─── AlertChannelRepository.ListByUserID ─────────────────────────────────────

func TestAlertChannelRepository_ListByUserID_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	telegramCfg, err := json.Marshal(model.TelegramChannelConfig{ChatID: "111"})
	require.NoError(t, err)
	emailCfg, err := json.Marshal(model.EmailChannelConfig{Email: "a@b.com"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).
		AddRow(uuid.New(), userID, model.AlertChannelTypeTelegram, model.AlertChannelStatusActive,
			true, true, 0, nil, telegramCfg, nil, nil, now, now).
		AddRow(uuid.New(), userID, model.AlertChannelTypeEmail, model.AlertChannelStatusActive,
			true, true, 0, nil, nil, emailCfg, nil, now, now)

	mock.ExpectQuery("SELECT").WithArgs(userID.String()).WillReturnRows(rows)

	result, err := repo.ListByUserID(context.Background(), userID.String())
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, model.AlertChannelTypeTelegram, result[0].Type)
	assert.Equal(t, model.AlertChannelTypeEmail, result[1].Type)
	require.NotNil(t, result[0].TelegramConfig)
	require.NotNil(t, result[1].EmailConfig)
}

func TestAlertChannelRepository_ListByUserID_Empty(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT").WithArgs(sqlmock.AnyArg()).WillReturnRows(rows)

	result, err := repo.ListByUserID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAlertChannelRepository_ListByUserID_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ListByUserID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list alert channels")
}

// ─── AlertChannelRepository.CountByUserID ────────────────────────────────────

func TestAlertChannelRepository_CountByUserID_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	userID := uuid.New()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(3)
	mock.ExpectQuery("SELECT COUNT").WithArgs(userID.String()).WillReturnRows(rows)

	count, err := repo.CountByUserID(context.Background(), userID.String())
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestAlertChannelRepository_CountByUserID_Error(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("db error"))

	_, err := repo.CountByUserID(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count alert channels by user")
}

// ─── AlertChannelRepository.Create ───────────────────────────────────────────

func TestAlertChannelRepository_Create_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeTelegram,
		Status:    model.AlertChannelStatusUnverified,
		Enabled:   true,
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Create_DuplicateError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnError(errors.New("duplicate key value violates unique constraint (23505)"))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeEmail,
		Status:    model.AlertChannelStatusUnverified,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.ErrorIs(t, err, model.ErrDuplicateChannel)
}

func TestAlertChannelRepository_Create_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("INSERT INTO alert_channels").WillReturnError(errors.New("connection refused"))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      model.AlertChannelTypeWebhook,
		Status:    model.AlertChannelStatusUnverified,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), channel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create alert channel")
}

// ─── AlertChannelRepository.Update ───────────────────────────────────────────

func TestAlertChannelRepository_Update_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		Status:    model.AlertChannelStatusActive,
		Enabled:   true,
		Verified:  true,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), channel)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Update_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(0, 0))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		Status:    model.AlertChannelStatusActive,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), channel)
	assert.ErrorIs(t, err, model.ErrAlertChannelNotFound)
}

func TestAlertChannelRepository_Update_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	channel := &model.AlertChannel{
		ID:        uuid.New(),
		Status:    model.AlertChannelStatusActive,
		UpdatedAt: time.Now(),
	}

	err := repo.Update(context.Background(), channel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update alert channel")
}

// ─── AlertChannelRepository.Delete ───────────────────────────────────────────

func TestAlertChannelRepository_Delete_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.NoError(t, err)
}

func TestAlertChannelRepository_Delete_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, model.ErrAlertChannelNotFound)
}

func TestAlertChannelRepository_Delete_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("DELETE FROM alert_channels").WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete alert channel")
}

// ─── AlertChannelRepository.MarkAsFailed ─────────────────────────────────────

func TestAlertChannelRepository_MarkAsFailed_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.MarkAsFailed(context.Background(), uuid.New().String(), 3)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_MarkAsFailed_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	err := repo.MarkAsFailed(context.Background(), uuid.New().String(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to mark channel as failed")
}

// ─── AlertChannelRepository.IncrementFailureCount ────────────────────────────

func TestAlertChannelRepository_IncrementFailureCount_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{"failure_count"}).AddRow(2)
	mock.ExpectQuery("UPDATE alert_channels").WillReturnRows(rows)

	count, err := repo.IncrementFailureCount(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestAlertChannelRepository_IncrementFailureCount_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	_, err := repo.IncrementFailureCount(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment channel failure count")
}

// ─── AlertChannelRepository.DisableChannel ───────────────────────────────────

func TestAlertChannelRepository_DisableChannel_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DisableChannel(context.Background(), uuid.New().String(), "too many failures")
	assert.NoError(t, err)
}

func TestAlertChannelRepository_DisableChannel_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectExec("UPDATE alert_channels").WillReturnError(errors.New("db error"))

	err := repo.DisableChannel(context.Background(), uuid.New().String(), "reason")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to disable channel")
}

// ─── AlertChannelRepository.ExistsDuplicate ──────────────────────────────────

func TestAlertChannelRepository_ExistsDuplicate_TelegramFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	userID := uuid.New()
	telegramCfg, err := json.Marshal(model.TelegramChannelConfig{ChatID: "chatX"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"telegram_config", "email_config", "webhook_config"}).
		AddRow(telegramCfg, nil, nil)

	mock.ExpectQuery("SELECT").WithArgs(userID.String(), model.AlertChannelTypeTelegram).WillReturnRows(rows)

	found, err := repo.ExistsDuplicate(context.Background(), userID.String(), model.AlertChannelTypeTelegram, "chatX")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestAlertChannelRepository_ExistsDuplicate_NotFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	userID := uuid.New()
	telegramCfg, err := json.Marshal(model.TelegramChannelConfig{ChatID: "other"})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"telegram_config", "email_config", "webhook_config"}).
		AddRow(telegramCfg, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	found, err := repo.ExistsDuplicate(context.Background(), userID.String(), model.AlertChannelTypeTelegram, "chatX")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestAlertChannelRepository_ExistsDuplicate_EmailFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	emailCfg, err := json.Marshal(model.EmailChannelConfig{Email: "x@y.com"})
	require.NoError(t, err)
	rows := sqlmock.NewRows([]string{"telegram_config", "email_config", "webhook_config"}).
		AddRow(nil, emailCfg, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	found, err := repo.ExistsDuplicate(context.Background(), uuid.New().String(), model.AlertChannelTypeEmail, "x@y.com")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestAlertChannelRepository_ExistsDuplicate_WebhookFound(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	webhookCfg, err := json.Marshal(model.WebhookChannelConfig{URL: "https://x.com"})
	require.NoError(t, err)
	rows := sqlmock.NewRows([]string{"telegram_config", "email_config", "webhook_config"}).
		AddRow(nil, nil, webhookCfg)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	found, err := repo.ExistsDuplicate(context.Background(), uuid.New().String(), model.AlertChannelTypeWebhook, "https://x.com")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestAlertChannelRepository_ExistsDuplicate_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.ExistsDuplicate(context.Background(), uuid.New().String(), model.AlertChannelTypeTelegram, "chatX")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check duplicate channel")
}

// ─── AlertChannelRepository.SetChannelPriorities ─────────────────────────────

func TestAlertChannelRepository_SetChannelPriorities_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	ruleID := uuid.New()
	channelID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").
		WithArgs(ruleID.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO alert_channel_priorities").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	priorities := []model.AlertChannelPriority{
		{
			AlertRuleID:    ruleID,
			AlertChannelID: channelID,
			Priority:       1,
		},
	}

	err := repo.SetChannelPriorities(context.Background(), ruleID.String(), priorities)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_SetChannelPriorities_Empty(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	ruleID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").
		WithArgs(ruleID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.SetChannelPriorities(context.Background(), ruleID.String(), nil)
	assert.NoError(t, err)
}

func TestAlertChannelRepository_SetChannelPriorities_BeginError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := repo.SetChannelPriorities(context.Background(), uuid.New().String(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestAlertChannelRepository_SetChannelPriorities_DeleteError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM alert_channel_priorities").
		WillReturnError(errors.New("delete error"))
	mock.ExpectRollback()

	err := repo.SetChannelPriorities(context.Background(), uuid.New().String(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete existing channel priorities")
}

// ─── AlertChannelRepository.GetChannelPriorities ─────────────────────────────

func TestAlertChannelRepository_GetChannelPriorities_Success(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	ruleID := uuid.New()
	p1ID := uuid.New()
	channelID1 := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{"id", "alert_rule_id", "alert_channel_id", "priority", "created_at"}).
		AddRow(p1ID, ruleID, channelID1, 1, now)

	mock.ExpectQuery("SELECT").WithArgs(ruleID.String()).WillReturnRows(rows)

	result, err := repo.GetChannelPriorities(context.Background(), ruleID.String())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, p1ID, result[0].ID)
	assert.Equal(t, 1, result[0].Priority)
}

func TestAlertChannelRepository_GetChannelPriorities_Empty(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	rows := sqlmock.NewRows([]string{"id", "alert_rule_id", "alert_channel_id", "priority", "created_at"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := repo.GetChannelPriorities(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAlertChannelRepository_GetChannelPriorities_DBError(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	_, err := repo.GetChannelPriorities(context.Background(), uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get channel priorities")
}

// ─── AlertChannelRepository.ListPendingForChannel ────────────────────────────

func TestAlertChannelRepository_ListPendingForChannel_ReturnsError(t *testing.T) {
	repo := NewAlertChannelRepository(nil)

	_, err := repo.ListPendingForChannel(context.Background(), uuid.New().String(), 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "use DeliveryAttemptRepository")
}

// ─── AlertChannelRepository.GetByID — WithLastFailureAt ──────────────────────

func TestAlertChannelRepository_GetByID_WithLastFailureAt(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := NewAlertChannelRepository(mockDB)

	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	lastFailure := now.Add(-time.Hour)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "type", "status", "enabled", "verified",
		"failure_count", "last_failure_at", "telegram_config", "email_config", "webhook_config",
		"created_at", "updated_at",
	}).AddRow(
		id, userID, model.AlertChannelTypeTelegram, model.AlertChannelStatusFailed,
		false, true, 5, &lastFailure, nil, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(id.String()).WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), id.String())
	require.NoError(t, err)
	assert.Equal(t, 5, result.FailureCount)
	require.NotNil(t, result.LastFailureAt)
	assert.Equal(t, lastFailure, *result.LastFailureAt)
}
