-- +goose Up
CREATE TABLE subscription_plans (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price_kopeks BIGINT NOT NULL DEFAULT 0,
    billing_period_days INTEGER NOT NULL DEFAULT 30,
    max_monitors INTEGER NOT NULL,
    min_check_interval_seconds INTEGER NOT NULL,
    max_alerts_per_day INTEGER NOT NULL,
    features JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscription_plans_active ON subscription_plans(is_active);

-- Вставка дефолтных планов
INSERT INTO subscription_plans (id, name, description, price_kopeks, billing_period_days, max_monitors, min_check_interval_seconds, max_alerts_per_day, features) VALUES
('TIER_FREE', 'Free', 'Бесплатный тариф для начинающих', 0, 30, 5, 300, 10, '[{"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},{"id":"email_alerts","name":"Email alerts","description":"Email notifications"}]'),
('TIER_STARTER', 'Starter', 'Идеально для небольших проектов', 29900, 30, 25, 60, 100, '[{"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},{"id":"email_alerts","name":"Email alerts","description":"Email notifications"},{"id":"sms_alerts","name":"SMS alerts","description":"SMS notifications"},{"id":"24h_retention","name":"24h retention","description":"24 hours data retention"}]'),
('TIER_PROFESSIONAL', 'Professional', 'Для растущих бизнесов', 99900, 30, 100, 30, 500, '[{"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},{"id":"email_alerts","name":"Email alerts","description":"Email notifications"},{"id":"sms_alerts","name":"SMS alerts","description":"SMS notifications"},{"id":"webhook_integrations","name":"Webhook integrations","description":"Custom webhook integrations"},{"id":"custom_domains","name":"Custom domains","description":"Custom domain monitoring"},{"id":"90d_retention","name":"90d retention","description":"90 days data retention"}]'),
('TIER_BUSINESS', 'Business', 'Масштабируемые решения', 299900, 30, -1, 10, -1, '[{"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},{"id":"email_alerts","name":"Email alerts","description":"Email notifications"},{"id":"sms_alerts","name":"SMS alerts","description":"SMS notifications"},{"id":"webhook_integrations","name":"Webhook integrations","description":"Custom webhook integrations"},{"id":"custom_domains","name":"Custom domains","description":"Custom domain monitoring"},{"id":"priority_support","name":"Priority support","description":"Priority support"},{"id":"unlimited_monitors","name":"Unlimited monitors","description":"Unlimited number of monitors"},{"id":"sla_guarantee","name":"SLA guarantee","description":"SLA guarantee"},{"id":"1y_retention","name":"1y retention","description":"1 year data retention"}]');

-- +goose Down
DROP INDEX IF EXISTS idx_subscription_plans_active;
DROP TABLE IF EXISTS subscription_plans;
