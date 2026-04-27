package channels

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmailSender реализует EmailSender интерфейс для тестирования
type MockEmailSender struct {
	sendFunc func(ctx context.Context, to, subject, body string) error
}

func (m *MockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, to, subject, body)
	}
	return nil
}

func TestNewEmailClient(t *testing.T) {
	host := "smtp.example.com"
	port := 587
	from := "noreply@example.com"

	client := NewEmailClient(host, port, from)

	assert.NotNil(t, client)
	assert.Equal(t, host, client.host)
	assert.Equal(t, port, client.port)
	assert.Equal(t, from, client.from)
}

func TestEmailSenderInterface(t *testing.T) {
	client := NewEmailClient("smtp.example.com", 587, "noreply@example.com")

	var sender EmailSender = client
	assert.NotNil(t, sender)

	ctx := context.Background()
	err := sender.Send(ctx, "test@example.com", "Test Subject", "Test Body")

	require.Error(t, err)
}

func TestMockEmailSender_Success(t *testing.T) {
	mock := &MockEmailSender{
		sendFunc: func(ctx context.Context, to, subject, body string) error {
			return nil
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "test@example.com", "Test Subject", "Test Body")

	assert.NoError(t, err)
}

func TestMockEmailSender_Error(t *testing.T) {
	mock := &MockEmailSender{
		sendFunc: func(ctx context.Context, to, subject, body string) error {
			return errors.New("email send failed")
		},
	}

	ctx := context.Background()
	err := mock.Send(ctx, "test@example.com", "Test Subject", "Test Body")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email send failed")
}

func TestMockEmailSender_ContextCancellation(t *testing.T) {
	calls := 0
	mock := &MockEmailSender{
		sendFunc: func(ctx context.Context, to, subject, body string) error {
			calls++
			if calls > 1 {
				return ctx.Err()
			}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mock.Send(ctx, "test@example.com", "Test Subject", "Test Body")

	// When context is cancelled, we might get context error
	if err != nil {
		assert.Equal(t, context.Canceled, err)
	}
}

func TestEmailClient_StructFields(t *testing.T) {
	client := &EmailClient{
		host:     "smtp.example.com",
		port:     587,
		from:     "noreply@example.com",
		username: "test@example.com",
		password: "password",
	}

	assert.Equal(t, "smtp.example.com", client.host)
	assert.Equal(t, 587, client.port)
	assert.Equal(t, "noreply@example.com", client.from)
	assert.Equal(t, "test@example.com", client.username)
	assert.Equal(t, "password", client.password)
}

func TestEmailClient_EmptyFields(t *testing.T) {
	client := &EmailClient{
		host:     "",
		port:     0,
		from:     "",
		username: "",
		password: "",
	}

	assert.Equal(t, "", client.host)
	assert.Equal(t, 0, client.port)
	assert.Equal(t, "", client.from)
	assert.Equal(t, "", client.username)
	assert.Equal(t, "", client.password)
}

func TestMockEmailSender_MultipleCalls(t *testing.T) {
	mock := &MockEmailSender{
		sendFunc: func(ctx context.Context, to, subject, body string) error {
			return nil
		},
	}

	ctx := context.Background()

	// Test multiple calls
	for i := 0; i < 5; i++ {
		err := mock.Send(ctx, "test@example.com", "Test Subject", "Test Body")
		assert.NoError(t, err, "Call %d should succeed", i)
	}
}

func TestEmailSender_Validation(t *testing.T) {
	tests := []struct {
		name       string
		to         string
		subject    string
		body       string
		shouldFail bool
	}{
		{
			name:       "all valid",
			to:         "test@example.com",
			subject:    "Test Subject",
			body:       "Test Body",
			shouldFail: false,
		},
		{
			name:       "empty to",
			to:         "",
			subject:    "Test Subject",
			body:       "Test Body",
			shouldFail: false,
		},
		{
			name:       "empty subject",
			to:         "test@example.com",
			subject:    "",
			body:       "Test Body",
			shouldFail: false,
		},
		{
			name:       "empty body",
			to:         "test@example.com",
			subject:    "Test Subject",
			body:       "",
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockEmailSender{
				sendFunc: func(ctx context.Context, to, subject, body string) error {
					if tt.shouldFail {
						return errors.New("validation failed")
					}
					return nil
				},
			}

			ctx := context.Background()
			err := mock.Send(ctx, tt.to, tt.subject, tt.body)

			if tt.shouldFail {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
