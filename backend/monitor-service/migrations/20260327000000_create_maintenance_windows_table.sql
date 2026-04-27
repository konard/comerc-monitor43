-- +goose Up
CREATE TABLE maintenance_windows (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL CHECK (end_time > start_time),
    status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED',
    recurrence VARCHAR(20) NOT NULL DEFAULT 'ONCE',
    is_global BOOLEAN NOT NULL DEFAULT false,
    pause_monitoring BOOLEAN NOT NULL DEFAULT true,
    suppress_alerts BOOLEAN NOT NULL DEFAULT true,
    safe_mode BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMP,
    completed_at TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1,
    CHECK (status IN ('SCHEDULED', 'ACTIVE', 'COMPLETED', 'CANCELLED', 'ORPHANED')),
    CHECK (recurrence IN ('ONCE', 'DAILY', 'WEEKLY', 'MONTHLY'))
);

CREATE INDEX idx_maintenance_windows_user_id ON maintenance_windows(user_id);
CREATE INDEX idx_maintenance_windows_status ON maintenance_windows(status);
CREATE INDEX idx_maintenance_windows_start_time ON maintenance_windows(start_time);
CREATE INDEX idx_maintenance_windows_end_time ON maintenance_windows(end_time);
CREATE INDEX idx_maintenance_windows_user_status_time ON maintenance_windows(user_id, status, start_time);

-- Junction table for many-to-many relationship between maintenance windows and monitors
CREATE TABLE maintenance_window_monitors (
    maintenance_window_id UUID NOT NULL,
    monitor_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (maintenance_window_id, monitor_id)
);

CREATE INDEX idx_maintenance_window_monitors_window_id ON maintenance_window_monitors(maintenance_window_id);
CREATE INDEX idx_maintenance_window_monitors_monitor_id ON maintenance_window_monitors(monitor_id);

-- Foreign keys
ALTER TABLE maintenance_window_monitors
    ADD CONSTRAINT fk_maintenance_window_monitors_window
    FOREIGN KEY (maintenance_window_id)
    REFERENCES maintenance_windows(id)
    ON DELETE CASCADE;

ALTER TABLE maintenance_window_monitors
    ADD CONSTRAINT fk_maintenance_window_monitors_monitor
    FOREIGN KEY (monitor_id)
    REFERENCES monitors(id)
    ON DELETE CASCADE;

-- +goose Down
DROP INDEX IF EXISTS idx_maintenance_window_monitors_monitor_id;
DROP INDEX IF EXISTS idx_maintenance_window_monitors_window_id;
DROP TABLE IF EXISTS maintenance_window_monitors;

DROP INDEX IF EXISTS idx_maintenance_windows_user_status_time;
DROP INDEX IF EXISTS idx_maintenance_windows_end_time;
DROP INDEX IF EXISTS idx_maintenance_windows_start_time;
DROP INDEX IF EXISTS idx_maintenance_windows_status;
DROP INDEX IF EXISTS idx_maintenance_windows_user_id;
DROP TABLE IF EXISTS maintenance_windows;
