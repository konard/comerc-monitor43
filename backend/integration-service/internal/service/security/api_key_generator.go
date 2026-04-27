package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// APIKeyGenerator генерирует API ключи.
type APIKeyGenerator struct{}

// NewAPIKeyGenerator создаёт новый APIKeyGenerator.
func NewAPIKeyGenerator() *APIKeyGenerator {
	return &APIKeyGenerator{}
}

// GenerateKey генерирует криптографически случайный ключ.
func (g *APIKeyGenerator) GenerateKey(bytes int) (string, error) {
	key := make([]byte, bytes)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	return hex.EncodeToString(key), nil
}

// GenerateAPIKey генерирует API ключ с префиксом.
func (g *APIKeyGenerator) GenerateAPIKey(keyType string) (string, error) {
	// Генерируем 32-байтовый ключ (64 hex символа)
	secretKey, err := g.GenerateKey(32)
	if err != nil {
		return "", err
	}

	// Добавляем префикс в зависимости от типа
	prefix := g.getPrefixForKeyType(keyType)
	fullKey := prefix + secretKey

	return fullKey, nil
}

// getPrefixForKeyType возвращает префикс для типа ключа.
func (g *APIKeyGenerator) getPrefixForKeyType(keyType string) string {
	switch strings.ToLower(keyType) {
	case "read_only", "readonly", "ro":
		return "baku_ro_"
	case "read_write", "readwrite", "rw":
		return "baku_rw_"
	case "admin":
		return "baku_admin_"
	default:
		return "baku_"
	}
}

// HashAPIKey создаёт SHA-256 хеш API ключа для хранения в БД.
func (g *APIKeyGenerator) HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKeyFormat проверяет формат API ключа.
func (g *APIKeyGenerator) ValidateAPIKeyFormat(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Проверяем префикс
	validPrefixes := []string{"baku_ro_", "baku_rw_", "baku_admin_"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(apiKey, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("invalid API key prefix")
	}

	// Проверяем длину: префикс (10) + 64 hex символа = 74
	expectedLen := 10 + 64
	if len(apiKey) != expectedLen {
		return fmt.Errorf("invalid API key length (expected %d, got %d)", expectedLen, len(apiKey))
	}

	// Проверяем, что остальная часть - валидный hex
	hexPart := apiKey[10:]
	if _, err := hex.DecodeString(hexPart); err != nil {
		return fmt.Errorf("API key contains invalid hex characters")
	}

	return nil
}

// ExtractKeyPrefix извлекает префикс для отображения.
func (g *APIKeyGenerator) ExtractKeyPrefix(apiKey string) string {
	if len(apiKey) >= 13 {
		// Префикс (10) + первые 3 символа ключа + "..."
		return apiKey[:13] + "..."
	}

	return apiKey[:10] + "..."
}

// GetAPIKeyTypeFromKey определяет тип API ключа из префикса.
func (g *APIKeyGenerator) GetAPIKeyTypeFromKey(apiKey string) string {
	if strings.HasPrefix(apiKey, "baku_ro_") {
		return "read_only"
	}
	if strings.HasPrefix(apiKey, "baku_rw_") {
		return "read_write"
	}
	if strings.HasPrefix(apiKey, "baku_admin_") {
		return "admin"
	}

	return ""
}

// GenerateSecretForWebhook генерирует секрет для webhook подписи.
func (g *APIKeyGenerator) GenerateSecretForWebhook() (string, error) {
	// Генерируем 32-байтовый секрет
	secret, err := g.GenerateKey(32)
	if err != nil {
		return "", err
	}

	return secret, nil
}

// VerifyAPIKey verifies API key against hash.
func (g *APIKeyGenerator) VerifyAPIKey(apiKey, keyHash string) bool {
	computedHash := g.HashAPIKey(apiKey)

	// Используем hmac.Equal для постоянного времени сравнения
	return hmac.Equal([]byte(computedHash), []byte(keyHash))
}
