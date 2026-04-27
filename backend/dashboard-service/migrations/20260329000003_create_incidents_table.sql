-- +goose Up
CREATE TABLE incidents (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    check_count INTEGER DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_monitor_started ON incidents(monitor_id, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_incidents_monitor_started;
DROP TABLE IF EXISTS incidents;
