-- +goose Up
CREATE TABLE scheduler_audit_log (
    id UUID PRIMARY KEY,
    action VARCHAR(100) NOT NULL,
    worker_id UUID,
    check_id UUID,
    monitor_id UUID,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheduler_audit_log_action ON scheduler_audit_log(action);
CREATE INDEX idx_scheduler_audit_log_worker_id ON scheduler_audit_log(worker_id);
CREATE INDEX idx_scheduler_audit_log_created_at ON scheduler_audit_log(created_at);

-- +goose Down
DROP TABLE IF EXISTS scheduler_audit_log;
