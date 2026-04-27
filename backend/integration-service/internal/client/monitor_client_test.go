package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer returns default", func(t *testing.T) {
		t.Parallel()
		result := getOrDefault(nil, 42)
		assert.Equal(t, 42, result)
	})

	t.Run("non-nil pointer returns value", func(t *testing.T) {
		t.Parallel()
		v := 100
		result := getOrDefault(&v, 42)
		assert.Equal(t, 100, result)
	})

	t.Run("zero value pointer returns zero not default", func(t *testing.T) {
		t.Parallel()
		v := 0
		result := getOrDefault(&v, 42)
		assert.Equal(t, 0, result)
	})
}

func TestGetDurationOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer returns default duration", func(t *testing.T) {
		t.Parallel()
		result := getDurationOrDefault(nil, 30)
		assert.Equal(t, 30*time.Second, result)
	})

	t.Run("non-nil pointer returns value duration", func(t *testing.T) {
		t.Parallel()
		v := 60
		result := getDurationOrDefault(&v, 30)
		assert.Equal(t, 60*time.Second, result)
	})
}

func TestGetOrDefaultString(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer returns default", func(t *testing.T) {
		t.Parallel()
		result := getOrDefaultString(nil, "default")
		assert.Equal(t, "default", result)
	})

	t.Run("non-nil pointer returns value", func(t *testing.T) {
		t.Parallel()
		s := "custom"
		result := getOrDefaultString(&s, "default")
		assert.Equal(t, "custom", result)
	})

	t.Run("empty string pointer returns empty not default", func(t *testing.T) {
		t.Parallel()
		s := ""
		result := getOrDefaultString(&s, "default")
		assert.Equal(t, "", result)
	})
}
