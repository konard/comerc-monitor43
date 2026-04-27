-- +goose Up
-- SQL в этой секции выполняется при миграции вверх.

CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    channel VARCHAR(50) NOT NULL CHECK (channel IN ('email', 'telegram', 'webhook', 'slack', 'discord', 'sms')),
    type VARCHAR(50) NOT NULL CHECK (type IN ('monitor_up', 'monitor_down', 'monitor_degraded', 'certificate_expiry', 'flapping_detected', 'incident_created', 'incident_resolved', 'maintenance_started', 'maintenance_ended', 'custom')),
    engine VARCHAR(50) NOT NULL DEFAULT 'gotemplate' CHECK (engine IN ('gotemplate', 'jinja2', 'handlebars')),
    subject TEXT,
    body TEXT NOT NULL,
    format VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (format IN ('text', 'html', 'markdown', 'json')),
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_system BOOLEAN NOT NULL DEFAULT false,
    version INT NOT NULL DEFAULT 1,
    parent_id UUID REFERENCES templates(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT templates_user_id_name_channel_type_unique UNIQUE (user_id, name, channel, type)
);

-- Индексы для быстрого поиска
CREATE INDEX idx_templates_user_id ON templates(user_id);
CREATE INDEX idx_templates_channel_type ON templates(channel, type);
CREATE INDEX idx_templates_is_default ON templates(is_default) WHERE is_default = true;
CREATE INDEX idx_templates_parent_id ON templates(parent_id) WHERE parent_id IS NOT NULL;

-- Триггер для автоматического обновления updated_at
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_templates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW
    EXECUTE FUNCTION update_templates_updated_at();

-- +goose Down
-- SQL в этой секции выполняется при откате миграции.

DROP TRIGGER IF EXISTS trigger_templates_updated_at ON templates;
DROP FUNCTION IF EXISTS update_templates_updated_at();
DROP INDEX IF EXISTS idx_templates_parent_id;
DROP INDEX IF EXISTS idx_templates_is_default;
DROP INDEX IF EXISTS idx_templates_channel_type;
DROP INDEX IF EXISTS idx_templates_user_id;
DROP TABLE IF EXISTS templates;
