package model

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONB_Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    JSONB
		wantNil bool
	}{
		{"nil_jsonb", nil, true},
		{"with_data", JSONB{"key": "value"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			val, err := tc.data.Value()
			if tc.wantNil {
				assert.Nil(t, val)
				assert.NoError(t, err)
			} else {
				require.NoError(t, err)
				var parsed map[string]any
				require.NoError(t, json.Unmarshal(val.([]byte), &parsed))
				assert.Equal(t, "value", parsed["key"])
			}
		})
	}
}

func TestJSONB_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil_value", func(t *testing.T) {
		t.Parallel()

		var j JSONB
		assert.NoError(t, j.Scan(nil))
		assert.Nil(t, j)
	})

	t.Run("bytes_value", func(t *testing.T) {
		t.Parallel()

		var j JSONB
		assert.NoError(t, j.Scan([]byte(`{"name":"test"}`)))
		assert.Equal(t, "test", j["name"])
	})

	t.Run("string_value", func(t *testing.T) {
		t.Parallel()

		var j JSONB
		assert.NoError(t, j.Scan(`{"name":"test"}`))
		assert.Equal(t, "test", j["name"])
	})

	t.Run("invalid_type", func(t *testing.T) {
		t.Parallel()

		var j JSONB
		assert.Error(t, j.Scan(12345))
	})
}

func TestJSONB_implements_interfaces(t *testing.T) {
	t.Parallel()

	var _ driver.Valuer = JSONB{}
}

func TestNewOAuthAccount(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	profileData := JSONB{"picture": "https://example.com/photo.jpg", "name": "Test"}
	oa := NewOAuthAccount(userID, "google", "google-123", "test@example.com", profileData)

	require.NotNil(t, oa)
	assert.NotEmpty(t, oa.ID)
	assert.Equal(t, userID, oa.UserID)
	assert.Equal(t, "google", oa.Provider)
	assert.Equal(t, "google-123", oa.ProviderUserID)
	assert.Equal(t, "test@example.com", oa.Email)
	assert.False(t, oa.CreatedAt.IsZero())
}

func TestOAuthAccount_ProfileDataValue(t *testing.T) {
	t.Parallel()

	t.Run("existing_key", func(t *testing.T) {
		t.Parallel()

		oa := &OAuthAccount{ProfileData: JSONB{"picture": "photo.jpg"}}
		val, ok := oa.ProfileDataValue("picture")
		assert.True(t, ok)
		assert.Equal(t, "photo.jpg", val)
	})

	t.Run("missing_key", func(t *testing.T) {
		t.Parallel()

		oa := &OAuthAccount{ProfileData: JSONB{"picture": "photo.jpg"}}
		val, ok := oa.ProfileDataValue("missing")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("nil_profile_data", func(t *testing.T) {
		t.Parallel()

		oa := &OAuthAccount{ProfileData: nil}
		val, ok := oa.ProfileDataValue("any")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}
