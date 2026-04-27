package channels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateChatID_ValidFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		chatID string
		valid  bool
	}{
		{name: "supergroup_negative", chatID: "-1001234567890", valid: true},
		{name: "regular_positive", chatID: "123456789", valid: true},
		{name: "simple_id", chatID: "42", valid: true},
		{name: "empty", chatID: "", valid: false},
		{name: "invalid_letters", chatID: "abc", valid: false},
		{name: "invalid_mixed", chatID: "invalid", valid: false},
		{name: "spaces", chatID: " 123", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chatIDPattern.MatchString(tt.chatID)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestValidateURL_ValidFormats(t *testing.T) {
	t.Parallel()

	service := &ChannelService{}

	tests := []struct {
		name    string
		url     string
		isValid bool
	}{
		{name: "https_url", url: "https://example.com", isValid: true},
		{name: "http_url", url: "http://example.com/webhook", isValid: true},
		{name: "https_with_path", url: "https://example.com/api/v1/webhook", isValid: true},
		{name: "empty_string", url: "", isValid: false},
		{name: "not_a_url", url: "not-a-url", isValid: false},
		{name: "ftp_scheme", url: "ftp://example.com", isValid: false},
		{name: "no_scheme", url: "example.com", isValid: false},
		{name: "url_injection", url: "https://evil.com/x SSTIInject", isValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateURL(tt.url)
			if tt.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
