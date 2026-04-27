-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE monitor_statuses (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    uptime_percentage DECIMAL(5,2) DEFAULT 0,
    last_checked_at TIMESTAMPTZ,
    last_response_time_ms DECIMAL(10,2),
    tags TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitor_statuses_user_id ON monitor_statuses(user_id);
CREATE INDEX idx_monitor_statuses_status ON monitor_statuses(status);
CREATE INDEX idx_monitor_statuses_name_trgm ON monitor_statuses USING gin (name gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS idx_monitor_statuses_name_trgm;
DROP INDEX IF EXISTS idx_monitor_statuses_status;
DROP INDEX IF EXISTS idx_monitor_statuses_user_id;
DROP TABLE IF EXISTS monitor_statuses;
