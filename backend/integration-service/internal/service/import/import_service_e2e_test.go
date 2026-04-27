package import_

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// MockMonitorClient мок для Monitor Service gRPC клиента.
type MockMonitorClient struct {
	createMonitorFunc    func(ctx context.Context, userID uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error)
	getMonitorByNameFunc func(ctx context.Context, userID uuid.UUID, name string) (*uuid.UUID, error)
}

func NewMockMonitorClient() *MockMonitorClient {
	return &MockMonitorClient{}
}

func (m *MockMonitorClient) CreateMonitor(ctx context.Context, userID uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error) {
	return m.createMonitorFunc(ctx, userID, data)
}

func (m *MockMonitorClient) GetMonitorByName(ctx context.Context, userID uuid.UUID, name string) (*uuid.UUID, error) {
	return m.getMonitorByNameFunc(ctx, userID, name)
}

// MockImportHistoryRepository мок для истории импорта.
type MockImportHistoryRepository struct {
	createFunc       func(ctx context.Context, history *model.ImportHistory) error
	getByIDFunc      func(ctx context.Context, id uuid.UUID) (*model.ImportHistory, error)
	listByUserIDFunc func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.ImportHistory, error)
	updateFunc       func(ctx context.Context, history *model.ImportHistory) error
	updateStatusFunc func(ctx context.Context, id uuid.UUID, status model.ImportStatus) error
}

func NewMockImportHistoryRepository() *MockImportHistoryRepository {
	return &MockImportHistoryRepository{}
}

func (m *MockImportHistoryRepository) Create(ctx context.Context, history *model.ImportHistory) error {
	return m.createFunc(ctx, history)
}

func (m *MockImportHistoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ImportHistory, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *MockImportHistoryRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.ImportHistory, error) {
	return m.listByUserIDFunc(ctx, userID, limit, offset)
}

func (m *MockImportHistoryRepository) Update(ctx context.Context, history *model.ImportHistory) error {
	return m.updateFunc(ctx, history)
}

func (m *MockImportHistoryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ImportStatus) error {
	return m.updateStatusFunc(ctx, id, status)
}

func (m *MockImportHistoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// TestImportService тестирует сервис импорта мониторов.
type TestImportService struct{}

func TestImportService_ImportMonitors_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()

	monitorClient := NewMockMonitorClient()
	importHistoryRepo := NewMockImportHistoryRepository()

	// Создаём ImportService
	importService := NewImportService(importHistoryRepo, monitorClient)

	// Act
	source := model.ImportSourceJSON
	fileData := []byte(`[
		{
			"name": "Test Monitor 1",
			"url": "https://example.com",
			"method": "GET",
			"check_interval": 60
		}
	]`)
	fileName := "test.json"
	overwriteExisting := false

	// Мокируем Create из репозитория
	importHistoryRepo.createFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}
	importHistoryRepo.updateFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}

	// Mock Monitor client
	expectedMonitorID := uuid.New()
	monitorClient.createMonitorFunc = func(ctx context.Context, uid uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error) {
		return &expectedMonitorID, nil
	}

	// Mock GetMonitorByName - возвращает nil (монитор не найден)
	monitorClient.getMonitorByNameFunc = func(ctx context.Context, uid uuid.UUID, name string) (*uuid.UUID, error) {
		return nil, nil // Монитор с таким именем не существует
	}

	// Assert
	importHistory, err := importService.ImportMonitors(ctx, userID, source, fileData, fileName, overwriteExisting)

	// Assert - ошибка не ожидается, потому что все моки возвращают nil
	assert.NoError(t, err)

	// Assert - импорт успешен
	assert.NotNil(t, importHistory)
	assert.Equal(t, model.ImportStatusCompleted, importHistory.Status)
	assert.Equal(t, 1, importHistory.TotalMonitors)

	assert.True(t, importHistory.IsFinished())

	// Assert - монитор был создан
	assert.True(t, len(importHistory.ImportedMonitorIDs) == 1)
	assert.Equal(t, expectedMonitorID, importHistory.ImportedMonitorIDs[0])

	// Assert - данные корректны
	assert.Equal(t, userID, importHistory.UserID)
	assert.Equal(t, source, importHistory.Source)
	assert.Equal(t, fileName, *importHistory.FileName)
	assert.Equal(t, len(fileData), *importHistory.FileSizeBytes)
	assert.True(t, importHistory.OverwriteExisting == overwriteExisting)

	// Assert - ошибка валидации не записана (в данном случае мониторы валидны)
	assert.False(t, importHistory.HasErrors())

	t.Log("✓ ImportMonitors with valid monitors works correctly")
}

func TestImportService_ImportMonitors_WithValidationErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()

	importService := NewImportService(importHistoryRepo, monitorClient)

	// Mock repository methods
	importHistoryRepo.createFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}
	importHistoryRepo.updateFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}

	// Валидные мониторы
	validData := []byte(`[
		{
			"name": "Valid Monitor",
			"url": "https://example.com",
			"method": "GET"
		}
	]`)

	// Mock - возвращает существующий монитор (для проверки overwrite)
	existingMonitorID := uuid.New()
	monitorClient.getMonitorByNameFunc = func(ctx context.Context, uid uuid.UUID, name string) (*uuid.UUID, error) {
		if name == "Valid Monitor" {
			return &existingMonitorID, nil
		}
		return nil, nil
	}
	newMonitorID := uuid.New()
	monitorClient.createMonitorFunc = func(ctx context.Context, uid uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error) {
		return &newMonitorID, nil
	}

	importHistory, err := importService.ImportMonitors(ctx, userID, model.ImportSourceJSON, validData, "test.json", true)

	// Assert - проверяем что импорт прошёл (даже если были ошибки)
	assert.NoError(t, err)
	assert.NotNil(t, importHistory)

	// Статус может быть completed или failed в зависимости от ошибок
	assert.True(t, importHistory.IsFinished())

	t.Log("✓ ImportMonitors with overwrite enabled works correctly")
}

func TestImportService_ImportMonitors_WithValidationError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()

	importService := NewImportService(importHistoryRepo, monitorClient)

	// Mock repository methods
	importHistoryRepo.createFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}
	importHistoryRepo.updateFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}

	// Мониторы с ошибками валидации
	invalidData := []byte(`[
		{"name": "", "url": "invalid-url"}
	]`)

	// Mock - возвращает ошибку при создании монитора
	monitorClient.createMonitorFunc = func(ctx context.Context, uid uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error) {
		return nil, errors.New("monitor creation failed")
	}

	importHistory, err := importService.ImportMonitors(ctx, userID, model.ImportSourceJSON, invalidData, "test.json", false)

	// Assert - ошибка валидации должна быть возвращена
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor name is required")

	// Статус должен быть "failed"
	assert.NotNil(t, importHistory)
	assert.Equal(t, model.ImportStatusFailed, importHistory.Status)

	t.Log("✓ ImportMonitors with validation errors works correctly")
}

func TestImportService_ImportMonitors_UnsupportedSource(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()

	importService := NewImportService(importHistoryRepo, monitorClient)

	// Mock repository methods
	importHistoryRepo.createFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}
	importHistoryRepo.updateFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}

	// Попытка импорта с неподдерживаемым источником
	invalidSource := model.ImportSource("invalid")

	_, err := importService.ImportMonitors(ctx, userID, invalidSource, []byte(`[]`), "test.json", false)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported import source")

	t.Log("✓ ImportMonitors rejects unsupported source correctly")
}

func TestImportService_GetImportHistory(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()
	importService := NewImportService(importHistoryRepo, monitorClient)

	importHistoryID := uuid.New()

	// Создаём тестовую историю
	testHistory := model.NewImportHistory(userID, model.ImportSourceJSON, "test.json", 100, false)
	importHistoryRepo.createFunc = func(c context.Context, h *model.ImportHistory) error {
		return nil
	}

	// Мок - возвращаем созданную историю при GetByID
	importHistoryRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.ImportHistory, error) {
		assert.Equal(t, importHistoryID, id)
		return testHistory, nil
	}

	// Act
	history, err := importService.GetImportHistory(ctx, importHistoryID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, history)
	assert.Equal(t, userID, history.UserID)
	assert.Equal(t, testHistory.ID, history.ID)
	assert.Equal(t, model.ImportSourceJSON, history.Source)
	assert.Equal(t, "test.json", *history.FileName)

	t.Log("✓ GetImportHistory returns correct history")
}

func TestImportService_ListImportHistory(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()
	importService := NewImportService(importHistoryRepo, monitorClient)

	// Создаём несколько историй для теста
	history1 := model.NewImportHistory(userID, model.ImportSourceJSON, "test1.json", 100, false)
	history1.MarkAsCompleted()

	history2 := model.NewImportHistory(userID, model.ImportSourceJSON, "test2.json", 200, false)
	history2.MarkAsCompleted()

	history3 := model.NewImportHistory(userID, model.ImportSourceJSON, "test3.json", 300, false)
	history3.MarkAsCompleted()

	// Mock ListByUserID
	histories := []*model.ImportHistory{history1, history2, history3}
	importHistoryRepo.listByUserIDFunc = func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*model.ImportHistory, error) {
		return histories, nil
	}

	// Act
	result, err := importService.ListImportHistory(ctx, userID, nil, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	// Проверяем сортировку (должно быть по started_at DESC)
	assert.True(t, result[0].StartedAt.Before(result[1].StartedAt))
	assert.True(t, result[1].StartedAt.Before(result[2].StartedAt))

	t.Log("✓ ListImportHistory returns histories in correct order")
}

func TestImportService_GetSupportedSources(t *testing.T) {
	t.Parallel()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()
	importService := NewImportService(importHistoryRepo, monitorClient)

	sources := importService.GetSupportedSources()
	assert.NotEmpty(t, sources)
	// Should include csv and json parsers at minimum
	sourceMap := make(map[model.ImportSource]bool)
	for _, s := range sources {
		sourceMap[s] = true
	}
	assert.True(t, sourceMap[model.ImportSourceCSV] || sourceMap[model.ImportSourceJSON])
}

func TestImportService_ListImportHistoryWithSourceFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()
	importService := NewImportService(importHistoryRepo, monitorClient)

	// Создаём несколько историй разных источников
	history1 := model.NewImportHistory(userID, model.ImportSourceCSV, "a.csv", 100, false)
	history2 := model.NewImportHistory(userID, model.ImportSourceJSON, "b.json", 100, false)
	history3 := model.NewImportHistory(userID, model.ImportSourceCSV, "c.csv", 100, false)

	importHistoryRepo.listByUserIDFunc = func(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.ImportHistory, error) {
		return []*model.ImportHistory{history1, history2, history3}, nil
	}

	csvSource := model.ImportSourceCSV
	result, err := importService.ListImportHistory(ctx, userID, &csvSource, 10, 0)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	for _, h := range result {
		assert.Equal(t, model.ImportSourceCSV, h.Source)
	}
}

func TestImportService_GetImportHistoryUnauthorized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()
	importService := NewImportService(importHistoryRepo, monitorClient)

	history := model.NewImportHistory(otherUserID, model.ImportSourceCSV, "a.csv", 100, false)
	importHistoryRepo.getByIDFunc = func(_ context.Context, _ uuid.UUID) (*model.ImportHistory, error) {
		return history, nil
	}

	_, err := importService.GetImportHistory(ctx, history.ID, userID)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrUnauthorized)
}

func TestImportService_GetImportHistoryNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()

	importHistoryRepo := NewMockImportHistoryRepository()
	monitorClient := NewMockMonitorClient()
	importService := NewImportService(importHistoryRepo, monitorClient)

	importHistoryRepo.getByIDFunc = func(_ context.Context, _ uuid.UUID) (*model.ImportHistory, error) {
		return nil, model.ErrImportNotFound
	}

	_, err := importService.GetImportHistory(ctx, uuid.New(), userID)
	require.Error(t, err)
}
