-- +goose Up
CREATE TABLE monitors (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    check_type VARCHAR(50) NOT NULL DEFAULT 'HTTP',
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds >= 30 AND interval_seconds <= 3600),
    timeout_seconds INTEGER NOT NULL DEFAULT 30 CHECK (timeout_seconds < interval_seconds),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    working_hours_start TIME,
    working_hours_end TIME,
    working_days VARCHAR(20)[],
    degraded_response_time_threshold INTEGER,
    degraded_failure_rate_threshold INTEGER,
    last_check_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_monitors_user_id ON monitors(user_id);
CREATE INDEX idx_monitors_status ON monitors(status);
CREATE INDEX idx_monitors_created_at ON monitors(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_monitors_created_at;
DROP INDEX IF EXISTS idx_monitors_status;
DROP INDEX IF EXISTS idx_monitors_user_id;
DROP TABLE IF EXISTS monitors;
