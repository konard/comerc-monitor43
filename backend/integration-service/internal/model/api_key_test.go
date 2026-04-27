package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKey(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name        string
		keyName     string
		keyType     APIKeyType
		scopes      []string
		wantErr     error
		checkPrefix string
	}{
		{
			name:        "valid read-only key",
			keyName:     "Read Only Key",
			keyType:     APIKeyTypeReadOnly,
			scopes:      []string{"read_monitors"},
			wantErr:     nil,
			checkPrefix: "baku_ro_",
		},
		{
			name:        "valid read-write key",
			keyName:     "Read Write Key",
			keyType:     APIKeyTypeReadWrite,
			scopes:      []string{"read_monitors", "write_monitors"},
			wantErr:     nil,
			checkPrefix: "baku_rw_",
		},
		{
			name:        "valid admin key",
			keyName:     "Admin Key",
			keyType:     APIKeyTypeAdmin,
			scopes:      []string{"admin"},
			wantErr:     nil,
			checkPrefix: "baku_admin_",
		},
		{
			name:    "empty name",
			keyName: "",
			keyType: APIKeyTypeReadOnly,
			scopes:  []string{"read_monitors"},
			wantErr: ErrEmptyAPIKeyName,
		},
		{
			name:    "name too long",
			keyName: string(make([]byte, 256)),
			keyType: APIKeyTypeReadOnly,
			scopes:  []string{"read_monitors"},
			wantErr: ErrAPIKeyNameTooLong,
		},
		{
			name:    "no scopes",
			keyName: "Test Key",
			keyType: APIKeyTypeReadOnly,
			scopes:  []string{},
			wantErr: ErrInvalidScope,
		},
		{
			name:    "invalid scope",
			keyName: "Test Key",
			keyType: APIKeyTypeReadOnly,
			scopes:  []string{"invalid_scope"},
			wantErr: ErrInvalidScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, fullKey, err := NewAPIKey(userID, tt.keyName, tt.keyType, tt.scopes)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, apiKey)
				assert.Empty(t, fullKey)
			} else {
				require.NoError(t, err)
				require.NotNil(t, apiKey)
				require.NotEmpty(t, fullKey)

				// Check basic fields
				assert.NotEqual(t, uuid.Nil, apiKey.ID)
				assert.Equal(t, userID, apiKey.UserID)
				assert.Equal(t, tt.keyName, apiKey.Name)
				assert.Equal(t, APIKeyStatusActive, apiKey.Status)

				// Check prefix
				assert.Contains(t, fullKey, tt.checkPrefix)
				assert.Contains(t, apiKey.KeyPrefix, tt.checkPrefix)

				// Check key format (prefix + 64 hex chars)
				assert.Equal(t, len(tt.checkPrefix)+64, len(fullKey))

				// Check scopes
				assert.Equal(t, tt.scopes, apiKey.Scopes)

				// Check that key hash is set
				assert.NotEmpty(t, apiKey.KeyHash)

				// Check that full key matches hash
				assert.Equal(t, HashAPIKey(fullKey), apiKey.KeyHash)

				// Check default values
				assert.Equal(t, 100, apiKey.RateLimitPerMinute)
				assert.Equal(t, 0, apiKey.TotalRequests)
			}
		})
	}
}

func TestAPIKey_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   APIKeyStatus
		expected bool
	}{
		{
			name:     "active",
			status:   APIKeyStatusActive,
			expected: true,
		},
		{
			name:     "inactive",
			status:   APIKeyStatusInactive,
			expected: false,
		},
		{
			name:     "disabled",
			status:   APIKeyStatusDisabled,
			expected: false,
		},
		{
			name:     "maintenance",
			status:   APIKeyStatusMaintenance,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{Status: tt.status}
			result := key.IsActive()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKey_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{
			name:      "not expired (nil)",
			expiresAt: nil,
			expected:  false,
		},
		{
			name:      "not expired (future)",
			expiresAt: &future,
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: &past,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{ExpiresAt: tt.expiresAt}
			result := key.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKey_IsValid(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		status    APIKeyStatus
		expiresAt *time.Time
		expected  bool
	}{
		{
			name:      "valid active key",
			status:    APIKeyStatusActive,
			expiresAt: &future,
			expected:  true,
		},
		{
			name:      "inactive key",
			status:    APIKeyStatusInactive,
			expiresAt: &future,
			expected:  false,
		},
		{
			name:      "expired key",
			status:    APIKeyStatusActive,
			expiresAt: &past,
			expected:  false,
		},
		{
			name:      "active key with no expiration",
			status:    APIKeyStatusActive,
			expiresAt: nil,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{
				Status:    tt.status,
				ExpiresAt: tt.expiresAt,
			}
			result := key.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKey_HasScope(t *testing.T) {
	tests := []struct {
		name          string
		scopes        []string
		requiredScope string
		expected      bool
	}{
		{
			name:          "has required scope",
			scopes:        []string{"read_monitors", "write_monitors"},
			requiredScope: "read_monitors",
			expected:      true,
		},
		{
			name:          "does not have scope",
			scopes:        []string{"read_monitors"},
			requiredScope: "write_monitors",
			expected:      false,
		},
		{
			name:          "admin has all scopes",
			scopes:        []string{"admin"},
			requiredScope: "any_scope",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{Scopes: tt.scopes}
			result := key.HasScope(tt.requiredScope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKey_HasAnyScope(t *testing.T) {
	key := &APIKey{Scopes: []string{"read_monitors", "write_monitors"}}

	assert.True(t, key.HasAnyScope("read_monitors"))
	assert.True(t, key.HasAnyScope("write_monitors"))
	assert.True(t, key.HasAnyScope("read_monitors", "write_alerts"))
	assert.False(t, key.HasAnyScope("delete_monitors"))
	assert.False(t, key.HasAnyScope("delete_monitors", "delete_alerts"))
}

func TestAPIKey_IsIPAllowed(t *testing.T) {
	tests := []struct {
		name        string
		ipWhitelist []string
		testIP      string
		expected    bool
	}{
		{
			name:        "empty whitelist allows all",
			ipWhitelist: []string{},
			testIP:      "192.168.1.1",
			expected:    true,
		},
		{
			name:        "exact match",
			ipWhitelist: []string{"192.168.1.1"},
			testIP:      "192.168.1.1",
			expected:    true,
		},
		{
			name:        "CIDR match",
			ipWhitelist: []string{"192.168.1.0/24"},
			testIP:      "192.168.1.100",
			expected:    true,
		},
		{
			name:        "IP not in whitelist",
			ipWhitelist: []string{"192.168.1.1"},
			testIP:      "192.168.1.2",
			expected:    false,
		},
		{
			name:        "multiple IPs",
			ipWhitelist: []string{"192.168.1.1", "10.0.0.1"},
			testIP:      "10.0.0.1",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{IPWhitelist: tt.ipWhitelist}
			result := key.IsIPAllowed(tt.testIP)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKey_MarkAsUsed(t *testing.T) {
	key := &APIKey{}

	endpoint := "/api/v1/monitors"
	ipAddr := "192.168.1.1"

	key.MarkAsUsed(endpoint, ipAddr, true)
	key.MarkAsUsed(endpoint, ipAddr, true)
	key.MarkAsUsed(endpoint, ipAddr, false)

	assert.Equal(t, 3, key.TotalRequests)
	assert.Equal(t, 2, key.SuccessfulRequests)
	assert.Equal(t, 1, key.FailedRequests)
	assert.NotNil(t, key.LastUsedAt)
	assert.Equal(t, ipAddr, *key.LastUsedIP)
	assert.NotNil(t, key.MostUsedEndpoint)
}

func TestAPIKey_GetSuccessRate(t *testing.T) {
	tests := []struct {
		name               string
		totalRequests      int
		successfulRequests int
		expectedRate       float64
	}{
		{
			name:               "100% success",
			totalRequests:      10,
			successfulRequests: 10,
			expectedRate:       100.0,
		},
		{
			name:               "50% success",
			totalRequests:      10,
			successfulRequests: 5,
			expectedRate:       50.0,
		},
		{
			name:               "0% success",
			totalRequests:      10,
			successfulRequests: 0,
			expectedRate:       0.0,
		},
		{
			name:               "no requests",
			totalRequests:      0,
			successfulRequests: 0,
			expectedRate:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{
				TotalRequests:      tt.totalRequests,
				SuccessfulRequests: tt.successfulRequests,
			}

			rate := key.GetSuccessRate()
			assert.Equal(t, tt.expectedRate, rate)
		})
	}
}

func TestAPIKey_Disable(t *testing.T) {
	key := &APIKey{Status: APIKeyStatusActive}

	key.Disable()

	assert.Equal(t, APIKeyStatusDisabled, key.Status)
}

func TestAPIKey_Enable(t *testing.T) {
	key := &APIKey{Status: APIKeyStatusDisabled}

	key.Enable()

	assert.Equal(t, APIKeyStatusActive, key.Status)
}

func TestAPIKey_Deactivate(t *testing.T) {
	key := &APIKey{Status: APIKeyStatusActive}

	key.Deactivate()

	assert.Equal(t, APIKeyStatusInactive, key.Status)
}

func TestAPIKey_SetMaintenance(t *testing.T) {
	key := &APIKey{Status: APIKeyStatusActive}

	key.SetMaintenance()

	assert.Equal(t, APIKeyStatusMaintenance, key.Status)
}

func TestAPIKey_RotateSecret(t *testing.T) {
	userID := uuid.New()
	key, _, err := NewAPIKey(userID, "Test Key", APIKeyTypeReadWrite, []string{"read_monitors"})
	require.NoError(t, err)

	oldHash := key.KeyHash
	oldPrefix := key.KeyPrefix

	newFullKey, err := key.RotateSecret()

	require.NoError(t, err)
	assert.NotEmpty(t, newFullKey)

	// Key hash should be different
	assert.NotEqual(t, oldHash, key.KeyHash)

	// Prefix should be different (after the 8 character display)
	assert.NotEqual(t, oldPrefix, key.KeyPrefix)

	// New key should hash to new hash
	assert.Equal(t, HashAPIKey(newFullKey), key.KeyHash)

	// LastRotatedAt should be set
	assert.NotNil(t, key.LastRotatedAt)
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{
			name:    "valid read-only key",
			key:     "baku_ro_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: nil,
		},
		{
			name:    "valid read-write key",
			key:     "baku_rw_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: nil,
		},
		{
			name:    "valid admin key",
			key:     "baku_admin_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: nil,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: ErrAPIKeyInvalid,
		},
		{
			name:    "invalid prefix",
			key:     "invalid_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: ErrAPIKeyInvalid,
		},
		{
			name:    "wrong length",
			key:     "baku_ro_too_short",
			wantErr: ErrAPIKeyInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAPIKeyTypeFromKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected APIKeyType
	}{
		{
			name:     "read-only key",
			key:      "baku_ro_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: APIKeyTypeReadOnly,
		},
		{
			name:     "read-write key",
			key:      "baku_rw_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: APIKeyTypeReadWrite,
		},
		{
			name:     "admin key",
			key:      "baku_admin_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: APIKeyTypeAdmin,
		},
		{
			name:     "unknown prefix",
			key:      "unknown_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAPIKeyTypeFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "baku_ro_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	hash1 := HashAPIKey(key)
	hash2 := HashAPIKey(key)

	// Same key should produce same hash
	assert.Equal(t, hash1, hash2)

	// Hash should be 64 hex chars (SHA-256)
	assert.Equal(t, 64, len(hash1))
}

func TestGenerateAPIKeyForType(t *testing.T) {
	tests := []struct {
		name        string
		keyType     APIKeyType
		checkPrefix string
	}{
		{
			name:        "read-only",
			keyType:     APIKeyTypeReadOnly,
			checkPrefix: "baku_ro_",
		},
		{
			name:        "read-write",
			keyType:     APIKeyTypeReadWrite,
			checkPrefix: "baku_rw_",
		},
		{
			name:        "admin",
			keyType:     APIKeyTypeAdmin,
			checkPrefix: "baku_admin_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateAPIKeyForType(tt.keyType)
			require.NoError(t, err)
			assert.True(t, len(key) >= 10) // At least prefix
			assert.Contains(t, key, tt.checkPrefix)

			// Validate generated key
			err = ValidateAPIKey(key)
			assert.NoError(t, err)
		})
	}
}

func TestNewAPIKeyUsageLog(t *testing.T) {
	apiKeyID := uuid.New()

	log := NewAPIKeyUsageLog(apiKeyID, "/api/v1/monitors", "GET", 200)

	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.Equal(t, apiKeyID, log.APIKeyID)
	assert.Equal(t, "/api/v1/monitors", log.Endpoint)
	assert.Equal(t, "GET", log.Method)
	assert.Equal(t, 200, log.HTTPStatusCode)
	assert.False(t, log.RateLimited)
	assert.NotNil(t, log.CreatedAt)
}

func TestAPIKeyUsageLog_MarkRateLimited(t *testing.T) {
	apiKeyID := uuid.New()

	log := NewAPIKeyUsageLog(apiKeyID, "/api/v1/monitors", "GET", 200)
	log.MarkRateLimited()

	assert.True(t, log.RateLimited)
}

func TestAPIKeyUsageLog_SetResponseDetails(t *testing.T) {
	apiKeyID := uuid.New()

	log := NewAPIKeyUsageLog(apiKeyID, "/api/v1/monitors", "GET", 200)

	responseTimeMs := 150
	ipAddress := "192.168.1.1"
	userAgent := "TestAgent/1.0"
	requestID := uuid.New()

	log.SetResponseDetails(responseTimeMs, ipAddress, userAgent, requestID)

	assert.Equal(t, &responseTimeMs, log.ResponseTimeMs)
	assert.Equal(t, &ipAddress, log.IPAddress)
	assert.Equal(t, &userAgent, log.UserAgent)
	assert.Equal(t, &requestID, log.RequestID)
}

func TestExtractKeyPrefix(t *testing.T) {
	t.Parallel()

	t.Run("key longer than 13 chars returns first 13", func(t *testing.T) {
		t.Parallel()
		key := "baku_rw_ABCDEFGHIJK"
		prefix := ExtractKeyPrefix(key)
		assert.Equal(t, "baku_rw_ABCDE", prefix)
		assert.Len(t, prefix, 13)
	})

	t.Run("short key returns first 10", func(t *testing.T) {
		t.Parallel()
		key := "baku_rw_AB" // exactly 10 chars
		prefix := ExtractKeyPrefix(key)
		assert.Equal(t, key, prefix)
	})
}
