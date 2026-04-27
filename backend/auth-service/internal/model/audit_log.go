package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an auth audit log entry for compliance (152-ФЗ)
type AuditLog struct {
	ID           uuid.UUID  `json:"id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	EventType    string     `json:"event_type"`         // login, logout, oauth_callback, session_created, session_revoked, account_locked, account_unlocked
	Provider     string     `json:"provider,omitempty"` // for OAuth events
	Success      bool       `json:"success"`
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(userID *uuid.UUID, eventType, provider string, success bool, ipAddress, userAgent, errorMessage string) *AuditLog {
	return &AuditLog{
		ID:           uuid.New(),
		UserID:       userID,
		EventType:    eventType,
		Provider:     provider,
		Success:      success,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ErrorMessage: errorMessage,
		CreatedAt:    time.Now(),
	}
}

// Event types for audit logging
const (
	EventTypeLogin           = "login"
	EventTypeLogout          = "logout"
	EventTypeOAuthCallback   = "oauth_callback"
	EventTypeSessionCreated  = "session_created"
	EventTypeSessionRevoked  = "session_revoked"
	EventTypeAccountLocked   = "account_locked"
	EventTypeAccountUnlocked = "account_unlocked"
	EventTypeTokenRefresh    = "token_refresh"
	EventTypeTokenValidate   = "token_validate"
	EventTypeRegister        = "register"

	// Team management event types (uc_05_02_*).
	EventTypeMemberInvited        = "member_invited"
	EventTypeInviteAccepted       = "invite_accepted"
	EventTypeInviteRevoked        = "invite_revoked"
	EventTypeInviteResent         = "invite_resent"
	EventTypeMemberRoleChanged    = "member_role_changed"
	EventTypeMemberRemoved        = "member_removed"
	EventTypeOwnershipTransferred = "ownership_transferred"
	EventTypeMemberLeft           = "member_left"
	EventTypeOrganizationCreated  = "organization_created"
)
