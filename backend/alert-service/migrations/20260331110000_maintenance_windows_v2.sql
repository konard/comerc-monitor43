-- +goose Up
ALTER TABLE maintenance_windows
    ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'scheduled',
    ADD COLUMN IF NOT EXISTS recurrence TEXT NOT NULL DEFAULT 'once',
    ADD COLUMN IF NOT EXISTS is_global BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS pause_monitoring BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS suppress_alerts BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS safe_mode BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS monitor_ids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS activated_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

-- Условные операции над monitor_id: колонка может отсутствовать, если таблица создана monitor-service
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='maintenance_windows' AND column_name='monitor_id') THEN
        ALTER TABLE maintenance_windows ALTER COLUMN monitor_id DROP NOT NULL;
        UPDATE maintenance_windows
            SET monitor_ids = ARRAY[monitor_id::text]
            WHERE monitor_id IS NOT NULL AND array_length(monitor_ids, 1) IS NULL;
    END IF;
END $$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_mw_status ON maintenance_windows(status);
CREATE INDEX IF NOT EXISTS idx_mw_user_id ON maintenance_windows(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_mw_status;
DROP INDEX IF EXISTS idx_mw_user_id;

ALTER TABLE maintenance_windows
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS recurrence,
    DROP COLUMN IF EXISTS is_global,
    DROP COLUMN IF EXISTS pause_monitoring,
    DROP COLUMN IF EXISTS suppress_alerts,
    DROP COLUMN IF EXISTS safe_mode,
    DROP COLUMN IF EXISTS monitor_ids,
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS completed_at,
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS updated_at;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='maintenance_windows' AND column_name='monitor_id') THEN
        ALTER TABLE maintenance_windows ALTER COLUMN monitor_id SET NOT NULL;
    END IF;
END $$;
-- +goose StatementEnd
