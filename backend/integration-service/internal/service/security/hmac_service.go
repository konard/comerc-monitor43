package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// HMACService сервис для работы с HMAC подписями.
type HMACService struct{}

// NewHMACService создаёт новый HMACService.
func NewHMACService() *HMACService {
	return &HMACService{}
}

// GenerateSignature генерирует HMAC-SHA256 подпись для payload.
func (s *HMACService) GenerateSignature(payload []byte, secret string, timestamp int64) string {
	// Создаём payload для подписи: timestamp.payload
	sigPayload := fmt.Sprintf("%d.%s", timestamp, payload)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(sigPayload))

	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// VerifySignature проверяет HMAC-SHA256 подпись.
func (s *HMACService) VerifySignature(payload []byte, secret string, timestamp int64, signature string) bool {
	expectedSig := s.GenerateSignature(payload, secret, timestamp)

	// Используем hmac.Equal для постоянного времени сравнения
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// ValidateTimestamp проверяет timestamp на актуальность (защита от replay атак).
func (s *HMACService) ValidateTimestamp(timestamp int64, tolerance time.Duration) bool {
	now := time.Now().Unix()

	// Проверяем, что timestamp в пределах допустимого диапазона
	diff := now - timestamp
	if diff < 0 {
		diff = -diff
	}

	return diff <= int64(tolerance.Seconds())
}

// GenerateTimestamp генерирует текущий timestamp для подписи.
func (s *HMACService) GenerateTimestamp() int64 {
	return time.Now().Unix()
}

// ParseTimestamp парсит timestamp из строки.
func (s *HMACService) ParseTimestamp(timestampStr string) (int64, error) {
	return strconv.ParseInt(timestampStr, 10, 64)
}

// SignWebhookPayload подписывает webhook payload.
func (s *HMACService) SignWebhookPayload(payload []byte, secretKey string) (string, int64) {
	timestamp := s.GenerateTimestamp()
	signature := s.GenerateSignature(payload, secretKey, timestamp)

	return signature, timestamp
}

// VerifyWebhookSignature верифицирует подпись webhook.
func (s *HMACService) VerifyWebhookSignature(payload []byte, secretKey string, signature string, timestamp int64) bool {
	// Проверяем timestamp (±5 минут)
	if !s.ValidateTimestamp(timestamp, 5*time.Minute) {
		return false
	}

	// Проверяем подпись
	return s.VerifySignature(payload, secretKey, timestamp, signature)
}

// GetSignatureHeaders возвращает заголовки для подписанного webhook.
func (s *HMACService) GetSignatureHeaders(payload []byte, secretKey string) map[string]string {
	timestamp := s.GenerateTimestamp()
	signature := s.GenerateSignature(payload, secretKey, timestamp)

	return map[string]string{
		"X-Webhook-Signature": signature,
		"X-Webhook-Timestamp": strconv.FormatInt(timestamp, 10),
	}
}
