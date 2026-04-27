package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/pure-golang/adapters/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/auth-service/internal/publisher/mocks"
)

// queuePublisher — частично применяемый интерфейс зависимости queue.Publisher.
// Содержит только метод, реально используемый PureGolangPublisher.
type queuePublisher interface {
	Publish(ctx context.Context, msgs ...queue.Message) error
}

// Проверяем, что *mocks.QueuePublisher удовлетворяет локальному интерфейсу.
var _ queuePublisher = (*mocks.QueuePublisher)(nil)

// --- NewEvent ---

func TestNewEvent(t *testing.T) {
	t.Parallel()

	event := NewEvent(EventTypeUserCreated, "user-123", map[string]any{
		"email": "test@example.com",
	})

	require.NotNil(t, event)
	assert.NotEmpty(t, event.ID)
	assert.Equal(t, EventTypeUserCreated, event.Type)
	assert.Equal(t, "auth-service", event.Source)
	assert.Equal(t, "user-123", event.Subject)
	assert.Equal(t, "test@example.com", event.Data["email"])
	assert.NotZero(t, event.Timestamp)
	assert.NotNil(t, event.Metadata)
}

func TestNewEvent_unique_ids(t *testing.T) {
	t.Parallel()

	e1 := NewEvent(EventTypeUserCreated, "u1", nil)
	e2 := NewEvent(EventTypeUserCreated, "u2", nil)

	assert.NotEqual(t, e1.ID, e2.ID)
}

// --- NoopPublisher ---

func TestNoopPublisher(t *testing.T) {
	t.Parallel()

	p := NewNoopPublisher()
	ctx := context.Background()

	assert.NoError(t, p.PublishUserCreated(ctx, "uid", "email"))
	assert.NoError(t, p.PublishUserLoggedIn(ctx, "uid", "email", "ip", "ua"))
	assert.NoError(t, p.PublishUserFailedLogin(ctx, "uid", "email", "ip", "reason"))
	assert.NoError(t, p.PublishUserLocked(ctx, "uid", "email", 3))
	assert.NoError(t, p.PublishUserUnlocked(ctx, "uid", "email"))
	assert.NoError(t, p.PublishOAuthAccountLinked(ctx, "uid", "email", "google", "provider-id"))
	assert.NoError(t, p.PublishOAuthAccountUnlinked(ctx, "uid", "email", "google"))
	assert.NoError(t, p.PublishSessionCreated(ctx, "uid", "email", "sess-id", 9999))
	assert.NoError(t, p.PublishSessionRevoked(ctx, "uid", "email", "sess-id", "logout"))
	assert.NoError(t, p.PublishPasswordChanged(ctx, "uid", "email", "reset"))
	assert.NoError(t, p.PublishPasswordReset(ctx, "uid", "email"))
}

// --- PureGolangPublisher ---

func newPureGolangPublisher(t *testing.T, qp *mocks.QueuePublisher) Publisher {
	t.Helper()
	return NewPureGolangPublisher(qp, slog.Default())
}

func TestPureGolangPublisher_PublishUserCreated(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishUserCreated(context.Background(), "uid", "email@example.com")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishUserCreated_error(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(fmt.Errorf("publish failed"))
	p := newPureGolangPublisher(t, qp)

	err := p.PublishUserCreated(context.Background(), "uid", "email@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish event")
}

func TestPureGolangPublisher_PublishUserLoggedIn(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishUserLoggedIn(context.Background(), "uid", "email", "127.0.0.1", "Chrome")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishUserFailedLogin(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishUserFailedLogin(context.Background(), "uid", "email", "ip", "bad password")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishUserLocked(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishUserLocked(context.Background(), "uid", "email", 5)
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishUserUnlocked(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishUserUnlocked(context.Background(), "uid", "email")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishOAuthAccountLinked(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishOAuthAccountLinked(context.Background(), "uid", "email", "google", "gid")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishOAuthAccountUnlinked(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishOAuthAccountUnlinked(context.Background(), "uid", "email", "google")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishSessionCreated(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishSessionCreated(context.Background(), "uid", "email", "sess-id", 9999)
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishSessionRevoked(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishSessionRevoked(context.Background(), "uid", "email", "sess-id", "logout")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishPasswordChanged(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishPasswordChanged(context.Background(), "uid", "email", "manual")
	require.NoError(t, err)
}

func TestPureGolangPublisher_PublishPasswordReset(t *testing.T) {
	t.Parallel()

	qp := mocks.NewQueuePublisher(t)
	qp.EXPECT().Publish(mock.Anything, mock.Anything).Return(nil)
	p := newPureGolangPublisher(t, qp)

	err := p.PublishPasswordReset(context.Background(), "uid", "email")
	require.NoError(t, err)
}
