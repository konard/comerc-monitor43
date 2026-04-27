-- +goose Up
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    plan_id VARCHAR(50) NOT NULL REFERENCES subscription_plans(id),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    canceled_at TIMESTAMP,
    auto_renew BOOLEAN NOT NULL DEFAULT TRUE,
    provider_payment_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_expires_at ON subscriptions(expires_at);
CREATE UNIQUE INDEX idx_subscriptions_user_active_pending_unique
    ON subscriptions(user_id, status)
    WHERE status IN ('ACTIVE', 'PENDING');

-- +goose Down
DROP INDEX IF EXISTS idx_subscriptions_user_active_pending_unique;
DROP INDEX IF EXISTS idx_subscriptions_expires_at;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_user_id;
DROP TABLE IF EXISTS subscriptions;
