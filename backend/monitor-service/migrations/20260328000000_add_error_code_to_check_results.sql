-- +goose Up
ALTER TABLE check_results ADD COLUMN error_code VARCHAR(64);

-- +goose Down
ALTER TABLE check_results DROP COLUMN IF EXISTS error_code;
