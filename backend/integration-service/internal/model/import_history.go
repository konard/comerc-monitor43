package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ImportSource источник импорта.
type ImportSource string

const (
	ImportSourceUptimeRobot ImportSource = "uptimerobot"
	ImportSourcePingdom     ImportSource = "pingdom"
	ImportSourceCSV         ImportSource = "csv"
	ImportSourceJSON        ImportSource = "json"
)

// ImportStatus статус импорта.
type ImportStatus string

const (
	ImportStatusPending   ImportStatus = "pending"
	ImportStatusRunning   ImportStatus = "running"
	ImportStatusCompleted ImportStatus = "completed"
	ImportStatusFailed    ImportStatus = "failed"
	ImportStatusPartial   ImportStatus = "partial" // Частичный успех
)

// ImportHistory представляет историю импорта мониторов.
type ImportHistory struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Source             ImportSource
	Status             ImportStatus
	TotalMonitors      int
	SuccessfulImports  int
	FailedImports      int
	SkippedImports     int
	OverwriteExisting  bool
	ImportedMonitorIDs []uuid.UUID
	ValidationErrors   []ValidationError
	ConflictResolution map[string]any
	FileName           *string
	FileSizeBytes      *int
	StartedAt          time.Time
	CompletedAt        *time.Time
	DurationMs         *int
	ErrorMessage       *string
}

// ValidationError представляет ошибку валидации при импорте.
type ValidationError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// NewImportHistory создаёт новую запись истории импорта.
func NewImportHistory(userID uuid.UUID, source ImportSource, fileName string, fileSizeBytes int, overwriteExisting bool) *ImportHistory {
	return &ImportHistory{
		ID:                 uuid.New(),
		UserID:             userID,
		Source:             source,
		Status:             ImportStatusPending,
		OverwriteExisting:  overwriteExisting,
		ImportedMonitorIDs: make([]uuid.UUID, 0),
		ValidationErrors:   make([]ValidationError, 0),
		ConflictResolution: make(map[string]any),
		FileName:           &fileName,
		FileSizeBytes:      &fileSizeBytes,
		StartedAt:          time.Now(),
	}
}

// MarkAsRunning помечает импорт как выполняющийся.
func (i *ImportHistory) MarkAsRunning() {
	i.Status = ImportStatusRunning
}

// MarkAsCompleted помечает импорт как завершённый.
func (i *ImportHistory) MarkAsCompleted() {
	now := time.Now()
	i.Status = ImportStatusCompleted
	i.CompletedAt = &now
	duration := int(now.Sub(i.StartedAt).Milliseconds())
	i.DurationMs = &duration
}

// MarkAsFailed помечает импорт как неудачный.
func (i *ImportHistory) MarkAsFailed(errorMessage string) {
	now := time.Now()
	i.Status = ImportStatusFailed
	i.CompletedAt = &now
	duration := int(now.Sub(i.StartedAt).Milliseconds())
	i.DurationMs = &duration
	i.ErrorMessage = &errorMessage
}

// MarkAsPartial помечает импорт как частично успешный.
func (i *ImportHistory) MarkAsPartial() {
	now := time.Now()
	i.Status = ImportStatusPartial
	i.CompletedAt = &now
	duration := int(now.Sub(i.StartedAt).Milliseconds())
	i.DurationMs = &duration
}

// AddSuccessfulImport добавляет успешный импорт.
func (i *ImportHistory) AddSuccessfulImport(monitorID uuid.UUID) {
	i.SuccessfulImports++
	i.TotalMonitors++
	i.ImportedMonitorIDs = append(i.ImportedMonitorIDs, monitorID)
}

// AddFailedImport добавляет неудачный импорт.
func (i *ImportHistory) AddFailedImport() {
	i.FailedImports++
	i.TotalMonitors++
}

// AddSkippedImport добавляет пропущенный импорт.
func (i *ImportHistory) AddSkippedImport() {
	i.SkippedImports++
	i.TotalMonitors++
}

// AddValidationError добавляет ошибку валидации.
func (i *ImportHistory) AddValidationError(row int, field, message, value string) {
	validationError := ValidationError{
		Row:     row,
		Field:   field,
		Message: message,
		Value:   value,
	}
	i.ValidationErrors = append(i.ValidationErrors, validationError)
	i.AddFailedImport()
}

// SetConflictResolution устанавливает детали разрешения конфликтов.
func (i *ImportHistory) SetConflictResolution(details map[string]any) {
	i.ConflictResolution = details
}

// GetSuccessRate возвращает процент успешных импортов.
func (i *ImportHistory) GetSuccessRate() float64 {
	if i.TotalMonitors == 0 {
		return 0
	}

	return (float64(i.SuccessfulImports) / float64(i.TotalMonitors)) * 100
}

// HasErrors возвращает true, если были ошибки.
func (i *ImportHistory) HasErrors() bool {
	return len(i.ValidationErrors) > 0
}

// IsFinished возвращает true, если импорт завершён.
func (i *ImportHistory) IsFinished() bool {
	return i.Status == ImportStatusCompleted ||
		i.Status == ImportStatusFailed ||
		i.Status == ImportStatusPartial
}

// GetValidationErrorsJSON возвращает ошибки валидации как JSON.
func (i *ImportHistory) GetValidationErrorsJSON() ([]byte, error) {
	return json.Marshal(i.ValidationErrors)
}

// GetImportedMonitorIDsJSON возвращает ID импортированных мониторов как JSON.
func (i *ImportHistory) GetImportedMonitorIDsJSON() ([]byte, error) {
	// Конвертируем UUID в строки для JSON
	ids := make([]string, len(i.ImportedMonitorIDs))
	for idx, id := range i.ImportedMonitorIDs {
		ids[idx] = id.String()
	}

	return json.Marshal(ids)
}

// GetConflictResolutionJSON возвращает детали разрешения конфликтов как JSON.
func (i *ImportHistory) GetConflictResolutionJSON() ([]byte, error) {
	return json.Marshal(i.ConflictResolution)
}

// ParseValidationErrorsFromJSON парсит ошибки валидации из JSON.
func ParseValidationErrorsFromJSON(data []byte) ([]ValidationError, error) {
	var validationErrors []ValidationError
	if err := json.Unmarshal(data, &validationErrors); err != nil {
		return nil, errors.Wrap(err, "failed to parse validation errors")
	}

	return validationErrors, nil
}

// ParseImportedMonitorIDsFromJSON парсит ID импортированных мониторов из JSON.
func ParseImportedMonitorIDsFromJSON(data []byte) ([]uuid.UUID, error) {
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, errors.Wrap(err, "failed to parse imported monitor IDs")
	}

	// Конвертируем строки в UUID
	uuids := make([]uuid.UUID, len(ids))
	for idx, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse UUID: %s", id)
		}
		uuids[idx] = parsed
	}

	return uuids, nil
}

// ParseConflictResolutionFromJSON парсит детали разрешения конфликтов из JSON.
func ParseConflictResolutionFromJSON(data []byte) (map[string]any, error) {
	var resolution map[string]any
	if err := json.Unmarshal(data, &resolution); err != nil {
		return nil, errors.Wrap(err, "failed to parse conflict resolution")
	}

	return resolution, nil
}

// ValidateImportSource проверяет валидность источника импорта.
func ValidateImportSource(source string) (ImportSource, error) {
	importSource := ImportSource(source)

	switch importSource {
	case ImportSourceUptimeRobot, ImportSourcePingdom, ImportSourceCSV, ImportSourceJSON:
		return importSource, nil
	default:
		return "", ErrUnsupportedImportSource
	}
}

// MonitorImportData представляет данные монитора для импорта.
type MonitorImportData struct {
	Name            string            `json:"name"`
	URL             string            `json:"url"`
	Method          string            `json:"method,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            *string           `json:"body,omitempty"`
	CheckInterval   *int              `json:"check_interval,omitempty"`
	Timeout         *int              `json:"timeout,omitempty"`
	ExpectedStatus  *int              `json:"expected_status,omitempty"`
	ExpectedPattern *string           `json:"expected_pattern,omitempty"`
	Enabled         *bool             `json:"enabled,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
}

// Validate валидирует данные монитора для импорта.
func (m *MonitorImportData) Validate() error {
	if m.Name == "" {
		return errors.New("monitor name is required")
	}

	if m.URL == "" {
		return errors.New("monitor URL is required")
	}

	// Валидация HTTP метода
	if m.Method != "" {
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
		}
		if !validMethods[m.Method] {
			return errors.New("invalid HTTP method")
		}
	}

	return nil
}

// ImportResult представляет результат импорта.
type ImportResult struct {
	ImportID         uuid.UUID
	Status           ImportStatus
	TotalMonitors    int
	SuccessfulCount  int
	FailedCount      int
	SkippedCount     int
	ValidationErrors []ValidationError
}

// NewImportResult создаёт результат импорта из истории.
func NewImportResult(history *ImportHistory) *ImportResult {
	return &ImportResult{
		ImportID:         history.ID,
		Status:           history.Status,
		TotalMonitors:    history.TotalMonitors,
		SuccessfulCount:  history.SuccessfulImports,
		FailedCount:      history.FailedImports,
		SkippedCount:     history.SkippedImports,
		ValidationErrors: history.ValidationErrors,
	}
}
