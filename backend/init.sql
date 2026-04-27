-- Инициализация дополнительных баз данных для локальной разработки.
-- Выполняется автоматически при первом создании volume.
-- Основная БД monitor создаётся через POSTGRES_DB в docker-compose.yml.

CREATE DATABASE billing;
CREATE DATABASE dashboard;
CREATE DATABASE notification_templates;
CREATE DATABASE integration_service;

-- Даём пользователю monitor доступ ко всем БД и их public-схемам
GRANT ALL PRIVILEGES ON DATABASE dashboard TO monitor;
GRANT ALL PRIVILEGES ON DATABASE notification_templates TO monitor;
GRANT ALL PRIVILEGES ON DATABASE integration_service TO monitor;

\c dashboard
GRANT ALL ON SCHEMA public TO monitor;

\c notification_templates
GRANT ALL ON SCHEMA public TO monitor;

\c integration_service
GRANT ALL ON SCHEMA public TO monitor;

\c monitor

-- Пользователи, специфичные для сервисов
CREATE USER billing WITH PASSWORD 'billing_secret';
GRANT ALL PRIVILEGES ON DATABASE billing TO billing;
\c billing
GRANT ALL ON SCHEMA public TO billing;

\c monitor
CREATE USER scheduler WITH PASSWORD 'scheduler';
GRANT ALL PRIVILEGES ON DATABASE monitor TO scheduler;
GRANT ALL ON SCHEMA public TO scheduler;
