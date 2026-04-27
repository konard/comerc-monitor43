-- +goose Up
CREATE TABLE monitor_audit_log (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    old_values TEXT,
    new_values TEXT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitor_audit_log_monitor_id ON monitor_audit_log(monitor_id, created_at DESC);
CREATE INDEX idx_monitor_audit_log_user_id ON monitor_audit_log(user_id, created_at DESC);
CREATE INDEX idx_monitor_audit_log_action ON monitor_audit_log(action, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_monitor_audit_log_action;
DROP INDEX IF EXISTS idx_monitor_audit_log_user_id;
DROP INDEX IF EXISTS idx_monitor_audit_log_monitor_id;
DROP TABLE IF EXISTS monitor_audit_log;
