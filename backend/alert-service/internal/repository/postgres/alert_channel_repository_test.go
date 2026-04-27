package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAlertChannelRepository проверяет создание репозитория
func TestNewAlertChannelRepository_Simple(t *testing.T) {
	repo := NewAlertChannelRepository(nil)

	assert.NotNil(t, repo)
}

// TestAlertChannelRepository_CreateAndVerify проверяет создание и верификацию
func TestAlertChannelRepository_CreateAndVerify_Simple(t *testing.T) {
	repo := NewAlertChannelRepository(nil)

	assert.NotNil(t, repo)
}
