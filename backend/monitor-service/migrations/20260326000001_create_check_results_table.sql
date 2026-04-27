-- +goose Up
CREATE TABLE check_results (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    response_time_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    checked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_check_results_monitor_id ON check_results(monitor_id, checked_at DESC);
CREATE INDEX idx_check_results_checked_at ON check_results(checked_at DESC);
CREATE INDEX idx_check_results_status ON check_results(status);

-- +goose Down
DROP INDEX IF EXISTS idx_check_results_status;
DROP INDEX IF EXISTS idx_check_results_checked_at;
DROP INDEX IF EXISTS idx_check_results_monitor_id;
DROP TABLE IF EXISTS check_results;
