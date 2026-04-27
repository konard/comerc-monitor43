package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewDB_Error тестирует ошибку при создании подключения.
func TestNewDB_Error(t *testing.T) {
	t.Run("invalid dsn", func(t *testing.T) {
		_, err := NewDB("invalid dsn")
		assert.Error(t, err)
	})

	t.Run("connection refused", func(t *testing.T) {
		_, err := NewDB("postgres://localhost:9999/test?sslmode=disable")
		assert.Error(t, err)
	})
}
