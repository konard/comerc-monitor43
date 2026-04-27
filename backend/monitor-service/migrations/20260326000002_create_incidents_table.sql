-- +goose Up
CREATE TABLE incidents (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_seconds INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_monitor_id ON incidents(monitor_id, start_time DESC);
CREATE INDEX idx_incidents_status ON incidents(status);
CREATE INDEX idx_incidents_start_time ON incidents(start_time DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_incidents_start_time;
DROP INDEX IF EXISTS idx_incidents_status;
DROP INDEX IF EXISTS idx_incidents_monitor_id;
DROP TABLE IF EXISTS incidents;
