-- +goose Up
CREATE TABLE IF NOT EXISTS alert_mutes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    monitor_id UUID,
    scope VARCHAR(20) NOT NULL DEFAULT 'user',
    muted_until TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    
    CONSTRAINT fk_mutes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_mutes_monitor FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    CONSTRAINT fk_mutes_creator FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_mutes_user_id ON alert_mutes(user_id);
CREATE INDEX idx_mutes_monitor_id ON alert_mutes(monitor_id);
CREATE INDEX idx_mutes_scope ON alert_mutes(scope);
CREATE INDEX idx_mutes_until ON alert_mutes(muted_until) WHERE muted_until IS NOT NULL;

CREATE TABLE IF NOT EXISTS alert_escalations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    escalated_to_channel_id UUID,
    reason VARCHAR(50) NOT NULL DEFAULT 'timeout_no_ack',
    timeout_minutes INTEGER NOT NULL DEFAULT 30,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_escalations_alert FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,
    CONSTRAINT fk_escalations_channel FOREIGN KEY (escalated_to_channel_id) REFERENCES alert_channels(id) ON DELETE SET NULL
);

CREATE INDEX idx_escalations_alert_id ON alert_escalations(alert_id);

-- +goose Down
DROP TABLE IF EXISTS alert_escalations;
DROP TABLE IF EXISTS alert_mutes;
