-- +goose Up
CREATE TABLE billing_audit_log (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_audit_user_id ON billing_audit_log(user_id);
CREATE INDEX idx_billing_audit_entity ON billing_audit_log(entity_type, entity_id);
CREATE INDEX idx_billing_audit_created_at ON billing_audit_log(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_billing_audit_created_at;
DROP INDEX IF EXISTS idx_billing_audit_entity;
DROP INDEX IF EXISTS idx_billing_audit_user_id;
DROP TABLE IF EXISTS billing_audit_log;
