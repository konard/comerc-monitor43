-- +goose Up
CREATE TABLE IF NOT EXISTS monitor_status_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL,
    user_id UUID NOT NULL,
    old_status VARCHAR(50) NOT NULL,
    new_status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_sc_monitor FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    CONSTRAINT fk_sc_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sc_monitor_id ON monitor_status_changes(monitor_id);
CREATE INDEX idx_sc_created_at ON monitor_status_changes(created_at DESC);
CREATE INDEX idx_sc_monitor_created ON monitor_status_changes(monitor_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS monitor_status_changes;
