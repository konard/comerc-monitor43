-- +goose Up
CREATE TABLE auth_audit_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id),
  event_type VARCHAR(50) NOT NULL,  -- login, logout, oauth_callback, session_created, session_revoked, account_locked, account_unlocked
  provider VARCHAR(50),  -- for OAuth events
  success BOOLEAN NOT NULL,
  ip_address INET,
  user_agent VARCHAR(500),
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_audit_log_user_id ON auth_audit_log(user_id);
CREATE INDEX idx_auth_audit_log_event_type ON auth_audit_log(event_type);
CREATE INDEX idx_auth_audit_log_created_at ON auth_audit_log(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_auth_audit_log_created_at;
DROP INDEX IF EXISTS idx_auth_audit_log_event_type;
DROP INDEX IF EXISTS idx_auth_audit_log_user_id;
DROP TABLE IF EXISTS auth_audit_log;
