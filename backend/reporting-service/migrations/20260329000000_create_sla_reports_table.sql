-- +goose Up
CREATE TABLE sla_reports (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL,
    user_id UUID NOT NULL,
    monitor_name VARCHAR(255) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    availability DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_checks INTEGER NOT NULL DEFAULT 0,
    up_checks INTEGER NOT NULL DEFAULT 0,
    down_checks INTEGER NOT NULL DEFAULT 0,
    degraded_checks INTEGER NOT NULL DEFAULT 0,
    paused_checks INTEGER NOT NULL DEFAULT 0,
    total_downtime_seconds BIGINT NOT NULL DEFAULT 0,
    incidents_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sla_reports_monitor_id ON sla_reports(monitor_id);
CREATE INDEX idx_sla_reports_user_id ON sla_reports(user_id);
CREATE INDEX idx_sla_reports_created_at ON sla_reports(created_at);
CREATE UNIQUE INDEX idx_sla_reports_monitor_period ON sla_reports(monitor_id, period_start, period_end);

-- +goose Down
DROP INDEX IF EXISTS idx_sla_reports_monitor_period;
DROP INDEX IF EXISTS idx_sla_reports_created_at;
DROP INDEX IF EXISTS idx_sla_reports_user_id;
DROP INDEX IF EXISTS idx_sla_reports_monitor_id;
DROP TABLE IF EXISTS sla_reports;
