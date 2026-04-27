-- +goose Up
ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS name VARCHAR(255);

-- +goose Down
ALTER TABLE alert_channels DROP COLUMN IF EXISTS name;
