package model

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashToken(t *testing.T) {
	t.Parallel()

	result := HashToken("my-secret-token")

	expected := sha256.Sum256([]byte("my-secret-token"))
	expectedHex := hex.EncodeToString(expected[:])
	assert.Equal(t, expectedHex, result)
}

func TestHashToken_deterministic(t *testing.T) {
	t.Parallel()

	assert.Equal(t, HashToken("same-input"), HashToken("same-input"))
}

func TestHashToken_different_inputs(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, HashToken("input-a"), HashToken("input-b"))
}

func TestHashToken_empty(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, HashToken(""))
}
