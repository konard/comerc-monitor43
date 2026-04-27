package repository

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// mustParseUUID is a helper function to parse UUID or fail the test
func mustParseUUID(t *testing.T, s string) uuid.UUID {
	uid, err := uuid.Parse(s)
	require.NoError(t, err)
	return uid
}

// generateTestUUID generates a valid test UUID with the given number
func generateTestUUID(num int) uuid.UUID {
	// Use a fixed base UUID and vary the last few digits based on num
	// This ensures deterministic UUID generation for tests
	// Standard UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 characters total)
	uuidStr := "99de8400-e29b-41d4-a716-446655440000"
	// Pad num to 2 digits with leading zeros (max 99 to fit within 2 digits)
	numStr := fmt.Sprintf("%02d", num%100)
	// Replace the last 2 characters of the base UUID with the num
	fullUUID := uuidStr[:34] + numStr
	return uuid.MustParse(fullUUID)
}
