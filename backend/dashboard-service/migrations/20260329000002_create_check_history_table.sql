-- +goose Up
CREATE TABLE check_history (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    status_code INTEGER,
    response_time_ms DECIMAL(10,2),
    error_message TEXT,
    checked_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_check_history_monitor_time ON check_history(monitor_id, checked_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_check_history_monitor_time;
DROP TABLE IF EXISTS check_history;
