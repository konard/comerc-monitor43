package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAlertRepositorySimple(t *testing.T) {
	repo := NewAlertRepository(nil)
	assert.NotNil(t, repo)
}
