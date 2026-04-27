package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptionService сервис для шифрования/дешифрования данных.
type EncryptionService struct {
	encryptionKey []byte
}

// NewEncryptionService создаёт новый EncryptionService.
func NewEncryptionService(encryptionKey string) (*EncryptionService, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes")
	}

	return &EncryptionService{
		encryptionKey: []byte(encryptionKey),
	}, nil
}

// Encrypt шифрует данные используя AES-256-GCM.
func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Создаём GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Генерируем случайный nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Шифруем данные
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Кодируем в base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt дешифрует данные используя AES-256-GCM.
func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
	// Декодируем из base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Создаём GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Извлекаем nonce и ciphertext
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// Дешифруем данные
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptAPIKey шифрует API ключ.
func (s *EncryptionService) EncryptAPIKey(apiKey string) (string, error) {
	return s.Encrypt(apiKey)
}

// DecryptAPIKey дешифрует API ключ.
func (s *EncryptionService) DecryptAPIKey(encryptedKey string) (string, error) {
	return s.Decrypt(encryptedKey)
}

// EncryptWebhookSecret шифрует секрет webhook.
func (s *EncryptionService) EncryptWebhookSecret(secret string) (string, error) {
	return s.Encrypt(secret)
}

// DecryptWebhookSecret дешифрует секрет webhook.
func (s *EncryptionService) DecryptWebhookSecret(encryptedSecret string) (string, error) {
	return s.Decrypt(encryptedSecret)
}
