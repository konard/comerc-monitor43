-- +goose Up
-- Migration: add enabled column to alerts
-- Description: алерт может быть временно отключён без удаления

ALTER TABLE alerts
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_alerts_enabled ON alerts(enabled) WHERE enabled = true;

-- +goose Down
DROP INDEX IF EXISTS idx_alerts_enabled;

ALTER TABLE alerts
    DROP COLUMN IF EXISTS enabled;
