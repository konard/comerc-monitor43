package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pure-golang/adapters/queue"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// Publisher defines event publisher interface
type Publisher interface {
	PublishUserCreated(ctx context.Context, userID, email string) error
	PublishUserLoggedIn(ctx context.Context, userID, email, ip, userAgent string) error
	PublishUserFailedLogin(ctx context.Context, userID, email, ip, reason string) error
	PublishUserLocked(ctx context.Context, userID, email string, attempts int) error
	PublishUserUnlocked(ctx context.Context, userID, email string) error
	PublishOAuthAccountLinked(ctx context.Context, userID, email, provider, providerID string) error
	PublishOAuthAccountUnlinked(ctx context.Context, userID, email, provider string) error
	PublishSessionCreated(ctx context.Context, userID, email, sessionID string, expiresAt int64) error
	PublishSessionRevoked(ctx context.Context, userID, email, sessionID, reason string) error
	PublishPasswordChanged(ctx context.Context, userID, email string, method string) error
	PublishPasswordReset(ctx context.Context, userID, email string) error
	PublishInviteCreated(ctx context.Context, inv *model.Invite) error
	PublishMemberRoleChanged(ctx context.Context, orgID, userID uuid.UUID, oldRole, newRole model.Role) error
	PublishMemberRemoved(ctx context.Context, orgID, userID uuid.UUID) error
}

// EventType represents type of event
type EventType string

const (
	EventTypeUserCreated          EventType = "user.created"
	EventTypeUserLoggedIn         EventType = "user.logged_in"
	EventTypeUserFailedLogin      EventType = "user.failed_login"
	EventTypeUserLocked           EventType = "user.locked"
	EventTypeUserUnlocked         EventType = "user.unlocked"
	EventTypeOAuthAccountLinked   EventType = "oauth_account.linked"
	EventTypeOAuthAccountUnlinked EventType = "oauth_account.unlinked"
	EventTypeSessionCreated       EventType = "session.created"
	EventTypeSessionRevoked       EventType = "session.revoked"
	EventTypePasswordChanged      EventType = "password.changed"
	EventTypePasswordReset        EventType = "password.reset"
	EventTypeInviteCreated        EventType = "team.invite.created"
	EventTypeMemberRoleChanged    EventType = "team.member.role_changed"
	EventTypeMemberRemoved        EventType = "team.member.removed"
)

// Event represents a domain event
type Event struct {
	ID        string         `json:"id"`
	Type      EventType      `json:"type"`
	Source    string         `json:"source"`
	Subject   string         `json:"subject"` // resource affected (e.g., user ID)
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, subject string, data map[string]any) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    "auth-service",
		Subject:   subject,
		Timestamp: time.Now().UTC(),
		Data:      data,
		Metadata:  make(map[string]any),
	}
}

// PureGolangPublisher implements Publisher interface using pure-golang RabbitMQ adapter
type PureGolangPublisher struct {
	publisher queue.Publisher
	logger    *slog.Logger
	exchange  string
}

// NewPureGolangPublisher creates a new event publisher using pure-golang RabbitMQ adapter
func NewPureGolangPublisher(publisher queue.Publisher, logger *slog.Logger) Publisher {
	return &PureGolangPublisher{
		publisher: publisher,
		logger:    logger,
		exchange:  "auth.events",
	}
}

// publish publishes an event to RabbitMQ
func (p *PureGolangPublisher) publish(ctx context.Context, event *Event, routingKey string) error {
	msg := queue.Message{
		Topic: routingKey,
		Body:  event,
		Headers: map[string]string{
			"event_id":   event.ID,
			"event_type": string(event.Type),
			"source":     event.Source,
			"subject":    event.Subject,
		},
	}

	err := p.publisher.Publish(ctx, msg)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to publish event",
			"event_type", string(event.Type),
			"event_id", event.ID,
			"error", err,
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.DebugContext(ctx, "Event published",
		"event_type", string(event.Type),
		"event_id", event.ID,
		"subject", event.Subject,
	)

	return nil
}

func (p *PureGolangPublisher) PublishUserCreated(ctx context.Context, userID, email string) error {
	event := NewEvent(EventTypeUserCreated, userID, map[string]any{
		"user_id": userID,
		"email":   email,
	})
	return p.publish(ctx, event, string(EventTypeUserCreated))
}

func (p *PureGolangPublisher) PublishUserLoggedIn(ctx context.Context, userID, email, ip, userAgent string) error {
	event := NewEvent(EventTypeUserLoggedIn, userID, map[string]any{
		"user_id":    userID,
		"email":      email,
		"ip":         ip,
		"user_agent": userAgent,
	})
	return p.publish(ctx, event, string(EventTypeUserLoggedIn))
}

func (p *PureGolangPublisher) PublishUserFailedLogin(ctx context.Context, userID, email, ip, reason string) error {
	event := NewEvent(EventTypeUserFailedLogin, userID, map[string]any{
		"user_id": userID,
		"email":   email,
		"ip":      ip,
		"reason":  reason,
	})
	return p.publish(ctx, event, string(EventTypeUserFailedLogin))
}

func (p *PureGolangPublisher) PublishUserLocked(ctx context.Context, userID, email string, attempts int) error {
	event := NewEvent(EventTypeUserLocked, userID, map[string]any{
		"user_id":  userID,
		"email":    email,
		"attempts": attempts,
	})
	return p.publish(ctx, event, string(EventTypeUserLocked))
}

func (p *PureGolangPublisher) PublishUserUnlocked(ctx context.Context, userID, email string) error {
	event := NewEvent(EventTypeUserUnlocked, userID, map[string]any{
		"user_id": userID,
		"email":   email,
	})
	return p.publish(ctx, event, string(EventTypeUserUnlocked))
}

func (p *PureGolangPublisher) PublishOAuthAccountLinked(ctx context.Context, userID, email, provider, providerID string) error {
	event := NewEvent(EventTypeOAuthAccountLinked, userID, map[string]any{
		"user_id":     userID,
		"email":       email,
		"provider":    provider,
		"provider_id": providerID,
	})
	return p.publish(ctx, event, string(EventTypeOAuthAccountLinked))
}

func (p *PureGolangPublisher) PublishOAuthAccountUnlinked(ctx context.Context, userID, email, provider string) error {
	event := NewEvent(EventTypeOAuthAccountUnlinked, userID, map[string]any{
		"user_id":  userID,
		"email":    email,
		"provider": provider,
	})
	return p.publish(ctx, event, string(EventTypeOAuthAccountUnlinked))
}

func (p *PureGolangPublisher) PublishSessionCreated(ctx context.Context, userID, email, sessionID string, expiresAt int64) error {
	event := NewEvent(EventTypeSessionCreated, userID, map[string]any{
		"user_id":    userID,
		"email":      email,
		"session_id": sessionID,
		"expires_at": expiresAt,
	})
	return p.publish(ctx, event, string(EventTypeSessionCreated))
}

func (p *PureGolangPublisher) PublishSessionRevoked(ctx context.Context, userID, email, sessionID, reason string) error {
	event := NewEvent(EventTypeSessionRevoked, userID, map[string]any{
		"user_id":    userID,
		"email":      email,
		"session_id": sessionID,
		"reason":     reason,
	})
	return p.publish(ctx, event, string(EventTypeSessionRevoked))
}

func (p *PureGolangPublisher) PublishPasswordChanged(ctx context.Context, userID, email, method string) error {
	event := NewEvent(EventTypePasswordChanged, userID, map[string]any{
		"user_id": userID,
		"email":   email,
		"method":  method,
	})
	return p.publish(ctx, event, string(EventTypePasswordChanged))
}

func (p *PureGolangPublisher) PublishPasswordReset(ctx context.Context, userID, email string) error {
	event := NewEvent(EventTypePasswordReset, userID, map[string]any{
		"user_id": userID,
		"email":   email,
	})
	return p.publish(ctx, event, string(EventTypePasswordReset))
}

func (p *PureGolangPublisher) PublishInviteCreated(ctx context.Context, inv *model.Invite) error {
	event := NewEvent(EventTypeInviteCreated, inv.ID.String(), map[string]any{
		"invite_id":       inv.ID.String(),
		"organization_id": inv.OrganizationID.String(),
		"inviter_id":      inv.InviterID.String(),
		"email":           inv.Email,
		"role":            string(inv.Role),
		"token":           inv.Token.String(),
		"expires_at":      inv.ExpiresAt.UTC().Format(time.RFC3339),
	})
	return p.publish(ctx, event, string(EventTypeInviteCreated))
}

func (p *PureGolangPublisher) PublishMemberRoleChanged(ctx context.Context, orgID, userID uuid.UUID, oldRole, newRole model.Role) error {
	event := NewEvent(EventTypeMemberRoleChanged, userID.String(), map[string]any{
		"organization_id": orgID.String(),
		"user_id":         userID.String(),
		"old_role":        string(oldRole),
		"new_role":        string(newRole),
	})
	return p.publish(ctx, event, string(EventTypeMemberRoleChanged))
}

func (p *PureGolangPublisher) PublishMemberRemoved(ctx context.Context, orgID, userID uuid.UUID) error {
	event := NewEvent(EventTypeMemberRemoved, userID.String(), map[string]any{
		"organization_id": orgID.String(),
		"user_id":         userID.String(),
	})
	return p.publish(ctx, event, string(EventTypeMemberRemoved))
}

// noopPublisher implements Publisher with no-op operations
type noopPublisher struct{}

// NewNoopPublisher creates a no-op publisher
func NewNoopPublisher() Publisher {
	return &noopPublisher{}
}

func (p *noopPublisher) PublishUserCreated(ctx context.Context, userID, email string) error {
	return nil
}
func (p *noopPublisher) PublishUserLoggedIn(ctx context.Context, userID, email, ip, userAgent string) error {
	return nil
}
func (p *noopPublisher) PublishUserFailedLogin(ctx context.Context, userID, email, ip, reason string) error {
	return nil
}
func (p *noopPublisher) PublishUserLocked(ctx context.Context, userID, email string, attempts int) error {
	return nil
}
func (p *noopPublisher) PublishUserUnlocked(ctx context.Context, userID, email string) error {
	return nil
}
func (p *noopPublisher) PublishOAuthAccountLinked(ctx context.Context, userID, email, provider, providerID string) error {
	return nil
}
func (p *noopPublisher) PublishOAuthAccountUnlinked(ctx context.Context, userID, email, provider string) error {
	return nil
}
func (p *noopPublisher) PublishSessionCreated(ctx context.Context, userID, email, sessionID string, expiresAt int64) error {
	return nil
}
func (p *noopPublisher) PublishSessionRevoked(ctx context.Context, userID, email, sessionID, reason string) error {
	return nil
}
func (p *noopPublisher) PublishPasswordChanged(ctx context.Context, userID, email, method string) error {
	return nil
}
func (p *noopPublisher) PublishPasswordReset(ctx context.Context, userID, email string) error {
	return nil
}
func (p *noopPublisher) PublishInviteCreated(ctx context.Context, inv *model.Invite) error {
	return nil
}
func (p *noopPublisher) PublishMemberRoleChanged(ctx context.Context, orgID, userID uuid.UUID, oldRole, newRole model.Role) error {
	return nil
}
func (p *noopPublisher) PublishMemberRemoved(ctx context.Context, orgID, userID uuid.UUID) error {
	return nil
}
