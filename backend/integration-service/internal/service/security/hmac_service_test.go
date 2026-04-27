package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHMACService(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()
	require.NotNil(t, svc)
}

func TestHMACServiceGenerateSignature(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()

	t.Run("generates sha256 prefix", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig := svc.GenerateSignature(payload, "secret", 1700000000)
		assert.True(t, len(sig) > 7)
		assert.Equal(t, "sha256=", sig[:7])
	})

	t.Run("same input produces same signature", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig1 := svc.GenerateSignature(payload, "secret", 1700000000)
		sig2 := svc.GenerateSignature(payload, "secret", 1700000000)
		assert.Equal(t, sig1, sig2)
	})

	t.Run("different secret produces different signature", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig1 := svc.GenerateSignature(payload, "secret1", 1700000000)
		sig2 := svc.GenerateSignature(payload, "secret2", 1700000000)
		assert.NotEqual(t, sig1, sig2)
	})

	t.Run("different timestamp produces different signature", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"test":true}`)
		sig1 := svc.GenerateSignature(payload, "secret", 1700000000)
		sig2 := svc.GenerateSignature(payload, "secret", 1700000001)
		assert.NotEqual(t, sig1, sig2)
	})
}

func TestHMACServiceVerifySignature(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()
	payload := []byte(`{"test":true}`)
	secret := "mysecret"
	timestamp := int64(1700000000)

	t.Run("valid signature passes verification", func(t *testing.T) {
		t.Parallel()
		sig := svc.GenerateSignature(payload, secret, timestamp)
		assert.True(t, svc.VerifySignature(payload, secret, timestamp, sig))
	})

	t.Run("tampered payload fails verification", func(t *testing.T) {
		t.Parallel()
		sig := svc.GenerateSignature(payload, secret, timestamp)
		tamperedPayload := []byte(`{"test":false}`)
		assert.False(t, svc.VerifySignature(tamperedPayload, secret, timestamp, sig))
	})

	t.Run("wrong secret fails verification", func(t *testing.T) {
		t.Parallel()
		sig := svc.GenerateSignature(payload, secret, timestamp)
		assert.False(t, svc.VerifySignature(payload, "wrongsecret", timestamp, sig))
	})

	t.Run("wrong timestamp fails verification", func(t *testing.T) {
		t.Parallel()
		sig := svc.GenerateSignature(payload, secret, timestamp)
		assert.False(t, svc.VerifySignature(payload, secret, timestamp+1, sig))
	})
}

func TestHMACServiceValidateTimestamp(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()

	t.Run("current timestamp is valid", func(t *testing.T) {
		t.Parallel()
		now := time.Now().Unix()
		assert.True(t, svc.ValidateTimestamp(now, 5*time.Minute))
	})

	t.Run("timestamp 4 minutes ago is valid with 5 min tolerance", func(t *testing.T) {
		t.Parallel()
		fourMinutesAgo := time.Now().Add(-4 * time.Minute).Unix()
		assert.True(t, svc.ValidateTimestamp(fourMinutesAgo, 5*time.Minute))
	})

	t.Run("timestamp 10 minutes ago is invalid with 5 min tolerance", func(t *testing.T) {
		t.Parallel()
		tenMinutesAgo := time.Now().Add(-10 * time.Minute).Unix()
		assert.False(t, svc.ValidateTimestamp(tenMinutesAgo, 5*time.Minute))
	})

	t.Run("future timestamp within tolerance is valid", func(t *testing.T) {
		t.Parallel()
		future := time.Now().Add(1 * time.Minute).Unix()
		assert.True(t, svc.ValidateTimestamp(future, 5*time.Minute))
	})
}

func TestHMACServiceGenerateTimestamp(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()

	before := time.Now().Unix()
	ts := svc.GenerateTimestamp()
	after := time.Now().Unix()

	assert.GreaterOrEqual(t, ts, before)
	assert.LessOrEqual(t, ts, after)
}

func TestHMACServiceParseTimestamp(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()

	t.Run("valid timestamp string", func(t *testing.T) {
		t.Parallel()
		ts, err := svc.ParseTimestamp("1700000000")
		require.NoError(t, err)
		assert.Equal(t, int64(1700000000), ts)
	})

	t.Run("invalid timestamp string", func(t *testing.T) {
		t.Parallel()
		_, err := svc.ParseTimestamp("not-a-number")
		require.Error(t, err)
	})
}

func TestHMACServiceSignWebhookPayload(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()
	payload := []byte(`{"event":"alert"}`)
	secret := "webhook-secret"

	sig, ts := svc.SignWebhookPayload(payload, secret)
	assert.NotEmpty(t, sig)
	assert.True(t, ts > 0)
	assert.Equal(t, "sha256=", sig[:7])
}

func TestHMACServiceVerifyWebhookSignature(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()
	payload := []byte(`{"event":"alert"}`)
	secret := "webhook-secret"

	sig, ts := svc.SignWebhookPayload(payload, secret)

	t.Run("valid signature passes", func(t *testing.T) {
		t.Parallel()
		assert.True(t, svc.VerifyWebhookSignature(payload, secret, sig, ts))
	})

	t.Run("old timestamp fails", func(t *testing.T) {
		t.Parallel()
		oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
		oldSig := svc.GenerateSignature(payload, secret, oldTimestamp)
		assert.False(t, svc.VerifyWebhookSignature(payload, secret, oldSig, oldTimestamp))
	})
}

func TestHMACServiceGetSignatureHeaders(t *testing.T) {
	t.Parallel()

	svc := NewHMACService()
	payload := []byte(`{"test":true}`)
	secret := "mysecret"

	headers := svc.GetSignatureHeaders(payload, secret)

	assert.Contains(t, headers, "X-Webhook-Signature")
	assert.Contains(t, headers, "X-Webhook-Timestamp")
	assert.True(t, len(headers["X-Webhook-Signature"]) > 7)
	assert.NotEmpty(t, headers["X-Webhook-Timestamp"])
}
