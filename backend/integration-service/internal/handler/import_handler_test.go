package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	integrationv1 "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	importsvc "github.com/raul/monitor/backend/integration-service/internal/service/import"
)

// TestModelToProtoImportHistory проверяет конвертацию модели ImportHistory в proto.
func TestModelToProtoImportHistory(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()
	historyID := uuid.New()
	completedAt := now.Add(1 * time.Second)
	fileName := "monitors.csv"
	fileSize := 1024
	durationMs := 500
	errMsg := ""
	monitor1ID := uuid.New()
	monitor2ID := uuid.New()

	history := &model.ImportHistory{
		ID:                 historyID,
		UserID:             userID,
		Source:             model.ImportSourceCSV,
		Status:             model.ImportStatusCompleted,
		TotalMonitors:      2,
		SuccessfulImports:  2,
		FailedImports:      0,
		SkippedImports:     0,
		OverwriteExisting:  false,
		ImportedMonitorIDs: []uuid.UUID{monitor1ID, monitor2ID},
		ValidationErrors:   []model.ValidationError{},
		FileName:           &fileName,
		FileSizeBytes:      &fileSize,
		StartedAt:          now,
		CompletedAt:        &completedAt,
		DurationMs:         &durationMs,
		ErrorMessage:       &errMsg,
	}

	result := modelToProtoImportHistory(history)

	require.NotNil(t, result)
	assert.Equal(t, historyID.String(), result.Id)
	assert.Equal(t, userID.String(), result.UserId)
	assert.Equal(t, string(model.ImportSourceCSV), result.Source)
	assert.Equal(t, string(model.ImportStatusCompleted), result.Status)
	assert.Equal(t, int32(2), result.TotalMonitors)
	assert.Equal(t, int32(2), result.SuccessfulImports)
	assert.Equal(t, int32(0), result.FailedImports)
	assert.Equal(t, int32(0), result.SkippedImports)
	assert.False(t, result.OverwriteExisting)
	assert.Equal(t, "monitors.csv", result.FileName)
	assert.Equal(t, int32(1024), result.FileSizeBytes)
	assert.Equal(t, int32(500), result.DurationMs)
	assert.Equal(t, "", result.ErrorMessage)
	require.NotNil(t, result.StartedAt)
	require.NotNil(t, result.CompletedAt)
	require.Len(t, result.ImportedMonitorIds, 2)
	assert.Equal(t, monitor1ID.String(), result.ImportedMonitorIds[0])
	assert.Equal(t, monitor2ID.String(), result.ImportedMonitorIds[1])
}

// TestModelToProtoImportHistoryWithValidationErrors проверяет конвертацию с ошибками валидации.
func TestModelToProtoImportHistoryWithValidationErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	historyID := uuid.New()
	now := time.Now()

	history := &model.ImportHistory{
		ID:                historyID,
		UserID:            userID,
		Source:            model.ImportSourceJSON,
		Status:            model.ImportStatusPartial,
		TotalMonitors:     3,
		SuccessfulImports: 1,
		FailedImports:     2,
		ValidationErrors: []model.ValidationError{
			{Row: 1, Field: "url", Message: "invalid URL", Value: "not-a-url"},
			{Row: 2, Field: "name", Message: "name required", Value: ""},
		},
		ImportedMonitorIDs: []uuid.UUID{uuid.New()},
		StartedAt:          now,
	}

	result := modelToProtoImportHistory(history)

	require.NotNil(t, result)
	require.Len(t, result.ValidationErrors, 2)
	assert.Equal(t, int32(1), result.ValidationErrors[0].Row)
	assert.Equal(t, "url", result.ValidationErrors[0].Field)
	assert.Equal(t, "invalid URL", result.ValidationErrors[0].Message)
	assert.Equal(t, "not-a-url", result.ValidationErrors[0].Value)
}

// TestModelToProtoImportHistoryNilOptionalFields проверяет nil optional fields.
func TestModelToProtoImportHistoryNilOptionalFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	history := &model.ImportHistory{
		ID:                 uuid.New(),
		UserID:             uuid.New(),
		Source:             model.ImportSourceUptimeRobot,
		Status:             model.ImportStatusPending,
		ImportedMonitorIDs: []uuid.UUID{},
		ValidationErrors:   []model.ValidationError{},
		StartedAt:          now,
	}

	result := modelToProtoImportHistory(history)

	require.NotNil(t, result)
	assert.Equal(t, "", result.FileName)
	assert.Equal(t, int32(0), result.FileSizeBytes)
	assert.Equal(t, int32(0), result.DurationMs)
	assert.Equal(t, "", result.ErrorMessage)
	assert.Nil(t, result.CompletedAt)
}

// TestCoalesceString проверяет вспомогательную функцию coalesceString.
func TestCoalesceString(t *testing.T) {
	t.Parallel()

	t.Run("nil returns empty string", func(t *testing.T) {
		t.Parallel()
		result := coalesceString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		t.Parallel()
		s := "hello"
		result := coalesceString(&s)
		assert.Equal(t, "hello", result)
	})

	t.Run("empty string pointer returns empty string", func(t *testing.T) {
		t.Parallel()
		s := ""
		result := coalesceString(&s)
		assert.Equal(t, "", result)
	})
}

// TestCoalesceInt проверяет вспомогательную функцию coalesceInt.
func TestCoalesceInt(t *testing.T) {
	t.Parallel()

	t.Run("nil returns zero", func(t *testing.T) {
		t.Parallel()
		result := coalesceInt(nil)
		assert.Equal(t, 0, result)
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		t.Parallel()
		i := 42
		result := coalesceInt(&i)
		assert.Equal(t, 42, result)
	})

	t.Run("zero int pointer returns zero", func(t *testing.T) {
		t.Parallel()
		i := 0
		result := coalesceInt(&i)
		assert.Equal(t, 0, result)
	})
}

// TestImportHandlerInvalidUUID проверяет обработку невалидных UUID.
func TestImportHandlerInvalidUUID(t *testing.T) {
	t.Parallel()

	h := &ImportHandler{}
	ctx := context.Background()

	t.Run("ImportMonitors invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.ImportMonitors(ctx, &integrationv1.ImportMonitorsRequest{
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("ImportMonitors invalid source", func(t *testing.T) {
		t.Parallel()
		_, err := h.ImportMonitors(ctx, &integrationv1.ImportMonitorsRequest{
			UserId: uuid.New().String(),
			Source: "invalid_source",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source")
	})

	t.Run("GetImportHistory invalid id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     "not-a-uuid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("GetImportHistory invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     uuid.New().String(),
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("ListImportHistory invalid user_id", func(t *testing.T) {
		t.Parallel()
		_, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId: "not-a-uuid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("ListImportHistory invalid source", func(t *testing.T) {
		t.Parallel()
		_, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId: uuid.New().String(),
			Source: "invalid_source_type",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source")
	})
}

// TestNewImportHandler проверяет создание нового обработчика.
func TestNewImportHandler(t *testing.T) {
	t.Parallel()

	h := NewImportHandler(nil)
	require.NotNil(t, h)
}

func TestImportHandlerGetImportHistoryValidation(t *testing.T) {
	t.Parallel()

	h := &ImportHandler{}
	ctx := context.Background()

	t.Run("invalid history id returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     "not-valid",
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("invalid user_id returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     uuid.New().String(),
			UserId: "not-valid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})
}

func TestImportHandlerListImportHistoryWithPageToken(t *testing.T) {
	t.Parallel()

	h := &ImportHandler{}
	ctx := context.Background()

	t.Run("invalid user_id returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId: "bad-uuid",
		})
		require.Error(t, err)
	})

	t.Run("invalid source returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId: uuid.New().String(),
			Source: "unsupported_source_xyz",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source")
	})
}

// handlerMockHistoryRepo реализует interfaces.ImportHistoryRepository.
type handlerMockHistoryRepo struct {
	histories map[uuid.UUID]*model.ImportHistory
}

func newHandlerMockHistoryRepo() *handlerMockHistoryRepo {
	return &handlerMockHistoryRepo{histories: make(map[uuid.UUID]*model.ImportHistory)}
}

func (m *handlerMockHistoryRepo) Create(_ context.Context, h *model.ImportHistory) error {
	m.histories[h.ID] = h
	return nil
}

func (m *handlerMockHistoryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.ImportHistory, error) {
	h, ok := m.histories[id]
	if !ok {
		return nil, model.ErrImportNotFound
	}
	return h, nil
}

func (m *handlerMockHistoryRepo) ListByUserID(_ context.Context, userID uuid.UUID, _, _ int) ([]*model.ImportHistory, error) {
	var result []*model.ImportHistory
	for _, h := range m.histories {
		if h.UserID == userID {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *handlerMockHistoryRepo) Update(_ context.Context, h *model.ImportHistory) error {
	m.histories[h.ID] = h
	return nil
}

func (m *handlerMockHistoryRepo) UpdateStatus(_ context.Context, id uuid.UUID, status model.ImportStatus) error {
	if h, ok := m.histories[id]; ok {
		h.Status = status
	}
	return nil
}

func (m *handlerMockHistoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.histories, id)
	return nil
}

// handlerMockMonitorClient реализует importsvc.MonitorServiceClient.
type handlerMockMonitorClient struct{}

func (m *handlerMockMonitorClient) CreateMonitor(_ context.Context, _ uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error) {
	id := uuid.New()
	return &id, nil
}

func (m *handlerMockMonitorClient) GetMonitorByName(_ context.Context, _ uuid.UUID, _ string) (*uuid.UUID, error) {
	return nil, nil
}

func newTestImportHandlerWithService(t *testing.T) (*ImportHandler, *handlerMockHistoryRepo) {
	t.Helper()
	repo := newHandlerMockHistoryRepo()
	svc := importsvc.NewImportService(repo, &handlerMockMonitorClient{})
	return NewImportHandler(svc), repo
}

func TestImportHandlerGetImportHistoryWithService(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	h, repo := newTestImportHandlerWithService(t)
	ctx := context.Background()

	history := &model.ImportHistory{
		ID:                 uuid.New(),
		UserID:             userID,
		Source:             model.ImportSourceJSON,
		Status:             model.ImportStatusCompleted,
		ImportedMonitorIDs: []uuid.UUID{},
		ValidationErrors:   []model.ValidationError{},
		StartedAt:          time.Now(),
	}
	repo.histories[history.ID] = history

	t.Run("get existing history", func(t *testing.T) {
		t.Parallel()
		result, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     history.ID.String(),
			UserId: userID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, history.ID.String(), result.Id)
	})

	t.Run("get history not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     uuid.New().String(),
			UserId: userID.String(),
		})
		require.Error(t, err)
	})

	t.Run("get history of other user returns unauthorized error", func(t *testing.T) {
		t.Parallel()
		_, err := h.GetImportHistory(ctx, &integrationv1.GetImportHistoryRequest{
			Id:     history.ID.String(),
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

func TestImportHandlerListImportHistoryWithService(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	h, repo := newTestImportHandlerWithService(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		history := &model.ImportHistory{
			ID:                 uuid.New(),
			UserID:             userID,
			Source:             model.ImportSourceCSV,
			Status:             model.ImportStatusCompleted,
			ImportedMonitorIDs: []uuid.UUID{},
			ValidationErrors:   []model.ValidationError{},
			StartedAt:          time.Now(),
		}
		repo.histories[history.ID] = history
	}

	t.Run("list with no page token returns histories", func(t *testing.T) {
		t.Parallel()
		result, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId:   userID.String(),
			PageSize: 50,
		})
		require.NoError(t, err)
		assert.Len(t, result.Imports, 3)
	})

	t.Run("list with valid page token", func(t *testing.T) {
		t.Parallel()
		token := base64.StdEncoding.EncodeToString([]byte("offset:10"))
		result, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId:    userID.String(),
			PageSize:  50,
			PageToken: token,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list with invalid page token defaults to offset=0", func(t *testing.T) {
		t.Parallel()
		result, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId:    userID.String(),
			PageSize:  50,
			PageToken: "not-base64!@#",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list with source filter", func(t *testing.T) {
		t.Parallel()
		result, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId:   userID.String(),
			PageSize: 50,
			Source:   "csv",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list with default page size", func(t *testing.T) {
		t.Parallel()
		result, err := h.ListImportHistory(ctx, &integrationv1.ListImportHistoryRequest{
			UserId:   userID.String(),
			PageSize: 0, // will use default of 50
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestImportHandlerImportMonitorsWithService(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	h, _ := newTestImportHandlerWithService(t)
	ctx := context.Background()

	t.Run("invalid source returns error", func(t *testing.T) {
		t.Parallel()
		_, err := h.ImportMonitors(ctx, &integrationv1.ImportMonitorsRequest{
			UserId:   userID.String(),
			Source:   "invalid_source",
			FileData: []byte(`[]`),
			FileName: "test.json",
		})
		require.Error(t, err)
	})

	t.Run("valid json import creates history", func(t *testing.T) {
		t.Parallel()
		jsonData := []byte(`[{"name": "Monitor 1", "url": "https://example.com", "method": "GET"}]`)
		result, err := h.ImportMonitors(ctx, &integrationv1.ImportMonitorsRequest{
			UserId:   userID.String(),
			Source:   "json",
			FileData: jsonData,
			FileName: "test.json",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, string(model.ImportStatusCompleted), result.Status)
	})
}

// TestListImportHistoryPageToken проверяет парсинг page token.
func TestListImportHistoryPageToken(t *testing.T) {
	t.Parallel()

	// Проверяем что корректный pageToken не вызывает паники при парсинге
	// (эта логика изолирована в ListImportHistory)
	validToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("offset:%d", 50)))
	assert.NotEmpty(t, validToken)

	decoded, err := base64.StdEncoding.DecodeString(validToken)
	require.NoError(t, err)
	assert.Equal(t, "offset:50", string(decoded))
}
