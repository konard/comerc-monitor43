package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// APIKeyStatus статус API ключа.
type APIKeyStatus string

const (
	APIKeyStatusActive      APIKeyStatus = "active"
	APIKeyStatusInactive    APIKeyStatus = "inactive"
	APIKeyStatusDisabled    APIKeyStatus = "disabled"
	APIKeyStatusMaintenance APIKeyStatus = "maintenance"
)

// APIKeyType тип API ключа (определяет префикс).
type APIKeyType string

const (
	APIKeyTypeReadOnly  APIKeyType = "read_only"
	APIKeyTypeReadWrite APIKeyType = "read_write"
	APIKeyTypeAdmin     APIKeyType = "admin"
)

// Доступные scopes для API ключей.
var ValidScopes = map[string]bool{
	"read_monitors":      true,
	"write_monitors":     true,
	"read_alerts":        true,
	"write_alerts":       true,
	"read_integrations":  true,
	"write_integrations": true,
	"read_api_keys":      true,
	"write_api_keys":     true,
	"read_imports":       true,
	"write_imports":      true,
	"admin":              true,
}

// APIKey представляет API ключ.
type APIKey struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Name               string
	Description        *string
	KeyHash            string
	KeyPrefix          string
	Scopes             []string
	Status             APIKeyStatus
	SecretKey          *string // Зашифрованный полный ключ
	ExpiresAt          *time.Time
	IPWhitelist        []string
	RateLimitPerMinute int
	// Statistics
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	LastUsedAt         *time.Time
	LastUsedIP         *string
	MostUsedEndpoint   *string
	// Timestamps
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastRotatedAt *time.Time
}

// NewAPIKey создаёт новый API ключ.
func NewAPIKey(userID uuid.UUID, name string, keyType APIKeyType, scopes []string) (*APIKey, string, error) {
	if err := validateAPIKeyName(name); err != nil {
		return nil, "", err
	}

	if err := validateScopes(scopes); err != nil {
		return nil, "", err
	}

	// Генерируем секретный ключ
	secretKey, err := generateSecretKey(32)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to generate secret key")
	}

	// Добавляем префикс в зависимости от типа
	prefix := getPrefixForKeyType(keyType)
	fullKey := prefix + secretKey

	// Создаём хеш для поиска
	keyHash := sha256Hash([]byte(fullKey))

	now := time.Now()

	apiKey := &APIKey{
		ID:                 uuid.New(),
		UserID:             userID,
		Name:               name,
		KeyHash:            keyHash,
		KeyPrefix:          prefix + secretKey[:8] + "...", // Показываем только префикс + первые 8 символов
		Scopes:             scopes,
		Status:             APIKeyStatusActive,
		SecretKey:          &fullKey, // Временно сохраняем полный ключ (будет зашифрован в репозитории)
		RateLimitPerMinute: 100,
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	return apiKey, fullKey, nil
}

// validateAPIKeyName проверяет валидность имени API ключа.
func validateAPIKeyName(name string) error {
	if name == "" {
		return ErrEmptyAPIKeyName
	}
	if len(name) > 255 {
		return ErrAPIKeyNameTooLong
	}
	return nil
}

// validateScopes проверяет валидность scopes.
func validateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return ErrInvalidScope
	}

	for _, scope := range scopes {
		if !ValidScopes[scope] {
			return ErrInvalidScope
		}
	}

	return nil
}

// generateSecretKey генерирует криптографически случайный ключ.
func generateSecretKey(bytes int) (string, error) {
	key := make([]byte, bytes)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return hex.EncodeToString(key), nil
}

// sha256Hash создаёт SHA-256 хеш.
func sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// getPrefixForKeyType возвращает префикс для типа ключа.
func getPrefixForKeyType(keyType APIKeyType) string {
	switch keyType {
	case APIKeyTypeReadOnly:
		return "baku_ro_"
	case APIKeyTypeReadWrite:
		return "baku_rw_"
	case APIKeyTypeAdmin:
		return "baku_admin_"
	default:
		return "baku_"
	}
}

// IsActive возвращает true, если ключ активен.
func (k *APIKey) IsActive() bool {
	return k.Status == APIKeyStatusActive
}

// IsExpired проверяет, истёк ли ключ.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}

	// Добавляем допуск в 1 минуту для пограничных случаев
	return time.Now().After(k.ExpiresAt.Add(1 * time.Minute))
}

// IsValid проверяет, может ли ключ использоваться.
func (k *APIKey) IsValid() bool {
	return k.IsActive() && !k.IsExpired()
}

// HasScope проверяет, есть ли у ключа указанный scope.
func (k *APIKey) HasScope(requiredScope string) bool {
	// Admin ключи имеют доступ ко всем scopes
	for _, scope := range k.Scopes {
		if scope == "admin" {
			return true
		}
	}

	for _, scope := range k.Scopes {
		if scope == requiredScope {
			return true
		}
	}

	return false
}

// HasAnyScope проверяет, есть ли у ключа хотя бы один из указанных scopes.
func (k *APIKey) HasAnyScope(requiredScopes ...string) bool {
	for _, required := range requiredScopes {
		if k.HasScope(required) {
			return true
		}
	}

	return false
}

// IsIPAllowed проверяет, разрешён ли IP адрес.
func (k *APIKey) IsIPAllowed(ipAddr string) bool {
	if len(k.IPWhitelist) == 0 {
		return true // Белый список пуст - разрешаем все
	}

	// Парсим входящий IP
	clientIP := net.ParseIP(ipAddr)
	if clientIP == nil {
		return false
	}

	// Проверяем каждый IP/подсеть в белом списке
	for _, allowed := range k.IPWhitelist {
		// Проверяем точное совпадение
		if allowedIP := net.ParseIP(allowed); allowedIP != nil {
			if allowedIP.Equal(clientIP) {
				return true
			}
		}

		// Проверяем CIDR
		if _, ipNet, err := net.ParseCIDR(allowed); err == nil {
			if ipNet.Contains(clientIP) {
				return true
			}
		}
	}

	return false
}

// MarkAsUsed помечает ключ как использованный.
func (k *APIKey) MarkAsUsed(endpoint string, ipAddr string, success bool) {
	now := time.Now()
	k.TotalRequests++
	k.LastUsedAt = &now
	k.LastUsedIP = &ipAddr

	// Обновляем наиболее используемый endpoint (простая логика: меняем только если другой)
	if k.MostUsedEndpoint == nil || *k.MostUsedEndpoint != endpoint {
		k.MostUsedEndpoint = &endpoint
	}

	if success {
		k.SuccessfulRequests++
	} else {
		k.FailedRequests++
	}

	k.UpdatedAt = now
}

// Disable отключает ключ.
func (k *APIKey) Disable() {
	k.Status = APIKeyStatusDisabled
	k.UpdatedAt = time.Now()
}

// Enable включает ключ.
func (k *APIKey) Enable() {
	k.Status = APIKeyStatusActive
	k.UpdatedAt = time.Now()
}

// Deactivate деактивирует ключ.
func (k *APIKey) Deactivate() {
	k.Status = APIKeyStatusInactive
	k.UpdatedAt = time.Now()
}

// SetMaintenance устанавливает режим технического обслуживания.
func (k *APIKey) SetMaintenance() {
	k.Status = APIKeyStatusMaintenance
	k.UpdatedAt = time.Now()
}

// RotateSecret ротирует секретный ключ.
func (k *APIKey) RotateSecret() (string, error) {
	// Генерируем новый секретный ключ
	secretKey, err := generateSecretKey(32)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate new secret key")
	}

	// Сохраняем префикс из текущего ключа
	prefix := strings.TrimSuffix(k.KeyPrefix, "...")
	fullKey := prefix + secretKey

	// Обновляем хеш
	k.KeyHash = sha256Hash([]byte(fullKey))

	// Обновляем префикс для отображения
	k.KeyPrefix = prefix + secretKey[:8] + "..."

	// Временно сохраняем полный ключ (будет зашифрован в репозитории)
	k.SecretKey = &fullKey

	now := time.Now()
	k.LastRotatedAt = &now
	k.UpdatedAt = now

	return fullKey, nil
}

// GetSuccessRate возвращает процент успешных запросов.
func (k *APIKey) GetSuccessRate() float64 {
	if k.TotalRequests == 0 {
		return 0
	}

	return (float64(k.SuccessfulRequests) / float64(k.TotalRequests)) * 100
}

// APIKeyUsageLog представляет запись лога использования API ключа.
type APIKeyUsageLog struct {
	ID             uuid.UUID
	APIKeyID       uuid.UUID
	Endpoint       string
	Method         string
	HTTPStatusCode int
	ResponseTimeMs *int
	IPAddress      *string
	UserAgent      *string
	RequestID      *uuid.UUID
	RateLimited    bool
	CreatedAt      time.Time
}

// NewAPIKeyUsageLog создаёт новую запись лога использования.
func NewAPIKeyUsageLog(apiKeyID uuid.UUID, endpoint, method string, statusCode int) *APIKeyUsageLog {
	return &APIKeyUsageLog{
		ID:             uuid.New(),
		APIKeyID:       apiKeyID,
		Endpoint:       endpoint,
		Method:         method,
		HTTPStatusCode: statusCode,
		RateLimited:    false,
		CreatedAt:      time.Now(),
	}
}

// MarkRateLimited помечает запрос как rate limited.
func (l *APIKeyUsageLog) MarkRateLimited() {
	l.RateLimited = true
}

// SetResponseDetails устанавливает детали ответа.
func (l *APIKeyUsageLog) SetResponseDetails(responseTimeMs int, ipAddress, userAgent string, requestID uuid.UUID) {
	l.ResponseTimeMs = &responseTimeMs
	l.IPAddress = &ipAddress
	l.UserAgent = &userAgent
	l.RequestID = &requestID
}

// ValidateAPIKey проверяет валидность API ключа.
func ValidateAPIKey(key string) error {
	if key == "" {
		return ErrAPIKeyInvalid
	}

	// Проверяем формат: должен начинаться с правильного префикса
	validPrefixes := map[string]bool{
		"baku_ro_":    true,
		"baku_rw_":    true,
		"baku_admin_": true,
	}

	var foundPrefix string
	for prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			foundPrefix = prefix
			break
		}
	}

	if foundPrefix == "" {
		return ErrAPIKeyInvalid
	}

	// Проверяем длину: префикс + 64 hex символа
	expectedLen := len(foundPrefix) + 64
	if len(key) != expectedLen {
		return ErrAPIKeyInvalid
	}

	// Проверяем что после префикса идут только hex символы
	keyPart := key[len(foundPrefix):]
	for _, c := range keyPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) { //nolint:staticcheck // QF1001: явная проверка диапазонов читается проще, чем De Morgan
			return ErrAPIKeyInvalid
		}
	}

	return nil
}

// ExtractKeyPrefix извлекает префикс из API ключа.
func ExtractKeyPrefix(key string) string {
	if len(key) >= 13 {
		return key[:13] // Префикс (10) + первые 3 символа ключа
	}

	return key[:10]
}

// GetAPIKeyTypeFromKey определяет тип API ключа из префикса.
func GetAPIKeyTypeFromKey(key string) APIKeyType {
	if strings.HasPrefix(key, "baku_ro_") {
		return APIKeyTypeReadOnly
	}
	if strings.HasPrefix(key, "baku_rw_") {
		return APIKeyTypeReadWrite
	}
	if strings.HasPrefix(key, "baku_admin_") {
		return APIKeyTypeAdmin
	}

	return ""
}

// HashAPIKey создаёт хеш API ключа для поиска в БД.
func HashAPIKey(key string) string {
	return sha256Hash([]byte(key))
}

// GenerateAPIKeyFromPrefix генерирует новый ключ на основе префикса типа.
func GenerateAPIKeyForType(keyType APIKeyType) (string, error) {
	secretKey, err := generateSecretKey(32)
	if err != nil {
		return "", err
	}

	prefix := getPrefixForKeyType(keyType)
	return prefix + secretKey, nil
}
