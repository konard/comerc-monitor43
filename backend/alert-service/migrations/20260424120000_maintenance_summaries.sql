-- +goose Up
CREATE TABLE IF NOT EXISTS maintenance_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    maintenance_window_id UUID NOT NULL,
    user_id UUID NOT NULL,
    suppressed_alert_count INTEGER NOT NULL DEFAULT 0,
    email_sent_to VARCHAR(255),
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_maintenance_summaries_window ON maintenance_summaries(maintenance_window_id);
CREATE INDEX idx_maintenance_summaries_user ON maintenance_summaries(user_id);

-- +goose Down
DROP TABLE IF EXISTS maintenance_summaries;
