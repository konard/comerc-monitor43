-- +goose Up
CREATE TABLE scheduled_checks (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL,
    worker_id UUID,
    priority VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
    scheduled_at TIMESTAMP NOT NULL,
    executed_at TIMESTAMP,
    completed_at TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheduled_checks_monitor_id ON scheduled_checks(monitor_id);
CREATE INDEX idx_scheduled_checks_worker_id ON scheduled_checks(worker_id);
CREATE INDEX idx_scheduled_checks_status ON scheduled_checks(status);
CREATE INDEX idx_scheduled_checks_scheduled_at ON scheduled_checks(scheduled_at);

-- +goose Down
DROP INDEX IF EXISTS idx_scheduled_checks_scheduled_at;
DROP INDEX IF EXISTS idx_scheduled_checks_status;
DROP INDEX IF EXISTS idx_scheduled_checks_worker_id;
DROP INDEX IF EXISTS idx_scheduled_checks_monitor_id;
DROP TABLE IF EXISTS scheduled_checks;
