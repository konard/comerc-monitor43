-- +goose Up
ALTER TABLE alerts
    ADD COLUMN IF NOT EXISTS acknowledged_by UUID,
    ADD COLUMN IF NOT EXISTS acknowledged_at TIMESTAMPTZ;

ALTER TABLE alerts
    ADD CONSTRAINT fk_alerts_acknowledged_by
        FOREIGN KEY (acknowledged_by) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged_by ON alerts(acknowledged_by) WHERE acknowledged_by IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);

-- +goose Down
DROP INDEX IF EXISTS idx_alerts_status;
DROP INDEX IF EXISTS idx_alerts_acknowledged_by;
ALTER TABLE alerts
    DROP CONSTRAINT IF EXISTS fk_alerts_acknowledged_by,
    DROP COLUMN IF EXISTS acknowledged_at,
    DROP COLUMN IF EXISTS acknowledged_by;
