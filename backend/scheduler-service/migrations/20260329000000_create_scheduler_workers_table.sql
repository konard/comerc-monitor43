-- +goose Up
CREATE TABLE scheduler_workers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    zone VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'IDLE',
    last_heartbeat TIMESTAMP NOT NULL DEFAULT NOW(),
    checks_completed INTEGER NOT NULL DEFAULT 0,
    checks_failed INTEGER NOT NULL DEFAULT 0,
    avg_check_duration_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheduler_workers_status ON scheduler_workers(status);
CREATE INDEX idx_scheduler_workers_zone ON scheduler_workers(zone);
CREATE INDEX idx_scheduler_workers_last_heartbeat ON scheduler_workers(last_heartbeat);

-- +goose Down
DROP INDEX IF EXISTS idx_scheduler_workers_last_heartbeat;
DROP INDEX IF EXISTS idx_scheduler_workers_zone;
DROP INDEX IF EXISTS idx_scheduler_workers_status;
DROP TABLE IF EXISTS scheduler_workers;
