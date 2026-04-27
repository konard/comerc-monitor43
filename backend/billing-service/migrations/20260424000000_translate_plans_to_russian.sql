-- +goose Up
-- Перевод названий тарифов и features на русский язык.
-- Идемпотентно: повторный запуск перезапишет теми же значениями.

UPDATE subscription_plans
SET name = 'Бесплатный',
    description = 'Бесплатный тариф для начинающих',
    features = '[
        {"id":"basic_monitoring","name":"Базовый мониторинг","description":"Базовый HTTP-мониторинг"},
        {"id":"email_alerts","name":"Email-уведомления","description":"Уведомления по электронной почте"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_FREE';

UPDATE subscription_plans
SET name = 'Стартовый',
    description = 'Идеально для небольших проектов',
    features = '[
        {"id":"basic_monitoring","name":"Базовый мониторинг","description":"Базовый HTTP-мониторинг"},
        {"id":"email_alerts","name":"Email-уведомления","description":"Уведомления по электронной почте"},
        {"id":"sms_alerts","name":"SMS-уведомления","description":"Уведомления по SMS"},
        {"id":"24h_retention","name":"Хранение 24 часа","description":"Хранение данных 24 часа"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_STARTER';

UPDATE subscription_plans
SET name = 'Профессиональный',
    description = 'Для растущих бизнесов',
    features = '[
        {"id":"basic_monitoring","name":"Базовый мониторинг","description":"Базовый HTTP-мониторинг"},
        {"id":"email_alerts","name":"Email-уведомления","description":"Уведомления по электронной почте"},
        {"id":"sms_alerts","name":"SMS-уведомления","description":"Уведомления по SMS"},
        {"id":"webhook_integrations","name":"Webhook-интеграции","description":"Произвольные webhook-интеграции"},
        {"id":"custom_domains","name":"Свои домены","description":"Мониторинг своих доменов"},
        {"id":"90d_retention","name":"Хранение 90 дней","description":"Хранение данных 90 дней"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_PROFESSIONAL';

UPDATE subscription_plans
SET name = 'Бизнес',
    description = 'Масштабируемые решения',
    features = '[
        {"id":"basic_monitoring","name":"Базовый мониторинг","description":"Базовый HTTP-мониторинг"},
        {"id":"email_alerts","name":"Email-уведомления","description":"Уведомления по электронной почте"},
        {"id":"sms_alerts","name":"SMS-уведомления","description":"Уведомления по SMS"},
        {"id":"webhook_integrations","name":"Webhook-интеграции","description":"Произвольные webhook-интеграции"},
        {"id":"custom_domains","name":"Свои домены","description":"Мониторинг своих доменов"},
        {"id":"priority_support","name":"Приоритетная поддержка","description":"Приоритетная техническая поддержка"},
        {"id":"unlimited_monitors","name":"Безлимит мониторов","description":"Неограниченное число мониторов"},
        {"id":"sla_guarantee","name":"SLA гарантия","description":"Гарантия SLA"},
        {"id":"1y_retention","name":"Хранение 1 год","description":"Хранение данных 1 год"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_BUSINESS';

-- +goose Down
-- Откат к английским названиям.

UPDATE subscription_plans
SET name = 'Free',
    description = 'Бесплатный тариф для начинающих',
    features = '[
        {"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},
        {"id":"email_alerts","name":"Email alerts","description":"Email notifications"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_FREE';

UPDATE subscription_plans
SET name = 'Starter',
    description = 'Идеально для небольших проектов',
    features = '[
        {"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},
        {"id":"email_alerts","name":"Email alerts","description":"Email notifications"},
        {"id":"sms_alerts","name":"SMS alerts","description":"SMS notifications"},
        {"id":"24h_retention","name":"24h retention","description":"24 hours data retention"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_STARTER';

UPDATE subscription_plans
SET name = 'Professional',
    description = 'Для растущих бизнесов',
    features = '[
        {"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},
        {"id":"email_alerts","name":"Email alerts","description":"Email notifications"},
        {"id":"sms_alerts","name":"SMS alerts","description":"SMS notifications"},
        {"id":"webhook_integrations","name":"Webhook integrations","description":"Custom webhook integrations"},
        {"id":"custom_domains","name":"Custom domains","description":"Custom domain monitoring"},
        {"id":"90d_retention","name":"90d retention","description":"90 days data retention"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_PROFESSIONAL';

UPDATE subscription_plans
SET name = 'Business',
    description = 'Масштабируемые решения',
    features = '[
        {"id":"basic_monitoring","name":"Basic monitoring","description":"Basic HTTP monitoring"},
        {"id":"email_alerts","name":"Email alerts","description":"Email notifications"},
        {"id":"sms_alerts","name":"SMS alerts","description":"SMS notifications"},
        {"id":"webhook_integrations","name":"Webhook integrations","description":"Custom webhook integrations"},
        {"id":"custom_domains","name":"Custom domains","description":"Custom domain monitoring"},
        {"id":"priority_support","name":"Priority support","description":"Priority support"},
        {"id":"unlimited_monitors","name":"Unlimited monitors","description":"Unlimited number of monitors"},
        {"id":"sla_guarantee","name":"SLA guarantee","description":"SLA guarantee"},
        {"id":"1y_retention","name":"1y retention","description":"1 year data retention"}
    ]'::jsonb,
    updated_at = NOW()
WHERE id = 'TIER_BUSINESS';
