package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	integrationv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/service/import"
)

// ImportHandler обрабатывает gRPC запросы для импорта мониторов.
type ImportHandler struct {
	integrationv1.UnimplementedImportServiceServer
	importService *import_.ImportService
}

// NewImportHandler создаёт новый ImportHandler.
func NewImportHandler(importService *import_.ImportService) *ImportHandler {
	return &ImportHandler{
		importService: importService,
	}
}

// ImportMonitors импортирует мониторы из файла.
func (h *ImportHandler) ImportMonitors(ctx context.Context, req *integrationv1.ImportMonitorsRequest) (*integrationv1.ImportHistory, error) {
	// 1. Парсим UUID
	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// 2. Валидируем источник импорта
	source, err := model.ValidateImportSource(req.Source)
	if err != nil {
		return nil, invalidArgument("source", "unsupported import source")
	}

	// 3. Импортируем мониторы
	history, err := h.importService.ImportMonitors(
		ctx,
		userID,
		source,
		req.FileData,
		req.FileName,
		req.OverwriteExisting,
	)
	if err != nil {
		return nil, errorToStatus(err)
	}

	// 4. Конвертируем модель в proto
	return modelToProtoImportHistory(history), nil
}

// GetImportHistory возвращает историю импорта по ID.
func (h *ImportHandler) GetImportHistory(ctx context.Context, req *integrationv1.GetImportHistoryRequest) (*integrationv1.ImportHistory, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, invalidArgument("id", "invalid UUID format")
	}

	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	history, err := h.importService.GetImportHistory(ctx, id, userID)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return modelToProtoImportHistory(history), nil
}

// ListImportHistory возвращает список истории импорта.
func (h *ImportHandler) ListImportHistory(ctx context.Context, req *integrationv1.ListImportHistoryRequest) (*integrationv1.ListImportHistoryResponse, error) {
	userID, err := resolveUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// Парсим фильтр по источнику
	var source *model.ImportSource
	if req.Source != "" {
		s, err := model.ValidateImportSource(req.Source)
		if err != nil {
			return nil, invalidArgument("source", "unsupported import source")
		}
		source = &s
	}

	// Парсим пагинацию
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 50 // default
	}

	pageToken := req.PageToken
	offset := 0
	if pageToken != "" {
		// Page token - это base64-encoded offset
		// Формат: "offset:<number>" закодированный в base64
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			// Пытаемся распарсить формат "offset:number"
			tokenStr := string(decoded)
			if len(tokenStr) > 7 && tokenStr[:7] == "offset:" {
				offsetStr := tokenStr[7:]
				if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
					offset = parsedOffset
				}
			}
		}
		// Если парсинг неудачен, используем offset=0
	}

	histories, err := h.importService.ListImportHistory(ctx, userID, source, pageSize, offset)
	if err != nil {
		return nil, errorToStatus(err)
	}

	protoHistories := make([]*integrationv1.ImportHistory, len(histories))
	for i, h := range histories {
		protoHistories[i] = modelToProtoImportHistory(h)
	}

	// Генерируем следующий page token, если есть ещё результаты
	var nextPageToken string
	if len(histories) == pageSize {
		nextOffset := offset + pageSize
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("offset:%d", nextOffset)))
	}

	return &integrationv1.ListImportHistoryResponse{
		Imports:       protoHistories,
		TotalCount:    int32(len(protoHistories)), //nolint:gosec // G115: len всегда неотрицателен
		NextPageToken: nextPageToken,
	}, nil
}

// modelToProtoImportHistory конвертирует модель ImportHistory в proto.
func modelToProtoImportHistory(history *model.ImportHistory) *integrationv1.ImportHistory {
	proto := &integrationv1.ImportHistory{
		Id:                history.ID.String(),
		UserId:            history.UserID.String(),
		Source:            string(history.Source),
		Status:            string(history.Status),
		TotalMonitors:     int32(history.TotalMonitors),     //nolint:gosec // G115: значение ограничено при создании
		SuccessfulImports: int32(history.SuccessfulImports), //nolint:gosec // G115: значение ограничено при создании
		FailedImports:     int32(history.FailedImports),     //nolint:gosec // G115: значение ограничено при создании
		SkippedImports:    int32(history.SkippedImports),    //nolint:gosec // G115: значение ограничено при создании
		OverwriteExisting: history.OverwriteExisting,
		FileName:          coalesceString(history.FileName),
		FileSizeBytes:     int32(coalesceInt(history.FileSizeBytes)), //nolint:gosec // G115: значение ограничено при создании
		DurationMs:        int32(coalesceInt(history.DurationMs)),    //nolint:gosec // G115: значение ограничено при создании
		ErrorMessage:      coalesceString(history.ErrorMessage),
		StartedAt:         timestamppb.New(history.StartedAt),
	}

	// Конвертируем imported_monitor_ids
	proto.ImportedMonitorIds = make([]string, len(history.ImportedMonitorIDs))
	for i, id := range history.ImportedMonitorIDs {
		proto.ImportedMonitorIds[i] = id.String()
	}

	// Конвертируем validation_errors
	proto.ValidationErrors = make([]*integrationv1.ValidationError, len(history.ValidationErrors))
	for i, ve := range history.ValidationErrors {
		proto.ValidationErrors[i] = &integrationv1.ValidationError{
			Row:     int32(ve.Row), //nolint:gosec // G115
			Field:   ve.Field,
			Message: ve.Message,
			Value:   ve.Value,
		}
	}

	// CompletedAt
	if history.CompletedAt != nil {
		proto.CompletedAt = timestamppb.New(*history.CompletedAt)
	}

	return proto
}

// coalesceString возвращает строку или пустую строку если nil.
func coalesceString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// coalesceInt возвращает int или 0 если nil.
func coalesceInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
