-- +goose Up
-- Migration: Initial schema for alert-service
-- Description: Creates tables for alert channels, rules, alerts, and delivery attempts

-- Alert channels table
CREATE TABLE IF NOT EXISTS alert_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'unverified',
    enabled BOOLEAN NOT NULL DEFAULT true,
    verified BOOLEAN NOT NULL DEFAULT false,
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_failure_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    telegram_config TEXT,
    email_config TEXT,
    webhook_config TEXT,

    CONSTRAINT fk_alert_channels_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Alert rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    monitor_id UUID NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    consecutive_failures INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_alert_rules_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_alert_rules_monitor FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    CONSTRAINT uq_user_monitor UNIQUE (user_id, monitor_id)
);

-- Alert channel priorities table
CREATE TABLE IF NOT EXISTS alert_channel_priorities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID NOT NULL,
    alert_channel_id UUID NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_acp_alert_rule FOREIGN KEY (alert_rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_acp_alert_channel FOREIGN KEY (alert_channel_id) REFERENCES alert_channels(id) ON DELETE CASCADE,
    CONSTRAINT uq_rule_channel UNIQUE (alert_rule_id, alert_channel_id)
);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    monitor_id UUID NOT NULL,
    alert_rule_id UUID,
    status VARCHAR(50) NOT NULL DEFAULT 'triggered',
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    threshold_ms INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    type VARCHAR(50) NOT NULL,
    config TEXT,

    CONSTRAINT fk_alerts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_alerts_monitor FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    CONSTRAINT fk_alerts_rule FOREIGN KEY (alert_rule_id) REFERENCES alert_rules(id) ON DELETE SET NULL
);

-- Delivery attempts table
CREATE TABLE IF NOT EXISTS delivery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL,
    alert_channel_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_delivery_alerts FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,
    CONSTRAINT fk_delivery_channels FOREIGN KEY (alert_channel_id) REFERENCES alert_channels(id) ON DELETE CASCADE
);

-- Indexes for alert_channels
CREATE INDEX IF NOT EXISTS idx_alert_channels_user_id ON alert_channels(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_channels_type ON alert_channels(type);
CREATE INDEX IF NOT EXISTS idx_alert_channels_status ON alert_channels(status);
CREATE INDEX IF NOT EXISTS idx_alert_channels_enabled ON alert_channels(enabled) WHERE enabled = true;

-- Indexes for alert_rules
CREATE INDEX IF NOT EXISTS idx_alert_rules_user_id ON alert_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_monitor_id ON alert_rules(monitor_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled) WHERE enabled = true;

-- Indexes for alert_channel_priorities
CREATE INDEX IF NOT EXISTS idx_acp_rule_id ON alert_channel_priorities(alert_rule_id);
CREATE INDEX IF NOT EXISTS idx_acp_channel_id ON alert_channel_priorities(alert_channel_id);

-- Indexes for alerts
CREATE INDEX IF NOT EXISTS idx_alerts_user_id ON alerts(user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_monitor_id ON alerts(monitor_id);
CREATE INDEX IF NOT EXISTS idx_alerts_rule_id ON alerts(alert_rule_id);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_user_status ON alerts(user_id, status);

-- Indexes for delivery_attempts
CREATE INDEX IF NOT EXISTS idx_delivery_alert_id ON delivery_attempts(alert_id);
CREATE INDEX IF NOT EXISTS idx_delivery_channel_id ON delivery_attempts(alert_channel_id);
CREATE INDEX IF NOT EXISTS idx_delivery_status ON delivery_attempts(status);
CREATE INDEX IF NOT EXISTS idx_delivery_next_retry ON delivery_attempts(next_retry_at) WHERE next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_delivery_status_retry ON delivery_attempts(status, next_retry_at);

-- Function to update updated_at timestamp
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Triggers for updated_at
CREATE TRIGGER update_alert_channels_updated_at
    BEFORE UPDATE ON alert_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_rules_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_delivery_attempts_updated_at
    BEFORE UPDATE ON delivery_attempts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS update_delivery_attempts_updated_at ON delivery_attempts;
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;
DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;
DROP TRIGGER IF EXISTS update_alert_channels_updated_at ON alert_channels;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS delivery_attempts;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS alert_channel_priorities;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS alert_channels;
