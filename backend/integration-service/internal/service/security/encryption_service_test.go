package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptionService(t *testing.T) {
	t.Parallel()

	t.Run("Valid encryption key", func(t *testing.T) {
		t.Parallel()

		// 32 bytes key (256 bits)
		key := strings.Repeat("a", 32)
		service, err := NewEncryptionService(key)

		assert.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotNil(t, service.encryptionKey)
		assert.Len(t, service.encryptionKey, 32)
	})

	t.Run("Invalid encryption key - too short", func(t *testing.T) {
		t.Parallel()

		key := "short"
		service, err := NewEncryptionService(key)

		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "must be exactly 32 bytes")
	})

	t.Run("Invalid encryption key - too long", func(t *testing.T) {
		t.Parallel()

		key := strings.Repeat("b", 64)
		service, err := NewEncryptionService(key)

		assert.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("Invalid encryption key - empty", func(t *testing.T) {
		t.Parallel()

		service, err := NewEncryptionService("")

		assert.Error(t, err)
		assert.Nil(t, service)
	})
}

func TestEncryptionService_Encrypt(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("x", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Encrypt valid plaintext", func(t *testing.T) {
		t.Parallel()

		plaintext := "Hello, World!"
		ciphertext, err := service.Encrypt(plaintext)

		assert.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, plaintext, ciphertext)    // ciphertext != plaintext
		assert.NotContains(t, ciphertext, plaintext) // plaintext not visible
	})

	t.Run("Encrypt empty string", func(t *testing.T) {
		t.Parallel()

		plaintext := ""
		ciphertext, err := service.Encrypt(plaintext)

		assert.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
	})

	t.Run("Encrypt special characters", func(t *testing.T) {
		t.Parallel()

		plaintext := "🔑 Secret Key: パスワード"
		ciphertext, err := service.Encrypt(plaintext)

		assert.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
	})

	t.Run("Encrypt long text", func(t *testing.T) {
		t.Parallel()

		plaintext := strings.Repeat("A", 10000)
		ciphertext, err := service.Encrypt(plaintext)

		assert.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
	})

	t.Run("Encrypt different plaintexts produce different ciphertexts", func(t *testing.T) {
		t.Parallel()

		plaintext1 := "Message 1"
		plaintext2 := "Message 2"

		ciphertext1, err1 := service.Encrypt(plaintext1)
		ciphertext2, err2 := service.Encrypt(plaintext2)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, ciphertext1, ciphertext2)
	})
}

func TestEncryptionService_Decrypt(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("y", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Decrypt valid ciphertext", func(t *testing.T) {
		t.Parallel()

		original := "Secret message"
		ciphertext, err := service.Encrypt(original)
		require.NoError(t, err)

		decrypted, err := service.Decrypt(ciphertext)

		assert.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("Decrypt empty string", func(t *testing.T) {
		t.Parallel()

		original := ""
		ciphertext, err := service.Encrypt(original)
		require.NoError(t, err)

		decrypted, err := service.Decrypt(ciphertext)

		assert.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("Decrypt special characters", func(t *testing.T) {
		t.Parallel()

		original := "🔐 128-bit: 鍵"
		ciphertext, err := service.Encrypt(original)
		require.NoError(t, err)

		decrypted, err := service.Decrypt(ciphertext)

		assert.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("Decrypt invalid base64", func(t *testing.T) {
		t.Parallel()

		invalidCiphertext := "not-valid-base64!!!"
		decrypted, err := service.Decrypt(invalidCiphertext)

		assert.Error(t, err)
		assert.Empty(t, decrypted)
		assert.Contains(t, err.Error(), "decode base64")
	})

	t.Run("Decrypt corrupted ciphertext", func(t *testing.T) {
		t.Parallel()

		// Valid base64 but corrupted ciphertext
		corruptedCiphertext := "YWJjZGVmZ2hpams=" // valid base64
		decrypted, err := service.Decrypt(corruptedCiphertext)

		assert.Error(t, err)
		assert.Empty(t, decrypted)
	})

	t.Run("Decrypt too short ciphertext", func(t *testing.T) {
		t.Parallel()

		shortCiphertext := "YWJj" // valid base64 but too short
		decrypted, err := service.Decrypt(shortCiphertext)

		assert.Error(t, err)
		assert.Empty(t, decrypted)
		assert.Contains(t, err.Error(), "too short")
	})
}

func TestEncryptionService_EncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("z", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Round trip encryption/decryption", func(t *testing.T) {
		t.Parallel()

		testCases := []string{
			"Simple message",
			"Message with numbers: 12345",
			"Special chars: !@#$%^&*()",
			"Multi\nLine\nMessage",
			strings.Repeat("A", 1000),
		}

		for _, original := range testCases {
			ciphertext, err := service.Encrypt(original)
			require.NoError(t, err)

			decrypted, err := service.Decrypt(ciphertext)
			require.NoError(t, err)

			assert.Equal(t, original, decrypted, "Round trip should preserve original")
		}
	})
}

func TestEncryptionService_EncryptAPIKey(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("a", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Encrypt API key", func(t *testing.T) {
		t.Parallel()

		apiKey := "sk_test_1234567890abcdefghijklmnop"
		encrypted, err := service.EncryptAPIKey(apiKey)

		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, apiKey, encrypted)
	})
}

func TestEncryptionService_DecryptAPIKey(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("b", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Decrypt API key", func(t *testing.T) {
		t.Parallel()

		originalKey := "sk_live_abcdef1234567890"
		encrypted, err := service.EncryptAPIKey(originalKey)
		require.NoError(t, err)

		decrypted, err := service.DecryptAPIKey(encrypted)

		assert.NoError(t, err)
		assert.Equal(t, originalKey, decrypted)
	})
}

func TestEncryptionService_EncryptWebhookSecret(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("c", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Encrypt webhook secret", func(t *testing.T) {
		t.Parallel()

		secret := "whsec_1234567890abcdefghijklmnopqrstuvwxyz"
		encrypted, err := service.EncryptWebhookSecret(secret)

		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, secret, encrypted)
	})
}

func TestEncryptionService_DecryptWebhookSecret(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("d", 32)
	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	t.Run("Decrypt webhook secret", func(t *testing.T) {
		t.Parallel()

		originalSecret := "whsec_ABCDEFGHIJKLMNOPQRSTUVWXYZ123456"
		encrypted, err := service.EncryptWebhookSecret(originalSecret)
		require.NoError(t, err)

		decrypted, err := service.DecryptWebhookSecret(encrypted)

		assert.NoError(t, err)
		assert.Equal(t, originalSecret, decrypted)
	})

	t.Run("Decrypt invalid webhook secret", func(t *testing.T) {
		t.Parallel()

		invalidSecret := "invalid-webhook-secret"
		decrypted, err := service.DecryptWebhookSecret(invalidSecret)

		assert.Error(t, err)
		assert.Empty(t, decrypted)
	})
}

func TestEncryptionService_DifferentKeys(t *testing.T) {
	t.Parallel()

	t.Run("Cannot decrypt with different key", func(t *testing.T) {
		t.Parallel()

		// Используем разные 32-байтовые ключи
		key1 := strings.Repeat("a", 32) // 32 bytes
		key2 := strings.Repeat("b", 32) // 32 bytes

		service1, err := NewEncryptionService(key1)
		require.NoError(t, err)

		service2, err := NewEncryptionService(key2)
		require.NoError(t, err)

		plaintext := "Secret message"
		ciphertext, err := service1.Encrypt(plaintext)
		require.NoError(t, err)

		// Try to decrypt with different key
		decrypted, err := service2.Decrypt(ciphertext)

		assert.Error(t, err)
		assert.Empty(t, decrypted)
		assert.Contains(t, err.Error(), "failed to decrypt")
	})
}
