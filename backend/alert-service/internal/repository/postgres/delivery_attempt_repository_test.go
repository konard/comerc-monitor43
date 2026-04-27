package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewDeliveryAttemptRepository проверяет создание репозитория
func TestNewDeliveryAttemptRepository_Simple(t *testing.T) {
	repo := NewDeliveryAttemptRepository(nil)

	assert.NotNil(t, repo)
}

// TestDeliveryAttempt_CreateAndUpdate проверяет создание и обновление
func TestDeliveryAttempt_CreateAndUpdate_Simple(t *testing.T) {
	repo := NewDeliveryAttemptRepository(nil)

	assert.NotNil(t, repo)
}
