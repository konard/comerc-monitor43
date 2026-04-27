package model

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken hashes a token using SHA-256 for secure storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
