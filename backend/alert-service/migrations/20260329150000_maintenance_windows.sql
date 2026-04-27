-- +goose Up
CREATE TABLE IF NOT EXISTS maintenance_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    monitor_id UUID NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_mw_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_mw_monitor FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
);

-- Индексы создаём условно: monitor-service владеет таблицей maintenance_windows и мог создать её с другой схемой
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='maintenance_windows' AND column_name='monitor_id') THEN
        CREATE INDEX IF NOT EXISTS idx_mw_monitor_id ON maintenance_windows(monitor_id);
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='maintenance_windows' AND column_name='starts_at')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='maintenance_windows' AND column_name='ends_at') THEN
        CREATE INDEX IF NOT EXISTS idx_mw_time_range ON maintenance_windows(starts_at, ends_at);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS maintenance_windows;
