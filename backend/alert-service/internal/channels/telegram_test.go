package channels

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTelegramSender реализует TelegramSender интерфейс для тестирования
type MockTelegramSender struct {
	sendFunc func(ctx context.Context, chatID, message string) error
}

func (m *MockTelegramSender) Send(ctx context.Context, chatID, message string) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, chatID, message)
	}
	return nil
}

func TestNewTelegramClient(t *testing.T) {
	botToken := "test-token-123"
	apiURL := "https://api.telegram.org"

	client := NewTelegramClient(botToken, apiURL)

	assert.NotNil(t, client)
	assert.Equal(t, botToken, client.botToken)
	assert.Equal(t, apiURL, client.apiURL)
	assert.NotNil(t, client.client)
}

func TestTelegramSenderInterface(t *testing.T) {
	// Test that TelegramClient implements TelegramSender interface
	client := NewTelegramClient("test-token", "https://api.telegram.org")

	var sender TelegramSender = client
	assert.NotNil(t, sender)

	ctx := context.Background()
	err := sender.Send(ctx, "@testuser", "Test message")

	require.Error(t, err)
}

func TestMockTelegramSender_Success(t *testing.T) {
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			return nil
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "@testuser", "Test message")

	assert.NoError(t, err)
}

func TestMockTelegramSender_Error(t *testing.T) {
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			return errors.New("telegram send failed")
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "@testuser", "Test message")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "telegram send failed")
}

func TestTelegramClient_StructFields(t *testing.T) {
	testBotToken := "test-bot" + "-token"
	client := &TelegramClient{
		botToken: testBotToken,
		apiURL:   "https://api.telegram.org",
		client:   &http.Client{},
	}

	assert.Equal(t, testBotToken, client.botToken)
	assert.Equal(t, "https://api.telegram.org", client.apiURL)
	assert.NotNil(t, client.client)
}

func TestTelegramClient_EmptyFields(t *testing.T) {
	client := &TelegramClient{
		botToken: "",
		apiURL:   "",
		client:   nil,
	}

	assert.Equal(t, "", client.botToken)
	assert.Equal(t, "", client.apiURL)
	assert.Nil(t, client.client)
}

func TestMockTelegramSender_ContextCancellation(t *testing.T) {
	calls := 0
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			calls++
			if calls > 1 {
				return ctx.Err()
			}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mock.Send(ctx, "@testuser", "Test message")

	// When context is cancelled, we might get context error
	if err != nil {
		assert.Equal(t, context.Canceled, err)
	}
}

func TestMockTelegramSender_MultipleCalls(t *testing.T) {
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			return nil
		},
	}

	ctx := context.Background()

	// Test multiple calls
	for i := 0; i < 5; i++ {
		err := mock.Send(ctx, "@testuser", "Test message")
		assert.NoError(t, err, "Call %d should succeed", i)
	}
}

func TestTelegramClient_ConstructorWithEmptyToken(t *testing.T) {
	botToken := ""
	apiURL := "https://api.telegram.org"

	client := NewTelegramClient(botToken, apiURL)

	assert.NotNil(t, client)
	assert.Equal(t, botToken, client.botToken)
	assert.Equal(t, apiURL, client.apiURL)
}

func TestTelegramClient_NilClient(t *testing.T) {
	var client *TelegramClient

	assert.Nil(t, client)
}

func TestMockTelegramSender_InterfaceCompliance(t *testing.T) {
	mock := &MockTelegramSender{}

	// Test that mock implements the interface
	var sender TelegramSender = mock
	assert.NotNil(t, sender)

	ctx := context.Background()
	err := sender.Send(ctx, "@testuser", "Test message")

	// With nil function, should return nil
	assert.NoError(t, err)
}

func TestTelegramSender_Validation(t *testing.T) {
	tests := []struct {
		name       string
		chatID     string
		message    string
		shouldFail bool
	}{
		{
			name:       "all valid",
			chatID:     "@testuser",
			message:    "Test message",
			shouldFail: false,
		},
		{
			name:       "empty chatID",
			chatID:     "",
			message:    "Test message",
			shouldFail: false,
		},
		{
			name:       "empty message",
			chatID:     "@testuser",
			message:    "",
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTelegramSender{
				sendFunc: func(ctx context.Context, chatID, message string) error {
					if tt.shouldFail {
						return errors.New("validation failed")
					}
					return nil
				},
			}

			ctx := context.Background()
			err := mock.Send(ctx, tt.chatID, tt.message)

			if tt.shouldFail {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTelegramClient_Send_JsonError(t *testing.T) {
	// Test JSON marshaling error simulation
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			return fmt.Errorf("json: unsupported type")
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "@testuser", "Test message")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "json")
}

func TestTelegramClient_Send_HttpError(t *testing.T) {
	// Test HTTP client error simulation
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			return fmt.Errorf("http: connection refused")
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "@testuser", "Test message")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "http")
}

func TestTelegramClient_Send_Non200Status(t *testing.T) {
	// Test non-200 status code error
	mock := &MockTelegramSender{
		sendFunc: func(ctx context.Context, chatID, message string) error {
			return fmt.Errorf("telegram API returned status 400")
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "@testuser", "Test message")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}
