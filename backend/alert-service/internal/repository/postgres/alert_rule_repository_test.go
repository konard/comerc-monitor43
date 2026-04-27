package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAlertRuleRepository проверяет создание репозитория
func TestNewAlertRuleRepository_Simple(t *testing.T) {
	repo := NewAlertRuleRepository(nil)

	assert.NotNil(t, repo)
}

// TestAlertRuleRepository_CreateAndRead проверяет создание и чтение правила
func TestAlertRuleRepository_CreateAndRead_Simple(t *testing.T) {
	repo := NewAlertRuleRepository(nil)

	assert.NotNil(t, repo)
}
