package import_

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/import/parser"
)

// tracer используется для трассировки операций сервиса импорта мониторов.
var tracer = otel.Tracer("github.com/raul/monitor/backend/integration-service/internal/service/import")

// MonitorServiceClient определяет интерфейс для создания мониторов.
// Это будет gRPC клиент для Monitor Service.
type MonitorServiceClient interface {
	// CreateMonitor создаёт новый монитор.
	CreateMonitor(ctx context.Context, userID uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error)

	// GetMonitorByName получает монитор по имени пользователя.
	GetMonitorByName(ctx context.Context, userID uuid.UUID, name string) (*uuid.UUID, error)
}

// ImportService предоставляет сервис для импорта мониторов.
type ImportService struct {
	importHistoryRepo interfaces.ImportHistoryRepository
	monitorClient     MonitorServiceClient
	parsers           map[model.ImportSource]parser.Parser
}

// NewImportService создаёт новый ImportService.
func NewImportService(
	importHistoryRepo interfaces.ImportHistoryRepository,
	monitorClient MonitorServiceClient,
) *ImportService {
	parsers := make(map[model.ImportSource]parser.Parser)
	parsers[model.ImportSourceUptimeRobot] = parser.NewUptimeRobotParser()
	parsers[model.ImportSourcePingdom] = parser.NewPingdomParser()
	parsers[model.ImportSourceCSV] = parser.NewCSVParser()
	parsers[model.ImportSourceJSON] = parser.NewJSONParser()

	return &ImportService{
		importHistoryRepo: importHistoryRepo,
		monitorClient:     monitorClient,
		parsers:           parsers,
	}
}

// ImportMonitors импортирует мониторы из данных.
func (s *ImportService) ImportMonitors(
	ctx context.Context,
	userID uuid.UUID,
	source model.ImportSource,
	fileData []byte,
	fileName string,
	overwriteExisting bool,
) (*model.ImportHistory, error) {
	ctx, span := tracer.Start(ctx, "ImportService.ImportMonitors")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("source", string(source)),
		attribute.String("file_name", fileName),
		attribute.Int("file_size_bytes", len(fileData)),
		attribute.Bool("overwrite_existing", overwriteExisting),
	)

	// 1. Создаём запись истории импорта
	history := model.NewImportHistory(userID, source, fileName, len(fileData), overwriteExisting)

	if err := s.importHistoryRepo.Create(ctx, history); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to create import history")
	}

	// 2. Получаем парсер
	p, ok := s.parsers[source]
	if !ok {
		history.MarkAsFailed(fmt.Sprintf("unsupported import source: %s", source))
		if updateErr := s.importHistoryRepo.Update(ctx, history); updateErr != nil {
			span.RecordError(updateErr)
		}
		return history, model.ErrUnsupportedImportSource
	}

	// 3. Валидируем данные
	if err := p.Validate(fileData); err != nil {
		span.RecordError(err)
		history.MarkAsFailed(fmt.Sprintf("validation failed: %v", err))
		if updateErr := s.importHistoryRepo.Update(ctx, history); updateErr != nil {
			span.RecordError(updateErr)
		}
		return history, errors.Wrap(err, "failed to validate import data")
	}

	// 4. Парсим мониторы
	monitors, err := p.Parse(fileData)
	if err != nil {
		span.RecordError(err)
		history.MarkAsFailed(fmt.Sprintf("parsing failed: %v", err))
		if updateErr := s.importHistoryRepo.Update(ctx, history); updateErr != nil {
			span.RecordError(updateErr)
		}
		return history, errors.Wrap(err, "failed to parse monitors")
	}

	span.SetAttributes(attribute.Int("monitors_to_import", len(monitors)))

	// 5. Помечаем как выполняющийся
	history.MarkAsRunning()
	if err := s.importHistoryRepo.Update(ctx, history); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to update import history status")
	}

	// 6. Импортируем мониторы
	for i, monitor := range monitors {
		if err := s.importMonitor(ctx, history, userID, monitor, overwriteExisting); err != nil {
			// Логируем ошибку, но продолжаем импорт
			history.AddValidationError(i, "import", err.Error(), monitor.Name)
		}
	}

	// 7. Завершаем импорт
	if history.HasErrors() {
		if history.SuccessfulImports > 0 {
			history.MarkAsPartial()
		} else {
			history.MarkAsFailed("all monitors failed to import")
		}
	} else {
		history.MarkAsCompleted()
	}

	span.AddEvent("import_completed", trace.WithAttributes(
		attribute.Int("successful", history.SuccessfulImports),
		attribute.Int("failed", history.FailedImports),
		attribute.Int("skipped", history.SkippedImports),
		attribute.String("status", string(history.Status)),
	))

	if err := s.importHistoryRepo.Update(ctx, history); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to update import history")
	}

	return history, nil
}

// importMonitor импортирует один монитор.
func (s *ImportService) importMonitor(
	ctx context.Context,
	history *model.ImportHistory,
	userID uuid.UUID,
	monitor *model.MonitorImportData,
	overwriteExisting bool,
) error {
	// Проверяем существующий монитор с таким же именем
	existingID, err := s.monitorClient.GetMonitorByName(ctx, userID, monitor.Name)

	if err == nil && existingID != nil {
		// Монитор уже существует
		if overwriteExisting {
			// Обновление существующего монитора требует реализации UpdateMonitor в Monitor Service
			// Пока возвращаем ошибку с понятным сообщением
			history.AddSkippedImport()
			return errors.Errorf("monitor '%s' already exists (ID: %s). Update functionality requires UpdateMonitor API in Monitor Service", monitor.Name, existingID.String())
		}
		history.AddSkippedImport()
		return errors.Errorf("monitor '%s' already exists (ID: %s)", monitor.Name, existingID.String())
	}

	// Создаём новый монитор
	monitorID, err := s.monitorClient.CreateMonitor(ctx, userID, monitor)
	if err != nil {
		return errors.Wrap(err, "failed to create monitor")
	}

	history.AddSuccessfulImport(*monitorID)
	return nil
}

// GetImportHistory получает историю импорта по ID.
func (s *ImportService) GetImportHistory(ctx context.Context, id, userID uuid.UUID) (*model.ImportHistory, error) {
	ctx, span := tracer.Start(ctx, "ImportService.GetImportHistory")
	defer span.End()

	span.SetAttributes(
		attribute.String("import_id", id.String()),
		attribute.String("user_id", userID.String()),
	)

	history, err := s.importHistoryRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to get import history")
	}

	// Проверяем, что пользователь имеет доступ к этой истории
	if history.UserID != userID {
		return nil, model.ErrUnauthorized
	}

	return history, nil
}

// ListImportHistory получает список истории импорта для пользователя.
func (s *ImportService) ListImportHistory(
	ctx context.Context,
	userID uuid.UUID,
	source *model.ImportSource,
	limit, offset int,
) ([]*model.ImportHistory, error) {
	ctx, span := tracer.Start(ctx, "ImportService.ListImportHistory")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)

	histories, err := s.importHistoryRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to list import history")
	}

	// Фильтруем по source если указан
	if source != nil {
		filtered := make([]*model.ImportHistory, 0)
		for _, history := range histories {
			if history.Source == *source {
				filtered = append(filtered, history)
			}
		}
		span.SetAttributes(attribute.Int("result_count", len(filtered)))
		return filtered, nil
	}

	span.SetAttributes(attribute.Int("result_count", len(histories)))

	return histories, nil
}

// GetSupportedSources возвращает список поддерживаемых источников импорта.
func (s *ImportService) GetSupportedSources() []model.ImportSource {
	sources := make([]model.ImportSource, 0, len(s.parsers))
	for source := range s.parsers {
		sources = append(sources, source)
	}
	return sources
}
