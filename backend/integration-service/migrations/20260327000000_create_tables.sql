-- +goose Up
-- Migration: Create integration service tables
-- Created: 2026-03-27

-- ============================================
-- Table: webhook_integrations
-- ============================================
CREATE TABLE IF NOT EXISTS webhook_integrations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers TEXT DEFAULT '{}',                     -- JSON stored as TEXT per DOD §4.1
    secret_key VARCHAR(255),                      -- HMAC secret (encrypted)
    enabled BOOLEAN NOT NULL DEFAULT true,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',

    -- Conditional sending
    severity_filter TEXT DEFAULT '["critical","warning","degraded"]',  -- JSON stored as TEXT per DOD §4.1

    -- Payload handling
    max_payload_size_bytes INTEGER NOT NULL DEFAULT 1048576,
    payload_handling_strategy VARCHAR(20) NOT NULL DEFAULT 'truncate',

    -- Statistics
    total_sent INTEGER NOT NULL DEFAULT 0,
    successful_sent INTEGER NOT NULL DEFAULT 0,
    failed_sent INTEGER NOT NULL DEFAULT 0,
    avg_response_time_ms INTEGER,
    last_sent_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,

    -- Failure tracking
    failure_count INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT webhook_integrations_user_id_name_unique UNIQUE (user_id, name),
    CONSTRAINT webhook_integrations_status_check CHECK (status IN ('active', 'inactive', 'failed', 'disabled')),
    CONSTRAINT webhook_integrations_priority_check CHECK (priority IN ('high', 'normal', 'low')),
    CONSTRAINT webhook_integrations_payload_strategy_check CHECK (payload_handling_strategy IN ('truncate', 'reject'))
);

-- Indexes for webhook_integrations
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_user_id ON webhook_integrations(user_id);
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_status ON webhook_integrations(status);
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_enabled ON webhook_integrations(enabled);
CREATE INDEX IF NOT EXISTS idx_webhook_integrations_priority ON webhook_integrations(priority);

-- ============================================
-- Table: webhook_delivery_attempts
-- ============================================
CREATE TABLE IF NOT EXISTS webhook_delivery_attempts (
    id UUID PRIMARY KEY,
    webhook_id UUID NOT NULL REFERENCES webhook_integrations(id) ON DELETE CASCADE,
    alert_id UUID NOT NULL,

    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE,

    http_status_code INTEGER,
    response_time_ms INTEGER,
    error_message TEXT,
    error_code VARCHAR(100),

    payload_size_bytes INTEGER,
    payload_truncated BOOLEAN DEFAULT false,
    signature_algorithm VARCHAR(50),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT webhook_delivery_attempts_status_check CHECK (status IN ('pending', 'sent', 'failed', 'retry_scheduled'))
);

-- Indexes for webhook_delivery_attempts
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_attempts_webhook_id ON webhook_delivery_attempts(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_attempts_status ON webhook_delivery_attempts(status);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_attempts_next_retry ON webhook_delivery_attempts(next_retry_at)
    WHERE next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_attempts_created_at ON webhook_delivery_attempts(created_at DESC);

-- ============================================
-- Table: api_keys
-- ============================================
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(50) NOT NULL,

    scopes TEXT NOT NULL DEFAULT '[]',            -- JSON stored as TEXT per DOD §4.1
    status VARCHAR(50) NOT NULL DEFAULT 'active',

    secret_key VARCHAR(255), -- Encrypted full key
    expires_at TIMESTAMP WITH TIME ZONE,

    ip_whitelist TEXT DEFAULT '[]',              -- JSON stored as TEXT per DOD §4.1
    rate_limit_per_minute INTEGER NOT NULL DEFAULT 100,

    -- Statistics
    total_requests INTEGER NOT NULL DEFAULT 0,
    successful_requests INTEGER NOT NULL DEFAULT 0,
    failed_requests INTEGER NOT NULL DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip VARCHAR(45),
    most_used_endpoint VARCHAR(255),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_rotated_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT api_keys_user_id_name_unique UNIQUE (user_id, name),
    CONSTRAINT api_keys_status_check CHECK (status IN ('active', 'inactive', 'disabled', 'maintenance'))
);

-- Indexes for api_keys
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at)
    WHERE expires_at IS NOT NULL;

-- ============================================
-- Table: api_key_usage_logs
-- ============================================
CREATE TABLE IF NOT EXISTS api_key_usage_logs (
    id UUID PRIMARY KEY,
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,

    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    http_status_code INTEGER NOT NULL,
    response_time_ms INTEGER,

    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id UUID,

    rate_limited BOOLEAN DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for api_key_usage_logs
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_api_key_id ON api_key_usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_created_at ON api_key_usage_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_rate_limited ON api_key_usage_logs(rate_limited)
    WHERE rate_limited = true;

-- ============================================
-- Table: import_history
-- ============================================
CREATE TABLE IF NOT EXISTS import_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,

    source VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    total_monitors INTEGER NOT NULL DEFAULT 0,
    successful_imports INTEGER NOT NULL DEFAULT 0,
    failed_imports INTEGER NOT NULL DEFAULT 0,
    skipped_imports INTEGER NOT NULL DEFAULT 0,

    overwrite_existing BOOLEAN NOT NULL DEFAULT false,

    imported_monitor_ids TEXT DEFAULT '[]',      -- JSON stored as TEXT per DOD §4.1
    validation_errors TEXT DEFAULT '[]',         -- JSON stored as TEXT per DOD §4.1
    conflict_resolution TEXT DEFAULT '{}',        -- JSON stored as TEXT per DOD §4.1

    file_name VARCHAR(255),
    file_size_bytes INTEGER,

    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,

    error_message TEXT,

    CONSTRAINT import_history_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'partial')),
    CONSTRAINT import_history_source_check CHECK (source IN ('uptimerobot', 'pingdom', 'csv', 'json'))
);

-- Indexes for import_history
CREATE INDEX IF NOT EXISTS idx_import_history_user_id ON import_history(user_id);
CREATE INDEX IF NOT EXISTS idx_import_history_source ON import_history(source);
CREATE INDEX IF NOT EXISTS idx_import_history_status ON import_history(status);
CREATE INDEX IF NOT EXISTS idx_import_history_started_at ON import_history(started_at DESC);

-- ============================================
-- Comments
-- ============================================
COMMENT ON TABLE webhook_integrations IS 'Webhook интеграции для отправки алертов во внешние системы';
COMMENT ON TABLE webhook_delivery_attempts IS 'Попытки доставки webhook с retry логикой';
COMMENT ON TABLE api_keys IS 'API ключи для программного доступа к системе';
COMMENT ON TABLE api_key_usage_logs IS 'Логи использования API ключей для аналитики';
COMMENT ON TABLE import_history IS 'История импорта мониторов из внешних систем';

-- +goose Down
-- Drop tables in reverse order due to foreign keys
DROP INDEX IF EXISTS idx_import_history_started_at;
DROP INDEX IF EXISTS idx_import_history_status;
DROP INDEX IF EXISTS idx_import_history_source;
DROP INDEX IF EXISTS idx_import_history_user_id;
DROP TABLE IF EXISTS import_history;

DROP INDEX IF EXISTS idx_api_key_usage_logs_rate_limited;
DROP INDEX IF EXISTS idx_api_key_usage_logs_created_at;
DROP INDEX IF EXISTS idx_api_key_usage_logs_api_key_id;
DROP TABLE IF EXISTS api_key_usage_logs;

DROP INDEX IF EXISTS idx_api_keys_expires_at;
DROP INDEX IF EXISTS idx_api_keys_status;
DROP INDEX IF EXISTS idx_api_keys_key_prefix;
DROP INDEX IF EXISTS idx_api_keys_key_hash;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP TABLE IF EXISTS api_keys;

DROP INDEX IF EXISTS idx_webhook_delivery_attempts_created_at;
DROP INDEX IF EXISTS idx_webhook_delivery_attempts_next_retry;
DROP INDEX IF EXISTS idx_webhook_delivery_attempts_status;
DROP INDEX IF EXISTS idx_webhook_delivery_attempts_webhook_id;
DROP TABLE IF EXISTS webhook_delivery_attempts;

DROP INDEX IF EXISTS idx_webhook_integrations_priority;
DROP INDEX IF EXISTS idx_webhook_integrations_enabled;
DROP INDEX IF EXISTS idx_webhook_integrations_status;
DROP INDEX IF EXISTS idx_webhook_integrations_user_id;
DROP TABLE IF EXISTS webhook_integrations;
